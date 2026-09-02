-- Usuarios iniciales para desarrollo y demostracion local.
-- Las contrasenas se almacenan como bcrypt, nunca en texto plano.
INSERT INTO customers (
    customer_id, first_name, last_name, full_name, document_id, document_photo_url,
    email, birth_date, address, username, password_hash, role, status
) VALUES
    (
        '00000000-0000-4000-8000-000000000001',
        'Administrador', 'Inicial', 'Administrador Inicial', 'ADMIN-INICIAL-001', '',
        'admin@ejemplo.com', '1990-01-01', 'Configuracion inicial', 'admin',
        '$2b$10$hZ2/rxlOMVONAgLHibCfQeC/EKIxgar8EIiWirEl6fOLmzabFokzq',
        'ADMIN', 'ACTIVO'
    ),
    (
        '00000000-0000-4000-8000-000000000002',
        'Cajero', 'Inicial', 'Cajero Inicial', 'TELLER-INICIAL-001', '',
        'teller@ejemplo.com', '1990-01-01', 'Configuracion inicial', 'teller',
        '$2b$10$VhFlY70cumFw3mEyzzJ/.uWOHwRCXZ7czy7iQO1ucWpcAAnaqzFIO',
        'TELLER', 'ACTIVO'
    )
ON CONFLICT DO NOTHING;
