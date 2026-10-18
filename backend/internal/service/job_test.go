package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/alv67/peculium/internal/model"
	"github.com/alv67/peculium/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// fakeJobRepo is an in-memory stand-in for repository.JobRepository that
// mirrors the queue semantics the runner and service rely on.
type fakeJobRepo struct {
	jobs []*model.Job

	lastListLimit  int
	lastListOffset int

	// openForAsset is the canned answer for FindOpenForAsset.
	openForAsset *model.Job
}

var _ repository.JobRepository = (*fakeJobRepo)(nil)

func (f *fakeJobRepo) Enqueue(ctx context.Context, job *model.Job) (*model.Job, error) {
	// Mirrors the repository-side dedup backed by uq_jobs_one_open_per_target:
	// an already-open job with the same type and target is returned instead
	// of stacking a duplicate.
	for _, j := range f.jobs {
		if j.Type == job.Type && j.TargetType == job.TargetType && sameTargetID(j.TargetID, job.TargetID) &&
			(j.Status == model.JobStatusQueued || j.Status == model.JobStatusRunning) {
			return j, nil
		}
	}
	queued := &model.Job{
		ID: uuid.New(), Type: job.Type, TargetType: job.TargetType, TargetID: job.TargetID,
		Status: model.JobStatusQueued, Total: job.Total, RequestedBy: job.RequestedBy,
		CreatedAt: time.Now(),
	}
	f.jobs = append(f.jobs, queued)
	return queued, nil
}

// sameTargetID compares nullable polymorphic target ids (NULL == NULL).
func sameTargetID(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func (f *fakeJobRepo) ClaimNext(ctx context.Context) (*model.Job, error) {
	var oldest *model.Job
	for _, j := range f.jobs {
		if j.Status == model.JobStatusQueued && (oldest == nil || j.CreatedAt.Before(oldest.CreatedAt)) {
			oldest = j
		}
	}
	if oldest == nil {
		return nil, nil
	}
	now := time.Now()
	oldest.Status = model.JobStatusRunning
	oldest.StartedAt = &now
	return oldest, nil
}

func (f *fakeJobRepo) find(id uuid.UUID) *model.Job {
	for _, j := range f.jobs {
		if j.ID == id {
			return j
		}
	}
	return nil
}

func (f *fakeJobRepo) UpdateProgress(ctx context.Context, id uuid.UUID, processed, total int) error {
	if j := f.find(id); j != nil {
		j.Processed, j.Total = processed, total
	}
	return nil
}

func (f *fakeJobRepo) SetCheckpoint(ctx context.Context, id uuid.UUID, checkpoint []byte) error {
	if j := f.find(id); j != nil {
		j.Checkpoint = checkpoint
	}
	return nil
}

func (f *fakeJobRepo) Finish(ctx context.Context, id uuid.UUID, status string, errMsg string) error {
	j := f.find(id)
	if j == nil {
		return pgx.ErrNoRows
	}
	now := time.Now()
	j.Status, j.Error, j.FinishedAt = status, errMsg, &now
	return nil
}

func (f *fakeJobRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Job, error) {
	if j := f.find(id); j != nil {
		return j, nil
	}
	return nil, pgx.ErrNoRows
}

func (f *fakeJobRepo) List(ctx context.Context, limit, offset int) ([]*model.Job, error) {
	f.lastListLimit, f.lastListOffset = limit, offset
	return f.jobs, nil
}

func (f *fakeJobRepo) FindOpenForAsset(ctx context.Context, assetID uuid.UUID) (*model.Job, error) {
	return f.openForAsset, nil
}

func (f *fakeJobRepo) DeleteBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	var kept []*model.Job
	var deleted int64
	for _, j := range f.jobs {
		at := j.CreatedAt
		if j.FinishedAt != nil {
			at = *j.FinishedAt
		}
		if !at.IsZero() && at.Before(cutoff) {
			deleted++
			continue
		}
		kept = append(kept, j)
	}
	f.jobs = kept
	return deleted, nil
}

func newFakeJobSvc() (*JobService, *fakeJobRepo) {
	repo := &fakeJobRepo{}
	return NewJobService(&repository.Repository{Job: repo}), repo
}

