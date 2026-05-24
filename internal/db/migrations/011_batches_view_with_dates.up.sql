DROP VIEW batches_with_stats;

CREATE VIEW batches_with_stats AS
SELECT
    b.id, b.course_id, b.instructor_user_id, b.created_by_user_id,
    b.batch_name, b.batch_code, b.academic_year, b.status,
    b.created_at, b.updated_at,
    c.course_name, c.course_code,
    u.first_name AS instructor_first_name,
    u.last_name  AS instructor_last_name,
    COUNT(e.id) FILTER (WHERE e.status != 'dropped')::bigint AS enrolled_count,
    MIN(s.effective_from)                              AS first_class_date,
    MAX(COALESCE(s.effective_until, s.effective_from)) AS last_class_date
FROM batches b
JOIN courses c ON c.id = b.course_id
JOIN users   u ON u.id = b.instructor_user_id
LEFT JOIN enrollments e ON e.batch_id = b.id
LEFT JOIN schedules   s ON s.batch_id = b.id
GROUP BY b.id, c.course_name, c.course_code, u.first_name, u.last_name;
