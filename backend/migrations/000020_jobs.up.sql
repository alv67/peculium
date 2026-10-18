-- UP
-- Postgres-backed queue for long-running external-site work (Yahoo,
-- python-service). Rows are claimed atomically by the worker and carry
-- per-item progress so later phases can report and resume.
CREATE TABLE jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type TEXT NOT NULL, -- 'price_refresh', 'history_backfill', 'meta_backfill', 'splits_fetch'
    target_type TEXT, -- 'asset' | 'portfolio' | 'global'
    target_id UUID, -- no FK: polymorphic reference keyed by target_type
    status TEXT NOT NULL DEFAULT 'queued' CHECK (status IN ('queued', 'running', 'done', 'failed', 'partial')),
    total INTEGER NOT NULL DEFAULT 0,
    processed INTEGER NOT NULL DEFAULT 0,
    checkpoint JSONB,
    error TEXT NOT NULL DEFAULT '',
    requested_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ
);

CREATE INDEX idx_jobs_status_created ON jobs (status, created_at);
CREATE INDEX idx_jobs_type ON jobs (type);

-- Lets health events be correlated with the job that produced them.
ALTER TABLE health_events ADD COLUMN job_id UUID REFERENCES jobs(id) ON DELETE SET NULL;
CREATE INDEX idx_health_events_job ON health_events (job_id);
