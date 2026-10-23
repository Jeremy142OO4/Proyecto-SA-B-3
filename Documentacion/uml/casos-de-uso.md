# Diagramas de casos de uso

## Catálogo y distribución de CDU

Cada CDU se documenta con su diagrama de caso de uso, flujo expandido, actores, precondiciones, postcondiciones y requisitos funcionales. Las validaciones internas se modelan con `include`, `extend` o escenarios alternativos del CDU principal.

| ID | Microservicio | Funcionalidad | Actor principal | RF cubiertos |
|---|---|---|---|---|
| CDU-CUS-01 | Customer Service | Registrar cliente | Cajero Receptor | RF-01, RF-02, RF-06, RF-07, RF-31 |
| CDU-CUS-02 | Customer Service | Actualizar datos del cliente | Cliente / Cajero Receptor | RF-03 |
| CDU-CUS-03 | Customer Service | Activar usuario | Cliente | RF-04, RF-32, RF-33 |
| CDU-CUS-04 | Customer Service | Iniciar sesión | Todos los roles | RF-05, RF-25, RF-26 |
| CDU-CUS-05 | Customer Service | Gestionar estado del usuario | Administrador | RF-33 |
| CDU-ACC-01 | Account Service | Crear cuenta bancaria | Cajero Receptor | RF-08, RF-09, RF-34 |
| CDU-ACC-02 | Account Service | Consultar saldo | Cliente / Cajero Receptor | RF-10 |
| CDU-ACC-03 | Account Service | Actualizar balance | Sistema | RF-11, RF-36 |
| CDU-ACC-04 | Account Service | Desactivar cuenta inactiva | Sistema | RF-12, RF-35 |
| CDU-ACC-05 | Account Service | Consultar estado de cuenta | Cliente / Cajero / Administrador | RF-35 |
| CDU-TRX-01 | Transaction Service | Realizar transferencia | Cliente / Cajero Receptor | RF-13, RF-14, RF-15, RF-37 |
| CDU-TRX-02 | Transaction Service | Ejecutar Saga de transferencia | Sistema | RF-16, RF-38 |
| CDU-TRX-03 | Transaction Service | Consultar estado de transferencia | Cliente / Cajero Receptor | RF-29, RF-38 |
| CDU-TRX-04 | Transaction Service | Consultar historial de transacciones | Cliente | RF-40 |
| CDU-PAY-01 | Payment Service | Procesar pago | Cliente / Cajero Receptor | RF-17, RF-18, RF-19, RF-20, RF-39 |
| CDU-PAY-02 | Payment Service | Consultar pagos | Cliente / Cajero Receptor | RF-29, RF-39, RF-40 |
| CDU-NOT-01 | Notification & Audit Service | Enviar notificación | Sistema | RF-21, RF-32 |
| CDU-NOT-02 | Notification & Audit Service | Registrar evento de auditoría | Sistema | RF-22, RF-23, RF-24 |
| CDU-CUS-KYC | Customer Service | Gestionar estado KYC del cliente | Administrador / Sistema | RF-41, RF-42, RF-43 |
| CDU-TRX-05 | Transaction Service | Consultar historial de transacciones filtrado | Cliente / Cajero Receptor | RF-46, RF-47, RF-48 |
| CDU-PAY-03 | Transaction Service / Payment Gateway (simulado), según documento base | Procesar pago externo simulado | Sistema | RF-49, RF-50 |
| CDU-NOT-03 | Notification & Audit Service | Clasificar y registrar evento por severidad | Sistema | RF-51, RF-52 |
| CDU-CUS-06 | Customer Service | Registrar trazabilidad del ciclo de vida del cliente | Sistema | Fase 3: correspondencia pendiente con la matriz |
| CDU-ACC-06 | Account Service | Detectar anomalías de uso de cuenta | Sistema | Fase 3: correspondencia pendiente con la matriz |
| CDU-TRX-06 | Transaction Service | Medir rendimiento del flujo de transferencias | Sistema | Fase 3: correspondencia pendiente con la matriz |
| CDU-PAY-04 | Payment Service | Simular fallos avanzados configurables | Administrador / Sistema | Fase 3: correspondencia pendiente con la matriz |
| CDU-TRX-07 | Transaction Service, con Account Service y adaptador interbancario | Enviar depósito interbancario | Cliente / Cajero Receptor | Fase 3: correspondencia pendiente con la matriz |
| CDU-TRX-08 | Transaction Service, con Account Service y entrada interbancaria | Recibir depósito interbancario | Banco pareja (BANK14) | Fase 3: correspondencia pendiente con la matriz |
| CDU-TRX-09 | Transaction Service | Consultar estado de depósito interbancario | Cliente / Cajero Receptor | Fase 3: correspondencia pendiente con la matriz |

El catálogo consolida **29 CDU: 22 existentes y 7 incorporados para Fase 3**. Los cuatro CDU del final del documento original ya existían aunque no figuraban en su catálogo inicial.

**Trazabilidad:** se conservan los RF históricos. La numeración RF-53 a RF-71 propuesta anteriormente no fue verificada y no se adopta como oficial. Los CDU nuevos deben contrastarse con los requerimientos del enunciado, sin asumir equivalencia automática entre numeraciones. La matriz de seguimiento se conserva únicamente en el chat de trabajo, fuera del repositorio.

**Alcance documental:** los CDU nuevos son especificaciones del comportamiento esperado, no certificaciones de implementación o pruebas. Sus notas señalan contratos y capacidades pendientes de contrastar. Los PNG se referencian por ruta: no están incrustados ni se generan con este archivo.


### Distribución entre integrantes

Se conserva la asignación original de los 18 CDU del catálogo base:

| Integrante | CDU asignados | Total |
|---|---|---:|
| Integrante 1 | CDU-CUS-01, CDU-CUS-03, CDU-CUS-05, CDU-ACC-01, CDU-TRX-01, CDU-NOT-01 | 6 |
| Integrante 2 | CDU-CUS-02, CDU-CUS-04, CDU-ACC-02, CDU-ACC-03, CDU-ACC-04, CDU-NOT-02 | 6 |
| Integrante 3 | CDU-ACC-05, CDU-TRX-02, CDU-TRX-03, CDU-TRX-04, CDU-PAY-01, CDU-PAY-02 | 6 |

La asignación de CDU-CUS-KYC, CDU-TRX-05, CDU-PAY-03, CDU-NOT-03 y los siete CDU de Fase 3 queda **[COMPLETAR por el equipo]**. No se modifican responsabilidades acordadas.

Las relaciones con otros microservicios se representan como sistemas externos al límite del CDU; no se dibujan relaciones directas entre bases de datos independientes.

## Casos de Uso de Alto Nivel y Expandidos

### Diagrama de CDU de alto nivel

![Diagrama de alto nivel](../Imagenes/Diagrama%20-%20AltoNivelCDU.png)

### Diagrama de CDU Expandido General

![Diagrama expandido](../Imagenes/Diagrama%20-Descomposicion.png)

## Casos de Uso de Customer Service
### Caso de Uso: Registrar Cliente

![Caso Expandido CUS-01](../Imagenes/Diagrama%20-%20CDUCSU01.png)


| Campo | Descripción |
|---|---|
| **ID Caso de Uso** | CDU-CUS-01 |
| **Nombre** | Registrar cliente |
| **Módulo al que pertenece** | Customer Service |
| **Actor Principal** | Cajero Receptor |
| **RF cubiertos** | RF-01, RF-02, RF-06, RF-07 y RF-31 |
| **Precondiciones** | - El Cajero Receptor debe haber iniciado sesión en el sistema.<br>- El cliente debe proporcionar sus datos personales.<br>- El cliente debe proporcionar su documento de identificación y la fotografía correspondiente.<br>- El cliente debe proporcionar una dirección de correo electrónico válida. |
| **Postcondiciones** | - Se crea un nuevo cliente en el sistema.<br>- Se genera un `customerId` único.<br>- Se genera el username correspondiente.<br>- Los datos quedan almacenados en Customer Service.<br>- El usuario queda registrado y pendiente del proceso de activación. |
| **Escenario Principal** | 1. El Cajero Receptor selecciona la opción **Registrar cliente**.<br>2. El sistema muestra el formulario de registro.<br>3. El Cajero Receptor ingresa los datos personales del cliente.<br>4. Adjunta la fotografía del documento de identificación.<br>5. El sistema valida los datos proporcionados.<br>6. El sistema valida la coherencia de la fecha de nacimiento.<br>7. El sistema genera el username del cliente.<br>8. El sistema genera un `customerId` único.<br>9. El sistema registra al cliente.<br>10. El sistema confirma que el registro fue realizado correctamente. |
| **Escenario Alternativo** | **1. Datos inválidos:** El sistema informa qué información debe corregirse.<br><br>**2. Documento incompleto:** Si no se proporciona la información necesaria de identificación, el registro no continúa.<br><br>**3. Correo inválido:** El sistema solicita ingresar un correo válido.<br><br>**4. Error durante el registro:** El sistema cancela la operación evitando almacenar información incompleta. |
| **Requerimientos** | - El sistema debe permitir registrar nuevos clientes.<br>- Debe validar la información de identificación.<br>- Debe validar la información ingresada antes de registrarla.<br>- Cada cliente debe poseer un `customerId` único.<br>- Los datos deben almacenarse únicamente en la base de datos de Customer Service.<br>- El proceso debe evitar registros duplicados. |

![Caso Expandido CUS-01](../Imagenes/Diagrama%20-%20CUS01.png)

---
### Caso de Uso: Actualizar datos del cliente

#### Diagrama de caso de uso

![Caso Expandido CUS-02](../Imagenes/Diagrama%20-%20CDUCSU02.png)

| Campo | Descripción |
|---|---|
| **ID Caso de Uso** | CDU-CUS-02 |
| **Nombre** | Actualizar datos del cliente |
| **Módulo al que pertenece** | Customer Service |
| **Actor Principal** | Cliente / Cajero Receptor |
| **RF cubiertos** | RF-03 |
| **Precondiciones** | - El actor debe haber iniciado sesión.<br>- El cliente debe estar registrado.<br>- El Cliente solo puede modificar sus propios datos.<br>- El Cajero Receptor debe poseer permisos para actualizar los datos solicitados. |
| **Postcondiciones** | - Los datos válidos quedan actualizados en Customer Service.<br>- Los datos no incluidos en la solicitud conservan su valor anterior.<br>- Se conserva el identificador único del cliente. |
| **Escenario Principal** | 1. El actor selecciona **Actualizar datos del cliente**.<br>2. El sistema identifica al cliente correspondiente.<br>3. Customer Service muestra los datos que pueden modificarse.<br>4. El actor ingresa la información actualizada.<br>5. El sistema verifica los permisos del actor.<br>6. El sistema valida el formato y la coherencia de los datos.<br>7. Customer Service almacena los cambios.<br>8. El sistema confirma que la actualización fue realizada. |
| **Escenario Alternativo** | **1. Cliente inexistente:** El sistema informa que el cliente no fue encontrado.<br><br>**2. Datos inválidos:** El sistema indica qué campos deben corregirse.<br><br>**3. Sin autorización:** La operación es rechazada porque el actor no puede modificar al cliente indicado.<br><br>**4. Datos duplicados:** Si un dato sujeto a unicidad ya pertenece a otro cliente, la actualización es rechazada.<br><br>**5. Error de persistencia:** Se conservan los datos anteriores y se informa el fallo. |
| **Requerimientos** | - Permitir la actualización controlada de datos personales.<br>- Validar los campos modificados.<br>- Comprobar que el actor tenga permiso sobre el cliente.<br>- Evitar actualizaciones parciales cuando ocurra un error.<br>- Conservar el `customerId` original. |

