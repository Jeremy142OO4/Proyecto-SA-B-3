# Estrategia de pruebas

La validación cubre los flujos funcionales, la comunicación asíncrona, la resiliencia, la trazabilidad y el despliegue completo.

## Pruebas funcionales

- Registro y activación de clientes mediante correo.
- Creación de cuentas y consulta de saldo.
- Depósitos de prueba y actualización de movimientos.
- Transferencias exitosas y rechazo por fondos insuficientes.
- Pagos internos o externos y consulta de su estado.
- Restricción de funciones según rol.

## Pruebas de mensajería y resiliencia

- Reenvío de eventos para comprobar idempotencia.
- Fallos controlados para validar compensaciones de la Saga.
- Retries limitados, backoff y envío a DLQ.
- Recuperación de consumidores sin duplicar operaciones financieras.

## Pruebas de trazabilidad

Cada flujo se inspecciona de extremo a extremo para comprobar que conserva el mismo `correlationId` en comandos, eventos, logs, auditoría y respuestas relacionadas.

## Pruebas de despliegue

El flujo completo se ejecuta en Kubernetes local, verificando frontend, API Gateway, RabbitMQ, los cinco microservicios y las bases PostgreSQL externas al clúster.

## Comandos de verificación

```bash
go test ./...       # dentro de cada módulo Go
npm run build       # frontend
kubectl -n bank-usac get pods
docker compose ps
```

