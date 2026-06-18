CREATE TABLE IF NOT EXISTS test_summaries (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT,
  status TEXT,
  analysis_id UUID REFERENCES analyses(id)
);