#### Diagrama de flujo — CDU-CUS-02

![Caso Expandido CUS-01](../Imagenes/Diagrama%20-%20CUS02.png)
### Caso de Uso: Activar Usuario

![Caso Expandido CUS-03](../Imagenes/Diagrama%20-%20CDUCSU03.png)


| Campo | Descripción |
|---|---|
| **ID Caso de Uso** | CDU-CUS-03 |
| **Nombre** | Activar usuario |
| **Módulo al que pertenece** | Customer Service |
| **Actor Principal** | Cliente |
| **RF cubiertos** | RF-04, RF-32 y RF-33 |
| **Precondiciones** | - El cliente debe estar previamente registrado.<br>- El usuario debe encontrarse pendiente de activación.<br>- El cliente debe poseer una dirección de correo electrónico registrada.<br>- Debe tratarse del proceso de activación correspondiente al primer ingreso del usuario. |
| **Postcondiciones** | - La identidad del usuario queda validada mediante el enlace correspondiente.<br>- El estado del usuario cambia a activo.<br>- El cliente queda habilitado para utilizar las funcionalidades permitidas del sistema. |
| **Escenario Principal** | 1. El cliente realiza el proceso correspondiente a su primer ingreso al sistema.<br>2. Customer Service identifica que el usuario todavía no se encuentra activo.<br>3. El sistema solicita el envío de un enlace de activación al correo registrado.<br>4. Se ejecuta el CDU-NOT-01 **Enviar notificación** para enviar el enlace al cliente.<br>5. El cliente recibe el correo electrónico.<br>6. El cliente accede al enlace de activación.<br>7. Customer Service valida el enlace recibido.<br>8. El sistema cambia el estado del usuario a activo.<br>9. El sistema informa que la cuenta fue activada correctamente. |
| **Escenario Alternativo** | **1. Enlace inválido:** Si el enlace no es válido, el sistema rechaza la activación.<br><br>**2. Enlace expirado:** El sistema informa al usuario y permite iniciar nuevamente el proceso correspondiente.<br><br>**3. Usuario ya activo:** El sistema informa que la cuenta ya se encuentra activada.<br><br>**4. Error en el proceso:** El usuario permanece en su estado anterior y el sistema informa que no fue posible completar la activación. |
| **Requerimientos** | - El sistema debe permitir activar a un usuario registrado.<br>- La activación debe realizarse mediante un enlace de validación.<br>- Debe enviarse el enlace al correo electrónico registrado.<br>- Una activación correcta debe modificar el estado del usuario.<br>- Para el envío del enlace se relaciona con `CDU-NOT-01 - Enviar notificación`. |

![Caso Expandido CUS-03](../Imagenes/Diagrama%20-%20CUS03.png)

---
### Caso de Uso: Iniciar sesión

#### Diagrama de caso de uso

![Caso Expandido CUS-02](../Imagenes/Diagrama%20-%20CDUCSU04.png)

| Campo | Descripción |
|---|---|
| **ID Caso de Uso** | CDU-CUS-04 |
| **Nombre** | Iniciar sesión |
| **Módulo al que pertenece** | Customer Service |
| **Actor Principal** | Administrador / Cajero Receptor / Cliente |
| **RF cubiertos** | RF-05, RF-25 y RF-26 |
| **Precondiciones** | - El usuario debe estar registrado.<br>- El usuario debe poseer credenciales.<br>- El sistema debe tener disponible la configuración utilizada para firmar el JWT. |
| **Postcondiciones** | - Si las credenciales y el estado son válidos, se genera un JWT.<br>- El token contiene la identidad y el rol del usuario.<br>- El actor queda habilitado únicamente para las funciones permitidas por su rol. |
| **Escenario Principal** | 1. El usuario selecciona **Iniciar sesión**.<br>2. El sistema solicita sus credenciales.<br>3. El usuario ingresa username o correo y contraseña.<br>4. Customer Service busca al usuario.<br>5. El servicio valida la contraseña.<br>6. El sistema comprueba que el usuario esté activo.<br>7. El servicio identifica el rol del usuario.<br>8. Customer Service genera un JWT firmado.<br>9. El sistema entrega el token y permite el acceso según el rol. |
| **Escenario Alternativo** | **1. Credenciales inválidas:** El sistema rechaza el inicio de sesión sin indicar cuál dato falló.<br><br>**2. Usuario pendiente:** Se informa que debe completar la activación.<br><br>**3. Usuario bloqueado:** El acceso es rechazado.<br><br>**4. Error al generar el token:** No se inicia la sesión y se informa el error.<br><br>**5. Token expirado en solicitudes posteriores:** El sistema solicita iniciar sesión nuevamente. |
| **Requerimientos** | - Autenticar usuarios mediante credenciales válidas.<br>- Generar un JWT firmado y con expiración.<br>- Incluir identidad y rol en el token.<br>- Validar el estado del usuario antes de conceder acceso.<br>- Aplicar los permisos de Administrador, Cajero Receptor y Cliente. |

#### Diagrama de flujo — CDU-CUS-04

![Caso Expandido CUS-03](../Imagenes/Diagrama%20-%20CUS04.png)

## Casos de Uso de Customer Service
### Caso de Uso: Gestionar Estado del Usuario

![Caso Expandido CUS-05](../Imagenes/Diagrama%20-%20CDUCSU05.png)


| Campo | Descripción |
|---|---|
| **ID Caso de Uso** | CDU-CUS-05 |
| **Nombre** | Gestionar estado del usuario |
| **Módulo al que pertenece** | Customer Service |
| **Actor Principal** | Administrador |
| **RF cubiertos** | RF-33 |
| **Precondiciones** | - El Administrador debe haber iniciado sesión en el sistema.<br>- El usuario que se desea gestionar debe estar registrado.<br>- El Administrador debe poseer permisos para realizar cambios sobre el estado de los usuarios. |
| **Postcondiciones** | - El estado del usuario queda actualizado según la acción realizada por el Administrador.<br>- La modificación queda almacenada en Customer Service.<br>- Las acciones disponibles para el usuario quedan determinadas por su nuevo estado. |
| **Escenario Principal** | 1. El Administrador selecciona la opción **Gestionar usuarios**.<br>2. El sistema muestra los usuarios registrados.<br>3. El Administrador selecciona un usuario.<br>4. El sistema muestra la información y estado actual del usuario.<br>5. El Administrador selecciona el nuevo estado permitido.<br>6. El sistema solicita confirmar la modificación.<br>7. El Administrador confirma el cambio.<br>8. Customer Service actualiza el estado del usuario.<br>9. El sistema confirma que la modificación fue realizada correctamente. |
| **Escenario Alternativo** | **1. Usuario inexistente:** Si el usuario ya no existe o no puede encontrarse, el sistema cancela la operación.<br><br>**2. Estado no válido:** Si se intenta establecer un estado no permitido, el sistema rechaza el cambio.<br><br>**3. Operación cancelada:** Si el Administrador no confirma la modificación, se conserva el estado anterior.<br><br>**4. Error durante la actualización:** El sistema conserva el estado anterior e informa que no fue posible realizar el cambio. |
| **Requerimientos** | - Solamente un Administrador autorizado debe poder gestionar el estado de los usuarios.<br>- Customer Service debe validar el estado antes de realizar el cambio.<br>- La modificación debe realizarse únicamente en la información administrada por Customer Service.<br>- Un error durante el proceso no debe dejar al usuario en un estado inconsistente. |

![Caso Expandido CUS-05](../Imagenes/Diagrama%20-%20CUS05.png)

---

## Casos de Uso de Account Service
### Caso de Uso: Crear Cuenta Bancaria

![Caso Expandido ACC-01](../Imagenes/Diagrama%20-%20CDUACC01.png)


| Campo | Descripción |
|---|---|
| **ID Caso de Uso** | CDU-ACC-01 |
| **Nombre** | Crear cuenta bancaria |
| **Módulo al que pertenece** | Account Service |
| **Actor Principal** | Cajero Receptor |
| **RF cubiertos** | RF-08, RF-09 y RF-34 |
| **Precondiciones** | - El Cajero Receptor debe haber iniciado sesión.<br>- El cliente debe estar registrado previamente.<br>- Debe existir un `customerId` correspondiente al cliente. |
| **Postcondiciones** | - Se crea una nueva cuenta bancaria.<br>- La cuenta queda asociada al `customerId` correspondiente.<br>- Se genera un `accountId` único.<br>- Se registra el tipo, balance y estado correspondiente de la cuenta. |
| **Escenario Principal** | 1. El Cajero Receptor selecciona **Crear cuenta bancaria**.<br>2. El sistema muestra el formulario correspondiente.<br>3. El Cajero Receptor identifica al cliente mediante su `customerId`.<br>4. Selecciona el tipo de cuenta: **Monetaria** o **Ahorro**.<br>5. Ingresa la información requerida para la creación.<br>6. El sistema valida los datos proporcionados.<br>7. Account Service genera un `accountId`.<br>8. El sistema asigna el estado inicial correspondiente.<br>9. Account Service registra la nueva cuenta.<br>10. El sistema informa que la cuenta fue creada correctamente. |
| **Escenario Alternativo** | **1. Cliente inválido:** La creación de la cuenta es rechazada.<br><br>**2. Tipo de cuenta no seleccionado:** El sistema solicita seleccionar un tipo válido.<br><br>**3. Datos inválidos:** El sistema solicita corregir la información proporcionada.<br><br>**4. Error durante la creación:** La operación se cancela sin dejar una cuenta parcialmente registrada. |
| **Requerimientos** | - El sistema debe permitir crear cuentas para clientes registrados.<br>- Cada cuenta debe poseer un `accountId` único.<br>- La cuenta debe estar asociada a un `customerId`.<br>- Deben contemplarse cuentas Monetarias y de Ahorro.<br>- Account Service debe utilizar exclusivamente su propia base de datos.<br>- La interacción necesaria con otros microservicios debe ser asíncrona. |

![Caso Expandido ACC-01](../Imagenes/Diagrama%20-%20ACC01.png)

---
### Caso de Uso: Consultar saldo

#### Diagrama de caso de uso — CDU-ACC-02

![Caso Expandido ACC-01](../Imagenes/Diagrama%20-%20CDUACC02.png)

| Campo | Descripción |
|---|---|
| **ID Caso de Uso** | CDU-ACC-02 |
| **Nombre** | Consultar saldo |
| **Módulo al que pertenece** | Account Service |
| **Actor Principal** | Cliente / Cajero Receptor |
| **RF cubiertos** | RF-10 |
| **Precondiciones** | - El actor debe haber iniciado sesión.<br>- La cuenta debe estar registrada.<br>- El Cliente solo puede consultar cuentas de su propiedad.<br>- El Cajero Receptor debe poseer autorización para realizar la consulta. |
| **Postcondiciones** | - El sistema muestra el saldo actual y la moneda de la cuenta.<br>- La información financiera no es modificada.<br>- Si la cuenta no puede consultarse, no se expone información sensible. |
| **Escenario Principal** | 1. El actor selecciona **Consultar saldo**.<br>2. El sistema solicita o recibe el identificador de la cuenta.<br>3. Account Service busca la cuenta.<br>4. El sistema valida que el actor pueda consultarla.<br>5. El servicio obtiene el saldo y la moneda.<br>6. El sistema muestra el saldo actual al actor. |
| **Escenario Alternativo** | **1. Cuenta inexistente:** Se informa que la cuenta no fue encontrada.<br><br>**2. Sin autorización:** Se rechaza la consulta.<br><br>**3. Cuenta cerrada o bloqueada:** Se muestra el estado correspondiente junto con la información permitida.<br><br>**4. Error de consulta:** Se informa que no fue posible obtener el saldo. |
| **Requerimientos** | - Consultar el saldo sin modificarlo.<br>- Validar la identidad y autorización del actor.<br>- Mostrar la moneda asociada.<br>- No permitir que un Cliente consulte cuentas ajenas. |

