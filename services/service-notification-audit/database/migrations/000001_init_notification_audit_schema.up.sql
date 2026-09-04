-- Extensión para generación de UUID v4
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. Tabla de Auditoría Distribuida
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id UUID NOT NULL,
    correlation_id UUID NOT NULL,
    causation_id UUID,
    event_type VARCHAR(100) NOT NULL,
    producer VARCHAR(100) NOT NULL,
    version INT NOT NULL DEFAULT 1,
    payload JSONB NOT NULL,
    occurred_at TIMESTAMP WITH TIME ZONE NOT NULL,
    recorded_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_audit_correlation_id ON audit_logs(correlation_id);
CREATE INDEX IF NOT EXISTS idx_audit_event_type ON audit_logs(event_type);
CREATE INDEX IF NOT EXISTS idx_audit_occurred_at ON audit_logs(occurred_at);

-- 2. Historial de Notificaciones Enviadas / Generadas
CREATE TABLE IF NOT EXISTS notification_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    correlation_id UUID NOT NULL,
    recipient VARCHAR(150) NOT NULL,              -- Correo o identificador de destino
    notification_type VARCHAR(50) NOT NULL,        -- ACTIVATION_EMAIL, TRANSFER_ALERT, etc.
    subject VARCHAR(200) NOT NULL,
    body_summary TEXT NOT NULL,                    -- Resumen sin credenciales ni datos confidenciales
    status VARCHAR(30) NOT NULL DEFAULT 'SENT' CHECK (status IN ('PENDING', 'SENT', 'FAILED')),
    error_detail TEXT,
    sent_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_notification_recipient ON notification_logs(recipient);
CREATE INDEX IF NOT EXISTS idx_notification_correlation ON notification_logs(correlation_id);


-- 3. Idempotencia: Registro de mensajes consumidos
CREATE TABLE IF NOT EXISTS processed_messages (
    message_id UUID PRIMARY KEY,
    consumer_name VARCHAR(100) NOT NULL,
    processed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    result_reference VARCHAR(255)
);