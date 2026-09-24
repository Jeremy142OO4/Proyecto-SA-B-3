# Infraestructura de Bases de Datos — Bank USAC (Fase 2)

## Propósito

Este documento describe la estrategia de persistencia de datos de Bank USAC en la Fase 2: cada microservicio posee una base de datos PostgreSQL exclusiva ejecutada en una Máquina Virtual (VM) de Google Compute Engine, fuera del clúster de Kubernetes. Este diseño cumple el requisito del proyecto de mantener las bases de datos en VMs y refuerza el principio de aislamiento de datos por servicio.

---

## Principio de diseño

Bank USAC sigue el patrón **Database per Service**: ningún microservicio accede directamente a la base de datos de otro. La comunicación entre servicios se realiza exclusivamente mediante eventos en RabbitMQ, nunca mediante consultas SQL cruzadas ni llaves foráneas entre bases.

```
Customer Service  →  VM: vm-db-customer    (customer_db)
Account Service   →  VM: vm-db-account     (cuentas_db)
Transaction Svc   →  VM: vm-db-transaction (transacciones_db)
Payment Service   →  VM: vm-db-payment     (pagos_db)
Notif. & Audit    →  VM: vm-db-audit       (audit_db)
```

---

## Inventario de bases de datos

| VM | Base de datos | Microservicio propietario | Puerto | Motor |
|---|---|---|---|---|
| `vm-db-customer` | `customer_db` | Customer Service | 5432 | PostgreSQL 17 |
| `vm-db-account` | `cuentas_db` | Account Service | 5432 | PostgreSQL 17 |
| `vm-db-transaction` | `transacciones_db` | Transaction Service | 5432 | PostgreSQL 17 |
| `vm-db-payment` | `pagos_db` | Payment Service | 5432 | PostgreSQL 17 |
| `vm-db-audit` | `audit_db` | Notification & Audit Service | 5432 | PostgreSQL 15 |

---

## Especificación de las VMs

Cada VM de base de datos es provisionada por Terraform con la siguiente configuración:

| Parámetro | Valor |
|---|---|
| Tipo de máquina | `e2-small` (2 vCPU, 2 GB RAM) |
| Sistema operativo | Debian 12 (Bookworm) |
| Disco de arranque | 30 GB SSD (`pd-ssd`) |
| Red | Subred interna `bank-usac-subnet-db` (`10.10.16.0/24`) |
| IP pública | **No** — acceso solo desde la VPC interna |
| Acceso SSH | Solo vía IAP (Identity-Aware Proxy) |

### IPs internas de referencia

| VM | IP interna (ejemplo) |
|---|---|
| `vm-db-customer` | `10.10.16.10` |
| `vm-db-account` | `10.10.16.11` |
| `vm-db-transaction` | `10.10.16.12` |
| `vm-db-payment` | `10.10.16.13` |
| `vm-db-audit` | `10.10.16.14` |

> Las IPs reales se asignan por GCP y se exportan como salidas de Terraform (`db_internal_ips`).

---

## Configuración de PostgreSQL por servicio

### `customer_db` — Customer Service

Almacena usuarios, clientes y el nuevo estado KYC introducido en la Fase 2.

**Tablas principales:**

| Tabla | Descripción |
|---|---|
| `users` | Credenciales de acceso y roles (ADMIN, CAJERO, CLIENTE) |
| `customers` | Datos personales del cliente + `kyc_status` + `kyc_updated_at` ★ |
| `outbox_messages` | Patrón Outbox para publicación confiable de eventos |

**Variables de conexión:**

```bash
POSTGRES_DB=customer_db
POSTGRES_USER=customer_user
POSTGRES_PASSWORD=<secreto-en-GSM>
POSTGRES_HOST=10.10.16.10
POSTGRES_PORT=5432
```

---

### `cuentas_db` — Account Service

Almacena cuentas, movimientos y las nuevas reglas de tipo de cuenta de la Fase 2.

**Tablas principales:**

| Tabla | Descripción |
|---|---|
| `accounts` | Cuentas con `account_type` (AHORRO/CORRIENTE), `min_balance`, `transaction_fee` ★ |
| `movimientos` | Débitos y créditos con `fee_applied` ★ |
| `solicitudes_creacion` | Solicitudes de apertura de cuenta pendientes de aprobación |
| `mensajes_salida` | Patrón Outbox |
| `mensajes_procesados` | Idempotencia de mensajes entrantes |

**Variables de conexión:**

```bash
POSTGRES_DB=cuentas_db
POSTGRES_USER=cuentas_usuario
POSTGRES_PASSWORD=<secreto-en-GSM>
POSTGRES_HOST=10.10.16.11
POSTGRES_PORT=5432
```

---