#### Diagrama de flujo — CDU-ACC-02

![Caso Expandido ACC-01](../Imagenes/Diagrama%20-%20ACC02.png)

### Caso de Uso: Actualizar balance

#### Diagrama de caso de uso — CDU-ACC-03

![Caso Expandido ACC-01](../Imagenes/Diagrama%20-%20CDUACC03.png)

| Campo | Descripción |
|---|---|
| **ID Caso de Uso** | CDU-ACC-03 |
| **Nombre** | Actualizar balance |
| **Módulo al que pertenece** | Account Service |
| **Actor Principal** | Sistema |
| **RF cubiertos** | RF-11 y RF-36 |
| **Precondiciones** | - Debe existir una operación financiera válida.<br>- Debe existir la cuenta afectada.<br>- El mensaje debe incluir identificadores de operación y correlación.<br>- El monto debe ser mayor que cero. |
| **Postcondiciones** | - El saldo se actualiza de forma atómica.<br>- Se registra el movimiento y los saldos anterior y nuevo.<br>- Se actualiza la última actividad de la cuenta.<br>- Se publica el resultado de la operación.<br>- El saldo nunca queda negativo. |
| **Escenario Principal** | 1. Account Service recibe un comando financiero mediante RabbitMQ.<br>2. El servicio valida que el mensaje no haya sido procesado.<br>3. Busca la cuenta indicada.<br>4. Valida el estado de la cuenta y el monto.<br>5. Si es débito, verifica fondos suficientes.<br>6. Calcula el nuevo saldo.<br>7. Actualiza el balance de forma atómica.<br>8. Registra el movimiento de cuenta.<br>9. Actualiza la fecha de última actividad.<br>10. Evalúa la frecuencia de uso conforme a CDU-ACC-06; la anomalía no revierte por sí sola el movimiento confirmado.<br>11. Publica el evento de operación completada. |
| **Escenario Alternativo** | **1. Cuenta inexistente:** Se publica el rechazo correspondiente.<br><br>**2. Cuenta no activa:** Se rechaza el débito.<br><br>**3. Fondos insuficientes:** No se modifica el saldo y se publica el rechazo.<br><br>**4. Operación duplicada:** Se devuelve el resultado previo sin aplicar nuevamente el movimiento.<br><br>**5. Conflicto de concurrencia:** La actualización se reintenta o se rechaza sin perder consistencia.<br><br>**6. Error de persistencia:** La transacción se revierte. |
| **Requerimientos** | - Actualizar balances únicamente por operaciones válidas.<br>- Impedir balances negativos.<br>- Garantizar atomicidad e idempotencia.<br>- Registrar cada movimiento financiero.<br>- Mantener la trazabilidad mediante `id_operacion` e `id_correlacion`. |

#### Diagrama de flujo — CDU-ACC-03

![Caso Expandido ACC-01](../Imagenes/Diagrama%20-%20ACC03.png)

### Caso de Uso: Desactivar cuenta inactiva

#### Diagrama de caso de uso — CDU-ACC-04

![Caso Expandido ACC-01](../Imagenes/Diagrama%20-%20CDUACC04.png)

| Campo | Descripción |
|---|---|
| **ID Caso de Uso** | CDU-ACC-04 |
| **Nombre** | Desactivar cuenta inactiva |
| **Módulo al que pertenece** | Account Service |
| **Actor Principal** | Sistema |
| **RF cubiertos** | RF-12 y RF-35 |
| **Precondiciones** | - Debe ejecutarse el proceso programado de inactividad.<br>- La cuenta debe encontrarse activa.<br>- La cuenta debe tener registrada su última actividad financiera. |
| **Postcondiciones** | - Las cuentas que cumplen todas las condiciones cambian a estado inactivo.<br>- Las cuentas que no cumplen las condiciones conservan su estado.<br>- Se registra o publica el cambio para mantener trazabilidad. |
| **Escenario Principal** | 1. El sistema inicia el proceso programado.<br>2. Account Service busca cuentas activas candidatas.<br>3. Para cada cuenta, consulta el saldo y la última actividad.<br>4. Verifica que el saldo sea menor a Q50.00.<br>5. Verifica que hayan transcurrido al menos seis meses sin actividad.<br>6. Cambia el estado de la cuenta a inactiva.<br>7. Guarda la actualización.<br>8. Registra o publica el cambio de estado.<br>9. Continúa con la siguiente cuenta. |
| **Escenario Alternativo** | **1. Saldo igual o mayor a Q50.00:** La cuenta permanece activa.<br><br>**2. Inactividad menor a seis meses:** La cuenta permanece activa.<br><br>**3. Cuenta bloqueada o cerrada:** No se modifica mediante este proceso.<br><br>**4. Sin fecha de actividad aplicable:** Se aplica la fecha de creación según la regla implementada o se omite la cuenta.<br><br>**5. Error en una cuenta:** Se registra el error y el proceso continúa con las demás. |
| **Requerimientos** | - Ejecutar periódicamente la evaluación de inactividad.<br>- Exigir simultáneamente saldo menor a Q50.00 y seis meses de inactividad.<br>- Manejar los estados activa, inactiva, bloqueada y cerrada.<br>- No detener todo el proceso por el fallo de una cuenta. |

#### Diagrama de flujo — CDU-ACC-04

![Caso Expandido ACC-01](../Imagenes/Diagrama%20-%20ACC04.png)

## Casos de Uso de Account Service (continuación)

#### Diagrama de caso de uso — CDU-ACC-05

![Caso Expandido ACC-05](../Imagenes/CDU-ACC-05%20-%20Caso%20de%20Uso.drawio.png)
## CDU-ACC-05 — Consultar estado de cuenta

| Campo | Descripción |
|---|---|
| **ID Caso de Uso** | CDU-ACC-05 |
| **Nombre** | Consultar estado de cuenta |
| **Módulo al que pertenece** | Account Service |
| **Actor Principal** | Cliente / Cajero Receptor / Administrador |
| **RF cubiertos** | RF-35 |
| **Precondiciones** | - El usuario debe estar autenticado mediante JWT válido.<br>- La cuenta bancaria debe existir en la base de datos de Account Service.<br>- El actor debe poseer autorización (ser el titular, Cajero Receptor o Administrador). |
| **Postcondiciones** | - Se presenta al usuario la información detallada de la cuenta: estado actual (ACTIVE, INACTIVE, BLOCKED, CLOSED), tipo de cuenta, balance actual y fecha/hora de la última actividad financiera.<br>- No se produce ninguna mutación de estado en la base de datos. |
| **Escenario Principal** | 1. El usuario selecciona la opción **Consultar estado de cuenta** e indica la cuenta.<br>2. El API Gateway intercepta la solicitud, valida la firma y expiración del JWT y corrobora los permisos según el rol.<br>3. Account Service consulta la cuenta en su base de datos (`account_db`).<br>4. El servicio extrae el balance, estado y fecha de última actividad registrada.<br>5. El sistema entrega la información al frontend para su visualización clara. |
| **Escenario Alternativo** | **1. Token inválido o expirado:** El sistema rechaza la petición con código 401 Unauthorized.<br><br>**2. Cuenta no perteneciente al cliente:** Si un cliente intenta ver la cuenta de otro usuario sin permisos de cajero/admin, el sistema responde 403 Forbidden.<br><br>**3. Cuenta inexistente:** El sistema informa que el identificador de cuenta no fue localizado.<br><br>**4. Cuenta inactiva o bloqueada:** El sistema muestra los datos y añade una advertencia explícita sobre las restricciones operativas vigentes. |
| **Requerimientos** | - Gestionar y reflejar con exactitud los 4 estados de cuenta: `ACTIVE`, `INACTIVE`, `BLOCKED`, `CLOSED`.<br>- Mantener y reportar la fecha de última actividad financiera.<br>- Garantizar el aislamiento de persistencia consultando únicamente `account_db`. |


#### Diagrama de flujo — CDU-ACC-05

![Caso flujo ACC-05](../Imagenes/CDU-ACC-05%20-%20Flujo.drawio.png)

## Casos de Uso de Transaction Service

### Caso de Uso: Realizar Transferencia

![Caso Expandido TRX-01](../Imagenes/Diagrama%20-%20CDUTRX01.png)


| Campo | Descripción |
|---|---|
| **ID Caso de Uso** | CDU-TRX-01 |
| **Nombre** | Realizar transferencia |
| **Módulo al que pertenece** | Transaction Service |
| **Actor Principal** | Cliente / Cajero Receptor |
| **RF cubiertos** | RF-13, RF-14, RF-15 y RF-37 |
| **Precondiciones** | - El actor debe haber iniciado sesión.<br>- Debe existir una cuenta origen.<br>- Debe indicarse una cuenta destino.<br>- Debe ingresarse un monto válido para la transferencia. |
| **Postcondiciones** | - La solicitud de transferencia queda registrada.<br>- Se genera un `transactionId` para identificar la operación.<br>- Se conserva un `correlationId` para mantener la trazabilidad del flujo.<br>- La transferencia queda con el estado correspondiente según el resultado del procesamiento.<br>- Se propaga el contexto W3C `traceparent` y se correlaciona la telemetría conforme a CDU-TRX-06. |
| **Escenario Principal** | 1. El Cliente o Cajero Receptor selecciona **Realizar transferencia**.<br>2. El sistema solicita la cuenta origen, la cuenta destino y el monto.<br>3. El actor ingresa la información solicitada.<br>4. El sistema valida el formato de los datos.<br>5. Transaction Service genera un `transactionId` y un `correlationId`.<br>6. Transaction Service registra la solicitud de transferencia.<br>7. Se incluye el `CDU-TRX-02 - Ejecutar Saga de transferencia` para realizar el procesamiento distribuido.<br>8. Transaction Service recibe el resultado correspondiente al flujo.<br>9. El estado de la transferencia es actualizado.<br>10. El sistema informa el resultado de la operación al actor. |
| **Escenario Alternativo** | **1. Monto inválido:** El sistema solicita corregir el monto antes de iniciar la operación.<br><br>**2. Datos incompletos:** El sistema indica qué información debe completarse.<br><br>**3. Transferencia rechazada:** Si el CDU-TRX-02 determina que la operación no puede realizarse, Transaction Service registra el estado correspondiente.<br><br>**4. Fallo durante el procesamiento:** El manejo distribuido del fallo y sus compensaciones corresponde al `CDU-TRX-02 - Ejecutar Saga de transferencia`.<br><br>**5. Operación duplicada:** El sistema debe evitar procesar nuevamente una transferencia ya registrada. |
| **Requerimientos** | - Transaction Service debe permitir iniciar transferencias entre cuentas.<br>- Cada operación debe contar con un `transactionId` único.<br>- Debe mantenerse un `correlationId` durante el flujo.<br>- La transferencia debe registrar cuenta origen, cuenta destino, monto y estado.<br>- La ejecución distribuida de la transferencia debe delegarse al `CDU-TRX-02`.<br>- La comunicación entre microservicios debe realizarse mediante eventos asíncronos.<br>- Transaction Service no debe consultar directamente las bases de datos de otros servicios.<br>- El procesamiento debe contemplar idempotencia. |

