# Catálogo de eventos

Este catálogo describe los mensajes intercambiados mediante RabbitMQ por los componentes de **Bank USAC**. Incluye comandos, eventos de dominio, respuestas de consulta y mensajes de soporte operacional.

## Convenciones

- Un **comando** solicita que un servicio ejecute una acción. Su routing key suele terminar en `.solicitada` o `.solicitado`.
- Un **evento** comunica un hecho que ya ocurrió, por ejemplo `cuenta.debitada`.
- Una **respuesta asíncrona** comunica el resultado de una consulta iniciada por el API Gateway.
- Las routing keys se escriben en minúsculas y separan el dominio y la acción mediante puntos.
- Cada mensaje utiliza el sobre común definido en [Contratos de eventos](contratos-eventos.md).
- Todos los mensajes de una misma operación conservan el mismo `idCorrelacion`.

## Customer Service

| Routing key | Clasificación | Productor principal | Consumidor principal | Propósito |
|---|---|---|---|---|
| `cliente.registro.solicitado` | Comando | API Gateway | Customer Service | Registrar un nuevo cliente y usuario. |
| `cliente.activacion.solicitada` | Comando | API Gateway | Customer Service | Activar un usuario mediante su token. |
| `cliente.login.solicitado` | Comando | API Gateway | Customer Service | Validar credenciales y generar la respuesta de autenticación. |
| `cliente.perfil.solicitado` | Comando de consulta | API Gateway | Customer Service | Consultar el perfil de un usuario. |
| `cliente.actualizacion.solicitada` | Comando | API Gateway | Customer Service | Actualizar los datos permitidos de un cliente. |
| `cliente.listado.solicitado` | Comando de consulta | API Gateway | Customer Service | Consultar clientes registrados. |
| `cliente.estado.solicitado` | Comando | API Gateway | Customer Service | Cambiar el estado de un usuario. |
| `cliente.validacion.solicitada` | Comando | Account Service | Customer Service | Validar que un cliente exista y esté activo. |
| `cliente.creado` | Evento | Customer Service | Notification & Audit Service | Informar el registro exitoso del cliente. |
| `cliente.activado` | Evento | Customer Service | Notification & Audit Service | Informar que el usuario fue activado. |
| `cliente.validado` | Evento de respuesta | Customer Service | Account Service | Confirmar que el cliente es válido y está activo. |
| `cliente.rechazado` | Evento de respuesta | Customer Service | Account Service | Rechazar la validación del cliente e indicar el motivo. |
| `notificacion.correo-activacion.solicitado` | Evento/solicitud | Customer Service | Notification & Audit Service | Solicitar el envío del correo de activación. |

## Account Service

| Routing key | Clasificación | Productor principal | Consumidor principal | Propósito |
|---|---|---|---|---|
| `cuenta.creacion.solicitada` | Comando | API Gateway | Account Service | Iniciar la creación de una cuenta. |
| `cuenta.consulta.solicitada` | Comando de consulta | API Gateway | Account Service | Consultar una cuenta y su saldo. |
| `cuenta.movimientos.solicitados` | Comando de consulta | API Gateway | Account Service | Consultar los movimientos de una cuenta. |
| `cuenta.historial.solicitado` | Comando de consulta | API Gateway | Account Service | Listar las cuentas pertenecientes a un cliente. |
| `cuenta.debito.solicitado` | Comando financiero | Transaction o Payment Service | Account Service | Debitar una cuenta si está habilitada y tiene fondos. |
| `cuenta.credito.solicitado` | Comando financiero | Transaction Service | Account Service | Acreditar fondos en la cuenta destino. |
| `cuenta.compensacion.solicitada` | Comando financiero | Transaction o Payment Service | Account Service | Revertir un débito previamente aplicado. |
| `cuenta.creada` | Evento | Account Service | API Gateway y Notification & Audit Service | Informar la creación exitosa de una cuenta. |
| `cuenta.creacion.rechazada` | Evento | Account Service | API Gateway y Notification & Audit Service | Informar que la cuenta no pudo crearse. |
| `cuenta.debitada` | Evento | Account Service | Transaction o Payment Service | Confirmar el débito. |
| `cuenta.debito.rechazado` | Evento | Account Service | Transaction o Payment Service | Informar el rechazo del débito. |
| `cuenta.acreditada` | Evento | Account Service | Transaction Service | Confirmar el crédito. |
| `cuenta.credito.rechazado` | Evento | Account Service | Transaction Service | Informar el rechazo del crédito. |
| `cuenta.compensada` | Evento | Account Service | Transaction o Payment Service | Confirmar la compensación. |
| `cuenta.compensacion.rechazada` | Evento | Account Service | Transaction Service | Informar que no se pudo compensar. |
| `cuenta.desactivada` | Evento | Account Service | Notification & Audit Service | Informar la desactivación automática de una cuenta. |
| `cuenta.consultada` | Respuesta asíncrona | Account Service | API Gateway | Entregar una cuenta consultada. |
| `cuenta.movimientos.consultados` | Respuesta asíncrona | Account Service | API Gateway | Entregar movimientos paginados. |
| `cuenta.historial.consultado` | Respuesta asíncrona | Account Service | API Gateway | Entregar las cuentas de un cliente. |

