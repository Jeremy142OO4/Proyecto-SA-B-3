# Contextos delimitados

Los contextos delimitados establecen qué conceptos, reglas y datos pertenecen a cada microservicio de **Bank USAC**. Dentro de cada contexto, los términos poseen un significado preciso y son controlados por un único propietario.

Esta división evita que un servicio modifique las tablas de otro o replique lógica que no le corresponde. Cuando un contexto necesita información externa, utiliza identificadores lógicos y contratos de mensajes publicados mediante RabbitMQ.


## Resumen de propiedad

| Contexto | Microservicio | Conceptos que posee | Datos que no posee |
|---|---|---|---|
| Clientes | `customer-service` | Cliente, credenciales, rol, estado y token de activación. | Cuentas, balances, transferencias y pagos. |
| Cuentas | `account-service` | Cuenta, movimiento, saldo, estado, última actividad y solicitud de creación. | Datos personales, credenciales, transferencias y pagos. |
| Transferencias | `transaction-service` | Transferencia, estado de la Saga, error y compensación. | Saldo real de las cuentas y datos personales. |
| Pagos | `payment-service` | Pago, intento, beneficiario, concepto, referencia y estado. | Saldo real, información personal y transferencias. |
| Notificaciones y Auditoría | `notification-audit-service` | Evento de auditoría, notificación y resultado del envío. | Entidades bancarias originales y sus reglas de modificación. |

## Contexto de Clientes

### Responsabilidad

Administrar la identidad y el acceso de las personas que utilizan el sistema.

### Funciones

- Registrar clientes.
- Validar la información y fotografía del documento de identificación.
- Evitar documentos, correos y usernames duplicados.
- Generar el username del cliente.
- Actualizar datos personales.
- Gestionar activación y estado del usuario.
- Autenticar credenciales y generar JWT.
- Asignar los roles Administrador, Cajero Receptor y Cliente.

### Entidades propias

- Cliente.
- Token de activación.
- Mensajes de salida y mensajes procesados del contexto.

### Límites

Customer Service no crea cuentas bancarias, no modifica balances y no procesa transferencias ni pagos. Cuando Account Service necesita validar a un cliente, la interacción se realiza mediante mensajes asíncronos y Customer Service responde con la información mínima necesaria.

## Contexto de Cuentas

### Responsabilidad

Mantener las cuentas bancarias y proteger la integridad de sus balances.

### Funciones

- Solicitar y completar la creación de una cuenta.
- Asociar lógicamente una cuenta con un cliente válido.
- Administrar cuentas monetarias y de ahorro.
- Consultar saldo y estado.
- Aplicar débitos, créditos y compensaciones.
- Registrar movimientos financieros.
- Impedir saldos negativos.
- Actualizar la última actividad.
- Desactivar cuentas con saldo menor a Q50.00 después de seis meses de inactividad.

### Entidades propias

- Cuenta.
- Movimiento de cuenta.
- Solicitud de creación de cuenta.
- Mensajes de salida y mensajes procesados del contexto.

### Límites

Account Service conserva `idCliente` como referencia lógica, pero no administra datos personales ni credenciales. Tampoco decide el estado global de una transferencia o pago; únicamente acepta o rechaza las operaciones que afectan el balance y publica el resultado.

## Contexto de Transferencias

### Responsabilidad

Registrar transferencias y coordinar su ejecución distribuida.

### Funciones

- Validar monto, moneda y diferencia entre cuentas.
- Crear una transferencia y conservar su estado.
- Coordinar la Saga de débito y crédito.
- Solicitar compensaciones cuando el flujo no pueda completarse.
- Registrar códigos de error.
- Permitir consultar estado e historial.

### Entidades propias

- Transferencia.
- Estado de la Saga representado por la transferencia.
- Mensajes de salida y mensajes procesados del contexto.

### Límites

Transaction Service no actualiza directamente los saldos. Emite comandos para Account Service y procesa sus respuestas. Los identificadores de cliente y cuentas son referencias lógicas; la propiedad de los balances permanece en el contexto de Cuentas.

## Contexto de Pagos

### Responsabilidad

Procesar y rastrear pagos internos o externos.

