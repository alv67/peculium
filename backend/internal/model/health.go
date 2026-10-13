package model

import (
	"time"

	"github.com/google/uuid"
)

type HealthEvent struct {
	ID         uuid.UUID  `json:"id"`
	AssetID    *uuid.UUID `json:"asset_id,omitempty"`
	JobID      *uuid.UUID `json:"job_id,omitempty"`
	EventType  string     `json:"event_type"`
	Status     string     `json:"status"`
	Code       string     `json:"code"`
	Message    string     `json:"message"`
	DurationMs int        `json:"duration_ms"`
	ErrorCode  string     `json:"error_code"`
	CreatedAt  time.Time  `json:"created_at"`
}

// ActivityItem is one row of the unified Data & Sync feed: a queue job or a
// health event that is not tied to one. The Kind decides which fields carry
// data — Processed/Total/Error are job-only, Message is event-only, and
// DurationMs is the wall-clock run for a job, the provider call time for an
// event. TargetLabel is filled in by the service, never by the query.
type ActivityItem struct {
	Kind        string     `json:"kind"`
	ID          uuid.UUID  `json:"id"`
	At          time.Time  `json:"at"`
	Type        string     `json:"type"`
	TargetType  string     `json:"target_type"`
	TargetLabel string     `json:"target_label"`
	TargetID    *uuid.UUID `json:"target_id"`
	Status      string     `json:"status"`
	Processed   int        `json:"processed"`
	Total       int        `json:"total"`
	DurationMs  int64      `json:"duration_ms"`
	Error       string     `json:"error"`
	Message     string     `json:"message"`
}

type HealthSummary struct {
	Period      string  `json:"period"`
	Successes   int     `json:"successes"`
	Failures    int     `json:"failures"`
	SuccessRate float64 `json:"success_rate"`
	RateLimited int     `json:"rate_limited"`
	HasData     bool    `json:"has_data"`
}
