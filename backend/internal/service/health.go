package service

import (
	"context"
	"time"

	"github.com/alv67/peculium/internal/model"
	"github.com/alv67/peculium/internal/repository"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

const (
	HealthPeriodToday = "today"
	HealthPeriod24h   = "24h"
	HealthPeriodLastN = "100"

	healthLastNEvents   = 100
	healthDefaultEvents = 100
	healthMaxEvents     = 500
)

type HealthService struct {
	repos *repository.Repository
}

func NewHealthService(repos *repository.Repository) *HealthService {
	return &HealthService{repos: repos}
}

// healthWindow is the DB window behind a summary period: either a time range
// (since non-zero) or the newest lastN events (lastN > 0).
type healthWindow struct {
	period string
	since  time.Time
	lastN  int
}

// healthWindowFor normalizes a requested period into a summary window.
// Unknown or empty values fall back to "today" (the current UTC calendar day).
func healthWindowFor(period string, now time.Time) healthWindow {
	switch period {
	case HealthPeriod24h:
		return healthWindow{period: HealthPeriod24h, since: now.Add(-24 * time.Hour)}
	case HealthPeriodLastN:
		return healthWindow{period: HealthPeriodLastN, lastN: healthLastNEvents}
	default:
		return healthWindow{period: HealthPeriodToday, since: now.Truncate(24 * time.Hour)}
	}
}

// normalizeHealthPage clamps the events pagination: non-positive limits fall
// back to the default page size, oversized limits are capped, and offsets
// never go below zero.
func normalizeHealthPage(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = healthDefaultEvents
	}
	if limit > healthMaxEvents {
		limit = healthMaxEvents
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

// RecordEvent stores a provider-interaction event. While a job executor
// runs, its id is bound to the context (JobRunner) and stamped here — the
// single chokepoint every event flows through — so the health page can
// correlate failures with the queue row that caused them. Request-path
// events carry no job and stay NULL. A context that is already expired or
// cancelled (a job that timed out) must not swallow the event describing
// that very timeout: the write is retried on a fresh short-lived context.
func (s *HealthService) RecordEvent(ctx context.Context, event *model.HealthEvent) error {
	if event.JobID == nil {
		if id, ok := JobIDFromContext(ctx); ok {
			event.JobID = &id
		}
	}
	if ctx.Err() != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
	}
	return s.repos.Health.RecordEvent(ctx, event)
}

// recordHealth logs one event through the optional Health service. nil-safe:
// deployments and tests without health tracking are unaffected, and a failed
// write never fails the surrounding operation.
func (s *Service) recordHealth(ctx context.Context, assetID *uuid.UUID, eventType, status, code, message string, since time.Time) {
	if s.Health == nil {
		return
	}
	if err := s.Health.RecordEvent(ctx, &model.HealthEvent{
		ID:         uuid.New(),
		AssetID:    assetID,
		EventType:  eventType,
		Status:     status,
		Code:       code,
		Message:    message,
		DurationMs: int(time.Since(since).Milliseconds()),
		CreatedAt:  time.Now().UTC(),
	}); err != nil {
		log.Warn().Err(err).Str("event_type", eventType).Msg("failed to record health event")
	}
}

func (s *HealthService) GetPriceHealth(ctx context.Context, period string, limit, offset int) (*model.HealthSummary, []*model.HealthEvent, int, error) {
	limit, offset = normalizeHealthPage(limit, offset)
	window := healthWindowFor(period, time.Now().UTC())

	var summary *model.HealthSummary
	var err error
	if window.lastN > 0 {
		summary, err = s.repos.Health.SummaryLastN(ctx, window.lastN)
	} else {
		summary, err = s.repos.Health.SummarySince(ctx, window.since)
	}
	if err != nil {
		return nil, nil, 0, err
	}

	summary.Period = window.period
	total := summary.Successes + summary.Failures
	summary.HasData = total > 0
	if total > 0 {
		summary.SuccessRate = float64(summary.Successes) / float64(total)
	}

	eventsTotal, err := s.repos.Health.CountEvents(ctx)
	if err != nil {
		return nil, nil, 0, err
	}

	events, err := s.repos.Health.GetEventsPage(ctx, limit, offset)
	if err != nil {
		return nil, nil, 0, err
	}

	return summary, events, eventsTotal, nil
}
