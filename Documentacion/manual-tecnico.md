# Manual técnico de Bank USAC

## 1. Alcance

Bank USAC es un sistema bancario académico compuesto por cinco microservicios Go, un API Gateway, un frontend React, RabbitMQ y cinco bases de datos PostgreSQL independientes. La aplicación se ejecuta localmente en Kubernetes (Minikube); las bases de datos se ejecutan como contenedores Docker/Podman fuera del clúster.

Este manual describe la arquitectura implementada, la configuración, el desarrollo, el despliegue y las verificaciones técnicas. Para instrucciones orientadas a los usuarios finales debe consultarse `manual-usuario.md`.

## 2. Estructura del repositorio

```text
gateway/api-gateway/                 API Gateway Go/Fiber
services/service-customer/           Clientes y autenticación
services/account-service/            Cuentas y movimientos
services/transaction-service/        Transferencias y Saga
services/payment-service/            Pagos
services/service-notification-audit/ Notificaciones y auditoría
frontend/bank-usac-web/              React + Vite + Nginx
infrastructure/kubernetes/           Namespace, Deployments, Services y Secrets
Documentacion/                        Diagramas y documentación Markdown
docker-compose.yml                    PostgreSQL y RabbitMQ para desarrollo
```

Cada servicio mantiene las capas `config`, `controllers`, `database`, `events`, `messaging`, `middleware`, `models`, `repositories`, `routes` y `services` según sus necesidades. Los nombres de carpetas están en inglés y las variables y funciones de negocio en español cuando corresponde.

## 3. Requisitos locales

- Docker o Podman con Compose.
- Minikube y kubectl.
- Go 1.22 o superior para desarrollo.
- Node.js y npm para el frontend.
- Una cuenta SMTP para probar correos de activación (la configuración actual usa Gmail con contraseña de aplicación).

No se deben subir archivos `.env`, contraseñas SMTP, API keys ni secretos JWT al repositorio.

## 4. Configuración

El despliegue utiliza el Secret `bank-usac-secrets` para RabbitMQ, URLs PostgreSQL y `JWT_SECRET`. `api-gateway`, `customer-service` y `notification-audit-service` deben recibir exactamente el mismo JWT secret para que los tokens firmados puedan validarse en todo el flujo.

`service-notification-audit/.env` define, en desarrollo local, `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_APP_PASSWORD` y `SMTP_FROM`. El script de despliegue copia esos valores al Secret de Kubernetes. La URL de activación se configura con `ACTIVATION_LINK_BASE` y, para una demostración en la laptop, debe ser `http://localhost:3000/activar`.

## 5. Ejecución con Docker Compose

Desde la raíz del repositorio:

```bash
docker compose up -d
docker compose ps
```

Compose levanta RabbitMQ y las bases PostgreSQL. Las migraciones de cada servicio se ejecutan mediante sus contenedores de migración. Las bases se exponen en los puertos 5432 (clientes), 5433 (cuentas), 5434 (pagos), 5435 (transacciones) y 5436 (auditoría).

## 6. Despliegue en Kubernetes

El script integral construye las imágenes, inicia Minikube, crea el Secret, aplica los manifiestos y espera el estado de cada Deployment:

```bash
chmod +x infrastructure/kubernetes/deploy.sh
infrastructure/kubernetes/deploy.sh
kubectl -n bank-usac get pods,services
```

El namespace es `bank-usac`. Dentro del clúster se ejecutan frontend, Gateway, RabbitMQ y los cinco microservicios. Las bases continúan fuera del clúster y son accesibles desde los Pods mediante `host.minikube.internal`.

Para acceder al frontend localmente:

```bash
kubectl -n bank-usac port-forward svc/frontend 3000:80
```

Después se abre `http://localhost:3000`. El Gateway escucha internamente en el puerto 8080 y RabbitMQ utiliza AMQP 5672; el panel de RabbitMQ usa 15672.

## 7. Comunicación y eventos

El Gateway publica comandos en `banco.comandos`. Los microservicios publican eventos en `banco.eventos` mediante Outbox y consumen con confirmación manual, idempotencia y DLQ. Las respuestas correlacionadas permiten que el Gateway actualice `/api/operaciones/:id`.

