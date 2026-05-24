-- name: CreateSchedule :one
INSERT INTO schedules (
    id, batch_id, course_id, instructor_user_id,
    day_of_week, start_time, end_time, room,
    recurrence, effective_from, effective_until
) VALUES (
    sqlc.arg('id'), sqlc.arg('batch_id'), sqlc.arg('course_id'),
    sqlc.narg('instructor_user_id'),
    sqlc.arg('day_of_week'), sqlc.arg('start_time'), sqlc.arg('end_time'),
    sqlc.narg('room'), sqlc.arg('recurrence'),
    sqlc.arg('effective_from'), sqlc.narg('effective_until')
)
RETURNING *;

-- name: ListSchedulesByBatch :many
SELECT * FROM schedules
WHERE batch_id = sqlc.arg('batch_id')
ORDER BY day_of_week, start_time;

-- name: GetScheduleByID :one
SELECT * FROM schedules WHERE id = sqlc.arg('id');

-- name: UpdateSchedule :one
UPDATE schedules SET
    instructor_user_id = sqlc.narg('instructor_user_id'),
    day_of_week        = sqlc.arg('day_of_week'),
    start_time         = sqlc.arg('start_time'),
    end_time           = sqlc.arg('end_time'),
    room               = sqlc.narg('room'),
    recurrence         = sqlc.arg('recurrence'),
    effective_from     = sqlc.arg('effective_from'),
    effective_until    = sqlc.narg('effective_until'),
    updated_at         = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: DeleteSchedule :exec
DELETE FROM schedules WHERE id = sqlc.arg('id');

-- name: ListSchedulesByBatchForCount :many
SELECT id, recurrence, day_of_week, effective_from, effective_until
FROM schedules
WHERE batch_id = sqlc.arg('batch_id');

-- name: GetBatchForSchedule :one
SELECT id, course_id, instructor_user_id FROM batches WHERE id = sqlc.arg('id');

-- name: GetCourseSessionCount :one
SELECT session_count FROM courses WHERE id = sqlc.arg('id');

-- name: ListSchedulesForWeek :many
SELECT
    s.id,
    s.batch_id,
    s.course_id,
    s.instructor_user_id,
    s.day_of_week,
    s.start_time,
    s.end_time,
    s.room,
    s.recurrence,
    s.effective_from,
    s.effective_until,
    b.batch_name,
    b.batch_code,
    b.instructor_user_id                AS batch_instructor_id,
    ub.first_name                       AS batch_instructor_first_name,
    ub.last_name                        AS batch_instructor_last_name,
    us.first_name                       AS sched_instructor_first_name,
    us.last_name                        AS sched_instructor_last_name,
    c.course_name,
    c.course_code,
    c.level                             AS course_level
FROM schedules s
JOIN batches b  ON b.id  = s.batch_id
JOIN courses c  ON c.id  = s.course_id
JOIN users   ub ON ub.id = b.instructor_user_id
LEFT JOIN users us ON us.id = s.instructor_user_id
WHERE s.effective_from <= sqlc.arg('week_end')::date
  AND (s.effective_until IS NULL OR s.effective_until >= sqlc.arg('week_start')::date)
  AND (sqlc.narg('course_id')::uuid IS NULL OR s.course_id = sqlc.narg('course_id'))
  AND (sqlc.narg('batch_id')::uuid  IS NULL OR s.batch_id  = sqlc.narg('batch_id'))
  AND (sqlc.narg('level')::text     IS NULL OR c.level     = sqlc.narg('level'))
ORDER BY s.day_of_week, s.start_time;

-- name: UpsertScheduleOverride :one
INSERT INTO schedule_overrides (
    id, schedule_id, original_date, override_type,
    new_date, new_start_time, new_end_time, new_room,
    new_instructor_user_id, reason, created_by_user_id
) VALUES (
    sqlc.arg('id'), sqlc.arg('schedule_id'), sqlc.arg('original_date'),
    sqlc.arg('override_type'),
    sqlc.narg('new_date'), sqlc.narg('new_start_time'), sqlc.narg('new_end_time'),
    sqlc.narg('new_room'), sqlc.narg('new_instructor_user_id'),
    sqlc.narg('reason'), sqlc.arg('created_by_user_id')
)
ON CONFLICT (schedule_id, original_date) DO UPDATE SET
    new_date               = EXCLUDED.new_date,
    new_start_time         = EXCLUDED.new_start_time,
    new_end_time           = EXCLUDED.new_end_time,
    new_room               = EXCLUDED.new_room,
    new_instructor_user_id = EXCLUDED.new_instructor_user_id,
    reason                 = EXCLUDED.reason,
    created_by_user_id     = EXCLUDED.created_by_user_id
RETURNING *;

-- name: GetOverridesByScheduleIDs :many
SELECT
    o.id, o.schedule_id, o.original_date, o.override_type,
    o.new_date, o.new_start_time, o.new_end_time, o.new_room,
    o.new_instructor_user_id, o.reason, o.created_by_user_id, o.created_at,
    u.first_name AS new_instructor_first_name,
    u.last_name  AS new_instructor_last_name
FROM schedule_overrides o
LEFT JOIN users u ON u.id = o.new_instructor_user_id
WHERE o.schedule_id = ANY(sqlc.arg('schedule_ids')::uuid[])
  AND (
      o.original_date BETWEEN sqlc.arg('week_start')::date AND sqlc.arg('week_end')::date
      OR (
          o.override_type = 'reschedule'
          AND o.new_date BETWEEN sqlc.arg('week_start')::date AND sqlc.arg('week_end')::date
      )
  );

-- name: GetOverrideByID :one
SELECT
    o.id, o.schedule_id, o.original_date, o.override_type,
    o.new_date, o.new_start_time, o.new_end_time, o.new_room,
    o.new_instructor_user_id, o.reason, o.created_by_user_id, o.created_at,
    u.first_name AS new_instructor_first_name,
    u.last_name  AS new_instructor_last_name
FROM schedule_overrides o
LEFT JOIN users u ON u.id = o.new_instructor_user_id
WHERE o.id = sqlc.arg('id');

-- name: UpdateScheduleOverride :one
UPDATE schedule_overrides SET
    new_date               = sqlc.narg('new_date'),
    new_start_time         = sqlc.narg('new_start_time'),
    new_end_time           = sqlc.narg('new_end_time'),
    new_room               = sqlc.narg('new_room'),
    new_instructor_user_id = sqlc.narg('new_instructor_user_id'),
    reason                 = sqlc.narg('reason')
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: DeleteScheduleOverride :exec
DELETE FROM schedule_overrides WHERE id = sqlc.arg('id');
