-- UP
-- A job and its health events are one unit: deleting a job must remove its
-- events. The original FK used ON DELETE SET NULL, which orphaned them.
ALTER TABLE health_events DROP CONSTRAINT IF EXISTS health_events_job_id_fkey;
ALTER TABLE health_events ADD CONSTRAINT health_events_job_id_fkey
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE;