func TestJobEnqueueClaimProgressFinish(t *testing.T) {
	svc, repo := newFakeJobSvc()
	ctx := context.Background()

	target := uuid.New()
	queued, err := svc.Enqueue(ctx, &model.Job{Type: model.JobTypeHistoryBackfill, TargetType: model.JobTargetAsset, TargetID: &target, Total: 5})
	if err != nil {
		t.Fatal(err)
	}
	if queued.Status != model.JobStatusQueued || queued.ID == uuid.Nil {
		t.Fatalf("queued job = %+v, want status queued with server-assigned id", queued)
	}

	runner := NewJobRunner(repo, time.Minute)
	runner.Register(model.JobTypeHistoryBackfill, func(ctx context.Context, job *model.Job) (string, error) {
		if job.TargetID == nil || *job.TargetID != target {
			t.Errorf("executor got target_id %v, want %v", job.TargetID, target)
		}
		if err := repo.UpdateProgress(ctx, job.ID, 3, 5); err != nil {
			return "", err
		}
		return "", nil // empty means done
	})

	ran, err := runner.RunOnce(ctx)
	if err != nil || !ran {
		t.Fatalf("RunOnce = (%v, %v), want (true, nil)", ran, err)
	}

	job, err := svc.Get(ctx, queued.ID)
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != model.JobStatusDone || job.Error != "" {
		t.Fatalf("status = %q error = %q, want done with no error", job.Status, job.Error)
	}
	if job.Processed != 3 || job.Total != 5 {
		t.Fatalf("progress = %d/%d, want 3/5", job.Processed, job.Total)
	}
	if job.StartedAt == nil || job.FinishedAt == nil {
		t.Fatalf("timestamps not stamped: started=%v finished=%v", job.StartedAt, job.FinishedAt)
	}

	if ran, err := runner.RunOnce(ctx); ran || err != nil {
		t.Fatalf("RunOnce on empty queue = (%v, %v), want (false, nil)", ran, err)
	}
}

func TestJobRunnerStatusClassification(t *testing.T) {
	boom := errors.New("yahoo exploded")

	tcs := []struct {
		name     string
		jobType  string
		exec     JobExecutor
		wantStat string
		wantErr  string
	}{
		{
			name: "executor error fails job", jobType: "a",
			exec:     func(context.Context, *model.Job) (string, error) { return model.JobStatusPartial, boom },
			wantStat: model.JobStatusFailed, wantErr: boom.Error(),
		},
		{
			name: "partial status kept", jobType: "b",
			exec:     func(context.Context, *model.Job) (string, error) { return model.JobStatusPartial, nil },
			wantStat: model.JobStatusPartial,
		},
		{
			name: "unknown type fails", jobType: "unknown", exec: nil,
			wantStat: model.JobStatusFailed, wantErr: "no executor for type unknown",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo := newFakeJobSvc()
			ctx := context.Background()
			if _, err := svc.Enqueue(ctx, &model.Job{Type: tc.jobType}); err != nil {
				t.Fatal(err)
			}
			runner := NewJobRunner(repo, time.Minute)
			if tc.exec != nil {
				runner.Register(tc.jobType, tc.exec)
			}
			if ran, err := runner.RunOnce(ctx); !ran || err != nil {
				t.Fatalf("RunOnce = (%v, %v), want (true, nil)", ran, err)
			}

			jobs, err := svc.List(ctx, 0, 0)
			if err != nil {
				t.Fatal(err)
			}
			if len(jobs) != 1 {
				t.Fatalf("got %d jobs, want 1", len(jobs))
			}
			if jobs[0].Status != tc.wantStat || jobs[0].Error != tc.wantErr {
				t.Fatalf("status/error = %q/%q, want %q/%q", jobs[0].Status, jobs[0].Error, tc.wantStat, tc.wantErr)
			}
			if jobs[0].FinishedAt == nil {
				t.Fatal("finished_at not stamped")
			}
		})
	}
}

