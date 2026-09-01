# Entrega de Persona 2: cuentas, pagos y frontend

## Alcance terminado

- **Account Service** en Go y Fiber, con PostgreSQL independiente y RabbitMQ.
- **Payment Service** en Go y Fiber, con PostgreSQL independiente y RabbitMQ.
- Frontend React organizado por funcionalidades para cuentas, movimientos y pagos.
- Dockerfiles, bases externas en Compose y componentes ejecutables desplegados en Kubernetes.
- Idempotencia de consumidores, patrón Outbox, reintentos limitados, DLQ y propagación de `correlationId`.
- Compensación de pagos externos fallidos y desactivación automática de cuentas inactivas.

## Contratos que debe implementar el API Gateway

El frontend usa exclusivamente estas rutas. El Gateway debe autenticar, autorizar y traducir cada petición a comandos o consultas asíncronas de RabbitMQ:

| Método | Ruta | Uso |
|---|---|---|
| GET | `/api/cuentas` | Listar cuentas del cliente autenticado |
| POST | `/api/cuentas` | Solicitar creación de cuenta |
| GET | `/api/cuentas/:idCuenta` | Consultar una cuenta y su saldo |
| GET | `/api/cuentas/:idCuenta/movimientos` | Consultar movimientos |
| GET | `/api/pagos` | Consultar historial de pagos |
| POST | `/api/pagos` | Solicitar un pago |
| GET | `/api/pagos/:idPago` | Consultar estado de un pago |

Las operaciones de escritura deben responder `202 Accepted` con el identificador de la operación. Las consultas deben esperar la respuesta correlacionada por RabbitMQ con un timeout controlado; el Gateway no debe conectarse a las bases de datos de los microservicios.

## Integraciones RabbitMQ requeridas a otros compañeros

### Customer Service

Debe consumir `cliente.validacion.solicitada` y publicar uno de estos eventos conservando el mismo `correlationId`:

- `cliente.validado`, cuando el cliente existe, está activo y puede tener cuentas.
- `cliente.rechazado`, incluyendo un motivo cuando no cumple las condiciones.

### Transaction Service

Para modificar saldos debe publicar los comandos de Account Service y escuchar sus resultados:

- Comandos: `cuenta.debito.solicitado`, `cuenta.credito.solicitado`, `cuenta.compensacion.solicitada`.
- Resultados: `cuenta.debitada`, `cuenta.debito.rechazado`, `cuenta.acreditada`, `cuenta.credito.rechazado`, `cuenta.compensada`, `cuenta.compensacion.rechazada`.

Cada movimiento debe tener un `idOperacion` estable. Reenviar el mismo mensaje o la misma operación no debe modificar el saldo dos veces.

### Notification & Audit Service

Debe enlazarse a los eventos de dominio de cuentas y pagos para guardar auditoría y generar notificaciones, sin llamar por HTTP a estos servicios.

## Reglas de datos importantes

- Los montos viajan como enteros en **centavos** y la moneda actual es `GTQ`.
- Los identificadores son UUID.
- Una cuenta no puede quedar con saldo negativo.
- Una cuenta activa se desactiva tras seis meses sin actividad si posee menos de Q50.00.
- Los pagos son `INTERNO` o `EXTERNO`; sus estados son `PROCESANDO`, `COMPENSANDO`, `COMPLETADO` o `RECHAZADO`.
- No se comparten tablas ni llaves foráneas entre bases de datos de distintos microservicios.

## Ejecución local

Las bases externas se levantan desde la raíz:

```bash
docker compose up -d
```

Los componentes se despliegan en Minikube con:

```bash
./infrastructure/kubernetes/deploy.sh
```

Servicios de Persona 2:

- Kubernetes: RabbitMQ, Account Service, Payment Service y frontend.
- Docker/Podman: PostgreSQL de cuentas en `5433` y PostgreSQL de pagos en `5434`.

El frontend mostrará errores de negocio mientras no exista el API Gateway; esto es esperado y no implica que React esté caído.

## Pendientes del equipo, fuera de Persona 2

- Customer Service, Transaction Service y Notification & Audit Service.
- API Gateway con JWT, roles y validación de propiedad del recurso.
- Incorporar esos componentes al Compose general.
- Despliegue Kubernetes y documentación/diagramas finales compartidos.
