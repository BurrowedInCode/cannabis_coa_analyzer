CREATE TABLE IF NOT EXISTS terpenes (
  id UUID PRIMARY KEY default gen_random_uuid(),
  name TEXT,
  value NUMERIC,
  unit TEXT,
  analysis_id UUID REFERENCES analyses(id)
);
