# Modelo de dominio

El modelo de dominio de **Bank USAC** describe los conceptos principales del negocio bancario, sus atributos, relaciones, estados y responsabilidades. El modelo se divide según los cinco microservicios del sistema para que cada concepto tenga un propietario claro y no sea modificado directamente desde otros dominios.

Este documento presenta una vista conceptual. Los diagramas entidad-relación de cada servicio detallan posteriormente la estructura física de sus tablas.


## Entidades principales

### Cliente

Representa a una persona o usuario registrado en el banco. Customer Service es responsable de su identidad, información personal, credenciales, rol, estado y estado KYC.

Roles admitidos:

- `ADMIN`: Administrador.
- `TELLER`: Cajero Receptor.
- `CLIENTE`: Cliente bancario.

Estados de acceso admitidos:

- `PENDIENTE_ACTIVACION`.
- `ACTIVO`.
- `BLOQUEADO`.

Estados KYC admitidos (Fase 2):

- `PENDING`: el cliente aún no ha completado el proceso de validación KYC.
- `VERIFIED`: el cliente ha sido verificado y puede realizar transferencias.
- `REJECTED`: la validación KYC fue rechazada; el cliente no puede transferir.

El estado de acceso y el estado KYC son independientes. Solamente un cliente con estado `ACTIVO` y KYC `VERIFIED` puede avanzar en la Saga de transferencia. El documento de identificación, correo y username deben ser únicos. La contraseña nunca se almacena en texto plano, sino como un hash.

### Token de activación

Representa el enlace de un solo uso enviado durante la activación del usuario. Pertenece a un cliente, posee una fecha de expiración y registra si ya fue utilizado. El valor original no se almacena directamente; se conserva su hash.

### Cuenta

Representa una cuenta bancaria asociada lógicamente con un cliente. Account Service es propietario de su número, tipo, reglas de negocio, saldo, moneda, estado y actividad financiera.

Tipos admitidos (Fase 2 extiende el modelo):

- `AHORRO`: cuenta de ahorro; aplica límite de saldo mínimo. No permite quedar por debajo del saldo mínimo configurado.
- `CORRIENTE`: cuenta corriente; puede aplicar comisión opcional por transacción. Mayor flexibilidad operativa.

Estados admitidos:

- `ACTIVA`.
- `INACTIVA`.
- `BLOQUEADA`.
- `CERRADA`.

Reglas de negocio por tipo (Fase 2):

- `min_balance`: saldo mínimo requerido, expresado en centavos. Se aplica a cuentas de tipo `AHORRO`.
- `transaction_fee`: comisión por transacción en centavos (opcional). Se aplica a cuentas de tipo `CORRIENTE` cuando está configurada.

El saldo se expresa en centavos y no puede quedar negativo ni por debajo del saldo mínimo del tipo de cuenta. La versión permite proteger las actualizaciones concurrentes.

### Movimiento de cuenta

Representa la evidencia de un débito, crédito o compensación aplicada sobre una cuenta. Conserva el saldo anterior, el nuevo saldo, el monto y los identificadores necesarios para idempotencia y trazabilidad.

Tipos admitidos:

- `DEBITO`.
- `CREDITO`.
- `COMPENSACION`.

### Solicitud de creación de cuenta

Representa el proceso asíncrono utilizado para crear una cuenta después de comprobar que el cliente exista y esté activo. Puede quedar `PENDIENTE_VALIDACION`, `COMPLETADA` o `RECHAZADA`.

### Transferencia

Representa el traslado de fondos entre dos cuentas diferentes. Transaction Service conserva el estado general de la operación y coordina la Saga necesaria para ejecutar el débito, el crédito y una eventual compensación.

Estados admitidos (Fase 2 incorpora estados intermedios):

- `PENDING`: la solicitud fue recibida y registrada.
- `PENDIENTE`: estado inicial de la Saga antes de iniciar validaciones.
- `VALIDANDO_KYC`: se está verificando el estado KYC del cliente origen (Fase 2).
- `VALIDANDO_CUENTAS`: se está verificando el tipo y reglas de las cuentas involucradas (Fase 2).
- `PROCESANDO`: la Saga está ejecutando el débito y el crédito.
- `APPROVED`: la transferencia fue aprobada y completada exitosamente (Fase 2).
- `FAILED`: la transferencia falló en alguna etapa (Fase 2).
- `COMPLETADA`: el proceso finalizó correctamente.
- `RECHAZADA`: la Saga determinó que la operación no puede ejecutarse.
- `COMPENSANDO`: se está revirtiendo un débito ya aplicado.
- `COMPENSADA`: la compensación se aplicó correctamente.
- `COMPENSACION_FALLIDA`: el intento de compensación no pudo completarse.

### Pago

Representa una instrucción de pago realizada desde una cuenta. Payment Service conserva el beneficiario, concepto, monto, moneda, tipo, resultado externo simulado, referencia externa y estado del proceso.

