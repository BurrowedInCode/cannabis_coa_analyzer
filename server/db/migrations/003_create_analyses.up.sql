CREATE TABLE IF NOT EXISTS analyses(
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  sample_name TEXT,
  seed_to_sale_number TEXT,
  sample_matrix TEXT,
  test_date TIMESTAMPTZ,
  overall_pass BOOLEAN,
  laboratory_id UUID REFERENCES laboratories(id)
)
