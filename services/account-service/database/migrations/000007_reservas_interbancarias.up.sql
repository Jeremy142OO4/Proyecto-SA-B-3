ALTER TABLE cuentas ADD COLUMN IF NOT EXISTS saldo_reservado_centavos BIGINT NOT NULL DEFAULT 0;
DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='cuentas'::regclass AND conname='ck_reserva_cuenta') THEN
 ALTER TABLE cuentas ADD CONSTRAINT ck_reserva_cuenta CHECK (saldo_reservado_centavos >= 0 AND saldo_reservado_centavos <= saldo_centavos);
END IF; END $$;
CREATE TABLE IF NOT EXISTS operaciones_interbancarias (
 id UUID PRIMARY KEY,
 cuenta TEXT NOT NULL,
 cliente TEXT NOT NULL DEFAULT '',
 monto BIGINT NOT NULL CHECK (monto > 0),
 comision BIGINT NOT NULL DEFAULT 0,
 estado TEXT NOT NULL CHECK (estado IN ('RESERVED','DEBITED','RELEASED','CREDITED','REJECTED')),
 motivo TEXT NOT NULL DEFAULT '',
 vence TIMESTAMPTZ NOT NULL,
 actualizado TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);
