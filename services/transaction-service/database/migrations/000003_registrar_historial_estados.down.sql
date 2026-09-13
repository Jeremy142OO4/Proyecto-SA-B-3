DROP TRIGGER IF EXISTS trg_historial_estado_transferencia ON transferencias;
DROP FUNCTION IF EXISTS registrar_cambio_estado_transferencia();
DROP TABLE IF EXISTS historial_estados_transferencia;
