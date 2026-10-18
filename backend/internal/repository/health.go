package repository

import (
	"context"
	"time"

	"github.com/alv67/peculium/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthRepository interface {
	RecordEvent(ctx context.Context, event *model.HealthEvent) error
	GetEventsPage(ctx context.Context, limit, offset int, eventType, status, jobID string) ([]*model.HealthEvent, error)
	CountEvents(ctx context.Context, eventType, status, jobID string) (int, error)
	GetActivityPage(ctx context.Context, limit, offset int, itemType, status string, since *time.Time) ([]*model.ActivityItem, error)
	CountActivity(ctx context.Context, itemType, status string, since *time.Time) (int, error)
	SummarySince(ctx context.Context, since *time.Time) (*model.HealthSummary, error)
	CountsByJobs(ctx context.Context, jobIDs []uuid.UUID) (map[uuid.UUID]*model.JobHealthCounts, error)
	DeleteStandaloneBefore(ctx context.Context, cutoff time.Time) (int64, error)
	ClearAll(ctx context.Context) (int64, error)
}

type healthRepo struct {
	db *pgxpool.Pool
}

func NewHealthRepository(db *pgxpool.Pool) HealthRepository {
	return &healthRepo{db}
}

func (r *healthRepo) RecordEvent(ctx context.Context, event *model.HealthEvent) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO health_events (id, asset_id, event_type, status, code, message, duration_ms, error_code, created_at, job_id) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		event.ID, event.AssetID, event.EventType, event.Status, event.Code, event.Message, event.DurationMs, event.ErrorCode, event.CreatedAt, event.JobID)
	return err
}

// GetEventsPage returns one page of events, newest first. Empty eventType,
// status or jobID means "no filter" on that dimension, so callers pass the raw
// query parameter through as-is. jobID restricts the page to one job's events
// (the activity list shows only jobless events, so the detail panel asks for
// the other side of that split).
func (r *healthRepo) GetEventsPage(ctx context.Context, limit, offset int, eventType, status, jobID string) ([]*model.HealthEvent, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, asset_id, event_type, status, code, message, duration_ms, error_code, created_at, job_id 
		 FROM health_events 
		 WHERE ($3 = '' OR event_type = $3) AND ($4 = '' OR status = $4)
		   AND ($5 = '' OR job_id::text = $5)
		 ORDER BY created_at DESC 
		 LIMIT $1 OFFSET $2`, limit, offset, eventType, status, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]*model.HealthEvent, 0)
	for rows.Next() {
		e := &model.HealthEvent{}
		var assetID *uuid.UUID
		err := rows.Scan(&e.ID, &assetID, &e.EventType, &e.Status, &e.Code, &e.Message, &e.DurationMs, &e.ErrorCode, &e.CreatedAt, &e.JobID)
		if err != nil {
			return nil, err
		}
		e.AssetID = assetID
		events = append(events, e)
	}
	return events, nil
}

// CountEvents counts the events matching the same optional filters as
// GetEventsPage, so events_total stays right under a filter.
func (r *healthRepo) CountEvents(ctx context.Context, eventType, status, jobID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM health_events 
		 WHERE ($1 = '' OR event_type = $1) AND ($2 = '' OR status = $2)
		   AND ($3 = '' OR job_id::text = $3)`,
		eventType, status, jobID).Scan(&count)
	return count, err
}

// sinceArg turns a nil window into a plain SQL NULL parameter: a typed nil
// pointer is not a driver NULL, and the queries key the "no lower bound" case
// off the parameter being null.
func sinceArg(since *time.Time) any {
	if since == nil {
		return nil
	}
	return *since
}

// activityUnion is the unified activity feed: every queue job plus the health
// events that are not tied to one (a job's own events belong to its detail
// panel, not to this list). Both branches are projected onto the same columns;
// column 3 is the sort timestamp (jobs: when the work actually started) and
// column 12 the duration — the wall clock between the job timestamps, the
// recorded provider call time for an event. The optional type/status filters
// are $1/$2 and the optional period lower bound is $3 (nil counts everything),
// so the paged and the counting query share it unchanged.
const activityUnion = `
		SELECT 'job', id, COALESCE(started_at, created_at), type, status,
		       COALESCE(target_type, ''), target_id,
		       processed, total, error, ''::text,
		       COALESCE((EXTRACT(EPOCH FROM (finished_at - started_at)) * 1000)::bigint, 0)
		 FROM jobs
		 WHERE ($1 = '' OR type = $1) AND ($2 = '' OR status = $2)
		   AND ($3::timestamptz IS NULL OR COALESCE(started_at, created_at) >= $3)
		 UNION ALL
		 SELECT 'event', id, created_at, event_type, status,
		       CASE WHEN asset_id IS NULL THEN 'global' ELSE 'asset' END, asset_id,
		       0, 0, ''::text, COALESCE(message, ''), COALESCE(duration_ms, 0)::bigint
		 FROM health_events
		 WHERE job_id IS NULL
		   AND ($1 = '' OR event_type = $1) AND ($2 = '' OR status = $2)
		   AND ($3::timestamptz IS NULL OR created_at >= $3)`

