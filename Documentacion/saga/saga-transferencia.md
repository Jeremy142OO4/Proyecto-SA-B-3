# Saga de transferencia bancaria

La transferencia bancaria es una operación distribuida que requiere debitar una cuenta origen y acreditar una cuenta destino. Como cada microservicio posee su propia base de datos y no se permiten transacciones distribuidas ni comunicación síncrona entre servicios, Bank USAC utiliza una **Saga orquestada mediante eventos asíncronos**.

`transaction-service` actúa como coordinador lógico de la Saga. Mantiene el estado general de la transferencia, publica comandos mediante RabbitMQ y procesa los eventos generados por `account-service`. No modifica directamente el saldo de ninguna cuenta.

## Objetivo

La Saga debe garantizar que una transferencia finalice en uno de estos resultados:

- **Completada:** el monto se debitó de la cuenta origen y se acreditó en la cuenta destino.
- **Rechazada:** no se aplicó ningún cambio financiero o el cambio parcial fue compensado.
- **Compensación fallida:** se detectó un cambio parcial que no pudo revertirse automáticamente y requiere revisión.

## Participantes

| Participante | Responsabilidad |
|---|---|
| API Gateway | Recibir la solicitud externa y publicar `transferencia.solicitada`. |
| RabbitMQ | Transportar comandos y eventos de forma asíncrona. |
| Transaction Service | Registrar la transferencia, coordinar la Saga y conservar su estado. |
| Customer Service | Confirmar que el cliente está activo y que su KYC está `VERIFIED`. |
| Account Service | Validar cuentas y aplicar débito, crédito o compensación. |
| Notification & Audit Service | Registrar los eventos y generar notificaciones cuando corresponda. |

## Datos de la transferencia

| Campo | Propósito |
|---|---|
| `idTransferencia` | Identificador único de la transferencia. |
| `idCliente` | Cliente que inicia la operación. |
| `idCuentaOrigen` | Cuenta desde la que se retiran los fondos. |
| `idCuentaDestino` | Cuenta que recibe los fondos. |
| `montoCentavos` | Monto positivo expresado en centavos. |
| `moneda` | Moneda de la operación; inicialmente `GTQ`. |
| `estado` | Situación actual de la Saga. |
| `idCorrelacion` | Identificador compartido por todos los mensajes del flujo. |
| `codigoError` | Motivo técnico o de negocio cuando la operación no finaliza exitosamente. |
| `resultadoExternoSimulado` | Escenario controlado `EXITO`, `FALLO` o `TIMEOUT` usado para demostrar compensación. |

## Estados

| Estado | Significado | Estado terminal |
|---|---|---:|
| `VALIDANDO_KYC` | Customer Service está validando el estado KYC del cliente. | No |
| `VALIDANDO_CUENTAS` | Account Service está validando propiedad, estado y tipos de ambas cuentas. | No |
| `PENDIENTE` | La transferencia fue registrada y todavía no inició el débito. | No |
| `PROCESANDO` | La Saga está ejecutando débito o crédito. | No |
| `COMPLETADA` | Débito y crédito finalizaron correctamente. | Sí |
| `RECHAZADA` | La operación no pudo ejecutarse sin dejar cambios pendientes. | Sí |
| `COMPENSANDO` | El crédito falló después del débito y se solicitó revertirlo. | No |
| `COMPENSADA` | El débito previo fue revertido correctamente. | Sí |
| `COMPENSACION_FALLIDA` | No fue posible revertir automáticamente el débito. | Sí, requiere revisión |

## Validaciones iniciales

Antes de registrar la transferencia se comprueba que:

1. `idTransferencia`, `idCliente`, `idCuentaOrigen` e `idCuentaDestino` sean válidos.
2. La cuenta origen y la cuenta destino sean diferentes.
3. `montoCentavos` sea mayor que cero.
4. La moneda sea `GTQ`.
5. El escenario externo sea `EXITO`, `FALLO` o `TIMEOUT`.
6. La operación no haya sido registrada previamente.

