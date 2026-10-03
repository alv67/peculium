package repository

import (
	"context"
	"errors"
	"time"

	"github.com/alv67/peculium/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// JobRepository backs the Postgres job queue drained by the worker.
type JobRepository interface {
	Enqueue(ctx context.Context, job *model.Job) (*model.Job, error)
	// ClaimNext atomically marks the oldest queued job as running and returns
	// it, or (nil, nil) when there is nothing to do. FOR UPDATE SKIP LOCKED
	// keeps concurrent consumers from picking the same row.
	ClaimNext(ctx context.Context) (*model.Job, error)
	UpdateProgress(ctx context.Context, id uuid.UUID, processed, total int) error
	SetCheckpoint(ctx context.Context, id uuid.UUID, checkpoint []byte) error
	// Finish moves a running job to its terminal status and stamps finished_at.
	Finish(ctx context.Context, id uuid.UUID, status string, errMsg string) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.Job, error)
	List(ctx context.Context, limit, offset int) ([]*model.Job, error)
	// DeleteFinishedBefore prunes done jobs finished before cutoff. Failed and
	// partial rows are kept so debugging history survives retention.
	DeleteFinishedBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

type jobRepo struct {
	db DBTX
}

func NewJobRepository(db DBTX) JobRepository {
	return &jobRepo{db}
}

const jobColumns = `id, type, target_type, target_id, status, total, processed, checkpoint, error, requested_by, created_at, started_at, finished_at`

func scanJob(row interface{ Scan(dest ...any) error }) (*model.Job, error) {
	j := &model.Job{}
	var checkpoint []byte
	err := row.Scan(&j.ID, &j.Type, &j.TargetType, &j.TargetID, &j.Status, &j.Total, &j.Processed,
		&checkpoint, &j.Error, &j.RequestedBy, &j.CreatedAt, &j.StartedAt, &j.FinishedAt)
	if err != nil {
		return nil, err
	}
	j.Checkpoint = checkpoint
	return j, nil
}

func (r *jobRepo) Enqueue(ctx context.Context, job *model.Job) (*model.Job, error) {
	row := r.db.QueryRow(ctx,
		`INSERT INTO jobs (type, target_type, target_id, total, requested_by)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING `+jobColumns,
		job.Type, job.TargetType, job.TargetID, job.Total, job.RequestedBy)
	return scanJob(row)
}

func (r *jobRepo) ClaimNext(ctx context.Context) (*model.Job, error) {
	row := r.db.QueryRow(ctx,
		`UPDATE jobs SET status = 'running', started_at = NOW()
		 WHERE id = (
			 SELECT id FROM jobs
			 WHERE status = 'queued'
			 ORDER BY created_at, id
			 FOR UPDATE SKIP LOCKED
			 LIMIT 1
		 )
		 RETURNING `+jobColumns)
	job, err := scanJob(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return job, err
}

func (r *jobRepo) UpdateProgress(ctx context.Context, id uuid.UUID, processed, total int) error {
	_, err := r.db.Exec(ctx, `UPDATE jobs SET processed = $2, total = $3 WHERE id = $1`, id, processed, total)
	return err
}

func (r *jobRepo) SetCheckpoint(ctx context.Context, id uuid.UUID, checkpoint []byte) error {
	_, err := r.db.Exec(ctx, `UPDATE jobs SET checkpoint = $2 WHERE id = $1`, id, checkpoint)
	return err
}

func (r *jobRepo) Finish(ctx context.Context, id uuid.UUID, status string, errMsg string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE jobs SET status = $2, error = $3, finished_at = NOW() WHERE id = $1`, id, status, errMsg)
	return err
}

func (r *jobRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Job, error) {
	row := r.db.QueryRow(ctx, `SELECT `+jobColumns+` FROM jobs WHERE id = $1`, id)
	return scanJob(row)
}

func (r *jobRepo) List(ctx context.Context, limit, offset int) ([]*model.Job, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+jobColumns+` FROM jobs ORDER BY created_at DESC, id DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]*model.Job, 0)
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (r *jobRepo) DeleteFinishedBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	tag, err := r.db.Exec(ctx, `DELETE FROM jobs WHERE status = 'done' AND finished_at < $1`, cutoff)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
