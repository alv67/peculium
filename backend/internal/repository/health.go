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
	GetEventsPage(ctx context.Context, limit, offset int) ([]*model.HealthEvent, error)
	CountEvents(ctx context.Context) (int, error)
	SummarySince(ctx context.Context, since time.Time) (*model.HealthSummary, error)
	SummaryLastN(ctx context.Context, n int) (*model.HealthSummary, error)
}

type healthRepo struct {
	db *pgxpool.Pool
}

func NewHealthRepository(db *pgxpool.Pool) HealthRepository {
	return &healthRepo{db}
}

func (r *healthRepo) RecordEvent(ctx context.Context, event *model.HealthEvent) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO health_events (id, asset_id, event_type, status, code, message, duration_ms, error_code, created_at) 
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		event.ID, event.AssetID, event.EventType, event.Status, event.Code, event.Message, event.DurationMs, event.ErrorCode, event.CreatedAt)
	return err
}

func (r *healthRepo) GetEventsPage(ctx context.Context, limit, offset int) ([]*model.HealthEvent, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, asset_id, event_type, status, code, message, duration_ms, error_code, created_at 
		 FROM health_events 
		 ORDER BY created_at DESC 
		 LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]*model.HealthEvent, 0)
	for rows.Next() {
		e := &model.HealthEvent{}
		var assetID *uuid.UUID
		err := rows.Scan(&e.ID, &assetID, &e.EventType, &e.Status, &e.Code, &e.Message, &e.DurationMs, &e.ErrorCode, &e.CreatedAt)
		if err != nil {
			return nil, err
		}
		e.AssetID = assetID
		events = append(events, e)
	}
	return events, nil
}

func (r *healthRepo) CountEvents(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM health_events`).Scan(&count)
	return count, err
}

func (r *healthRepo) SummarySince(ctx context.Context, since time.Time) (*model.HealthSummary, error) {
	return r.scanSummary(r.db.QueryRow(ctx,
		`SELECT COUNT(*) FILTER (WHERE status = 'success'),
		        COUNT(*) FILTER (WHERE status <> 'success'),
		        COUNT(*) FILTER (WHERE code = 'rate_limited')
		 FROM health_events
		 WHERE created_at >= $1`, since))
}

func (r *healthRepo) SummaryLastN(ctx context.Context, n int) (*model.HealthSummary, error) {
	return r.scanSummary(r.db.QueryRow(ctx,
		`SELECT COUNT(*) FILTER (WHERE status = 'success'),
		        COUNT(*) FILTER (WHERE status <> 'success'),
		        COUNT(*) FILTER (WHERE code = 'rate_limited')
		 FROM (
			 SELECT status, code
			 FROM health_events
			 ORDER BY created_at DESC
			 LIMIT $1
		 ) recent`, n))
}

func (r *healthRepo) scanSummary(row pgx.Row) (*model.HealthSummary, error) {
	summary := &model.HealthSummary{}
	if err := row.Scan(&summary.Successes, &summary.Failures, &summary.RateLimited); err != nil {
		return nil, err
	}
	return summary, nil
}
