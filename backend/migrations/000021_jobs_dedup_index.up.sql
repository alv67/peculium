-- UP
-- Dedup: at most one open (queued/running) job per (type, target). A unique
-- index beats a SELECT ... FOR UPDATE transaction because the repository
-- needs no lock and no round trip to enforce it: concurrent enqueues of the
-- same work collapse onto the existing row via ON CONFLICT DO NOTHING.
-- COALESCE normalizes the NULL "global" target to a fixed UUID so Postgres
-- compares it for equality (NULLs are never equal under unique indexes).
CREATE UNIQUE INDEX uq_jobs_one_open_per_target ON jobs (
    type,
    COALESCE(target_type, ''),
    COALESCE(target_id, '00000000-0000-0000-0000-000000000000'::uuid)
) WHERE status IN ('queued', 'running');