// GetActivityPage returns one page of the unified activity feed, newest
// first. Empty itemType or status means "no filter" on that dimension; a nil
// since means no lower time bound.
func (r *healthRepo) GetActivityPage(ctx context.Context, limit, offset int, itemType, status string, since *time.Time) ([]*model.ActivityItem, error) {
	rows, err := r.db.Query(ctx,
		`SELECT * FROM (`+activityUnion+`) a
		 ORDER BY 3 DESC, 2 DESC LIMIT $4 OFFSET $5`,
		itemType, status, sinceArg(since), limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*model.ActivityItem, 0)
	for rows.Next() {
		it := &model.ActivityItem{}
		err := rows.Scan(&it.Kind, &it.ID, &it.At, &it.Type, &it.Status,
			&it.TargetType, &it.TargetID, &it.Processed, &it.Total, &it.Error, &it.Message, &it.DurationMs)
		if err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// CountActivity counts the feed rows matching the same optional filters as
// GetActivityPage, so items_total stays right under a filter.
func (r *healthRepo) CountActivity(ctx context.Context, itemType, status string, since *time.Time) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM (`+activityUnion+`) a`, itemType, status, sinceArg(since)).Scan(&count)
	return count, err
}

// SummarySince counts provider outcomes (success vs failure) of a time
// window. Job lifecycle events are meta rows about the queue, not external
// interactions, and stay out of the provider-health picture. A nil since
// counts every event except those job rows.
func (r *healthRepo) SummarySince(ctx context.Context, since *time.Time) (*model.HealthSummary, error) {
	return r.scanSummary(r.db.QueryRow(ctx,
		`SELECT COUNT(*) FILTER (WHERE status = 'success'),
		        COUNT(*) FILTER (WHERE status <> 'success'),
		        COUNT(*) FILTER (WHERE code = 'rate_limited')
		 FROM health_events
		 WHERE event_type <> 'job'
		   AND ($1::timestamptz IS NULL OR created_at >= $1)`, sinceArg(since)))
}

// CountsByJobs rolls up the item-level health events of the given jobs
// (lifecycle rows excluded): one grouped query serves a whole page of jobs.
func (r *healthRepo) CountsByJobs(ctx context.Context, jobIDs []uuid.UUID) (map[uuid.UUID]*model.JobHealthCounts, error) {
	if len(jobIDs) == 0 {
		return map[uuid.UUID]*model.JobHealthCounts{}, nil
	}
	rows, err := r.db.Query(ctx,
		`SELECT job_id,
		        COUNT(*) FILTER (WHERE status = 'success'),
		        COUNT(*) FILTER (WHERE status = 'failure')
		 FROM health_events
		 WHERE job_id = ANY($1) AND event_type <> 'job'
		 GROUP BY job_id`, jobIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[uuid.UUID]*model.JobHealthCounts)
	for rows.Next() {
		var id uuid.UUID
		c := &model.JobHealthCounts{}
		if err := rows.Scan(&id, &c.OK, &c.Failed); err != nil {
			return nil, err
		}
		out[id] = c
	}
	return out, rows.Err()
}

func (r *healthRepo) DeleteStandaloneBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	tag, err := r.db.Exec(ctx, `DELETE FROM health_events WHERE job_id IS NULL AND created_at < $1`, cutoff)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// ClearAll wipes every event and every job. Jobs and their events are one
// unit (the cascade FK); the endpoint is meant to empty both together.
func (r *healthRepo) ClearAll(ctx context.Context) (int64, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `DELETE FROM health_events`)
	if err != nil {
		return 0, err
	}
	events := tag.RowsAffected()
	if _, err := tx.Exec(ctx, `DELETE FROM jobs`); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return events, nil
}

func (r *healthRepo) scanSummary(row pgx.Row) (*model.HealthSummary, error) {
	summary := &model.HealthSummary{}
	if err := row.Scan(&summary.Successes, &summary.Failures, &summary.RateLimited); err != nil {
		return nil, err
	}
	return summary, nil
}