![Caso Expandido TRX-01](../Imagenes/Diagrama%20-%20TRX01.png)



## CDU-TRX-02 — Ejecutar Saga de transferencia
![Caso Expandido TRX-02](../Imagenes/CDU-TRX-02%20-%20Caso%20de%20Uso.drawio.png)

| Campo | Descripción |
|---|---|
| **ID Caso de Uso** | CDU-TRX-02 |
| **Nombre** | Ejecutar Saga de transferencia |
| **Módulo al que pertenece** | Transaction Service |
| **Actor Principal** | Sistema |
| **RF cubiertos** | RF-16 y RF-38 |
| **Precondiciones** | - Debe existir una solicitud válida de transferencia con cuentas origen/destino, monto e identificador de correlación.<br>- Customer Service y Account Service deben poder atender las validaciones y operaciones mediante RabbitMQ.<br>- Debe propagarse el contexto de traza distribuida. |
| **Postcondiciones** | - Si el flujo es exitoso, Account Service confirma el débito y el crédito y la transferencia termina en `COMPLETADA`.<br>- Si falla una validación o se rechaza el débito sin afectar fondos, termina en `RECHAZADA`.<br>- Si el crédito falla después del débito, se solicita la restitución a Account Service; al confirmarse, termina en `COMPENSADA`.<br>- Si falla la restitución, queda en `COMPENSACION_FALLIDA` para atención operativa.<br>- Se conserva el historial de estados y su correlación con la traza. |
| **Escenario Principal** | 1. Transaction Service recibe la solicitud e inicia la Saga en `VALIDANDO_KYC`.<br>2. Solicita la validación KYC a Customer Service.<br>3. Con KYC válido, cambia a `VALIDANDO_CUENTAS` y solicita validar las cuentas a Account Service.<br>4. Con cuentas válidas, cambia a `DEBITANDO` y solicita el débito a Account Service.<br>5. Account Service realiza el débito de forma atómica e idempotente y confirma el resultado.<br>6. Transaction Service cambia a `ACREDITANDO` y solicita el crédito en la cuenta destino.<br>7. Account Service confirma la acreditación.<br>8. Transaction Service registra `COMPLETADA` y publica el resultado para auditoría y notificación.<br>9. Registra los spans o eventos de cada etapa conforme a CDU-TRX-06. |
| **Escenario Alternativo** | **1. KYC, cuenta o fondos inválidos:** La transferencia termina en `RECHAZADA`, sin compensación cuando no se aplicó débito.<br><br>**2. Crédito fallido tras débito exitoso:** Transaction Service cambia a `COMPENSANDO` y solicita a Account Service restituir el monto; al confirmarse, cambia a `COMPENSADA`.<br><br>**3. Compensación fallida:** Se registra `COMPENSACION_FALLIDA` y el contexto necesario para atención operativa.<br><br>**4. Mensaje duplicado:** Se recupera el resultado previo sin repetir movimientos financieros.<br><br>**5. Dependencia sin respuesta:** Se registra el fallo con su correlación; se aplica la política de timeout y recuperación configurada, sin asumir que una solicitud sin respuesta no fue ejecutada. |
| **Requerimientos** | - Coordinar la Saga desde Transaction Service mediante RabbitMQ, sin acceso directo a bases de datos ajenas.<br>- Utilizar los estados `VALIDANDO_KYC`, `VALIDANDO_CUENTAS`, `DEBITANDO`, `ACREDITANDO`, `COMPLETADA`, `RECHAZADA`, `COMPENSANDO`, `COMPENSADA` y `COMPENSACION_FALLIDA`.<br>- Delegar los movimientos financieros exclusivamente a Account Service.<br>- Propagar `correlationId` y `traceparent` y registrar cada cambio de estado.<br>- Mantener idempotencia y publicación confiable de eventos.<br>- Usar los contratos reales del repositorio; los nombres de eventos en inglés del documento anterior no se consideran contratos confirmados. |

---

#### Diagrama de flujo — CDU-TRX-02

![Caso Flujo 02](../Imagenes/CDU-TRX-02%20-%20Flujo.drawio.png)

---

## CDU-TRX-03 — Consultar estado de transferencia
![Caso Expandido TRX-03](../Imagenes/CDU-TRX-03%20-%20Caso%20de%20Uso.drawio.png)

| Campo | Descripción |
|---|---|
| **ID Caso de Uso** | CDU-TRX-03 |
| **Nombre** | Consultar estado de transferencia |
| **Módulo al que pertenece** | Transaction Service |
| **Actor Principal** | Cliente / Cajero Receptor |
| **RF cubiertos** | RF-29 y RF-38 |
| **Precondiciones** | - El usuario debe disponer de una sesión válida con JWT.<br>- Debe contarse con el identificador de la operación (`operationId` o `idTransferencia`). |
| **Postcondiciones** | - El sistema retorna el estado actual de la transferencia (`VALIDANDO_KYC`, `VALIDANDO_CUENTAS`, `DEBITANDO`, `ACREDITANDO`, `COMPLETADA`, `RECHAZADA`, `COMPENSANDO`, `COMPENSADA`, `COMPENSACION_FALLIDA`).<br>- No se altera la información financiera ni de auditoría. |
| **Escenario Principal** | 1. El cliente o cajero solicita consultar el estado de una transferencia a través del frontend.<br>2. El API Gateway valida el token JWT y los permisos del usuario.<br>3. El API Gateway consulta a Transaction Service por el estado asociado al `operationId`.<br>4. Transaction Service verifica la existencia y pertenencia de la transferencia en su base de datos (`transaction_db`).<br>5. El servicio retorna los datos: identificador, monto, cuentas involucradas, fecha y estado actual.<br>6. El frontend renderiza el resultado permitiendo al usuario conocer el progreso de la operación asíncrona. |
| **Escenario Alternativo** | **1. Operación no encontrada:** Se responde con error 404 indicando que el identificador no corresponde a ninguna transferencia registrada.<br><br>**2. Usuario no autorizado:** Si el cliente autenticado no es el emisor de la transferencia, el sistema deniega el acceso con 403 Forbidden.<br><br>**3. Operación en curso:** En `VALIDANDO_KYC`, `VALIDANDO_CUENTAS`, `DEBITANDO`, `ACREDITANDO` o `COMPENSANDO`, puede actualizarse la consulta hasta alcanzar un estado final. La actualización automática debe validarse contra la interfaz implementada. |
| **Requerimientos** | - Permitir el monitoreo del ciclo de vida asíncrono originado por respuestas `202 Accepted`.<br>- Validar estrictamente la pertenencia del recurso según la identidad contenida en el JWT.<br>- Consultar exclusivamente la persistencia de Transaction Service. |


---

#### Diagrama de flujo — CDU-TRX-03

![Caso Flujo 03](../Imagenes/CDU-TRX-03%20-%20Flujo.drawio.png)


## CDU-TRX-04 — Consultar historial de transacciones
![Caso Expandido TRX-04](../Imagenes/CDU-TRX-04%20-%20Caso%20de%20Uso.drawio.png)

| Campo | Descripción |
|---|---|
| **ID Caso de Uso** | CDU-TRX-04 |
| **Nombre** | Consultar historial de transacciones |
| **Módulo al que pertenece** | Transaction Service |
| **Actor Principal** | Cliente |
| **RF cubiertos** | RF-40 |
| **Precondiciones** | - El cliente debe estar autenticado con un token JWT válido.<br>- El cliente debe poseer al menos una cuenta bancaria asociada a su perfil. |
| **Postcondiciones** | - Se entrega un listado paginado y cronológico de todas las transferencias donde el cliente figure como emisor o receptor.<br>- Los datos son exclusivamente de lectura. |
| **Escenario Principal** | 1. El cliente accede a la pestaña de **Historial de Transferencias** en la aplicación web.<br>2. El API Gateway valida la sesión y propaga la identidad del cliente extraída del JWT.<br>3. Transaction Service consulta en `transaction_db` todas las transacciones asociadas a las cuentas del cliente.<br>4. El servicio ordena los registros de manera descendente por fecha (`occurredAt`).<br>5. El frontend recibe la colección de transacciones y renderiza la tabla con montos, fechas, cuentas y estados finales. |
| **Escenario Alternativo** | **1. Sesión caducada:** El frontend redirige a la pantalla de inicio de sesión.<br><br>**2. Sin transferencias previas:** El sistema responde exitosamente con un arreglo vacío y la interfaz muestra el mensaje informativo "No hay transferencias registradas".<br><br>**3. Error de conexión:** Se despliega un mensaje amigable solicitando reintentar la consulta sin exponer detalles internos de la base de datos. |
| **Requerimientos** | - Filtrar estrictamente las operaciones para que ningún cliente pueda visualizar transferencias de terceros.<br>- Proveer orden cronológico y soporte para paginación.<br>- Desplegar montos convertidos a formato legible a partir de centavos en GTQ. |


---

#### Diagrama de flujo — CDU-TRX-04

![Caso Flujo 04](../Imagenes/CDU-TRX-04%20-%20Flujo.drawio.png)


## Casos de Uso de Payment Service

## CDU-PAY-01 — Procesar pago
![Caso Expandido PAY-01](../Imagenes/CDU-PAY-01%20-%20Caso%20de%20Uso.drawio.png)

| Campo | Descripción |
|---|---|
| **ID Caso de Uso** | CDU-PAY-01 |
| **Nombre** | Procesar pago |
| **Módulo al que pertenece** | Payment Service |
| **Actor Principal** | Cliente / Cajero Receptor |
| **RF cubiertos** | RF-17, RF-18, RF-19, RF-20 y RF-39 |
| **Precondiciones** | - El usuario debe estar autenticado en el sistema.<br>- La cuenta de débito debe existir, estar activa y pertenecer al cliente.<br>- El cliente debe tener KYC `VERIFIED`.<br>- Los datos del servicio y monto deben cumplir las validaciones de formato. |
| **Postcondiciones** | - Si el pago es exitoso: el balance de la cuenta queda debitado, el pago se registra en estado `COMPLETADO` y se publica el evento de confirmación.<br>- Si falla la llamada externa: se compensa el débito en la cuenta y el pago queda en estado `RECHAZADO`.<br>- Se emiten los eventos de auditoría correspondientes. |
| **Escenario Principal** | 1. El usuario selecciona **Nuevo Pago**, ingresa la cuenta origen, tipo de pago (`INTERNO` o `EXTERNO`), proveedor/servicio y monto. Para un pago externo selecciona el resultado simulado (`EXITO`, `FALLO` o `TIMEOUT`).<br>2. El sistema valida el formato de entrada y el API Gateway responde con `202 Accepted` conteniendo el `operationId`.<br>3. Payment Service registra el pago con estado `VALIDANDO_KYC` y publica `cliente.kyc.validacion.solicitada`.<br>4. Customer Service responde `cliente.kyc.verificado`; Payment Service cambia a `VALIDANDO_CUENTA` y publica `cuenta.transferencia.validacion.solicitada`.<br>5. Account Service responde `cuenta.transferencia.validada`; Payment Service cambia a `PROCESANDO` y publica `cuenta.debito.solicitado`.<br>6. Account Service valida fondos, realiza el débito y emite `cuenta.debitada`.<br>7. Si el pago es `EXTERNO`, Payment Service simula la respuesta del proveedor; `EXITO` conserva el débito y genera una referencia externa. En Fase 3, los parámetros de fallo, latencia y timeout se documentan en CDU-PAY-04; la selección manual del escenario se conserva donde siga disponible en la implementación.<br>8. Payment Service actualiza el pago a `COMPLETADO`, emite `pago.completado` y deja el resultado disponible para consulta. |
| **Escenario Alternativo** | **1. Fondos insuficientes en la cuenta:** Account Service emite `cuenta.debito.rechazada`. Payment Service marca el pago como `RECHAZADO` y finaliza.<br><br>**2. Fallo simulado:** Payment Service registra el código `PROVEEDOR_EXTERNO`, transiciona a `COMPENSANDO`, publica `cuenta.compensacion.solicitada` y finaliza como `RECHAZADO` después de restituir los fondos.<br><br>**3. Timeout simulado:** Payment Service registra `TIMEOUT_PROVEEDOR` y ejecuta el mismo flujo de compensación para evitar perder el dinero debitado.<br><br>**4. Datos o escenario inválidos:** El sistema rechaza la solicitud antes de publicarla. |
| **Requerimientos** | - Soportar tipos de pagos `INTERNO` y `EXTERNO`.<br>- Validar KYC y reglas de cuenta antes del débito.<br>- Gestionar los estados: `VALIDANDO_KYC`, `VALIDANDO_CUENTA`, `PROCESANDO`, `COMPLETADO`, `COMPENSANDO`, `RECHAZADO`.<br>- Manejar fallos externos mediante compensación asíncrona hacia Account Service.<br>- Garantizar persistencia con Outbox e idempotencia operativa. |

