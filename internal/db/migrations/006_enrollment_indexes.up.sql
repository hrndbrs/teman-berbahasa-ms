CREATE INDEX idx_enrollments_created_at ON enrollments(created_at DESC);

DROP INDEX idx_respondents_email;
CREATE UNIQUE INDEX idx_respondents_email ON respondents(email) WHERE email IS NOT NULL;
