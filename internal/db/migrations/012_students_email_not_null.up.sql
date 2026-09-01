-- Student email is now mandatory. Existing NULL rows must be backfilled before
-- this migration can apply — there is no safe synthetic default for an identity field.
ALTER TABLE students ALTER COLUMN email SET NOT NULL;