func TestJobRunnerTimeoutStillRecordsFailure(t *testing.T) {
	svc, repo := newFakeJobSvc()
	ctx := context.Background()
	queued, err := svc.Enqueue(ctx, &model.Job{Type: "hang"})
	if err != nil {
		t.Fatal(err)
	}

	runner := NewJobRunner(repo, 50*time.Millisecond)
	runner.Register("hang", func(jobCtx context.Context, job *model.Job) (string, error) {
		<-jobCtx.Done()
		return "", jobCtx.Err()
	})

	if ran, err := runner.RunOnce(ctx); !ran || err != nil {
		t.Fatalf("RunOnce = (%v, %v), want (true, nil)", ran, err)
	}
	// The executor blew its deadline, yet the terminal state must still persist.
	job, err := svc.Get(ctx, queued.ID)
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != model.JobStatusFailed || job.Error != context.DeadlineExceeded.Error() {
		t.Fatalf("status/error = %q/%q, want failed with deadline exceeded", job.Status, job.Error)
	}
}

func TestJobServicePaginationAndNotFound(t *testing.T) {
	svc, repo := newFakeJobSvc()
	ctx := context.Background()

	if _, err := svc.Get(ctx, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get unknown = %v, want ErrNotFound", err)
	}

	if _, err := svc.List(ctx, -5, -1); err != nil {
		t.Fatal(err)
	}
	if repo.lastListLimit != 50 || repo.lastListOffset != 0 {
		t.Fatalf("clamped page = %d/%d, want default 50/0", repo.lastListLimit, repo.lastListOffset)
	}
}

func TestJobEnqueueDedupsOpenTarget(t *testing.T) {
	svc, repo := newFakeJobSvc()
	ctx := context.Background()
	target := uuid.New()

	first, err := svc.Enqueue(ctx, &model.Job{Type: model.JobTypeHistoryBackfill, TargetType: model.JobTargetAsset, TargetID: &target})
	if err != nil {
		t.Fatal(err)
	}
	again, err := svc.Enqueue(ctx, &model.Job{Type: model.JobTypeHistoryBackfill, TargetType: model.JobTargetAsset, TargetID: &target})
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != first.ID || len(repo.jobs) != 1 {
		t.Fatalf("dedup miss: ids %v/%v, %d rows", first.ID, again.ID, len(repo.jobs))
	}

	other := uuid.New()
	if _, err := svc.Enqueue(ctx, &model.Job{Type: model.JobTypeHistoryBackfill, TargetType: model.JobTargetAsset, TargetID: &other}); err != nil {
		t.Fatal(err)
	}
	if len(repo.jobs) != 2 {
		t.Fatalf("rows = %d, want 2 (different asset target)", len(repo.jobs))
	}

	// NULL-target (global) jobs dedup against each other too.
	g1, err := svc.Enqueue(ctx, &model.Job{Type: model.JobTypeMetaBackfill, TargetType: model.JobTargetGlobal})
	if err != nil {
		t.Fatal(err)
	}
	g2, err := svc.Enqueue(ctx, &model.Job{Type: model.JobTypeMetaBackfill, TargetType: model.JobTargetGlobal})
	if err != nil {
		t.Fatal(err)
	}
	if g2.ID != g1.ID {
		t.Fatalf("global dedup miss: %v != %v", g2.ID, g1.ID)
	}

	// A finished job frees the slot for the next request.
	if err := repo.Finish(ctx, first.ID, model.JobStatusDone, ""); err != nil {
		t.Fatal(err)
	}
	fresh, err := svc.Enqueue(ctx, &model.Job{Type: model.JobTypeHistoryBackfill, TargetType: model.JobTargetAsset, TargetID: &target})
	if err != nil {
		t.Fatal(err)
	}
	if fresh.ID == first.ID {
		t.Fatal("finished job must not block a new enqueue")
	}
}

func TestJobDeleteBeforePrunesAnyOldStatus(t *testing.T) {
	svc, repo := newFakeJobSvc()
	ctx := context.Background()
	older := time.Now().Add(-91 * 24 * time.Hour)
	recent := time.Now().Add(-time.Hour)
	cutoff := time.Now().Add(-90 * 24 * time.Hour)

	seed := func(status string, finished *time.Time) {
		j := &model.Job{ID: uuid.New(), Status: status, FinishedAt: finished}
		repo.jobs = append(repo.jobs, j)
	}
	seed(model.JobStatusDone, &older)
	seed(model.JobStatusFailed, &older)
	seed(model.JobStatusDone, &recent)
	seed(model.JobStatusQueued, nil)

	n, err := svc.DeleteBefore(ctx, cutoff)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("deleted = %d, want 2", n)
	}
	if len(repo.jobs) != 2 {
		t.Fatalf("kept %d jobs, want 2 (young done and never-timestamped queued survive)", len(repo.jobs))
	}
}