---

#### Diagrama de flujo — CDU-PAY-01

![Caso Flujo 01](../Imagenes/CDU-PAY-01%20-%20Flujo.drawio.png)


## CDU-PAY-02 — Consultar pagos
![Caso Expandido PAY-02](../Imagenes/CDU-PAY-02%20-%20Caso%20de%20Uso.drawio.png)


| Campo | Descripción |
|---|---|
| **ID Caso de Uso** | CDU-PAY-02 |
| **Nombre** | Consultar pagos |
| **Módulo al que pertenece** | Payment Service |
| **Actor Principal** | Cliente / Cajero Receptor |
| **RF cubiertos** | RF-29, RF-39 y RF-40 |
| **Precondiciones** | - El usuario debe estar autenticado con token JWT válido.<br>- Para consulta individual, se debe proporcionar el `idPago` u `operationId`. Para listado general, se requiere la identidad del cliente. |
| **Postcondiciones** | - Se entrega el detalle del pago individual o el listado histórico de pagos asociados al actor.<br>- La operación es de sólo lectura. |
| **Escenario Principal** | 1. El usuario solicita consultar su historial de pagos o el estado de un pago específico desde el frontend.<br>2. El API Gateway valida el JWT y comprueba las reglas de autorización.<br>3. Payment Service realiza la búsqueda en `payment_db` filtrando por el ID de pago o por las cuentas del cliente.<br>4. Payment Service recupera los registros: identificador, tipo (`INTERNO`/`EXTERNO`), proveedor, monto, estado actual y fecha de creación.<br>5. El sistema entrega la información y el frontend la presenta en la tabla o tarjeta de detalle de pago. |
| **Escenario Alternativo** | **1. Pago no encontrado:** El sistema responde con código 404 indicando que el registro no existe.<br><br>**2. Consulta no autorizada:** Si el cliente intenta consultar un pago perteneciente a otro usuario, se responde 403 Forbidden.<br><br>**3. Sin pagos previos:** Se responde con lista vacía y el frontend muestra el mensaje descriptivo correspondiente. |
| **Requerimientos** | - Consultar estado de operaciones individuales originadas de forma asíncrona.<br>- Listar historial financiero de pagos del cliente.<br>- Consultar exclusivamente la persistencia de Payment Service (`payment_db`). |


---

#### Diagrama de flujo — CDU-PAY-02

![Caso Flujo 02](../Imagenes/CDU-PAY-02%20-%20Flujo.drawio.png)
---

## Casos de Uso de Notification & Audit Service
### Caso de Uso: Enviar Notificación

![Caso Expandido NOT-01](../Imagenes/Diagrama%20-%20CDUNOT01.png)


| Campo | Descripción |
|---|---|
| **ID Caso de Uso** | CDU-NOT-01 |
| **Nombre** | Enviar notificación |
| **Módulo al que pertenece** | Notification & Audit Service |
| **Actor Principal** | Sistema |
| **RF cubiertos** | RF-21 y RF-32 |
| **Precondiciones** | - Debe existir un evento que requiera generar una notificación.<br>- El evento debe haber sido publicado mediante el mecanismo de mensajería.<br>- El evento debe contener la información necesaria para generar la notificación. |
| **Postcondiciones** | - La solicitud de notificación queda procesada.<br>- La notificación correspondiente es generada y enviada al destinatario.<br>- Notification & Audit Service finaliza el procesamiento del evento recibido. |
| **Escenario Principal** | 1. Se genera un evento que requiere notificar a un usuario.<br>2. El evento es publicado mediante el broker de mensajería.<br>3. Notification & Audit Service consume el evento.<br>4. El servicio identifica el tipo de notificación requerida.<br>5. El sistema obtiene la información necesaria del evento.<br>6. Se genera la notificación correspondiente.<br>7. La notificación es enviada al destinatario.<br>8. El servicio confirma el procesamiento correspondiente. |
| **Escenario Alternativo** | **1. Evento inválido:** Si el evento no contiene la información necesaria, la notificación no es enviada.<br><br>**2. Tipo de notificación no reconocido:** El sistema rechaza el procesamiento correspondiente.<br><br>**3. Error durante el envío:** El sistema registra el fallo correspondiente para permitir su manejo posterior.<br><br>**4. Evento duplicado:** El sistema debe evitar enviar dos veces una misma notificación.<br><br>**5. Fallo temporal:** El evento puede ser procesado nuevamente mediante el mecanismo de reintentos definido. |
| **Requerimientos** | - Notification & Audit Service debe permitir generar notificaciones a partir de eventos.<br>- La comunicación debe realizarse mediante mecanismos asíncronos.<br>- El servicio debe permanecer desacoplado de los demás microservicios.<br>- El procesamiento debe contemplar idempotencia.<br>- El fallo en el envío de una notificación no debe afectar directamente la operación de negocio que originó el evento.<br>- El registro formal de auditoría corresponde al `CDU-NOT-02 - Registrar evento de auditoría`. |

![Caso Expandido TRX-01](../Imagenes/Diagrama%20-%20NOT01.png)

### Caso de Uso: Registrar evento de auditoría

#### Diagrama de caso de uso — CDU-NOT-02

![Caso Expandido NOT-01](../Imagenes/Diagrama%20-%20CDUNOT02.png)

| Campo | Descripción |
|---|---|
| **ID Caso de Uso** | CDU-NOT-02 |
| **Nombre** | Registrar evento de auditoría |
| **Módulo al que pertenece** | Notification & Audit Service |
| **Actor Principal** | Sistema |
| **RF cubiertos** | RF-22, RF-23 y RF-24 |
| **Precondiciones** | - Un microservicio debe haber publicado un evento en RabbitMQ.<br>- El evento debe contener un identificador, tipo, fecha y payload.<br>- Debe incluir un `correlationId` que permita relacionarlo con la operación distribuida. |
| **Postcondiciones** | - El evento queda almacenado para fines de auditoría.<br>- Se conserva su información original relevante.<br>- El registro puede relacionarse con otros eventos mediante `correlationId`.<br>- El mismo mensaje no se registra dos veces.<br>- Se conserva la correlación con `traceId`, `spanId` y `traceparent` cuando vienen en el contexto del mensaje; la correspondencia con columnas persistidas debe validarse contra el esquema real. |
| **Escenario Principal** | 1. Un microservicio publica un evento de dominio.<br>2. RabbitMQ entrega el evento a Notification & Audit Service.<br>3. El servicio valida el contrato del mensaje.<br>4. Comprueba que el evento no haya sido procesado.<br>5. Extrae el identificador, tipo, fecha, payload y `correlationId`, junto con el contexto de traza disponible, sin registrar secretos ni datos personales innecesarios.<br>6. Guarda el registro de auditoría.<br>7. Marca el mensaje como procesado.<br>8. Confirma el procesamiento al broker. |
| **Escenario Alternativo** | **1. Evento inválido:** El mensaje se rechaza o se dirige al mecanismo de fallos.<br><br>**2. Evento duplicado:** Se reconoce sin crear otro registro.<br><br>**3. Falta `correlationId`:** El evento se registra como inválido o se rechaza según el contrato.<br><br>**4. Error de base de datos:** No se confirma el mensaje para permitir su reintento.<br><br>**5. Máximo de reintentos alcanzado:** El mensaje se envía a la cola de mensajes muertos. |
| **Requerimientos** | - Consumir eventos de forma asíncrona.<br>- Conservar identificador, tipo, fecha, payload y `correlationId`.<br>- Garantizar idempotencia.<br>- Permitir la trazabilidad de operaciones distribuidas.<br>- Aplicar reintentos y cola de mensajes muertos cuando corresponda. |

#### Diagrama de flujo — CDU-NOT-02

![Caso Expandido TRX-01](../Imagenes/Diagrama%20-%20NOT02.png)

---

### Caso de Uso: Gestionar Estado KYC del Cliente

![Caso Expandido CUS-KYC](../Imagenes/CDU-CUS-KYC_%20Gestionar%20Estado%20KYC%20del%20Cliente.drawio.png)


| Campo | Descripción |
|---|---|
| **ID Caso de Uso** | CDU-CUS-KYC |
| **Nombre** | Gestionar estado KYC del cliente |
| **Módulo al que pertenece** | Customer Service |
| **Actor Principal** | Administrador / Sistema |
| **RF cubiertos** | RF-41, RF-42 y RF-43 |
| **Precondiciones** | - El cliente debe estar registrado en el sistema con estado activo.<br>- El Administrador debe haber iniciado sesión y poseer los permisos correspondientes.<br>- El cliente debe tener el campo `kyc_status` en estado PENDING o REJECTED. |
| **Postcondiciones** | - El estado KYC del cliente (`kyc_status`) queda actualizado a VERIFIED o REJECTED.<br>- El campo `kyc_updated_at` refleja la fecha y hora de la última modificación.<br>- Si el resultado es VERIFIED, el cliente queda habilitado para realizar transferencias.<br>- Si el resultado es REJECTED, las operaciones de alto valor quedan bloqueadas para el cliente.<br>- El evento `customer.kyc.updated` es publicado hacia el broker de mensajería.<br>- El cambio de ciclo de vida se correlaciona mediante `correlationId` y, cuando existe contexto activo, `traceId` y `spanId`, conforme a CDU-CUS-06. |
| **Escenario Principal** | 1. El Administrador selecciona la opción **Gestionar KYC de clientes**.<br>2. El sistema muestra los clientes con `kyc_status` en PENDING.<br>3. El Administrador selecciona un cliente y revisa su documentación.<br>4. El Administrador elige la acción: **Aprobar** o **Rechazar**.<br>5. El sistema solicita confirmación de la acción.<br>6. El Administrador confirma la decisión.<br>7. Customer Service actualiza `kyc_status` al valor correspondiente.<br>8. Customer Service actualiza el campo `kyc_updated_at` con la fecha actual.<br>9. Customer Service publica el evento `customer.kyc.updated`.<br>10. El sistema informa que el estado KYC fue actualizado correctamente. |
| **Escenario Alternativo** | **1. Cliente no encontrado:** El sistema informa que el cliente no existe y cancela la operación.<br><br>**2. Estado KYC no elegible:** Si el cliente ya tiene `kyc_status = VERIFIED`, el sistema rechaza una nueva aprobación redundante.<br><br>**3. Acción cancelada por el Administrador:** El estado KYC permanece sin cambios.<br><br>**4. Error durante la actualización:** Customer Service conserva el estado anterior y el evento no es publicado.<br><br>**5. Sistema automatizado:** El proceso puede ser iniciado por un sistema externo de validación documental que publique el resultado como evento; Customer Service consume dicho evento y aplica el mismo flujo a partir del paso 7. |
| **Requerimientos** | - Customer Service debe almacenar el estado KYC (`kyc_status`) para cada cliente.<br>- Los valores válidos de `kyc_status` son: PENDING, VERIFIED, REJECTED.<br>- Solo un Administrador autorizado o un sistema externo de validación puede modificar el estado KYC.<br>- La actualización debe ser atómica y registrar la fecha de modificación.<br>- El evento `customer.kyc.updated` debe publicarse tras cada cambio exitoso.<br>- Las operaciones de alto valor deben verificar `kyc_status = VERIFIED` antes de ejecutarse. |

