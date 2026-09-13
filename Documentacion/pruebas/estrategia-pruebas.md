# Estrategia de pruebas

La validación cubre los flujos funcionales, la comunicación asíncrona, la resiliencia, la trazabilidad y el despliegue completo.

## Pruebas funcionales

- Registro y activación de clientes mediante correo.
- Creación de cuentas y consulta de saldo.
- Depósitos de prueba y actualización de movimientos.
- Transferencias exitosas y rechazo por fondos insuficientes.
- Pagos internos o externos y consulta de su estado.
- Pagos externos simulados con resultados `EXITO`, `FALLO` y `TIMEOUT`.
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

## Verificación de la simulación de pagos externos

La prueba de integración de Payment Service crea los tres escenarios contra PostgreSQL y procesa los eventos de débito y compensación mediante los mismos métodos utilizados por los consumidores AMQP.

| Escenario | Estado esperado | Código esperado | Saldo esperado |
|---|---|---|---|
| `EXITO` | `COMPLETADO` | `OK` | El débito permanece aplicado. |
| `FALLO` | `RECHAZADO` | `PROVEEDOR_EXTERNO` | El débito es compensado. |
| `TIMEOUT` | `RECHAZADO` | `TIMEOUT_PROVEEDOR` | El débito es compensado. |

La prueba automatizada se ejecuta con `URL_BASE_DATOS_PRUEBAS` y el caso `TestIntegracionEscenariosSimulados`.

## Verificación de la Saga de fase 2

El caso `TestIntegracionSagaFase2` comprueba contra PostgreSQL las transiciones de KYC, validación de cuentas, débito, crédito y compensación. Se verifican tres escenarios:

- `EXITO`: termina `COMPLETADA` después del crédito.
- `FALLO`: registra `FALLO_EXTERNO` y termina `COMPENSADA`.
- `TIMEOUT`: registra `TIMEOUT_EXTERNO` y termina `COMPENSADA`.

Las pruebas integrales también deben confirmar que KYC no verificado y una cuenta origen ajena se rechazan antes del débito, que el saldo se conserva tras una compensación y que toda la traza mantiene el mismo `idCorrelacion`.

