# Diagrama entidad-relación — Account Service

El diagrama representa la estructura persistente administrada exclusivamente por **Account Service**. Incluye las cuentas bancarias, sus movimientos, las solicitudes de creación y las tablas técnicas utilizadas para idempotencia y publicación confiable de eventos.

> **Fase 2:** Se añaden los campos `account_type`, `min_balance` y `transaction_fee` a la tabla `accounts` para dar soporte a los tipos **AHORRO** y **CORRIENTE**, cada uno con saldo mínimo y comisión por operación diferenciados.

![Diagrama entidad-relación de Account Service](../../Imagenes/ER%20Account%20Service.png)

## Entidades principales

### `accounts`

Almacena cada cuenta bancaria activa o histórica dentro del sistema.

| Columna          | Tipo            | Restricciones                                  | Descripción                                                                 |
|------------------|-----------------|------------------------------------------------|-----------------------------------------------------------------------------|
| `id`             | UUID            | PK, NOT NULL, DEFAULT gen_random_uuid()        | Identificador único de la cuenta (`accountId`).                             |
| `customer_id`    | UUID            | NOT NULL                                       | Referencia lógica al cliente en Customer Service (sin FK inter-BD).         |
| `account_number` | VARCHAR(20)     | NOT NULL, UNIQUE                               | Número de cuenta generado por el sistema.                                   |
| `account_type`   | VARCHAR(20)     | NOT NULL, CHECK IN ('AHORRO','CORRIENTE')       | **[Fase 2]** Tipo de cuenta: AHORRO o CORRIENTE.                            |
| `balance`        | NUMERIC(18,2)   | NOT NULL, DEFAULT 0.00                         | Saldo disponible actual.                                                    |
| `min_balance`    | NUMERIC(18,2)   | NOT NULL, DEFAULT 0.00                         | **[Fase 2]** Saldo mínimo requerido según el tipo de cuenta.                |
| `transaction_fee`| NUMERIC(10,4)   | NOT NULL, DEFAULT 0.0000                       | **[Fase 2]** Comisión aplicada a cada operación de retiro o transferencia.  |
| `currency`       | VARCHAR(3)      | NOT NULL, DEFAULT 'GTQ'                        | Moneda de la cuenta (ISO 4217).                                             |
| `status`         | VARCHAR(20)     | NOT NULL, DEFAULT 'PENDING'                    | Estado de la cuenta: PENDING, ACTIVE, SUSPENDED, CLOSED.                   |
| `created_at`     | TIMESTAMPTZ     | NOT NULL, DEFAULT NOW()                        | Fecha y hora de creación del registro.                                      |
| `updated_at`     | TIMESTAMPTZ     | NOT NULL, DEFAULT NOW()                        | Fecha y hora de la última modificación.                                     |

**Restricciones de negocio (Fase 2):**
- `balance` no puede quedar por debajo de `min_balance` tras una operación de débito.
- Si `account_type = 'CORRIENTE'`, `transaction_fee` puede ser mayor que cero y se descuenta automáticamente al aplicar el movimiento.
- Si `account_type = 'AHORRO'`, `min_balance` define el monto mínimo que debe permanecer intocable.

---

### `movimientos`

Registra cada operación de crédito o débito aplicada sobre una cuenta.

| Columna          | Tipo            | Restricciones                                  | Descripción                                                                  |
|------------------|-----------------|------------------------------------------------|------------------------------------------------------------------------------|
| `id`             | UUID            | PK, NOT NULL, DEFAULT gen_random_uuid()        | Identificador único del movimiento.                                          |
| `account_id`     | UUID            | NOT NULL, FK → accounts(id)                    | Cuenta sobre la que se aplica el movimiento.                                 |
| `transaction_id` | UUID            | NOT NULL                                       | Referencia lógica a la transacción origen en Transaction Service.            |
| `type`           | VARCHAR(10)     | NOT NULL, CHECK IN ('CREDIT','DEBIT')          | Tipo de movimiento.                                                          |
| `amount`         | NUMERIC(18,2)   | NOT NULL, CHECK > 0                            | Monto del movimiento (siempre positivo; el signo lo indica `type`).          |
| `fee_applied`    | NUMERIC(10,4)   | NOT NULL, DEFAULT 0.0000                       | **[Fase 2]** Comisión efectivamente descontada en este movimiento.           |
| `balance_before` | NUMERIC(18,2)   | NOT NULL                                       | Saldo de la cuenta antes de aplicar el movimiento.                          |
| `balance_after`  | NUMERIC(18,2)   | NOT NULL                                       | Saldo de la cuenta después de aplicar el movimiento.                        |
| `description`    | TEXT            | NULLABLE                                       | Descripción o referencia del movimiento.                                     |
| `created_at`     | TIMESTAMPTZ     | NOT NULL, DEFAULT NOW()                        | Fecha y hora del movimiento.                                                 |

---

### `solicitudes_creacion`

Almacena solicitudes de apertura de cuenta mientras se espera validación asíncrona del cliente.