// fakeJobHealth collects the lifecycle events a runner emits.
type fakeJobHealth struct {
	events []*model.HealthEvent
}

func (f *fakeJobHealth) RecordEvent(ctx context.Context, ev *model.HealthEvent) error {
	f.events = append(f.events, ev)
	return nil
}

func TestJobRunnerEmitsLifecycleEvents(t *testing.T) {
	ctx := context.Background()

	tcs := []struct {
		name       string
		status     string
		execErr    error
		wantCode   string
		wantStatus string
	}{
		{"done", model.JobStatusDone, nil, "job_completed", "success"},
		{"partial", model.JobStatusPartial, nil, "job_partial", model.JobStatusPartial},
		{"failed", "", errors.New("yahoo exploded"), "job_failed", "failure"},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			svc, repo := newFakeJobSvc()
			target := uuid.New()
			job, _ := svc.Enqueue(ctx, &model.Job{Type: "work", TargetType: model.JobTargetAsset, TargetID: &target, Total: 3})
			rec := &fakeJobHealth{}
			runner := NewJobRunner(repo, time.Minute)
			runner.WithHealth(rec)
			runner.Register("work", func(c context.Context, j *model.Job) (string, error) {
				_ = repo.UpdateProgress(c, j.ID, 2, 3)
				return tc.status, tc.execErr
			})

			if ran, err := runner.RunOnce(ctx); !ran || err != nil {
				t.Fatalf("RunOnce = (%v, %v)", ran, err)
			}
			if len(rec.events) != 2 {
				t.Fatalf("events = %d, want 2 (started + terminal)", len(rec.events))
			}
			started, terminal := rec.events[0], rec.events[1]
			if started.Code != "job_started" || started.Status != model.JobStatusRunning {
				t.Fatalf("started event = %q/%q", started.Code, started.Status)
			}
			if started.JobID == nil || *started.JobID != job.ID || started.AssetID == nil || *started.AssetID != target {
				t.Fatalf("started event job/asset attribution = %+v", started)
			}
			if started.Message == "" || !strings.Contains(started.Message, "work") {
				t.Fatalf("started event = %+v, want type in message", started)
			}
			if terminal.Code != tc.wantCode || terminal.Status != tc.wantStatus {
				t.Fatalf("terminal = %q/%q, want %q/%q", terminal.Code, terminal.Status, tc.wantCode, tc.wantStatus)
			}
			if terminal.JobID == nil || *terminal.JobID != job.ID {
				t.Fatalf("terminal job id = %v", terminal.JobID)
			}
			if !strings.Contains(terminal.Message, "2/3") {
				t.Fatalf("terminal message = %q, want processed/total counts", terminal.Message)
			}
			if tc.execErr != nil && !strings.Contains(terminal.Message, tc.execErr.Error()) {
				t.Fatalf("terminal message = %q, want it to carry the error", terminal.Message)
			}
		})
	}
}

func TestJobRunnerTerminalEventSurvivesJobTimeout(t *testing.T) {
	svc, repo := newFakeJobSvc()
	ctx := context.Background()
	if _, err := svc.Enqueue(ctx, &model.Job{Type: "hang"}); err != nil {
		t.Fatal(err)
	}

	rec := &fakeJobHealth{}
	runner := NewJobRunner(repo, 20*time.Millisecond)
	runner.WithHealth(rec)
	runner.Register("hang", func(jobCtx context.Context, _ *model.Job) (string, error) {
		<-jobCtx.Done()
		return "", jobCtx.Err()
	})

	if ran, err := runner.RunOnce(ctx); !ran || err != nil {
		t.Fatalf("RunOnce = (%v, %v)", ran, err)
	}
	// jobCtx is dead by now; the terminal event must still have been written.
	terminal := rec.events[len(rec.events)-1]
	if terminal.Code != "job_failed" || terminal.Status != "failure" {
		t.Fatalf("terminal = %q/%q, want job_failed/failure", terminal.Code, terminal.Status)
	}
	if !strings.Contains(terminal.Message, context.DeadlineExceeded.Error()) {
		t.Fatalf("terminal message = %q, want the deadline error", terminal.Message)
	}
}

