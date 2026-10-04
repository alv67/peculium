package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Job lifecycle statuses: queued -> running -> done|failed|partial.
const (
	JobStatusQueued  = "queued"
	JobStatusRunning = "running"
	JobStatusDone    = "done"
	JobStatusFailed  = "failed"
	JobStatusPartial = "partial"
)

// Job types cover every long-running fetch against an external site.
const (
	JobTypePriceRefresh    = "price_refresh"
	JobTypeHistoryBackfill = "history_backfill"
	JobTypeMetaBackfill    = "meta_backfill"
	JobTypeSplitsFetch     = "splits_fetch"
)

// Job target types; target_id is a polymorphic reference to the entity.
const (
	JobTargetAsset     = "asset"
	JobTargetPortfolio = "portfolio"
	JobTargetGlobal    = "global"
)

// JobHealthCounts is the item-level health rollup of a job, derived from its
// health events at read time (lifecycle rows excluded — they are meta, not
// work).
type JobHealthCounts struct {
	OK     int `json:"ok"`
	Failed int `json:"failed"`
}

// Job is one unit of external-site work queued in Postgres and drained by the
// worker. Processed/Total track per-item progress and Checkpoint lets a long
// job resume where a previous attempt stopped.
type Job struct {
	ID          uuid.UUID       `json:"id"`
	Type        string          `json:"type"`
	TargetType  string          `json:"target_type,omitempty"`
	TargetID    *uuid.UUID      `json:"target_id,omitempty"`
	Status      string          `json:"status"`
	Total       int             `json:"total"`
	Processed   int             `json:"processed"`
	Checkpoint  json.RawMessage `json:"checkpoint,omitempty"`
	Error       string          `json:"error,omitempty"`
	RequestedBy *uuid.UUID      `json:"requested_by,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	StartedAt   *time.Time      `json:"started_at,omitempty"`
	FinishedAt  *time.Time      `json:"finished_at,omitempty"`

	// Derived fields, filled by JobService.Get/List and never stored: the
	// wall-clock run duration (finished - started) and the health-event
	// rollup. Zero/nil while the job has not started or finished.
	DurationMs int64            `json:"duration_ms,omitempty"`
	Summary    *JobHealthCounts `json:"summary,omitempty"`
}