| Columna          | Tipo            | Restricciones                                  | Descripción                                                                  |
|------------------|-----------------|------------------------------------------------|------------------------------------------------------------------------------|
| `id`             | UUID            | PK, NOT NULL, DEFAULT gen_random_uuid()        | Identificador único de la solicitud.                                         |
| `customer_id`    | UUID            | NOT NULL                                       | Cliente que solicita la apertura.                                            |
| `account_type`   | VARCHAR(20)     | NOT NULL, CHECK IN ('AHORRO','CORRIENTE')       | **[Fase 2]** Tipo de cuenta solicitado.                                      |
| `currency`       | VARCHAR(3)      | NOT NULL, DEFAULT 'GTQ'                        | Moneda solicitada.                                                           |
| `initial_deposit`| NUMERIC(18,2)   | NOT NULL, DEFAULT 0.00                         | Depósito inicial indicado en la solicitud.                                   |
| `status`         | VARCHAR(20)     | NOT NULL, DEFAULT 'PENDING'                    | Estado: PENDING, APPROVED, REJECTED.                                         |
| `correlation_id` | UUID            | NOT NULL, UNIQUE                               | Identificador de correlación del flujo saga.                                 |
| `created_at`     | TIMESTAMPTZ     | NOT NULL, DEFAULT NOW()                        | Fecha de creación de la solicitud.                                           |
| `updated_at`     | TIMESTAMPTZ     | NOT NULL, DEFAULT NOW()                        | Fecha de última actualización.                                               |

---

### `mensajes_salida`

Implementa el patrón **Transactional Outbox** para garantizar la publicación confiable de eventos hacia RabbitMQ.

| Columna          | Tipo            | Restricciones                                  | Descripción                                                                  |
|------------------|-----------------|------------------------------------------------|------------------------------------------------------------------------------|
| `id`             | UUID            | PK, NOT NULL, DEFAULT gen_random_uuid()        | Identificador único del mensaje pendiente.                                   |
| `exchange`       | VARCHAR(100)    | NOT NULL                                       | Exchange de RabbitMQ de destino.                                             |
| `routing_key`    | VARCHAR(200)    | NOT NULL                                       | Clave de enrutamiento del mensaje.                                           |
| `payload`        | JSONB           | NOT NULL                                       | Cuerpo del mensaje serializado.                                              |
| `status`         | VARCHAR(20)     | NOT NULL, DEFAULT 'PENDING'                    | Estado: PENDING, SENT, FAILED.                                               |
| `retry_count`    | INTEGER         | NOT NULL, DEFAULT 0                            | Número de intentos de publicación realizados.                                |
| `created_at`     | TIMESTAMPTZ     | NOT NULL, DEFAULT NOW()                        | Fecha de creación del mensaje.                                               |
| `sent_at`        | TIMESTAMPTZ     | NULLABLE                                       | Fecha en que el mensaje fue publicado exitosamente.                          |

---

### `mensajes_procesados`

Garantiza **idempotencia** en el consumo de eventos: registra cada `message_id` procesado para evitar aplicar dos veces el mismo comando.

| Columna          | Tipo            | Restricciones                                  | Descripción                                                                  |
|------------------|-----------------|------------------------------------------------|------------------------------------------------------------------------------|
| `id`             | UUID            | PK, NOT NULL, DEFAULT gen_random_uuid()        | Identificador interno del registro.                                          |
| `message_id`     | UUID            | NOT NULL, UNIQUE                               | Identificador del mensaje ya procesado (proveniente del evento).             |
| `processed_at`   | TIMESTAMPTZ     | NOT NULL, DEFAULT NOW()                        | Fecha y hora en que el mensaje fue procesado.                                |

---

## Relaciones principales

- Una cuenta (`accounts`) puede registrar múltiples movimientos (`movimientos`). Relación: **1 a N**.
- Una solicitud de creación (`solicitudes_creacion`) puede dar origen a una cuenta cuando la validación del cliente finaliza correctamente. Relación: **1 a 0..1**.
- `customer_id` funciona como referencia lógica a Customer Service; no es una clave foránea hacia otra base de datos, preservando el aislamiento entre microservicios.
- `mensajes_procesados` evita aplicar dos veces un mismo comando recibido desde el broker.
- `mensajes_salida` implementa el patrón **Transactional Outbox**: los eventos se insertan en esta tabla dentro de la misma transacción de negocio y un proceso separado los publica hacia RabbitMQ.
- El campo `fee_applied` en `movimientos` registra la comisión efectivamente cobrada al momento del débito, lo que permite auditar y reconstruir el historial de costos operativos por cuenta.

---

## Cambios introducidos en Fase 2

| Campo / Tabla       | Cambio                                        | Motivación                                                              |
|---------------------|-----------------------------------------------|-------------------------------------------------------------------------|
| `accounts.account_type`  | Nuevo campo ENUM: AHORRO / CORRIENTE     | Soporte a tipos de cuenta con reglas diferenciadas (RF-44, RF-45).     |
| `accounts.min_balance`   | Nuevo campo NUMERIC                      | Control de saldo mínimo por tipo de cuenta (RF-44).                    |
| `accounts.transaction_fee` | Nuevo campo NUMERIC                    | Comisión por operación configurable por tipo (RF-45).                  |
| `movimientos.fee_applied`  | Nuevo campo NUMERIC                    | Registro auditable de la comisión aplicada en cada movimiento.         |
| `solicitudes_creacion.account_type` | Nuevo campo ENUM              | La solicitud debe especificar el tipo de cuenta desde el inicio.       |
