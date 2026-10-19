DO $$ BEGIN IF EXISTS (SELECT 1 FROM operaciones_interbancarias WHERE original IS NOT NULL) THEN RAISE EXCEPTION 'refund receipts must be retained'; END IF; END $$;
ALTER TABLE operaciones_interbancarias DROP COLUMN original;
ALTER TABLE operaciones_interbancarias DROP COLUMN accion;
