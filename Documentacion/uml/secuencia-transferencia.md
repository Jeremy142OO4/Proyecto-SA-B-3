# Secuencia de transferencia bancaria

Flujo de transferencia, Saga, fallos y compensaciones.

![Diagrama de secuencia transferencia](../Imagenes/secuencia%20transferencia.png)

La transferencia utiliza una Saga coordinada lógicamente por Transaction Service y ejecutada con eventos asíncronos; no existen llamadas HTTP entre microservicios.

El cliente ingresa las cuentas y el monto. API Gateway publica `transferencia.solicitada` con el `idCorrelacion`.

Transaction Service registra `VALIDANDO_KYC` y publica `cliente.kyc.validacion.solicitada`. Customer Service responde `cliente.kyc.verificado` o `cliente.kyc.rechazado`.

Con KYC aprobado, Transaction Service registra `VALIDANDO_CUENTAS` y publica `cuenta.transferencia.validacion.solicitada`. Account Service valida propiedad, estado, tipo y fondos preliminares, y responde `cuenta.transferencia.validada` o `cuenta.transferencia.rechazada`.

Con ambas validaciones aprobadas, Transaction Service cambia a `PENDIENTE` y solicita `cuenta.debito.solicitado`. Account Service publica `cuenta.debitada` o `cuenta.debito.rechazado`.

Para `EXITO`, Transaction Service pasa a `PROCESANDO`, solicita el crédito y finaliza como `COMPLETADA` al recibir `cuenta.acreditada`. Para `FALLO` o `TIMEOUT`, registra `COMPENSANDO`, solicita `cuenta.compensacion.solicitada` y finaliza como `COMPENSADA` o `COMPENSACION_FALLIDA`.

Notification & Audit Service registra cada evento y genera la notificación final. El historial de transiciones y la auditoría permiten reconstruir la operación completa.
