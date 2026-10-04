package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/alv67/peculium/internal/model"
	"github.com/alv67/peculium/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

const (
	// DefaultJobTimeout bounds a single job's external work. It is a whole
	// order of magnitude larger than the server's 30s request timeout because
	// the worker runs jobs on their own detached context, not a caller's.
	DefaultJobTimeout = 15 * time.Minute
	// DefaultJobPollInterval is how often the worker checks for new work when
	// the queue is idle.
	DefaultJobPollInterval = 3 * time.Second
	// JobRetention is how long finished (done) jobs are kept before pruning.
	JobRetention = 30 * 24 * time.Hour
	// jobEventTimeout bounds a health write made outside any request/job
	// context: the lifecycle event must land even when jobCtx already died.
	jobEventTimeout = 3 * time.Second
)

// JobService is the thin business layer over the Postgres job queue. Phase 1
// exposes only the primitives the worker and read-only listings need; wiring
// to HTTP routes is deferred to phase 2.
type JobService struct {
	repos *repository.Repository
}

func NewJobService(repos *repository.Repository) *JobService {
	return &JobService{repos: repos}
}

func (s *JobService) Enqueue(ctx context.Context, job *model.Job) (*model.Job, error) {
	return s.repos.Job.Enqueue(ctx, job)
}

func (s *JobService) List(ctx context.Context, limit, offset int) ([]*model.Job, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	jobs, err := s.repos.Job.List(ctx, limit, offset)
	if err == nil {
		s.decorate(ctx, jobs)
	}
	return jobs, err
}

func (s *JobService) Get(ctx context.Context, id uuid.UUID) (*model.Job, error) {
	job, err := s.repos.Job.GetByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err == nil && job != nil {
		s.decorate(ctx, []*model.Job{job})
	}
	return job, err
}

// decorate fills the read-only derived job fields (duration_ms, summary) that
// no DB column backs. The duration is finished-started once both timestamps
// exist; the health rollup is queried for every claimed job (running or
// terminal) in one grouped call. A rollup failure only drops the summary:
// the queue row itself must always be served.
func (s *JobService) decorate(ctx context.Context, jobs []*model.Job) {
	ids := make([]uuid.UUID, 0, len(jobs))
	for _, j := range jobs {
		if j.StartedAt != nil && j.FinishedAt != nil {
			j.DurationMs = j.FinishedAt.Sub(*j.StartedAt).Milliseconds()
		}
		if j.StartedAt != nil {
			ids = append(ids, j.ID)
		}
	}
	if len(ids) == 0 || s.repos.Health == nil {
		return
	}
	counts, err := s.repos.Health.CountsByJobs(ctx, ids)
	if err != nil {
		log.Warn().Err(err).Msg("job health summary query failed")
		return
	}
	for _, j := range jobs {
		if c, ok := counts[j.ID]; ok {
			j.Summary = c
		}
	}
}

// DeleteFinishedBefore prunes done jobs finished before cutoff, returning how
// many rows were removed. Called by the worker on startup and in its hourly
// tick so the queue table stays bounded.
func (s *JobService) DeleteFinishedBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	return s.repos.Job.DeleteFinishedBefore(ctx, cutoff)
}

// jobIDKey tags a context with the job being executed. HealthService reads it
// so every event recorded while a job runs is attributed to its queue row.
type jobIDKey struct{}

// WithJobID binds the job id to ctx for health-event attribution downstream.
func WithJobID(ctx context.Context, jobID uuid.UUID) context.Context {
	return context.WithValue(ctx, jobIDKey{}, jobID)
}

// JobIDFromContext returns the job id bound by WithJobID, if any.
func JobIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	id, ok := ctx.Value(jobIDKey{}).(uuid.UUID)
	return id, ok
}

// JobExecutor runs a single claimed job. It reports its terminal status via
// the returned string (done/partial/failed); an error is recorded on the job.
// A returned status of "" is treated as done. The context passed in is the
// job's own detached context, not the caller's.
type JobExecutor func(ctx context.Context, job *model.Job) (string, error)

// healthRecorder is the slice of the health service the runner consumes for
// job lifecycle events. Keeping it an interface (and optional) means callers
// and tests that never wired health tracking keep working: nil emits nothing.
type healthRecorder interface {
	RecordEvent(ctx context.Context, event *model.HealthEvent) error
}

// JobRunner claims queued jobs and dispatches them to a registered executor by
// type. It is consumed by the worker's polling goroutine.
type JobRunner struct {
	repo      repository.JobRepository
	executors map[string]JobExecutor
	timeout   time.Duration
	health    healthRecorder
}

func NewJobRunner(repo repository.JobRepository, timeout time.Duration) *JobRunner {
	return &JobRunner{repo: repo, executors: make(map[string]JobExecutor), timeout: timeout}
}

