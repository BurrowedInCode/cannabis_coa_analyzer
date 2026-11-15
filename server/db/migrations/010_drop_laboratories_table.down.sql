CREATE TABLE IF NOT EXISTS laboratories (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  address TEXT NOT NULL,
  phone TEXT NOT NULL,
  certification TEXT NOT NULL
);

ALTER TABLE analyses
  ADD COLUMN laboratory_id UUID REFERENCES laboratories(id);

INSERT INTO laboratories (id, name, address, phone, certification)
SELECT id, laboratory_name, laboratory_address, laboratory_phone, laboratory_certification
FROM analyses;

UPDATE analyses
SET laboratory_id = id;
