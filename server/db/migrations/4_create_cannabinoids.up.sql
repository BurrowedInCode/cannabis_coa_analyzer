CREATE TABLE IF NOT EXISTS cannabinoids (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT,
  value NUMERIC,
  unit TEXT,
  analysis_id UUID REFERENCES analyses(id)
);