Los pagos pueden ser `INTERNO` o `EXTERNO` y pasan por estados como `PENDIENTE`, `PROCESANDO`, `COMPENSANDO`, `COMPLETADO` o `RECHAZADO`.

En un pago externo, `resultadoSimulado` admite `EXITO`, `FALLO` y `TIMEOUT`. Un fallo o timeout posterior al débito obliga a compensar la cuenta antes de cerrar el pago como rechazado.

### Intento de pago

Representa cada intento de procesar un pago. Permite conservar el número de intento, respuesta obtenida, detalle del error y tiempos de ejecución sin sobrescribir la historia de intentos anteriores.

### Evento de auditoría

Es una copia inmutable de un evento relevante ocurrido en cualquier dominio. Conserva el identificador del evento, productor, tipo, versión, payload y fechas. El `idCorrelacion` permite reconstruir una operación distribuida completa.

Clasificación de severidad (Fase 2):

- `INFO`: operaciones exitosas y flujos normales (por ejemplo, transferencia completada, pago procesado).
- `WARNING`: rechazos recuperables, compensaciones ejecutadas o reintentos (por ejemplo, KYC rechazado, pago con fallo externo y compensado).
- `ERROR`: fallos no recuperables, timeouts sin compensación posible o mensajes en DLQ.

La clasificación es asignada por Notification & Audit Service al consumir cada evento; no modifica la routing key original del evento.

### Notificación

Representa el historial de una comunicación dirigida a un usuario. Conserva el destinatario, tipo, asunto, resumen seguro del contenido, severidad del evento que la originó y resultado del envío. Sus estados son `PENDING`, `SENT` y `FAILED`.

La generación de la notificación se adapta según la severidad del evento: los eventos `INFO` generan notificaciones informativas, los `WARNING` generan alertas y los `ERROR` generan notificaciones de error que pueden requerir acción del usuario o del administrador.

## Reglas e invariantes del dominio

1. Cada cliente posee un identificador único y sus datos de identidad sujetos a unicidad no pueden repetirse.
2. Solamente un usuario activo puede utilizar las funciones protegidas del sistema.
3. Una cuenta pertenece lógicamente a un único cliente, aunque un cliente puede poseer varias cuentas.
4. Toda cuenta utiliza inicialmente la moneda `GTQ`.
5. Los montos se representan mediante centavos enteros y deben ser mayores que cero para operaciones financieras.
6. Un débito solo puede aplicarse sobre una cuenta habilitada y con fondos suficientes.
7. El saldo de una cuenta nunca puede quedar negativo.
8. Cada movimiento conserva el saldo anterior y el saldo resultante.
9. La cuenta origen y la cuenta destino de una transferencia deben ser diferentes.
10. Toda transferencia conserva un estado que representa el avance o resultado de su Saga.
11. Un pago registra al menos beneficiario, concepto, monto, tipo y estado.
12. Un mensaje repetido no debe volver a aplicar una operación financiera.
13. Los eventos relacionados con una misma operación conservan el mismo `idCorrelacion`.
14. Los datos sensibles, credenciales y tokens completos no forman parte de logs ni payloads de auditoría.
15. Solamente un cliente con estado KYC `VERIFIED` puede avanzar en la Saga de transferencia (Fase 2).
16. Un débito sobre una cuenta `AHORRO` no puede dejar el saldo por debajo del `min_balance` configurado (Fase 2).
17. Si el tipo de cuenta `CORRIENTE` tiene `transaction_fee` configurada, la comisión se descuenta del saldo origen en la misma operación del débito (Fase 2).
18. Todo evento procesado por Notification & Audit Service debe clasificarse como `INFO`, `WARNING` o `ERROR` antes de generar la notificación (Fase 2).
19. Un pago externo simulado con resultado `FALLO` o `TIMEOUT` que ya aplicó un débito debe compensar la cuenta antes de cerrar el pago como rechazado (Fase 2).

## Elementos técnicos de soporte

Además de las entidades del negocio, los servicios utilizan elementos técnicos para garantizar la comunicación confiable:

- **Mensaje de salida:** conserva un comando o evento pendiente de publicar como parte del patrón Outbox.
- **Mensaje procesado:** registra qué consumidor ya atendió un mensaje y garantiza idempotencia.
- **Identificador de correlación:** relaciona los mensajes producidos durante una operación distribuida.
- **Identificador de causación:** indica cuál mensaje o acción produjo un evento posterior.

Estos elementos respaldan el dominio, pero no sustituyen las entidades bancarias principales.
El estado de acceso (`PENDIENTE_ACTIVACION`, `ACTIVO`, `BLOQUEADO`) se conserva separado del estado KYC (`PENDING`, `VERIFIED`, `REJECTED`). Solamente un cliente activo con KYC `VERIFIED` puede avanzar en la Saga de transferencia.

La fase 2 incorpora los estados `VALIDANDO_KYC` y `VALIDANDO_CUENTAS`. También conserva `resultadoExternoSimulado` para demostrar `EXITO`, `FALLO` y `TIMEOUT`; los dos últimos obligan a compensar cualquier débito ya aplicado.