## Transaction Service

| Routing key | Clasificación | Productor principal | Consumidor principal | Propósito |
|---|---|---|---|---|
| `transferencia.solicitada` | Comando | API Gateway | Transaction Service | Registrar e iniciar una transferencia. |
| `transfer.requested` | Comando compatible | API Gateway | Transaction Service | Aceptar el contrato anterior en inglés durante la integración. |
| `transferencia.consulta.solicitada` | Comando de consulta | API Gateway | Transaction Service | Consultar una transferencia. |
| `transferencia.historial.solicitado` | Comando de consulta | API Gateway | Transaction Service | Consultar el historial de un cliente. |
| `transferencia.procesando` | Evento | Transaction Service | API Gateway y Notification & Audit Service | Informar que inició la Saga. |
| `transferencia.completada` | Evento | Transaction Service | API Gateway y Notification & Audit Service | Informar la finalización exitosa. |
| `transferencia.rechazada` | Evento | Transaction Service | API Gateway y Notification & Audit Service | Informar que la transferencia fue rechazada. |
| `transferencia.compensando` | Evento | Transaction Service | API Gateway y Notification & Audit Service | Informar el inicio de una compensación. |
| `transferencia.compensada` | Evento | Transaction Service | API Gateway y Notification & Audit Service | Informar que la operación fue revertida. |
| `transferencia.compensacion.fallida` | Evento | Transaction Service | API Gateway y Notification & Audit Service | Informar un fallo al compensar. |
| `transferencia.consultada` | Respuesta asíncrona | Transaction Service | API Gateway | Entregar el detalle de una transferencia. |
| `transferencia.historial.consultado` | Respuesta asíncrona | Transaction Service | API Gateway | Entregar el historial solicitado. |

Transaction Service también produce `cuenta.debito.solicitado`, `cuenta.credito.solicitado` y `cuenta.compensacion.solicitada`, y consume los resultados financieros publicados por Account Service.

## Payment Service

| Routing key | Clasificación | Productor principal | Consumidor principal | Propósito |
|---|---|---|---|---|
| `pago.procesamiento.solicitado` | Comando | API Gateway | Payment Service | Registrar e iniciar un pago. |
| `pago.consulta.solicitada` | Comando de consulta | API Gateway | Payment Service | Consultar un pago. |
| `pago.historial.solicitado` | Comando de consulta | API Gateway | Payment Service | Consultar los pagos de un cliente. |
| `pago.completado` | Evento | Payment Service | API Gateway y Notification & Audit Service | Informar la finalización exitosa de un pago. |
| `pago.rechazado` | Evento | Payment Service | API Gateway y Notification & Audit Service | Informar el rechazo o compensación del pago. |
| `pago.consultado` | Respuesta asíncrona | Payment Service | API Gateway | Entregar el detalle de un pago. |
| `pago.historial.consultado` | Respuesta asíncrona | Payment Service | API Gateway | Entregar el historial de pagos. |

Payment Service produce comandos de débito y compensación para Account Service y consume `cuenta.debitada`, `cuenta.debito.rechazado` y `cuenta.compensada`.

## Notification & Audit Service

| Routing key | Clasificación | Productor principal | Consumidor principal | Propósito |
|---|---|---|---|---|
| `auditoria.registros.solicitados` | Comando de consulta | API Gateway | Notification & Audit Service | Consultar registros de auditoría. |
| `auditoria.traza.solicitada` | Comando de consulta | API Gateway | Notification & Audit Service | Reconstruir una operación por `idCorrelacion`. |
| `auditoria.notificaciones.solicitadas` | Comando de consulta | API Gateway | Notification & Audit Service | Consultar el historial de notificaciones. |

Este servicio también consume los eventos de dominio publicados por los demás microservicios. Cada evento válido se almacena como evidencia de auditoría; los eventos que requieren comunicación al usuario generan además una notificación.

## Mensajes fallidos

| Routing key | Servicio asociado | Uso |
|---|---|---|
| `cliente.validacion.fallida` | Customer Service | Mensajes de cliente que agotaron el procesamiento esperado. |
| `cuenta.comando.fallido` | Account Service | Comandos de cuenta no procesados después de los reintentos. |
| `transaccion.mensaje.fallido` | Transaction Service | Mensajes de transferencias enviados a la DLQ. |
| `pago.mensaje.fallido` | Payment Service | Mensajes de pagos enviados a la DLQ. |
| `notification-audit.dlq` | Notification & Audit Service | Eventos o consultas de auditoría que no pudieron procesarse. |