![Flujo CUS-KYC](../Imagenes/CDU-CUS-KYC_%20Flujo%20de%20Gestión%20de%20Estado%20KYC.drawio.png)

---

### Caso de Uso: Consultar Historial de Transacciones Filtrado

![Caso Expandido TRX-05](../Imagenes/CDU-TRX-05_%20Consultar%20Historial%20de%20Transacciones%20Filtrado.drawio.png)


| Campo | Descripción |
|---|---|
| **ID Caso de Uso** | CDU-TRX-05 |
| **Nombre** | Consultar historial de transacciones filtrado |
| **Módulo al que pertenece** | Transaction Service |
| **Actor Principal** | Cliente / Cajero Receptor |
| **RF cubiertos** | RF-46, RF-47 y RF-48 |
| **Precondiciones** | - El actor debe haber iniciado sesión.<br>- Debe existir al menos una cuenta asociada al actor.<br>- Transaction Service debe tener acceso a los registros históricos de transacciones. |
| **Postcondiciones** | - El sistema devuelve el listado de transacciones que cumplen los criterios de filtrado aplicados.<br>- La respuesta incluye: fecha, tipo, monto, cuenta origen, cuenta destino y estado de cada transacción.<br>- Si no hay resultados, el sistema informa que no se encontraron registros para los filtros aplicados. |
| **Escenario Principal** | 1. El Cliente o Cajero Receptor selecciona **Consultar historial de transacciones**.<br>2. El sistema solicita los parámetros de filtrado disponibles: rango de fechas, tipo de transacción, estado y monto.<br>3. El actor aplica uno o más filtros.<br>4. El sistema valida que los parámetros ingresados sean coherentes (fechas válidas, montos positivos).<br>5. Transaction Service ejecuta la consulta con los filtros aplicados.<br>6. El sistema devuelve los resultados paginados.<br>7. El actor puede navegar entre páginas o exportar el resultado.<br>8. El sistema confirma la finalización de la consulta. |
| **Escenario Alternativo** | **1. Parámetros inválidos:** El sistema informa el error y solicita corregir los filtros antes de ejecutar la consulta.<br><br>**2. Sin resultados:** El sistema informa que no se encontraron transacciones para los criterios aplicados.<br><br>**3. Error durante la consulta:** El sistema informa el fallo y permite reintentar la operación.<br><br>**4. Consulta sin filtros:** El sistema devuelve el historial completo paginado, comenzando por las transacciones más recientes. |
| **Requerimientos** | - Transaction Service debe permitir consultar el historial de transacciones con filtros combinables.<br>- Los filtros soportados incluyen: rango de fechas, tipo de transacción, estado y monto mínimo/máximo.<br>- La consulta debe devolver resultados paginados.<br>- Transaction Service no debe consultar directamente bases de datos de otros microservicios.<br>- La operación debe ser de solo lectura; no puede modificar ningún registro. |

![Flujo TRX-05](../Imagenes/CDU-TRX-05_%20Flujo%20de%20Consulta%20de%20Historial%20Filtrado.drawio.png)

---

### Caso de Uso: Procesar Pago Externo Simulado

![Caso Expandido PAY-03](../Imagenes/CDU-PAY-03_%20Procesar%20Pago%20Externo%20Simulado.drawio.png)


| Campo | Descripción |
|---|---|
| **ID Caso de Uso** | CDU-PAY-03 |
| **Nombre** | Procesar pago externo simulado |
| **Módulo al que pertenece** | Transaction Service / Payment Gateway (simulado) |
| **Actor Principal** | Sistema |
| **RF cubiertos** | RF-49 y RF-50 |
| **Precondiciones** | - La cuenta origen debe estar activa y con saldo suficiente para cubrir el monto más la comisión aplicable.<br>- El cliente debe tener `kyc_status = VERIFIED`.<br>- Debe existir un identificador de pago externo (`externalPaymentId`) provisto por el solicitante.<br>- El servicio de simulación de pasarela de pago debe estar disponible. |
| **Postcondiciones** | - El pago queda registrado en Transaction Service con su `transactionId` único.<br>- La cuenta origen refleja el débito correspondiente al monto más la comisión.<br>- El evento `payment.processed` o `payment.failed` es publicado en el broker.<br>- El resultado del pago es notificado al cliente mediante Notification & Audit Service. |
| **Escenario Principal** | 1. El sistema recibe una solicitud de pago externo con `externalPaymentId`, `accountId`, monto y referencia.<br>2. Transaction Service valida que el `externalPaymentId` no haya sido procesado anteriormente (idempotencia).<br>3. Transaction Service verifica que la cuenta origen está activa y que el cliente tiene KYC verificado.<br>4. Transaction Service verifica que el saldo disponible cubre el monto más la comisión aplicable.<br>5. Transaction Service invoca el servicio de simulación de pasarela de pago.<br>6. La pasarela simulada devuelve el resultado: APPROVED o REJECTED.<br>7. Si el resultado es APPROVED, Transaction Service aplica el débito sobre la cuenta mediante Account Service.<br>8. Transaction Service registra la transacción con el estado correspondiente.<br>9. Transaction Service publica el evento resultante en el broker.<br>10. Notification & Audit Service procesa el evento y notifica al cliente. |
| **Escenario Alternativo** | **1. Pago duplicado:** Si el `externalPaymentId` ya existe, el sistema retorna el resultado original sin reprocesar.<br><br>**2. KYC no verificado:** El pago es rechazado y se registra el motivo; el evento `payment.failed` es publicado.<br><br>**3. Saldo insuficiente:** El sistema rechaza el pago, registra el fallo y publica `payment.failed`.<br><br>**4. Pasarela rechaza el pago:** El sistema registra el estado REJECTED y publica `payment.failed`.<br><br>**5. Fallo en Account Service:** Transaction Service aplica la compensación correspondiente y registra el error. |
| **Requerimientos** | - Transaction Service debe validar idempotencia usando `externalPaymentId`.<br>- Debe verificar KYC del cliente antes de procesar el pago.<br>- La pasarela de pago externa es simulada; debe retornar APPROVED o REJECTED de forma configurable.<br>- La comisión aplicable debe descontarse junto con el monto principal.<br>- El resultado debe publicarse como evento asíncrono en el broker.<br>- El fallo no debe dejar la cuenta en estado inconsistente (compensación obligatoria). |

![Flujo PAY-03](../Imagenes/CDU-PAY-03_%20Flujo%20de%20Procesamiento%20de%20Pago%20Externo%20Simulado.drawio.png)

---

### Caso de Uso: Clasificar y Registrar Evento por Severidad

![Caso Expandido NOT-03](../Imagenes/CDU-NOT-03_%20Clasificar%20y%20Registrar%20Evento%20por%20Severidad.drawio.png)


| Campo | Descripción |
|---|---|
| **ID Caso de Uso** | CDU-NOT-03 |
| **Nombre** | Clasificar y registrar evento por severidad |
| **Módulo al que pertenece** | Notification & Audit Service |
| **Actor Principal** | Sistema |
| **RF cubiertos** | RF-51 y RF-52 |
| **Precondiciones** | - Debe haberse recibido un evento de auditoría válido (procesado por CDU-NOT-02).<br>- El evento debe incluir un campo `severity` o ser clasificable según las reglas de negocio definidas.<br>- La tabla de bitácora de auditoría debe estar disponible para escritura. |
| **Postcondiciones** | - El evento queda registrado con su nivel de severidad asignado: INFO, WARNING o ERROR.<br>- Los eventos con severidad ERROR generan una alerta adicional hacia el canal de monitoreo configurado.<br>- El historial de eventos queda consultable por nivel de severidad. |
| **Escenario Principal** | 1. Notification & Audit Service recibe el evento de auditoría procesado por CDU-NOT-02.<br>2. El servicio evalúa el campo `severity` incluido en el evento.<br>3. Si el campo `severity` no está presente, el servicio aplica las reglas de clasificación automática según el tipo de evento.<br>4. El servicio asigna el nivel de severidad correspondiente: INFO, WARNING o ERROR; las anomalías de uso de cuenta de CDU-ACC-06 se clasifican como WARNING según la regla de Fase 3. El nombre técnico del evento debe coincidir con el contrato implementado.<br>5. El evento es persistido en la bitácora con el nivel de severidad asignado.<br>6. Si el nivel es ERROR, el sistema publica una alerta hacia el canal de monitoreo (cola o webhook configurado).<br>7. El servicio confirma el registro exitoso del evento clasificado. |
| **Escenario Alternativo** | **1. Severidad no reconocida:** Si el valor de `severity` no corresponde a INFO, WARNING o ERROR, el servicio asigna INFO por defecto y registra una advertencia de clasificación.<br><br>**2. Error al publicar alerta:** Si falla la publicación hacia el canal de monitoreo, el evento permanece registrado en la bitácora; el fallo en la alerta no revierte el registro.<br><br>**3. Error de persistencia:** El servicio registra el fallo y el evento es reencolado para reintento posterior.<br><br>**4. Eventos en cascada ERROR:** Si múltiples eventos ERROR llegan en un intervalo corto, el sistema los agrupa para evitar saturar el canal de monitoreo. |
| **Requerimientos** | - Notification & Audit Service debe soportar tres niveles de severidad: INFO, WARNING y ERROR.<br>- Debe aplicar reglas automáticas de clasificación cuando el campo `severity` no está presente en el evento.<br>- Los eventos con severidad ERROR deben generar notificación adicional hacia el canal de monitoreo.<br>- El registro de severidad debe ser parte del registro inmutable de auditoría.<br>- La operación de clasificación no debe bloquear el registro del evento si la alerta falla. |

![Flujo NOT-03](../Imagenes/CDU-NOT-03_%20Flujo%20de%20Clasificación%20y%20Registro%20por%20Severidad.drawio.png)


## Casos de Uso de Fase 3

Los casos de observabilidad y monitoreo se documentan como comportamientos de soporte, siguiendo la convención del documento base que utiliza Sistema como actor. Si el límite del diagrama representa a Bank USAC completo, sus microservicios internos no son actores externos del banco. Las relaciones include se reservan para comportamiento obligatorio; los fallos condicionales se expresan como escenarios alternativos o extend, no como pasos obligatorios.