### `transacciones_db` — Transaction Service

Almacena transferencias y el historial completo con estados intermedios (Fase 2).

**Tablas principales:**

| Tabla | Descripción |
|---|---|
| `transfers` | Transferencias con estados PENDING / APPROVED / FAILED ★ |
| `transfer_events` | Historial de cambios de estado para trazabilidad ★ |
| `idempotency_keys` | Previene procesamiento duplicado de eventos |

**Variables de conexión:**

```bash
POSTGRES_DB=transacciones_db
POSTGRES_USER=transacciones_usuario
POSTGRES_PASSWORD=<secreto-en-GSM>
POSTGRES_HOST=10.10.16.12
POSTGRES_PORT=5432
```

---

### `pagos_db` — Payment Service

Almacena pagos externos y resultados de la simulación (Fase 2).

**Tablas principales:**

| Tabla | Descripción |
|---|---|
| `payments` | Pagos con resultado simulado (SUCCESS / FAILED / TIMEOUT) ★ |
| `external_provider_log` | Registro de llamadas al proveedor externo simulado ★ |
| `outbox_messages` | Patrón Outbox |

**Variables de conexión:**

```bash
POSTGRES_DB=pagos_db
POSTGRES_USER=pagos_usuario
POSTGRES_PASSWORD=<secreto-en-GSM>
POSTGRES_HOST=10.10.16.13
POSTGRES_PORT=5432
```

---

### `audit_db` — Notification & Audit Service

Almacena eventos de auditoría clasificados por severidad (Fase 2).

**Tablas principales:**

| Tabla | Descripción |
|---|---|
| `audit_events` | Eventos con `severity` (INFO / WARNING / ERROR) ★, `correlation_id`, `service_source` |
| `notification_log` | Historial de notificaciones enviadas por correo/SMTP |

**Variables de conexión:**

```bash
POSTGRES_DB=audit_db
POSTGRES_USER=audit_user
POSTGRES_PASSWORD=<secreto-en-GSM>
POSTGRES_HOST=10.10.16.14
POSTGRES_PORT=5432
```

> ★ Campos nuevos introducidos en la Fase 2.

---

## Gestión de migraciones

Las migraciones de esquema se ejecutan con la herramienta [`golang-migrate`](https://github.com/golang-migrate/migrate) (imagen Docker `migrate/migrate:v4.18.3`). Cada microservicio mantiene su propio directorio de migraciones:

```
services/
├── service-customer/database/migrations/
├── account-service/database/migrations/
├── transaction-service/database/migrations/
├── payment-service/database/migrations/
└── service-notification-audit/database/migrations/
```

Las migraciones se ejecutan como un **Job de Kubernetes** durante el despliegue, antes de arrancar el pod del microservicio. El Job se conecta a la VM correspondiente vía la IP interna.

```bash
# Ejecutar migraciones manualmente (dev)
migrate -path ./database/migrations \
        -database "postgres://customer_user:<pass>@10.10.16.10:5432/customer_db?sslmode=disable" \
        up
```

---

## Seguridad y acceso

| Medida | Descripción |
|---|---|
| Sin IP pública | Las VMs de base de datos no tienen interfaz pública; solo accesibles desde la VPC |
| Firewall | Puerto 5432 abierto únicamente desde la subred GKE (`10.10.0.0/20`) hacia la subred DB (`10.10.16.0/24`) |
| SSH vía IAP | Acceso administrativo mediante Identity-Aware Proxy de GCP (no requiere VPN ni bastion host) |
| Credenciales en GSM | Contraseñas almacenadas en Google Secret Manager; Terraform las lee con `data "google_secret_manager_secret_version"` |
| `sslmode=disable` | Solo para entorno de desarrollo dentro de la VPC. En producción se configura `sslmode=require` con certificados autofirmados generados por `pg_ssl` |

---

## Respaldo y recuperación

| Estrategia | Configuración |
|---|---|
| Snapshots de disco GCP | Política de snapshots diarios con retención de 7 días (configurada en Terraform con `google_compute_resource_policy`) |
| `pg_dump` periódico | CronJob de Kubernetes ejecuta `pg_dump` diariamente y sube el dump a un bucket GCS |
| RTO objetivo | < 1 hora (restaurar snapshot + replay de dump) |
| RPO objetivo | < 24 horas (último snapshot exitoso) |

---

## Monitoreo de bases de datos

| Herramienta | Uso |
|---|---|
| Google Cloud Monitoring | Métricas de CPU, memoria y disco de las VMs |
| `pg_stat_activity` | Consultas activas y bloqueos (acceso vía SSH IAP) |
| Alertas GCP | Alerta si disco > 80 % o CPU > 90 % por más de 5 minutos |

---