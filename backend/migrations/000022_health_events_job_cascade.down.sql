-- DOWN
ALTER TABLE health_events DROP CONSTRAINT IF EXISTS health_events_job_id_fkey;
ALTER TABLE health_events ADD CONSTRAINT health_events_job_id_fkey
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE SET NULL;
