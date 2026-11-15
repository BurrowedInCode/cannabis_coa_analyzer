ALTER TABLE analyses 
  ADD COLUMN laboratory_name TEXT,
  ADD COLUMN laboratory_address TEXT,
  ADD COLUMN laboratory_phone TEXT,
  ADD COLUMN laboratory_certification TEXT;

UPDATE analyses AS a
SET laboratory_name = l.name,
    laboratory_address = l.address,
    laboratory_phone = l.phone,
    laboratory_certification = l.certification
FROM laboratories as l
WHERE a.laboratory_id = l.id;
