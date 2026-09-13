ALTER TABLE transferencias DROP CONSTRAINT IF EXISTS transferencias_estado_check;
ALTER TABLE transferencias
    ADD CONSTRAINT transferencias_estado_check
    CHECK (estado IN ('VALIDANDO_KYC','VALIDANDO_CUENTAS','PENDIENTE','PROCESANDO','COMPLETADA','RECHAZADA','COMPENSANDO','COMPENSADA','COMPENSACION_FALLIDA'));

ALTER TABLE transferencias
    ADD COLUMN resultado_externo_simulado VARCHAR(20) NOT NULL DEFAULT 'EXITO';
ALTER TABLE transferencias
    ADD CONSTRAINT ck_transferencia_resultado_externo
    CHECK (resultado_externo_simulado IN ('EXITO','FALLO','TIMEOUT'));
