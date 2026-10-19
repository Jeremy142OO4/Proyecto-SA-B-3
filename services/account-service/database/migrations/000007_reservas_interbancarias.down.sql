-- Financial idempotency receipts cannot be discarded by an automatic rollback.
DO $$ BEGIN IF EXISTS (SELECT 1 FROM operaciones_interbancarias) THEN RAISE EXCEPTION 'interbank receipts must be retained'; END IF; END $$;
DROP TABLE operaciones_interbancarias;
ALTER TABLE cuentas DROP COLUMN saldo_reservado_centavos;