### Funciones

- Registrar la solicitud de pago.
- Validar beneficiario, concepto, monto, moneda y tipo.
- Solicitar la afectación de la cuenta origen.
- Mantener el estado del pago.
- Registrar cada intento de procesamiento.
- Conservar la referencia de un proveedor externo cuando corresponda.
- Manejar rechazos, fallos y compensaciones.
- Permitir consultar pagos e historial.

### Entidades propias

- Pago.
- Intento de pago.
- Mensajes de salida y mensajes procesados del contexto.

### Límites

Payment Service no modifica directamente la tabla de cuentas ni autentica al cliente. Solicita las operaciones financieras mediante RabbitMQ y utiliza identificadores externos únicamente como referencias.

## Contexto de Notificaciones y Auditoría

### Responsabilidad

Observar los eventos del sistema, conservar evidencia auditable y generar notificaciones sin acoplar los servicios de negocio a un canal de envío.

### Funciones

- Consumir eventos publicados por los demás contextos.
- Registrar identificador, tipo, productor, versión, fecha y payload del evento.
- Relacionar eventos mediante `idCorrelacion` e `idCausacion`.
- Evitar registros duplicados.
- Generar notificaciones de activación y operaciones bancarias.
- Registrar el resultado del envío.
- Mantener resúmenes sin información sensible.

### Entidades propias

- Evento de auditoría.
- Notificación.
- Mensaje procesado del contexto.

### Límites

Este servicio no modifica clientes, cuentas, transferencias o pagos. El registro de auditoría representa lo ocurrido, pero no se convierte en la fuente de verdad del evento original. Un fallo de notificación tampoco revierte la operación bancaria que originó el mensaje.

## Relaciones entre contextos

| Contexto productor | Contexto consumidor | Información intercambiada | Propósito |
|---|---|---|---|
| API Gateway | Cualquier contexto de negocio | Comando con identidad, payload e `idCorrelacion`. | Iniciar una operación solicitada externamente. |
| Cuentas | Clientes | Solicitud de validación del cliente. | Comprobar que el propietario exista y esté activo antes de crear una cuenta. |
| Clientes | Cuentas | Resultado de validación. | Completar o rechazar la solicitud de creación. |
| Transferencias | Cuentas | Comandos de débito, crédito o compensación. | Ejecutar las etapas de la Saga. |
| Cuentas | Transferencias | Resultado de la operación sobre la cuenta. | Avanzar, rechazar o compensar la transferencia. |
| Pagos | Cuentas | Solicitud de afectación o compensación. | Aplicar el monto asociado con un pago. |
| Cuentas | Pagos | Resultado financiero. | Actualizar el estado del pago. |
| Todos los contextos | Notificaciones y Auditoría | Eventos de dominio. | Conservar trazabilidad y producir notificaciones. |

La tabla describe dependencias mediante contratos de eventos. No implica llamadas síncronas ni acceso a datos internos del contexto consumidor.

## Reglas de integración

1. Toda comunicación de negocio entre contextos utiliza RabbitMQ.
2. Cada mensaje posee un identificador único y un `idCorrelacion`.
3. Cada contexto publica eventos usando su propia tabla Outbox cuando corresponde.
4. Los consumidores son idempotentes y registran los mensajes procesados.
5. Un contexto nunca escribe en la base de datos de otro.
6. Los contratos transportan únicamente la información necesaria para ejecutar el flujo.
7. Los fallos temporales utilizan reintentos y los fallos agotados pueden dirigirse a una DLQ.
8. Los nombres de eventos deben expresar hechos ocurridos o comandos explícitos, sin revelar detalles internos de persistencia.

## Consistencia y autonomía

Cada contexto garantiza consistencia fuerte únicamente dentro de su propia transacción y base de datos. Entre contextos se utiliza **consistencia eventual**. Esto significa que una solicitud puede permanecer temporalmente pendiente o en procesamiento mientras los mensajes son atendidos.

La autonomía de cada servicio se conserva mediante tres reglas principales: propiedad exclusiva de datos, contratos de eventos versionados y ausencia de dependencias síncronas entre microservicios.