func TestJobRunnerUnknownTypeEmitsFailedLifecycle(t *testing.T) {
	svc, repo := newFakeJobSvc()
	ctx := context.Background()
	if _, err := svc.Enqueue(ctx, &model.Job{Type: "ghost"}); err != nil {
		t.Fatal(err)
	}
	rec := &fakeJobHealth{}
	runner := NewJobRunner(repo, time.Minute)
	runner.WithHealth(rec)
	if ran, err := runner.RunOnce(ctx); !ran || err != nil {
		t.Fatalf("RunOnce = (%v, %v)", ran, err)
	}
	if len(rec.events) != 2 || rec.events[1].Code != "job_failed" {
		t.Fatalf("events = %+v, want started + job_failed", rec.events)
	}
}

func TestJobGetDecoratesDurationAndSummary(t *testing.T) {
	ctx := context.Background()
	jr := &fakeJobRepo{}
	health := &fakeHealthRepo{countsByJobs: map[uuid.UUID]*model.JobHealthCounts{}}
	svc := NewJobService(&repository.Repository{Job: jr, Health: health})

	id := uuid.New()
	started := time.Now().Add(-2500 * time.Millisecond)
	finished := started.Add(2500 * time.Millisecond)
	jr.jobs = append(jr.jobs,
		&model.Job{ID: id, Status: model.JobStatusDone, StartedAt: &started, FinishedAt: &finished},
		&model.Job{ID: uuid.New(), Status: model.JobStatusQueued},
	)
	health.countsByJobs[id] = &model.JobHealthCounts{OK: 7, Failed: 2}

	job, err := svc.Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if job.DurationMs != 2500 {
		t.Fatalf("duration_ms = %d, want 2500", job.DurationMs)
	}
	if job.Summary == nil || job.Summary.OK != 7 || job.Summary.Failed != 2 {
		t.Fatalf("summary = %+v, want ok=7 failed=2", job.Summary)
	}
	if len(health.countsIDs) != 1 || health.countsIDs[0] != id {
		t.Fatalf("counts queried for %v, want only the started job", health.countsIDs)
	}

	// A never-started job derives no duration and no summary.
	list, err := svc.List(ctx, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	queued := list[1]
	if queued.DurationMs != 0 || queued.Summary != nil {
		t.Fatalf("queued job decorated anyway: %+v", queued)
	}
}

func TestJobListResolvesTargetLabel(t *testing.T) {
	ctx := context.Background()
	asset := &model.Asset{ID: uuid.New(), Ticker: "AAPL"}
	pf := &model.Portfolio{ID: uuid.New(), Name: "Mediolanum"}
	repo := &fakeJobRepo{}
	svc := NewJobService(&repository.Repository{
		Job:       repo,
		Asset:     &fakeAssetRepo{assets: []*model.Asset{asset}},
		Portfolio: &fakePortfolioRepo{portfolio: pf},
	})
	if _, err := svc.Enqueue(ctx, &model.Job{Type: model.JobTypeHistoryBackfill, TargetType: model.JobTargetAsset, TargetID: &asset.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Enqueue(ctx, &model.Job{Type: model.JobTypePriceRefresh, TargetType: model.JobTargetPortfolio, TargetID: &pf.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Enqueue(ctx, &model.Job{Type: model.JobTypeAssetSync, TargetType: model.JobTargetGlobal}); err != nil {
		t.Fatal(err)
	}

	jobs, err := svc.List(ctx, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	byType := map[string]string{}
	for _, j := range jobs {
		byType[j.Type] = j.TargetLabel
	}
	if byType[model.JobTypeHistoryBackfill] != "AAPL" {
		t.Fatalf("asset label = %q, want AAPL", byType[model.JobTypeHistoryBackfill])
	}
	if byType[model.JobTypePriceRefresh] != "Mediolanum" {
		t.Fatalf("portfolio label = %q, want Mediolanum", byType[model.JobTypePriceRefresh])
	}
	if byType[model.JobTypeAssetSync] != "" {
		t.Fatalf("global label = %q, want empty", byType[model.JobTypeAssetSync])
	}
}
