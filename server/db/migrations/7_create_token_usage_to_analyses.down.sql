ALTER TABLE analyses
  DROP COLUMN IF EXISTS input_tokens,
  DROP COLUMN IF EXISTS output_tokens;
  DROP COLUMN IF EXISTS created_at;