// WithHealth attaches the health recorder used to emit job lifecycle events
// (job_started plus one terminal job_completed/job_failed/job_partial).
func (r *JobRunner) WithHealth(hr healthRecorder) { r.health = hr }

// emit writes one job lifecycle event. The context is always a fresh,
// short-lived one because the job's own context may already be expired when
// the terminal event is recorded. Lifecycle rows carry event_type "job" so
// the health summary can keep them out of the provider-interaction counts.
func (r *JobRunner) emit(job *model.Job, status, code, message string, since time.Time) {
	if r.health == nil {
		return
	}
	ev := &model.HealthEvent{
		ID:         uuid.New(),
		JobID:      &job.ID,
		EventType:  "job",
		Status:     status,
		Code:       code,
		Message:    message,
		DurationMs: int(time.Since(since).Milliseconds()),
		CreatedAt:  time.Now().UTC(),
	}
	if job.TargetType == model.JobTargetAsset && job.TargetID != nil {
		ev.AssetID = job.TargetID
	}
	ctx, cancel := context.WithTimeout(context.Background(), jobEventTimeout)
	defer cancel()
	if err := r.health.RecordEvent(ctx, ev); err != nil {
		log.Warn().Err(err).Str("job_id", job.ID.String()).Msg("failed to record job lifecycle event")
	}
}

// Register binds an executor to a job type, overwriting any previous one.
func (r *JobRunner) Register(jobType string, exec JobExecutor) {
	r.executors[jobType] = exec
}

// RunOnce claims and executes a single job, returning ran=true when it
// processed something. When the queue is empty it returns ran=false with no
// error. Unknown job types are failed so they do not wedge the queue.
func (r *JobRunner) RunOnce(ctx context.Context) (ran bool, err error) {
	job, err := r.repo.ClaimNext(ctx)
	if err != nil || job == nil {
		return false, err
	}

	start := time.Now()
	target := job.TargetType
	if job.TargetID != nil {
		target = fmt.Sprintf("%s %s", job.TargetType, job.TargetID)
	}
	r.emit(job, model.JobStatusRunning, "job_started", fmt.Sprintf("%s job started, target %s", job.Type, target), start)

	exec, ok := r.executors[job.Type]
	if !ok {
		log.Warn().Str("job_id", job.ID.String()).Str("type", job.Type).Msg("no executor registered for job type")
		msg := "no executor for type " + job.Type
		err := r.repo.Finish(context.Background(), job.ID, model.JobStatusFailed, msg)
		r.emit(job, model.JobStatusFailed, "job_failed", msg, start)
		return true, err
	}

	// Each job runs on its own context with a generous timeout, detached from
	// the caller so a long fetch is not cut by the server's request deadline.
	// The job id is bound to it so health events recorded deep inside the
	// fetchers are attributed to this job.
	// ponytail: no checkpoint resume yet; add it when an executor needs to
	// survive worker restarts mid-run.
	jobCtx, cancel := context.WithTimeout(WithJobID(context.Background(), job.ID), r.timeout)
	defer cancel()

	status, execErr := exec(jobCtx, job)
	if status == "" {
		status = model.JobStatusDone
	}

	finishStatus := status
	finishErr := ""
	eventStatus, eventCode := "success", "job_completed"
	switch finishStatus {
	case model.JobStatusPartial:
		eventStatus, eventCode = model.JobStatusPartial, "job_partial"
	}
	if execErr != nil {
		finishStatus = model.JobStatusFailed
		finishErr = execErr.Error()
		eventStatus, eventCode = "failure", "job_failed"
	}

	// Finish on a fresh context: jobCtx may already be expired/cancelled but
	// the terminal state must still be persisted.
	if err := r.repo.Finish(context.Background(), job.ID, finishStatus, finishErr); err != nil {
		return true, err
	}

	// Re-read to report the final persisted progress in the terminal event.
	if fresh, err := r.repo.GetByID(context.Background(), job.ID); err == nil && fresh != nil {
		job = fresh
	}
	message := fmt.Sprintf("%s finished %d/%d", job.Type, job.Processed, job.Total)
	if finishErr != "" {
		message += ": " + finishErr
	}
	r.emit(job, eventStatus, eventCode, message, start)
	return true, nil
}

// Loop polls for work until ctx is cancelled, sleeping pollInterval when the
// queue is idle so an empty queue does not busy-loop.
func (r *JobRunner) Loop(ctx context.Context, pollInterval time.Duration) {
	t := time.NewTicker(pollInterval)
	defer t.Stop()

	for {
		ran, err := r.RunOnce(ctx)
		if err != nil {
			log.Error().Err(err).Msg("job run failed")
		}
		if !ran {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
			continue
		}
		// Drain promptly but still honor cancellation between jobs.
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}