**Diagramas:** se mantienen los nombres de páginas del XML compartido para facilitar la exportación. Las notas y reglas de esta versión deben reflejarse en esos diagramas antes de entregar; el XML anterior no queda actualizado automáticamente al generar este Markdown.

### Caso de Uso: Registrar trazabilidad del ciclo de vida del cliente

![Caso Expandido CDU-CUS-06](../imgF3/CDU-CUS-06%20-%20Caso%20de%20Uso.drawio.png)

| Campo | Descripción |
|---|---|
| **ID Caso de Uso** | CDU-CUS-06 |
| **Nombre** | Registrar trazabilidad del ciclo de vida del cliente |
| **Módulo al que pertenece** | Customer Service |
| **Actor Principal** | Sistema |
| **RF cubiertos** |   - . |
| **Precondiciones** | - Debe recibirse una solicitud de creación de cliente o producirse un cambio KYC.<br>- Debe estar habilitada la instrumentación del servicio y su exportación al Collector. |
| **Postcondiciones** | - La creación, verificación o rechazo queda correlacionado con la operación de origen.<br>- Se emite telemetría sin datos sensibles y se actualizan las métricas aplicables.<br>- La operación de negocio conserva su resultado independientemente de un fallo de exportación de telemetría. |
| **Escenario Principal** | 1. Customer Service recibe la operación de creación o cambio KYC.<br>2. Recupera el identificador de correlación y el contexto de traza disponible.<br>3. Inicia el span correspondiente a la operación.<br>4. Ejecuta la operación de negocio y registra su resultado.<br>5. Omite o enmascara documento, correo, fotografía y tokens antes de emitir telemetría.<br>6. Emite el log estructurado con identificación de servicio, ambiente, versión, correlación y contexto de traza disponible.<br>7. Actualiza el conteo de clientes verificados o rechazados según el cambio realizado.<br>8. En una verificación, observa el tiempo transcurrido desde la creación para calcular percentiles.<br>9. Cierra el span y exporta la telemetría al Collector. |
| **Escenario Alternativo** | **1. Creación rechazada:** Se registra el fallo sin suponer la existencia de un cliente persistido.<br><br>**2. Contexto ausente o inválido:** Se aplica la política de correlación del servicio y se identifica la pérdida de continuidad de traza.<br><br>**3. Datos sensibles:** Se excluyen antes de registrar logs o atributos de spans.<br><br>**4. Exportación fallida:** Se aplican los reintentos y límites del exportador; no se garantiza conservación indefinida de telemetría.<br><br>**5. Rechazo KYC:** Se registra el rechazo sin contabilizarlo como verificación exitosa. |
| **Requerimientos** | - Cubrir creación y cambios KYC sin alterar sus reglas de negocio.<br>- Correlacionar logs y trazas con `correlationId`, `traceId` y `spanId` cuando estén disponibles.<br>- Propagar contexto W3C `traceparent` en las interacciones distribuidas.<br>- Medir resultados KYC y distribución de tiempos de verificación.<br>- No usar datos personales como etiquetas de métricas. |
| **Notas de validación** | La observabilidad es un comportamiento transversal de CDU-CUS-01 y CDU-CUS-KYC. Los nombres técnicos de eventos y métricas deben verificarse contra el repositorio; no se introducen nombres de contrato nuevos. |

#### Diagrama de flujo — CDU-CUS-06

![Flujo CDU-CUS-06](../imgF3/CDU-CUS-06%20-%20Flujo.drawio.png)

### Caso de Uso: Detectar anomalías de uso de cuenta

![Caso Expandido CDU-ACC-06](../imgF3/CDU-ACC-06%20-%20Caso%20de%20Uso.drawio.png)

| Campo | Descripción |
|---|---|
| **ID Caso de Uso** | CDU-ACC-06 |
| **Nombre** | Detectar anomalías de uso de cuenta |
| **Módulo al que pertenece** | Account Service |
| **Actor Principal** | Sistema |
| **RF cubiertos** |   - . |
| **Precondiciones** | - Debe existir un movimiento válido de cuenta.<br>- Deben estar disponibles los registros necesarios para evaluar su frecuencia de uso.<br>- Los umbrales y la ventana temporal deben estar definidos en la configuración del ambiente. |
| **Postcondiciones** | - El movimiento confirmado conserva su resultado.<br>- Se evalúa la frecuencia de uso y, cuando corresponde, se registra o publica una anomalía.<br>- La anomalía se correlaciona con la operación y se clasifica como WARNING para auditoría.<br>- La detección por sí sola no implica bloqueo automático en esta especificación. |
| **Escenario Principal** | 1. Account Service procesa el débito o crédito conforme a CDU-ACC-03.<br>2. Registra la operación confirmada y su fecha.<br>3. Obtiene el umbral de operaciones y la ventana temporal configurados.<br>4. Cuenta las operaciones elegibles de la cuenta dentro de la ventana.<br>5. Compara el resultado con el criterio de detección configurado.<br>6. Cuando se cumple el criterio, registra o publica la anomalía con su correlación.<br>7. Notification & Audit Service clasifica el evento como WARNING conforme a CDU-NOT-03.<br>8. Actualiza la métrica de anomalías y registra el detalle permitido en logs. |
| **Escenario Alternativo** | **1. Umbral no superado:** Finaliza la evaluación sin emitir una anomalía.<br><br>**2. Configuración ausente o inválida:** Se aplica la política real del servicio y se registra el problema; no se presume un valor por defecto.<br><br>**3. Operación duplicada:** No se vuelve a aplicar el movimiento; la deduplicación de alertas debe respetar la política implementada.<br><br>**4. Fallo al publicar:** Se usa el mecanismo de entrega confiable disponible, sin revertir un movimiento ya confirmado solo por el fallo de telemetría.<br><br>**5. Error de evaluación:** Se registra el error para atención operativa sin inventar una anomalía. |
| **Requerimientos** | - Evaluar uso por cuenta mediante umbral y ventana configurables.<br>- Correlacionar la anomalía con la operación que la originó.<br>- Delegar clasificación y notificación en los mecanismos de Notification & Audit Service.<br>- Mantener los identificadores individuales fuera de etiquetas de métricas para evitar alta cardinalidad.<br>- Separar la detección de la decisión de bloquear una cuenta. |
| **Notas de validación** | Confirmar umbrales, ventana, comparador, nombre del evento y política de deduplicación. Los valores de 10 operaciones en 5 minutos, Outbox y recarga de ConfigMap propuestos anteriormente no se consideran hechos verificados para este flujo. |

#### Diagrama de flujo — CDU-ACC-06

![Flujo CDU-ACC-06](../imgF3/CDU-ACC-06%20-%20Flujo.drawio.png)

### Caso de Uso: Medir rendimiento del flujo de transferencias

![Caso Expandido CDU-TRX-06](../imgF3/CDU-TRX-06%20-%20Caso%20de%20Uso.drawio.png)

| Campo | Descripción |
|---|---|
| **ID Caso de Uso** | CDU-TRX-06 |
| **Nombre** | Medir rendimiento del flujo de transferencias |
| **Módulo al que pertenece** | Transaction Service |
| **Actor Principal** | Sistema |
| **RF cubiertos** |   - . |
| **Precondiciones** | - Debe iniciarse una transferencia o recibirse una transición de su Saga.<br>- La instrumentación de trazas y métricas debe estar habilitada. |
| **Postcondiciones** | - Las etapas y el resultado quedan correlacionados con el contexto de traza disponible.<br>- Se observa la duración y se actualizan los contadores de resultado sin duplicar operaciones.<br>- Las compensaciones se identifican dentro del contexto distribuido. |
| **Escenario Principal** | 1. Transaction Service recibe la transferencia e inicia o continúa el contexto de traza.<br>2. Propaga `traceparent` y correlación a las dependencias.<br>3. Registra spans o eventos en cada transición de la Saga.<br>4. Observa las duraciones por etapa y el tiempo hasta el resultado terminal.<br>5. Actualiza los histogramas y contadores correspondientes al resultado.<br>6. Si hubo compensación, registra su solicitud y resultado en la traza.<br>7. Exporta trazas y métricas por el mecanismo configurado.<br>8. Permite consultar latencia y tasa de éxito o fallo desde los dashboards. |
| **Escenario Alternativo** | **1. Saga estancada:** Se registra la última transición y la duración observada; la alerta solo se considera operativa si existe una regla configurada y validada.<br><br>**2. Compensación fallida:** Se identifica `COMPENSACION_FALLIDA` y su contexto operativo.<br><br>**3. Contexto no propagado:** Se registra la discontinuidad de trazabilidad.<br><br>**4. Exportador indisponible:** Se aplican sus límites de buffer y reintento sin bloquear la operación financiera.<br><br>**5. Mensaje duplicado:** No se duplica el efecto financiero ni se contabiliza como nueva operación de negocio. |
| **Requerimientos** | - Exponer distribuciones de duración para consultar p50, p95 y p99.<br>- Medir éxito y fallo con criterios consistentes de resultado terminal.<br>- Incluir compensaciones y etapas intermedias en la trazabilidad.<br>- No incluir identificadores de transferencia como etiquetas de métricas.<br>- Mantener separados los estados de Saga interna y los del contrato interbancario. |
| **Notas de validación** | Este CDU amplía CDU-TRX-01 y CDU-TRX-02. Los nombres y etiquetas de métricas, así como el alcance exacto de los spans, deben coincidir con la instrumentación real. |

#### Diagrama de flujo — CDU-TRX-06

![Flujo CDU-TRX-06](../imgF3/CDU-TRX-06%20-%20Flujo.drawio.png)

### Caso de Uso: Simular fallos avanzados configurables

![Caso Expandido CDU-PAY-04](../imgF3/CDU-PAY-04%20-%20Caso%20de%20Uso.drawio.png)

| Campo | Descripción |
|---|---|
| **ID Caso de Uso** | CDU-PAY-04 |
| **Nombre** | Simular fallos avanzados configurables |
| **Módulo al que pertenece** | Payment Service |
| **Actor Principal** | Administrador / Sistema |
| **RF cubiertos** |   - . |
| **Precondiciones** | - Payment Service debe estar operativo en un ambiente autorizado para pruebas.<br>- Debe existir un mecanismo de configuración de simulaciones.<br>- El operador debe poseer permisos acordes con ese mecanismo. |
| **Postcondiciones** | - La configuración válida queda aplicada o se conserva la anterior si la solicitud es rechazada.<br>- Las operaciones de prueba reflejan el fallo, latencia o timeout configurado.<br>- Cuando hubo débito y el pago no se completa, se verifica la compensación correspondiente.<br>- Queda evidencia correlacionada para el escenario ejecutado. |
| **Escenario Principal** | 1. El operador selecciona el ambiente y registra la configuración inicial.<br>2. Define los parámetros disponibles de fallo, latencia y timeout.<br>3. El mecanismo de administración valida autorización y valores permitidos.<br>4. Payment Service aplica la configuración mediante el mecanismo implementado.<br>5. Se ejecuta un pago de prueba conforme a CDU-PAY-01.<br>6. El simulador aplica el comportamiento configurado al proveedor externo simulado.<br>7. Payment Service registra el resultado y solicita compensación a Account Service cuando corresponde.<br>8. El operador verifica estado final, saldo, logs, métricas y traza.<br>9. Se restaura la configuración inicial y se documenta el resultado. |
| **Escenario Alternativo** | **1. Sin autorización:** No se modifica la configuración.<br><br>**2. Parámetros inválidos:** Se rechaza el cambio y se conserva la configuración válida.<br><br>**3. Recarga no soportada:** Se identifica expresamente la necesidad de reinicio; no se presenta como cambio en caliente.<br><br>**4. Fallo o timeout simulado:** Se verifica la compensación si existió débito; no se presupone su éxito.<br><br>**5. Compensación fallida:** Se conserva evidencia del estado pendiente de atención.<br><br>**6. Pago exitoso:** No se ejecuta una compensación innecesaria. |
| **Requerimientos** | - Permitir los escenarios de fallo, respuesta lenta y timeout soportados.<br>- Controlar permisos y alcance de las pruebas.<br>- Medir resultados y latencia del proveedor simulado.<br>- Verificar consistencia financiera tras cada escenario.<br>- Si el requisito exige cambios sin redespliegue, demostrar ese comportamiento y no darlo por cumplido solo por editar un ConfigMap. |
| **Notas de validación** | Validar el mecanismo real de configuración antes de indicar endpoint ADMIN o recarga dinámica. La simulación de Payment Service no debe confundirse con el laboratorio interbancario aislado: el Transaction Service normal no alterna su destino entre BANK14 y el mock. |

