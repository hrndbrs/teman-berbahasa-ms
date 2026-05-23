DROP INDEX IF EXISTS idx_respondents_email;
CREATE INDEX IF NOT EXISTS idx_respondents_email ON respondents(email);
