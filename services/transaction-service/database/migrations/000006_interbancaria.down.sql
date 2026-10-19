DO $$ BEGIN IF EXISTS (SELECT 1 FROM depositos_interbancarios) THEN RAISE EXCEPTION 'interbank receipts must be retained'; END IF; END $$;
DROP TABLE depositos_interbancarios;