#### Diagrama de flujo — CDU-PAY-04

![Flujo CDU-PAY-04](../imgF3/CDU-PAY-04%20-%20Flujo.drawio.png)

### Caso de Uso: Enviar depósito interbancario

![Caso Expandido CDU-TRX-07](../imgF3/CDU-TRX-07%20-%20Caso%20de%20Uso.drawio.png)

| Campo | Descripción |
|---|---|
| **ID Caso de Uso** | CDU-TRX-07 |
| **Nombre** | Enviar depósito interbancario |
| **Módulo al que pertenece** | Transaction Service, con Account Service y adaptador interbancario |
| **Actor Principal** | Cliente / Cajero Receptor |
| **RF cubiertos** | Fase 3: . |
| **Precondiciones** | - El actor debe estar autenticado y autorizado para operar la cuenta origen.<br>- Deben cumplirse las validaciones KYC, estado de cuenta, monto y fondos aplicables.<br>- El contrato y las credenciales con BANK14 deben estar configurados.<br>- La solicitud debe disponer de identificación e idempotencia conforme al contrato acordado. |
| **Postcondiciones** | - La operación y su historial quedan registrados.<br>- Los movimientos financieros se realizan exclusivamente mediante Account Service.<br>- La aceptación inicial de transporte no se confunde con la confirmación financiera.<br>- Si existe rechazo o incertidumbre, se conserva el estado y la información de conciliación o compensación aplicable. |
| **Escenario Principal** | 1. El actor solicita un depósito hacia BANK14.<br>2. El sistema valida identidad, autorización y datos de la solicitud.<br>3. Transaction Service verifica idempotencia y registra la operación.<br>4. Solicita las validaciones y la reserva de fondos a Account Service mediante RPC por RabbitMQ, según el flujo implementado.<br>5. Envía la solicitud a BANK14 por HTTPS a través del adaptador real de integración, propagando la correlación acordada.<br>6. Registra la respuesta inicial del banco pareja.<br>7. Obtiene el resultado de negocio mediante el mecanismo definido por el contrato, sin presumir callback cuando no esté confirmado.<br>8. Al confirmar aceptación, coordina con Account Service la confirmación financiera correspondiente.<br>9. Registra el resultado y lo deja disponible para consulta, auditoría y notificación. |
| **Escenario Alternativo** | **1. Validaciones o fondos insuficientes:** Rechaza la operación antes de enviarla cuando aún no hay efecto financiero.<br><br>**2. Solicitud repetida:** Recupera el resultado existente sin reservar o debitar dos veces.<br><br>**3. BANK14 rechaza:** Registra el motivo y solicita liberar o restituir fondos según la etapa alcanzada.<br><br>**4. Timeout:** Registra la incertidumbre y aplica la política real de consulta, conciliación y compensación; un timeout no prueba que el banco destino no haya acreditado.<br><br>**5. Contrato o credenciales incompatibles:** Registra el fallo de integración sin presentarlo como depósito exitoso.<br><br>**6. Respuesta tardía o duplicada:** Evita aplicar dos veces efectos financieros y reconcilia el resultado con el historial existente. |
| **Requerimientos** | - Conservar idempotencia y trazabilidad de la operación interbancaria.<br>- Usar el contrato acordado sin inventar campos, rutas o responsabilidades de servicios.<br>- Delegar reserva, débito y liberación de fondos a Account Service.<br>- Separar recepción HTTP y aceptación financiera.<br>- No mezclar los estados del contrato interbancario con la Saga interna.<br>- Mantener separado el laboratorio del flujo real con BANK14. |
| **Notas de validación** | Confirmar en el contrato el uso exacto de `interbankTransferId`, `idempotencyKey`, estados y mecanismo de resultado. El flujo exitoso de extremo a extremo debe respaldarse con evidencia; la existencia de este CDU no demuestra una integración satisfactoria. |

#### Diagrama de flujo — CDU-TRX-07

![Flujo CDU-TRX-07](../imgF3/CDU-TRX-07%20-%20Flujo.drawio.png)

### Caso de Uso: Recibir depósito interbancario

![Caso Expandido CDU-TRX-08](../imgF3/CDU-TRX-08%20-%20Caso%20de%20Uso.drawio.png)

| Campo | Descripción |
|---|---|
| **ID Caso de Uso** | CDU-TRX-08 |
| **Nombre** | Recibir depósito interbancario |
| **Módulo al que pertenece** | Transaction Service, con Account Service y entrada interbancaria |
| **Actor Principal** | Banco pareja (BANK14) |
| **RF cubiertos** |   - . |
| **Precondiciones** | - Debe estar disponible la entrada interbancaria mediante el proxy HTTPS de Cloud Run y el recorrido interno configurado en GKE.<br>- Deben existir el contrato de recepción y el mecanismo de autenticación acordado.<br>- Account Service debe poder validar y acreditar la cuenta destino. |
| **Postcondiciones** | - La recepción y su resultado quedan registrados sin doble acreditación.<br>- La aceptación financiera solo se confirma cuando Account Service confirma la acreditación.<br>- Las solicitudes rechazadas conservan el motivo permitido por el contrato.<br>- Se conserva la correlación del banco origen cuando resulta válida. |
| **Escenario Principal** | 1. BANK14 envía la solicitud a la entrada interbancaria publicada.<br>2. El proxy HTTPS de Cloud Run la encamina al recorrido interno configurado; no es procesamiento en la interfaz del frontend.<br>3. La capa responsable valida autenticación y contrato.<br>4. Transaction Service, o el componente receptor confirmado por el código, verifica la idempotencia y registra la operación.<br>5. Solicita a Account Service validar cuenta destino y monto.<br>6. Solicita la acreditación mediante RabbitMQ con identificación de la operación.<br>7. Account Service aplica la acreditación de forma atómica e idempotente y devuelve el resultado.<br>8. El receptor registra el resultado final.<br>9. Comunica el resultado a BANK14 mediante el mecanismo definido en el contrato.<br>10. Emite la información necesaria para auditoría y notificación. |
| **Escenario Alternativo** | **1. Autenticación inválida:** Rechaza la solicitud sin acreditar fondos.<br><br>**2. Contrato inválido:** Comunica el error definido sin crear una operación financiera parcial.<br><br>**3. Solicitud duplicada:** Devuelve o recupera el resultado original sin nueva acreditación.<br><br>**4. Cuenta o monto inválidos:** Registra rechazo con el motivo contractual aplicable.<br><br>**5. Acreditación fallida o incierta:** No confirma aceptación financiera hasta verificar el resultado; evita una segunda acreditación al reintentar.<br><br>**6. Comunicación de resultado fallida:** Conserva el resultado para recuperación o reintento según el contrato, sin repetir el crédito. |
| **Requerimientos** | - Autenticar al banco emisor y proteger credenciales.<br>- Garantizar idempotencia también ante solicitudes concurrentes y reintentos.<br>- Acreditar exclusivamente a través de Account Service.<br>- Distinguir respuesta inicial y resultado de negocio si el contrato los separa.<br>- Preservar historial y contexto de correlación.<br>- No asumir cifrado HTTPS extremo a extremo en el recorrido interno cuando existe HTTP en GKE. |
| **Notas de validación** | Validar el componente receptor exacto, autenticación, campos y mecanismo de respuesta con código y contrato. No se da por probado el flujo entrante de BANK14 sin evidencia; los ensayos contra el laboratorio se identifican por separado. |

#### Diagrama de flujo — CDU-TRX-08

![Flujo CDU-TRX-08](../imgF3/CDU-TRX-08%20-%20Flujo.drawio.png)

### Caso de Uso: Consultar estado de depósito interbancario

![Caso Expandido CDU-TRX-09](../imgF3/CDU-TRX-09%20-%20Caso%20de%20Uso.drawio.png)

| Campo | Descripción |
|---|---|
| **ID Caso de Uso** | CDU-TRX-09 |
| **Nombre** | Consultar estado de depósito interbancario |
| **Módulo al que pertenece** | Transaction Service |
| **Actor Principal** | Cliente / Cajero Receptor |
| **RF cubiertos** |   - . |
| **Precondiciones** | - El actor debe estar autenticado y autorizado para consultar la operación.<br>- Debe proporcionarse el identificador admitido por el contrato de consulta. |
| **Postcondiciones** | - Se informa el estado registrado y el resultado conocido de la operación.<br>- La consulta no modifica saldos ni dispara una nueva transferencia.<br>- Se muestra la correlación de traza únicamente si está disponible y autorizada en la interfaz. |
| **Escenario Principal** | 1. El actor solicita consultar un depósito interbancario.<br>2. El sistema valida sesión, identificador y autorización.<br>3. Transaction Service recupera la operación de su persistencia.<br>4. Obtiene estado, resultado, fechas e historial disponibles.<br>5. Devuelve los datos permitidos para el actor.<br>6. La interfaz presenta el resultado distinguiendo operación en curso, confirmada, rechazada o pendiente de conciliación según los estados reales.<br>7. Permite actualizar la consulta sin reprocesar el depósito. |
| **Escenario Alternativo** | **1. Identificador inválido:** Solicita corregir los datos.<br><br>**2. Operación inexistente:** Devuelve la respuesta de recurso no encontrado definida por la API.<br><br>**3. Sin autorización:** No expone los datos de la operación.<br><br>**4. Operación en curso o incierta:** Presenta el último estado conocido sin declararlo éxito financiero.<br><br>**5. Error de consulta:** Informa el fallo y permite reintentar sin efectos financieros.<br><br>**6. Traza no disponible:** Muestra los datos de negocio disponibles sin inventar un `traceId`. |
| **Requerimientos** | - Restringir acceso según identidad y permisos.<br>- Consultar exclusivamente la persistencia responsable de la operación.<br>- Exponer los estados reales y el historial disponible.<br>- Mantener la consulta como solo lectura.<br>- Correlacionar con observabilidad sin exponer secretos ni datos ajenos. |
| **Notas de validación** | Validar el identificador, la presentación de `traceId` y la actualización automática contra la API y el frontend reales. Los estados interbancarios de la propuesta anterior no sustituyen el contrato confirmado. |

#### Diagrama de flujo — CDU-TRX-09

![Flujo CDU-TRX-09](../imgF3/CDU-TRX-09%20-%20Flujo.drawio.png)
