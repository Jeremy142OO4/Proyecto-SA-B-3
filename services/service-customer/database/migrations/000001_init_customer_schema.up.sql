-- Extensión para generación de UUID v4
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";


-- 1. Tabla de Clientes / Usuarios
CREATE TABLE IF NOT EXISTS customers (
    customer_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    full_name VARCHAR(200) NOT NULL,
    document_id VARCHAR(50) UNIQUE NOT NULL,       
    document_photo_url VARCHAR(500),               
    email VARCHAR(150) UNIQUE NOT NULL,
    birth_date DATE NOT NULL,
    address TEXT NOT NULL,
    username VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) NOT NULL CHECK (role IN ('ADMIN', 'TELLER', 'CUSTOMER')),
    status VARCHAR(30) NOT NULL DEFAULT 'PENDING_ACTIVATION' CHECK (status IN ('PENDING_ACTIVATION', 'ACTIVE', 'BLOCKED')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_customers_email ON customers(email);
CREATE INDEX IF NOT EXISTS idx_customers_username ON customers(username);
CREATE INDEX IF NOT EXISTS idx_customers_document_id ON customers(document_id);


-- 2. Tokens de Activación de un solo uso
CREATE TABLE IF NOT EXISTS activation_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    customer_id UUID NOT NULL REFERENCES customers(customer_id) ON DELETE CASCADE,
    token_hash VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    is_used BOOLEAN NOT NULL DEFAULT FALSE,
    used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_activation_tokens_customer ON activation_tokens(customer_id);
CREATE INDEX IF NOT EXISTS idx_activation_tokens_lookup ON activation_tokens(token_hash, is_used, expires_at);


-- 3. Idempotencia: Registro de mensajes consumidos
CREATE TABLE IF NOT EXISTS processed_messages (
    message_id UUID PRIMARY KEY,
    consumer_name VARCHAR(100) NOT NULL,
    processed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    result_reference VARCHAR(255)
);

-- 4. Transactional Outbox: Registro de eventos para publicación
CREATE TABLE IF NOT EXISTS outbox_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    correlation_id UUID NOT NULL,
    causation_id UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMP WITH TIME ZONE,
    attempts INT DEFAULT 0,
    last_error TEXT
);

CREATE INDEX IF NOT EXISTS idx_outbox_unpublished ON outbox_messages(created_at) WHERE published_at IS NULL;