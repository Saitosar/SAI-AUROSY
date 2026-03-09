-- Add optional location for fleet grouping (e.g. "Warehouse A", "Floor 2")
ALTER TABLE robots ADD COLUMN location TEXT DEFAULT '';