Después del registro, Customer Service valida KYC y Account Service valida propiedad, estado y tipo de las cuentas antes de permitir cualquier movimiento financiero.

## Flujo exitoso

| Paso | Responsable | Mensaje o acción | Resultado |
|---:|---|---|---|
| 1 | API Gateway | Publica `transferencia.solicitada`. | Inicia la operación asíncrona. |
| 2 | Transaction Service | Registra la transferencia como `VALIDANDO_KYC` y publica `cliente.kyc.validacion.solicitada`. | La solicitud queda persistida sin mover fondos. |
| 3 | Customer Service | Comprueba acceso activo y KYC `VERIFIED`; publica `cliente.kyc.verificado`. | Autoriza continuar. |
| 4 | Transaction Service | Cambia a `VALIDANDO_CUENTAS` y publica `cuenta.transferencia.validacion.solicitada`. | Solicita validación financiera. |
| 5 | Account Service | Valida propiedad de la cuenta origen, estados y tipos de ambas cuentas; publica `cuenta.transferencia.validada`. | Autoriza el débito. |
| 6 | Transaction Service | Cambia a `PENDIENTE` y publica `cuenta.debito.solicitado`. | Inicia el movimiento. |
| 7 | Account Service | Valida fondos, aplica el débito y publica `cuenta.debitada`. | El saldo origen disminuye. |
| 8 | Transaction Service | Para `EXITO`, cambia a `PROCESANDO` y publica `cuenta.credito.solicitado`. | Inicia el crédito. |
| 9 | Account Service | Acredita la cuenta destino y publica `cuenta.acreditada`. | El saldo destino aumenta. |
| 10 | Transaction Service | Cambia a `COMPLETADA` y publica `transferencia.completada`. | Finaliza la Saga. |
| 11 | Notification & Audit Service | Registra la traza y la notificación final. | El resultado queda auditable. |

Todos los comandos y eventos conservan el mismo `idCorrelacion`. Cada evento derivado puede utilizar como `idCausa` el identificador del mensaje anterior.

## Fallo antes del débito

Si Account Service determina que la cuenta origen no existe, no está activa, el monto es inválido o no hay fondos suficientes:

1. No modifica el balance.
2. Publica `cuenta.debito.rechazado` con el código y motivo.
3. Transaction Service cambia la transferencia a `RECHAZADA`.
4. Publica `transferencia.rechazada`.
5. No se ejecuta compensación porque no existe un cambio parcial.

## Fallo durante el crédito

Si el débito fue aplicado, pero Account Service rechaza el crédito de la cuenta destino:

1. Account Service publica `cuenta.credito.rechazado`.
2. Transaction Service cambia el estado a `COMPENSANDO`.
3. Transaction Service publica `transferencia.compensando`.
4. Registra en Outbox `cuenta.compensacion.solicitada` para la cuenta origen.
5. Account Service procesa la compensación como un movimiento independiente e idempotente.

La compensación no elimina el movimiento original. Registra un nuevo movimiento que devuelve el monto a la cuenta origen y conserva la trazabilidad financiera.

## Compensación exitosa

Cuando Account Service revierte el débito:

1. Publica `cuenta.compensada`.
2. Transaction Service actualiza la transferencia a `COMPENSADA`.
3. Publica `transferencia.compensada`.
4. El usuario puede consultar que la transferencia no se completó, pero los fondos fueron devueltos.

## Compensación fallida

Cuando Account Service no puede revertir el débito:

1. Publica `cuenta.compensacion.rechazada`.
2. Transaction Service cambia el estado a `COMPENSACION_FALLIDA`.
3. Publica `transferencia.compensacion.fallida`.
4. Notification & Audit Service conserva la secuencia completa de eventos.
5. La operación debe ser revisada manualmente utilizando el `idCorrelacion`.

