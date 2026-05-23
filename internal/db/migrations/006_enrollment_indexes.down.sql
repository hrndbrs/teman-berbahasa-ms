DROP INDEX IF EXISTS idx_enrollments_created_at;

DROP INDEX IF EXISTS idx_respondents_email;
CREATE INDEX idx_respondents_email ON respondents(email);
