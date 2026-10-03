package service

import (
	"context"
	"errors"
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
	return s.repos.Job.List(ctx, limit, offset)
}

func (s *JobService) Get(ctx context.Context, id uuid.UUID) (*model.Job, error) {
	job, err := s.repos.Job.GetByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return job, err
}

// DeleteFinishedBefore prunes done jobs finished before cutoff, returning how
// many rows were removed. Called by the worker on startup and in its hourly
// tick so the queue table stays bounded.
func (s *JobService) DeleteFinishedBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	return s.repos.Job.DeleteFinishedBefore(ctx, cutoff)
}

// JobExecutor runs a single claimed job. It reports its terminal status via
// the returned string (done/partial/failed); an error is recorded on the job.
// A returned status of "" is treated as done. The context passed in is the
// job's own detached context, not the caller's.
type JobExecutor func(ctx context.Context, job *model.Job) (string, error)

// JobRunner claims queued jobs and dispatches them to a registered executor by
// type. It is consumed by the worker's polling goroutine.
type JobRunner struct {
	repo      repository.JobRepository
	executors map[string]JobExecutor
	timeout   time.Duration
}

func NewJobRunner(repo repository.JobRepository, timeout time.Duration) *JobRunner {
	return &JobRunner{repo: repo, executors: make(map[string]JobExecutor), timeout: timeout}
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

	exec, ok := r.executors[job.Type]
	if !ok {
		log.Warn().Str("job_id", job.ID.String()).Str("type", job.Type).Msg("no executor registered for job type")
		return true, r.repo.Finish(context.Background(), job.ID, model.JobStatusFailed, "no executor for type "+job.Type)
	}

	// Each job runs on its own context with a generous timeout, detached from
	// the caller so a long fetch is not cut by the server's request deadline.
	// ponytail: no checkpoint resume yet; add it when an executor needs to
	// survive worker restarts mid-run.
	jobCtx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	status, execErr := exec(jobCtx, job)
	if status == "" {
		status = model.JobStatusDone
	}

	finishStatus := status
	finishErr := ""
	if execErr != nil {
		finishStatus = model.JobStatusFailed
		finishErr = execErr.Error()
	}

	// Finish on a fresh context: jobCtx may already be expired/cancelled but
	// the terminal state must still be persisted.
	return true, r.repo.Finish(context.Background(), job.ID, finishStatus, finishErr)
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
