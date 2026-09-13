ALTER TABLE pagos DROP CONSTRAINT IF EXISTS ck_pagos_resultado_simulado;
ALTER TABLE pagos DROP COLUMN IF EXISTS resultado_simulado;
