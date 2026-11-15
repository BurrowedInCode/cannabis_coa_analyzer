ALTER TABLE analyses
ADD COLUMN input_tokens BIGINT,
ADD COLUMN output_tokens BIGINT,
ADD COLUMN created_at TIMESTAMPTZ DEFAULT now();
