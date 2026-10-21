#!/bin/sh
set -eu
# Runs once against the new sandbox volume, not against any external database.
for pair in accounts:account-service transactions:transaction-service customers:service-customer; do
  database=${pair%%:*}
  module=${pair#*:}
  psql -v ON_ERROR_STOP=1 -U postgres -c "CREATE DATABASE $database"
  for migration in /schemas/"$module"/database/migrations/*.up.sql; do
    psql -v ON_ERROR_STOP=1 -U postgres -d "$database" -f "$migration"
  done
done
psql -v ON_ERROR_STOP=1 -U postgres -d customers <<'SQL'
CREATE EXTENSION IF NOT EXISTS pgcrypto;
INSERT INTO customers(customer_id,first_name,last_name,full_name,document_id,email,birth_date,address,username,password_hash,role,status,kyc_status)
VALUES
 ('00000000-0000-4000-8000-000000000004','Admin','Sandbox','Administrador Sandbox','SANDBOX-ADMIN','admin@sandbox.invalid','1990-01-01','Laboratorio local','sandbox-admin',crypt('SandboxDemo2026!',gen_salt('bf')),'ADMIN','ACTIVO','VERIFIED'),
 ('00000000-0000-4000-8000-000000000003','Cliente','Sandbox','Cliente Sandbox','SANDBOX-CLIENT','cliente@sandbox.invalid','1990-01-01','Laboratorio local','sandbox-cliente',crypt('SandboxDemo2026!',gen_salt('bf')),'CLIENTE','ACTIVO','VERIFIED');
UPDATE customers SET document_photo_url='' WHERE username IN ('sandbox-admin','sandbox-cliente');
SQL
psql -v ON_ERROR_STOP=1 -U postgres -d accounts <<'SQL'
INSERT INTO cuentas(id_cuenta,id_cliente,numero_cuenta,tipo_cuenta,saldo_centavos,moneda,estado)
VALUES('00000000-0000-4000-8000-000000000005','00000000-0000-4000-8000-000000000003','900000000003','CORRIENTE',10000000,'GTQ','ACTIVA');
SQL