Eventos relevantes:

- `cliente.registro.solicitado` y `notificacion.correo-activacion.solicitado`.
- `cuenta.creacion.solicitada`, `cuenta.credito.solicitado`, `cuenta.debitada` y `cuenta.acreditada`.
- `transferencia.solicitada`, `transferencia.completada`, `transferencia.rechazada` y eventos de compensación.
- `pago.procesamiento.solicitado`, `pago.completado` y `pago.rechazado`.

El correo de activación es la notificación externa exigida: Customer Service genera el enlace y Notification & Audit Service lo envía por SMTP y registra el resultado `SENT` en su base.

## 8. API principal

Todas las rutas de negocio se consumen a través del Gateway y requieren `Authorization: Bearer <JWT>` salvo login y activación:

| Método | Ruta | Propósito |
|---|---|---|
| POST | `/api/clientes/login` | Iniciar sesión y obtener JWT |
| POST | `/api/clientes/registro` | Registrar cliente (TELLER) |
| GET | `/api/clientes/activacion?token=...` | Activar cliente |
| GET | `/api/cuentas` | Listar cuentas propias |
| POST | `/api/cuentas` | Crear cuenta (TELLER) |
| POST | `/api/cuentas/:idCuenta/deposito` | Acreditar fondos de prueba |
| GET | `/api/cuentas/:idCuenta/movimientos` | Consultar movimientos |
| POST | `/api/transferencias` | Solicitar transferencia |
| POST | `/api/pagos` | Solicitar pago |
| GET | `/api/auditoria/notificaciones` | Consultar notificaciones (ADMIN) |

El depósito de prueba usa el comando asíncrono `cuenta.credito.solicitado`; no modifica directamente la base desde el Gateway.

## 9. Roles y datos de demostración

Los usuarios iniciales se crean mediante la migración de Customer Service:

| Rol | Correo | Usuario | Contraseña |
|---|---|---|---|
| ADMIN | `admin@ejemplo.com` | `admin` | `Admin123!` |
| TELLER | `teller@ejemplo.com` | `teller` | `Teller123!` |

Las contraseñas son únicamente para demostración local y deben cambiarse en cualquier entorno real.

## 10. Verificación técnica

Antes de entregar se recomienda ejecutar:

```bash
go test ./...                         # dentro de cada módulo Go
npm run build                         # frontend
kubectl -n bank-usac get pods         # todos Running/Ready
kubectl -n bank-usac get secret bank-usac-secrets
```

El flujo mínimo de aceptación es: login TELLER, registro, envío SMTP, activación, login CLIENTE, creación de cuenta, depósito de prueba, consulta de saldo, pago rechazado por fondos insuficientes, transferencia y consulta de auditoría. Para una transferencia exitosa debe existir saldo suficiente en la cuenta de origen.

## 11. Diagnóstico rápido

- **Pod no inicia:** revisar `kubectl describe pod` y que existan todas las claves del Secret.
- **Correo no llega:** revisar Spam/Promociones y los logs de `notification-audit-service`; `SENT` significa que el servidor SMTP aceptó el mensaje.
- **JWT rechazado:** comprobar que Gateway, Customer y Notification usen el mismo `JWT_SECRET`.
- **Operación pendiente:** consultar RabbitMQ, la DLQ y el `correlationId` en los logs de los consumidores.
- **Base no accesible:** verificar que Compose esté activo y que el manifiesto use `host.minikube.internal` con el puerto correcto.

## 12. Documentación relacionada

- Arquitectura: `arquitectura/c4-contexto.md`, `c4-contenedores.md`, `c4-componentes.md` y `despliegue.md`.
- Dominio: `dominio/modelo-dominio.md`, `dominio/contextos-delimitados.md` y diagramas ER.
- Eventos: `eventos/catalogo-eventos.md` y `eventos/contratos-eventos.md`.
- Saga y SRE: `saga/saga-transferencia.md` y `sre/sli.md`, `slo.md`, `sla.md`.
- Casos de uso y secuencias: `uml/casos-de-uso.md` y los diagramas de secuencia.