No se marca la transferencia como completada ni compensada cuando existe incertidumbre sobre el saldo.

## Matriz de fallos y compensaciones

| Punto de fallo | Cambio aplicado | Evento recibido | Acción | Estado final esperado |
|---|---|---|---|---|
| KYC pendiente o rechazado | Ninguno | `cliente.kyc.rechazado` | Rechazar antes del débito. | `RECHAZADA` |
| Cuenta ajena, inactiva o tipo no admitido | Ninguno | `cuenta.transferencia.rechazada` | Rechazar antes del débito. | `RECHAZADA` |
| Fallo externo simulado | Débito en origen | Escenario `FALLO` | Registrar `FALLO_EXTERNO` y compensar. | `COMPENSADA` o `COMPENSACION_FALLIDA` |
| Timeout externo simulado | Débito en origen | Escenario `TIMEOUT` | Registrar `TIMEOUT_EXTERNO` y compensar. | `COMPENSADA` o `COMPENSACION_FALLIDA` |
| Validación inicial | Ninguno | No aplica | Rechazar la solicitud. | `RECHAZADA` |
| Débito | Ninguno | `cuenta.debito.rechazado` | Finalizar sin compensar. | `RECHAZADA` |
| Crédito | Débito en origen | `cuenta.credito.rechazado` | Solicitar compensación del débito. | `COMPENSADA` o `COMPENSACION_FALLIDA` |
| Compensación | Débito pendiente de revertir | `cuenta.compensacion.rechazada` | Registrar incidente para revisión. | `COMPENSACION_FALLIDA` |
| RabbitMQ o consumidor temporalmente indisponible | Depende de la etapa | Timeout o ausencia temporal de respuesta | Reintentar sin duplicar efectos. | Se mantiene el estado no terminal |
| Mensaje malformado o no recuperable | Ninguno o estado conocido | Error permanente | Enviar a DLQ y registrar el fallo. | Según la última etapa confirmada |

## Idempotencia

RabbitMQ puede entregar un mensaje más de una vez. Por ello:

- Cada comando y evento posee un `idMensaje` único.
- Cada consumidor registra la combinación de mensaje y nombre del consumidor.
- Account Service identifica cada movimiento mediante `idOperacion` y tipo de movimiento.
- Transaction Service no vuelve a registrar una transferencia existente.
- Un evento repetido no debe avanzar dos veces el estado ni aplicar un segundo movimiento.

## Reintentos y DLQ

Los fallos transitorios, como una desconexión temporal de la base de datos, utilizan reintentos limitados. El mensaje conserva sus identificadores durante cada intento. Cuando se alcanza el máximo configurado, se envía a la DLQ del servicio correspondiente.

Los errores de negocio, como fondos insuficientes, no deben reintentarse porque repetir el mensaje no modifica la condición que provocó el rechazo.

## Consistencia

Cada modificación local se ejecuta mediante una transacción ACID dentro de la base del servicio propietario. Entre servicios se utiliza consistencia eventual. Durante el procesamiento, la transferencia puede permanecer temporalmente en `PENDIENTE`, `PROCESANDO` o `COMPENSANDO`.

El patrón Outbox evita guardar un cambio de estado sin conservar también la intención de publicar el siguiente mensaje. La auditoría por `idCorrelacion` permite reconstruir el orden de toda la Saga.

## Criterios de aceptación

- Una transferencia exitosa produce exactamente un débito y un crédito.
- Una transferencia sin fondos no modifica ninguna cuenta.
- Un crédito rechazado después del débito inicia obligatoriamente la compensación.
- Una compensación exitosa devuelve el monto completo a la cuenta origen.
- Los mensajes duplicados no producen movimientos duplicados.
- Todo el flujo conserva el mismo `idCorrelacion`.
- Los estados y eventos coinciden con la última etapa confirmada.
- Ninguna etapa utiliza HTTP ni accede directamente a la base de otro microservicio.

