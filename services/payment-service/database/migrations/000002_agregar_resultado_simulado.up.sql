ALTER TABLE pagos
    ADD COLUMN resultado_simulado VARCHAR(20) NOT NULL DEFAULT 'EXITO';

ALTER TABLE pagos
    ADD CONSTRAINT ck_pagos_resultado_simulado
    CHECK (resultado_simulado IN ('EXITO', 'FALLO', 'TIMEOUT'));
