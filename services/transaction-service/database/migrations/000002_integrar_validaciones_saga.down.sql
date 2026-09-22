UPDATE transferencias SET estado='PENDIENTE' WHERE estado IN ('VALIDANDO_KYC','VALIDANDO_CUENTAS');
DROP TRIGGER IF EXISTS trg_historial_estado_transferencia ON transferencias;
DROP FUNCTION IF EXISTS registrar_cambio_estado_transferencia();
DROP TABLE IF EXISTS historial_estados_transferencia;
ALTER TABLE transferencias DROP CONSTRAINT IF EXISTS transferencias_estado_check;
ALTER TABLE transferencias ADD CONSTRAINT transferencias_estado_check CHECK (estado IN ('PENDIENTE','PROCESANDO','COMPLETADA','RECHAZADA','COMPENSANDO','COMPENSADA','COMPENSACION_FALLIDA'));
ALTER TABLE transferencias DROP CONSTRAINT IF EXISTS ck_transferencia_resultado_externo;
ALTER TABLE transferencias DROP COLUMN IF EXISTS resultado_externo_simulado;
