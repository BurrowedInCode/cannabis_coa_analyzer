CREATE TABLE IF NOT EXISTS laboratories (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  address TEXT NOT NULL,
  phone TEXT NOT NULL,
  certification TEXT NOT NULL,
)
