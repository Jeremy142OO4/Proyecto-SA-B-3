# Transaction Service

Propietario de las transferencias y de la Saga por coreografía. No modifica balances ni consulta bases de datos ajenas: solicita débito, crédito y compensación mediante RabbitMQ.

## Flujo

1. Consume `transfer.requested` (contrato del plan maestro) o `transferencia.solicitada` (compatibilidad interna).
2. Guarda la transferencia `PENDIENTE` y publica `cuenta.debito.solicitado` mediante Outbox.
3. Ante `cuenta.debitada`, pasa a `PROCESANDO` y solicita el crédito.
4. Ante `cuenta.acreditada`, termina `COMPLETADA`.
5. Si el crédito es rechazado después del débito, pasa a `COMPENSANDO` y solicita compensación.
6. Termina `COMPENSADA` o `COMPENSACION_FALLIDA` según la respuesta.

El `idTransferencia` se utiliza como `idOperacion` estable en los tres movimientos. Todos los eventos conservan el `idCorrelacion` original.

## Persistencia y ejecución

PostgreSQL independiente en `localhost:5435`, fuera de Kubernetes. Variables disponibles en `.env.example`. El único endpoint HTTP es `GET /salud`; toda funcionalidad de negocio usa RabbitMQ.

> Los exchanges y sobres en español coinciden con Account Service ya entregado: `banco.comandos`, `banco.eventos` y `banco.fallidos`.
