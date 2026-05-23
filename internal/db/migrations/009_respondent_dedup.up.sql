-- Deduplicate: keep oldest row per email before creating unique index
DELETE FROM respondents r1
USING respondents r2
WHERE r1.email = r2.email
  AND r1.email IS NOT NULL
  AND r1.created_at > r2.created_at;

-- Replace non-unique index with unique partial index
DROP INDEX IF EXISTS idx_respondents_email;
CREATE UNIQUE INDEX IF NOT EXISTS idx_respondents_email ON respondents(email) WHERE email IS NOT NULL;
