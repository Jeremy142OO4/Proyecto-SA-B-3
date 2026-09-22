CREATE TABLE IF NOT EXISTS historial_estados_transferencia (
    id_historial UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    id_transferencia UUID NOT NULL REFERENCES transferencias(id_transferencia) ON DELETE CASCADE,
    estado_anterior VARCHAR(30),
    estado_nuevo VARCHAR(30) NOT NULL,
    codigo_error VARCHAR(100),
    fecha_creacion TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_historial_transferencia_fecha
    ON historial_estados_transferencia (id_transferencia, fecha_creacion);
CREATE OR REPLACE FUNCTION registrar_cambio_estado_transferencia() RETURNS trigger AS $$
BEGIN
    IF TG_OP = 'INSERT' OR OLD.estado IS DISTINCT FROM NEW.estado THEN
        INSERT INTO historial_estados_transferencia(id_transferencia, estado_anterior, estado_nuevo, codigo_error)
        VALUES (NEW.id_transferencia, CASE WHEN TG_OP = 'INSERT' THEN NULL ELSE OLD.estado END, NEW.estado, NEW.codigo_error);
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS trg_historial_estado_transferencia ON transferencias;
CREATE TRIGGER trg_historial_estado_transferencia
AFTER INSERT OR UPDATE OF estado ON transferencias
FOR EACH ROW EXECUTE FUNCTION registrar_cambio_estado_transferencia();
