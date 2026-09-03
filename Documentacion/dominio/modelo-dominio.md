# Modelo de dominio

El modelo de dominio de **Bank USAC** describe los conceptos principales del negocio bancario, sus atributos, relaciones, estados y responsabilidades. El modelo se divide según los cinco microservicios del sistema para que cada concepto tenga un propietario claro y no sea modificado directamente desde otros dominios.

Este documento presenta una vista conceptual. Los diagramas entidad-relación de cada servicio detallan posteriormente la estructura física de sus tablas.


## Entidades principales

### Cliente

Representa a una persona o usuario registrado en el banco. Customer Service es responsable de su identidad, información personal, credenciales, rol y estado.

Roles admitidos:

- `ADMIN`: Administrador.
- `TELLER`: Cajero Receptor.
- `CLIENTE`: Cliente bancario.

Estados admitidos:

- `PENDIENTE_ACTIVACION`.
- `ACTIVO`.
- `BLOQUEADO`.

El documento de identificación, correo y username deben ser únicos. La contraseña nunca se almacena en texto plano, sino como un hash.

### Token de activación

Representa el enlace de un solo uso enviado durante la activación del usuario. Pertenece a un cliente, posee una fecha de expiración y registra si ya fue utilizado. El valor original no se almacena directamente; se conserva su hash.

### Cuenta

Representa una cuenta bancaria asociada lógicamente con un cliente. Account Service es propietario de su número, tipo, saldo, moneda, estado y actividad financiera.

Tipos admitidos:

- `MONETARIA`.
- `AHORRO`.

Estados admitidos:

- `ACTIVA`.
- `INACTIVA`.
- `BLOQUEADA`.
- `CERRADA`.

El saldo se expresa en centavos y no puede quedar negativo. La versión permite proteger las actualizaciones concurrentes.

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

Estados admitidos:

- `PENDIENTE`.
- `PROCESANDO`.
- `COMPLETADA`.
- `RECHAZADA`.
- `COMPENSANDO`.
- `COMPENSADA`.
- `COMPENSACION_FALLIDA`.

### Pago

Representa una instrucción de pago realizada desde una cuenta. Payment Service conserva el beneficiario, concepto, monto, moneda, tipo, referencia externa y estado del proceso.

Los pagos pueden ser `INTERNO` o `EXTERNO` y pasan por estados como `PENDIENTE`, `PROCESANDO`, `COMPENSANDO`, `COMPLETADO` o `RECHAZADO`.

### Intento de pago

Representa cada intento de procesar un pago. Permite conservar el número de intento, respuesta obtenida, detalle del error y tiempos de ejecución sin sobrescribir la historia de intentos anteriores.

### Evento de auditoría

Es una copia inmutable de un evento relevante ocurrido en cualquier dominio. Conserva el identificador del evento, productor, tipo, versión, payload y fechas. El `idCorrelacion` permite reconstruir una operación distribuida completa.

### Notificación

Representa el historial de una comunicación dirigida a un usuario. Conserva el destinatario, tipo, asunto, resumen seguro del contenido y resultado del envío. Sus estados son `PENDING`, `SENT` y `FAILED`.

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

## Elementos técnicos de soporte

Además de las entidades del negocio, los servicios utilizan elementos técnicos para garantizar la comunicación confiable:

- **Mensaje de salida:** conserva un comando o evento pendiente de publicar como parte del patrón Outbox.
- **Mensaje procesado:** registra qué consumidor ya atendió un mensaje y garantiza idempotencia.
- **Identificador de correlación:** relaciona los mensajes producidos durante una operación distribuida.
- **Identificador de causación:** indica cuál mensaje o acción produjo un evento posterior.

Estos elementos respaldan el dominio, pero no sustituyen las entidades bancarias principales.
