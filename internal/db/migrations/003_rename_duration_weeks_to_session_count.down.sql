DROP VIEW IF EXISTS courses_with_stats;

ALTER TABLE courses RENAME COLUMN session_count TO duration_weeks;

CREATE VIEW courses_with_stats AS
SELECT
  c.id,
  c.course_name,
  c.course_code,
  c.description,
  c.subject,
  c.level,
  c.duration_weeks,
  c.price,
  c.max_capacity,
  c.status,
  c.created_at,
  c.updated_at,
  COUNT(b.id)::bigint                                       AS batch_count,
  COUNT(b.id) FILTER (WHERE b.status = 'ongoing')::bigint   AS ongoing_batch_count,
  COALESCE((
    SELECT COUNT(*)::bigint
    FROM enrollments e
    WHERE e.course_id = c.id AND e.status != 'dropped'
  ), 0)                                                      AS enrolled_count
FROM courses c
LEFT JOIN batches b ON b.course_id = c.id
GROUP BY
  c.id, c.course_name, c.course_code, c.description, c.subject,
  c.level, c.duration_weeks, c.price, c.max_capacity, c.status,
  c.created_at, c.updated_at;
