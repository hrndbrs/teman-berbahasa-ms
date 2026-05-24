UPDATE courses SET session_count = 0 WHERE session_count IS NULL;
ALTER TABLE courses ALTER COLUMN session_count SET NOT NULL;

ALTER TABLE schedule_overrides
  DROP CONSTRAINT schedule_overrides_override_type_check,
  ADD CONSTRAINT schedule_overrides_override_type_check
    CHECK (override_type IN ('reschedule', 'instructor_change'));

ALTER TABLE schedule_overrides
  DROP CONSTRAINT schedule_overrides_schedule_id_fkey,
  ADD CONSTRAINT schedule_overrides_schedule_id_fkey
    FOREIGN KEY (schedule_id) REFERENCES schedules(id) ON DELETE CASCADE;
