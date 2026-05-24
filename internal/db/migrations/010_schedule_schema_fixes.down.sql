ALTER TABLE schedule_overrides
  DROP CONSTRAINT schedule_overrides_schedule_id_fkey,
  ADD CONSTRAINT schedule_overrides_schedule_id_fkey
    FOREIGN KEY (schedule_id) REFERENCES schedules(id);

ALTER TABLE schedule_overrides
  DROP CONSTRAINT schedule_overrides_override_type_check,
  ADD CONSTRAINT schedule_overrides_override_type_check
    CHECK (override_type IN ('reschedule', 'cancellation', 'instructor_change'));

ALTER TABLE courses ALTER COLUMN session_count DROP NOT NULL;
