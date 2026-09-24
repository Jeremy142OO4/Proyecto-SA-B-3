# Requisitos funcionales

Los requisitos RF-01 a RF-30 corresponden a las capacidades principales del enunciado de la Fase 1. Los RF-31 a RF-40 son funciones complementarias incorporadas para completar los flujos bancarios. Los RF-41 a RF-52 corresponden a las nuevas capacidades incorporadas en la Fase 2.

| ID | Requisito funcional |
|---|---|
| RF-01 | Registrar clientes. |
| RF-02 | Validar la identidad mediante fotografía del documento. |
| RF-03 | Actualizar datos del cliente. |
| RF-04 | Activar al usuario mediante enlace enviado por correo. |
| RF-05 | Autenticar y validar usuarios mediante JWT. |
| RF-06 | Validar la coherencia de la fecha de nacimiento. |
| RF-07 | Generar un `username` a partir del nombre del cliente. |
| RF-08 | Crear cuentas bancarias asociadas a un cliente. |
| RF-09 | Crear cuentas monetarias y de ahorro. |
| RF-10 | Consultar el saldo de una cuenta. |
| RF-11 | Actualizar el balance como resultado de una operación válida. |
| RF-12 | Desactivar cuentas con balance menor a Q50.00 después de seis meses de inactividad. |
| RF-13 | Realizar transferencias entre cuentas. |
| RF-14 | Validar fondos suficientes antes de una transferencia. |
| RF-15 | Registrar las transacciones y su estado. |
| RF-16 | Ejecutar transferencias mediante una Saga con fallos y compensaciones. |
| RF-17 | Procesar pagos internos o externos. |
| RF-18 | Validar una operación financiera antes de confirmar el pago. |
| RF-19 | Registrar cada pago y su estado. |
| RF-20 | Manejar fallos durante la interacción con sistemas de pago externos. |
| RF-21 | Enviar notificaciones relacionadas con operaciones bancarias. |
| RF-22 | Registrar eventos para auditoría. |
| RF-23 | Conservar identificador, tipo, fecha, hora y payload en la auditoría. |
| RF-24 | Relacionar eventos mediante `correlationId`. |
| RF-25 | Reconocer los roles Administrador, Cajero Receptor y Cliente. |
| RF-26 | Aplicar permisos según el rol autenticado. |
| RF-27 | Recibir solicitudes externas mediante el API Gateway. |
| RF-28 | Iniciar flujos mediante mensajería asíncrona. |
| RF-29 | Consultar el resultado de operaciones asíncronas. |
| RF-30 | Proporcionar una interfaz intuitiva y funcional para los cinco microservicios. |
| RF-31 | Impedir registros duplicados de documento, correo y `username`. |
| RF-32 | Usar enlaces de activación de un solo uso, con expiración y reenvío controlado. |
| RF-33 | Manejar usuarios pendientes de activación, activos y bloqueados. |
| RF-34 | Validar que el cliente exista y esté activo antes de crearle una cuenta. |
| RF-35 | Manejar cuentas activas, inactivas, bloqueadas y cerradas, registrando su última actividad. |
| RF-36 | Rechazar débitos sobre cuentas no activas e impedir balances negativos. |
| RF-37 | Validar monto positivo, cuentas diferentes y cuentas habilitadas en transferencias. |
| RF-38 | Manejar estados pendiente, procesando, completada, rechazada y compensada. |
| RF-39 | Registrar beneficiario, concepto, monto, tipo y estado de cada pago. |
| RF-40 | Permitir al cliente consultar su historial de transacciones y pagos. |
| RF-41 | Registrar el estado KYC de un cliente con los valores `PENDING`, `VERIFIED` y `REJECTED`. |
| RF-42 | Actualizar el estado KYC de un cliente desde el sistema. |
| RF-43 | Validar que el cliente tenga estado KYC `VERIFIED` antes de autorizar una transferencia. |
| RF-44 | Crear cuentas bancarias de tipo `AHORRO` o `CORRIENTE` con reglas de negocio específicas por tipo. |
| RF-45 | Aplicar límite de saldo mínimo y comisión por transacción según el tipo de cuenta. |
| RF-46 | Consultar el historial de transacciones asociadas a una cuenta. |
| RF-47 | Filtrar el historial de transacciones por fecha y por estado. |
| RF-48 | Registrar estados intermedios `PENDING`, `APPROVED` y `FAILED` durante el procesamiento de una transacción. |
| RF-49 | Procesar pagos externos mediante simulación de integración con un sistema externo. |
| RF-50 | Simular las respuestas `EXITO`, `FALLO` y `TIMEOUT` para pagos externos; compensar la cuenta cuando corresponda. |
| RF-51 | Clasificar los eventos del sistema como `INFO`, `WARNING` o `ERROR` en el servicio de notificaciones y auditoría. |
| RF-52 | Generar notificaciones diferenciadas según la clasificación del evento (`INFO`, `WARNING`, `ERROR`). |

