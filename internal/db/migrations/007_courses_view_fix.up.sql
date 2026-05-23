DROP VIEW IF EXISTS courses_with_stats;

CREATE VIEW courses_with_stats AS
SELECT
  c.id,
  c.course_name,
  c.course_code,
  c.description,
  c.subject,
  c.level,
  c.session_count,
  c.price,
  c.max_capacity,
  c.status,
  c.created_at,
  c.updated_at,
  COUNT(b.id)::bigint                                       AS batch_count,
  COUNT(b.id) FILTER (WHERE b.status = 'ongoing')::bigint   AS ongoing_batch_count,
  COALESCE(ec.enrolled_count, 0)                            AS enrolled_count
FROM courses c
LEFT JOIN batches b ON b.course_id = c.id
LEFT JOIN (
    SELECT course_id,
           COUNT(*) FILTER (WHERE status != 'dropped')::bigint AS enrolled_count
    FROM enrollments
    GROUP BY course_id
) ec ON ec.course_id = c.id
GROUP BY
  c.id, c.course_name, c.course_code, c.description, c.subject,
  c.level, c.session_count, c.price, c.max_capacity, c.status,
  c.created_at, c.updated_at,
  ec.enrolled_count;
