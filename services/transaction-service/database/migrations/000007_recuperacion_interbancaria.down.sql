DO $$ BEGIN IF EXISTS (SELECT 1 FROM reversos_interbancarios) OR EXISTS (SELECT 1 FROM cancelaciones_interbancarias) THEN RAISE EXCEPTION 'financial recovery receipts must be retained'; END IF; END $$;
DROP TABLE conciliacion_items_interbancarios;
DROP TABLE conciliaciones_interbancarias;
DROP TABLE incidentes_interbancarios;
DROP TABLE reversos_interbancarios;
DROP TABLE cancelaciones_interbancarias;
ALTER TABLE depositos_interbancarios DROP COLUMN refund_of;