## Matriz de trazabilidad

| Requisito | Responsable | Actor principal | Caso de uso |
|---|---|---|---|
| RF-01, RF-02, RF-06, RF-07, RF-31 | Customer Service | Cajero Receptor | CDU-CUS-01 Registrar cliente |
| RF-03 | Customer Service | Cliente / Cajero Receptor | CDU-CUS-02 Actualizar datos |
| RF-04, RF-32, RF-33 | Customer Service / Notification & Audit | Cliente | CDU-CUS-03 Activar usuario |
| RF-05, RF-25, RF-26 | Customer Service / API Gateway | Todos los roles | CDU-CUS-04 Iniciar sesión |
| RF-08, RF-09, RF-34 | Account Service | Cajero Receptor | CDU-ACC-01 Crear cuenta |
| RF-10 | Account Service | Cliente / Cajero Receptor | CDU-ACC-02 Consultar saldo |
| RF-11, RF-36 | Account Service | Sistema | CDU-ACC-03 Actualizar balance |
| RF-12, RF-35 | Account Service | Sistema | CDU-ACC-04 Desactivar cuenta inactiva |
| RF-35 | Account Service | Cliente / Cajero / Administrador | CDU-ACC-05 Consultar estado |
| RF-13, RF-14, RF-15, RF-37 | Transaction Service | Cliente / Cajero Receptor | CDU-TRX-01 Realizar transferencia |
| RF-16, RF-38 | Transaction Service | Sistema | CDU-TRX-02 Ejecutar Saga |
| RF-29, RF-38 | Transaction Service | Cliente / Cajero Receptor | CDU-TRX-03 Consultar transferencia |
| RF-40 | Transaction Service | Cliente | CDU-TRX-04 Consultar historial |
| RF-17, RF-18, RF-19, RF-20, RF-39 | Payment Service | Cliente / Cajero Receptor | CDU-PAY-01 Procesar pago |
| RF-29, RF-39, RF-40 | Payment Service | Cliente / Cajero Receptor | CDU-PAY-02 Consultar pagos |
| RF-21, RF-32 | Notification & Audit Service | Sistema | CDU-NOT-01 Enviar notificación |
| RF-22, RF-23, RF-24 | Notification & Audit Service | Sistema | CDU-NOT-02 Registrar auditoría |
| RF-27, RF-28, RF-29, RF-30 | API Gateway / Frontend | Todos los roles | Acceso y operación del sistema |
| RF-41, RF-42, RF-43 | Customer Service | Administrador / Sistema | CDU-CUS-KYC Gestionar estado KYC |
| RF-44, RF-45 | Account Service | Cajero Receptor | CDU-ACC-01 Crear cuenta (extensión Fase 2) |
| RF-46, RF-47, RF-48 | Transaction Service | Cliente / Cajero Receptor | CDU-TRX-05 Consultar historial filtrado |
| RF-49, RF-50 | Payment Service | Sistema | CDU-PAY-03 Procesar pago externo simulado |
| RF-51, RF-52 | Notification & Audit Service | Sistema | CDU-NOT-02 Clasificar y registrar evento |

