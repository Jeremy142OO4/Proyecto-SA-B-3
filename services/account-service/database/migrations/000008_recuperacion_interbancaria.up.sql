ALTER TABLE operaciones_interbancarias ADD COLUMN IF NOT EXISTS original UUID;
ALTER TABLE operaciones_interbancarias ADD COLUMN IF NOT EXISTS accion TEXT NOT NULL DEFAULT '';
CREATE UNIQUE INDEX IF NOT EXISTS uq_reintegro_interbancario ON operaciones_interbancarias(original) WHERE original IS NOT NULL;
