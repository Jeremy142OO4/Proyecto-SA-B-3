# Diagramas de casos de uso

## Casos de Uso de Alto Nivel y Expandidos

### Diagram de CDU de Alto nivel

![Diagrama de alto nivel](../Imagenes/Diagrama%20-%20AltoNivelCDU.png)

### Diagrama de CDU Exapandido General

![Daigamra expandido](../Imagenes/Diagrama%20-Descomposicion.png)

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

## Casos de Uso de Account Service
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
| **Escenario Principal** | 1. Account Service recibe un comando financiero mediante RabbitMQ.<br>2. El servicio valida que el mensaje no haya sido procesado.<br>3. Busca la cuenta indicada.<br>4. Valida el estado de la cuenta y el monto.<br>5. Si es débito, verifica fondos suficientes.<br>6. Calcula el nuevo saldo.<br>7. Actualiza el balance de forma atómica.<br>8. Registra el movimiento de cuenta.<br>9. Actualiza la fecha de última actividad.<br>10. Publica el evento de operación completada. |
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

## Casos de Uso de Transaction Service


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


#### Diagrama de flujo — CDU-ACC-04

![Caso flujo ACC-05](../Imagenes/CDU-ACC-05%20-%20Flujo.drawio.png)

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
| **Postcondiciones** | - La solicitud de transferencia queda registrada.<br>- Se genera un `transactionId` para identificar la operación.<br>- Se conserva un `correlationId` para mantener la trazabilidad del flujo.<br>- La transferencia queda con el estado correspondiente según el resultado del procesamiento. |
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
| **Precondiciones** | - Debe recibirse el evento o comando `transfer.requested` en RabbitMQ con `correlationId`, cuentas origen/destino y monto en centavos GTQ.<br>- Las cuentas origen y destino deben existir en el sistema. |
| **Postcondiciones** | - Si el flujo es exitoso: el débito y crédito son confirmados, la transacción pasa a estado `COMPLETED` y se publica `transfer.completed`.<br>- Si el débito falla: la transacción pasa a `REJECTED` y se publica `transfer.rejected`.<br>- Si el crédito falla tras un débito exitoso: se orquesta la compensación, se reembolsa el monto a la cuenta origen, el estado final pasa a `COMPENSATED` y se emite `transfer.compensated`. |
| **Escenario Principal** | 1. Transaction Service consume el comando `transfer.requested` desde su cola en RabbitMQ.<br>2. El servicio registra la transferencia en su base de datos con estado `PENDING`.<br>3. Transaction Service publica el comando `account.debit.requested` con el mismo `correlationId`.<br>4. Account Service ejecuta el débito y publica `account.debited`.<br>5. Transaction Service actualiza el estado a `PROCESSING` y publica el comando `account.credit.requested`.<br>6. Account Service aplica el crédito en la cuenta destino y emite `account.credited`.<br>7. Transaction Service actualiza la transacción a estado `COMPLETED` y publica el evento final `transfer.completed`.<br>8. Notification & Audit Service consume el evento para auditoría y notificación. |
| **Escenario Alternativo** | **1. Débito rechazado (fondos insuficientes o cuenta bloqueada):** Account Service emite `account.debit.rejected`. Transaction Service marca la transferencia como `REJECTED`, emite `transfer.rejected` y finaliza sin compensar.<br><br>**2. Falla en el crédito tras débito exitoso:** Account Service emite `account.credit.rejected`. Transaction Service transiciona a `COMPENSATING` y publica `account.compensation.requested`. Account Service acredita el monto de vuelta en la cuenta origen y publica `account.compensated`. Transaction Service actualiza a `COMPENSATED`.<br><br>**3. Mensaje duplicado:** El servicio comprueba el `messageId` en su registro de idempotencia y descarta la ejecución redundante. |
| **Requerimientos** | - Implementar Saga basada en coreografía mediante RabbitMQ.<br>- Manejar la máquina de estados completa: `PENDING`, `PROCESSING`, `COMPLETED`, `REJECTED`, `COMPENSATING`, `COMPENSATED`, `COMPENSATION_FAILED`.<br>- Propagar estrictamente el `correlationId` en cada comando y evento.<br>- Asegurar idempotencia y persistencia mediante Transactional Outbox. |

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
| **Postcondiciones** | - El sistema retorna el estado actual de la transferencia (`PENDING`, `PROCESSING`, `COMPLETED`, `REJECTED`, `COMPENSATED`).<br>- No se altera la información financiera ni de auditoría. |
| **Escenario Principal** | 1. El cliente o cajero solicita consultar el estado de una transferencia a través del frontend.<br>2. El API Gateway valida el token JWT y los permisos del usuario.<br>3. El API Gateway consulta a Transaction Service por el estado asociado al `operationId`.<br>4. Transaction Service verifica la existencia y pertenencia de la transferencia en su base de datos (`transaction_db`).<br>5. El servicio retorna los datos: identificador, monto, cuentas involucradas, fecha y estado actual.<br>6. El frontend renderiza el resultado permitiendo al usuario conocer el progreso de la operación asíncrona. |
| **Escenario Alternativo** | **1. Operación no encontrada:** Se responde con error 404 indicando que el identificador no corresponde a ninguna transferencia registrada.<br><br>**2. Usuario no autorizado:** Si el cliente autenticado no es el emisor de la transferencia, el sistema deniega el acceso con 403 Forbidden.<br><br>**3. Operación en curso:** Si el estado es `PENDING` o `PROCESSING`, el frontend mantiene una consulta periódica (polling) hasta alcanzar un estado final. |
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
| **Precondiciones** | - El usuario debe estar autenticado en el sistema.<br>- La cuenta de débito debe existir, estar activa y pertenecer al cliente.<br>- Los datos del servicio y monto deben cumplir las validaciones de formato. |
| **Postcondiciones** | - Si el pago es exitoso: el balance de la cuenta queda debitado, el pago se registra en estado `COMPLETADO` y se publica el evento de confirmación.<br>- Si falla la llamada externa: se compensa el débito en la cuenta y el pago queda en estado `RECHAZADO`.<br>- Se emiten los eventos de auditoría correspondientes. |
| **Escenario Principal** | 1. El usuario selecciona **Nuevo Pago**, ingresa la cuenta origen, tipo de pago (`INTERNO` o `EXTERNO`), proveedor/servicio y monto.<br>2. El sistema valida el formato de entrada y el API Gateway responde con `202 Accepted` conteniendo el `operationId`.<br>3. Payment Service registra el pago en su base de datos con estado `PROCESANDO`.<br>4. Payment Service publica el comando de débito `cuenta.debito.solicitado` por RabbitMQ con `correlationId`.<br>5. Account Service valida fondos, realiza el débito y emite `cuenta.debitada`.<br>6. Si el pago es `EXTERNO`, Payment Service invoca la API / pasarela externa correspondiente.<br>7. Al confirmarse el pago externo (o en caso de pago `INTERNO` inmediato), Payment Service actualiza el estado a `COMPLETADO`.<br>8. El servicio emite el evento de pago completado y el usuario es notificado del éxito. |
| **Escenario Alternativo** | **1. Fondos insuficientes en la cuenta:** Account Service emite `cuenta.debito.rechazada`. Payment Service marca el pago como `RECHAZADO` y finaliza.<br><br>**2. Fallo o Timeout en el proveedor externo:** Si la pasarela externa falla o no responde dentro del tiempo límite, Payment Service transiciona a `COMPENSANDO`, publica `cuenta.compensacion.solicitada` hacia Account Service para restituir los fondos y finalmente marca el pago como `RECHAZADO`.<br><br>**3. Datos de servicio inválidos:** El sistema valida antes de publicar y rechaza la solicitud de inmediato. |
| **Requerimientos** | - Soportar tipos de pagos `INTERNO` y `EXTERNO`.<br>- Gestionar los estados: `PROCESANDO`, `COMPLETADO`, `COMPENSANDO`, `RECHAZADO`.<br>- Manejar fallos externos mediante compensación asíncrona hacia Account Service.<br>- Garantizar persistencia con Outbox e idempotencia operativa. |

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

![Caso Flujo 02](../Imagenes/CDU-TRX-02%20-%20Flujo.drawio.png)
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
| **Postcondiciones** | - El evento queda almacenado para fines de auditoría.<br>- Se conserva su información original relevante.<br>- El registro puede relacionarse con otros eventos mediante `correlationId`.<br>- El mismo mensaje no se registra dos veces. |
| **Escenario Principal** | 1. Un microservicio publica un evento de dominio.<br>2. RabbitMQ entrega el evento a Notification & Audit Service.<br>3. El servicio valida el contrato del mensaje.<br>4. Comprueba que el evento no haya sido procesado.<br>5. Extrae el identificador, tipo, fecha, payload y `correlationId`.<br>6. Guarda el registro de auditoría.<br>7. Marca el mensaje como procesado.<br>8. Confirma el procesamiento al broker. |
| **Escenario Alternativo** | **1. Evento inválido:** El mensaje se rechaza o se dirige al mecanismo de fallos.<br><br>**2. Evento duplicado:** Se reconoce sin crear otro registro.<br><br>**3. Falta `correlationId`:** El evento se registra como inválido o se rechaza según el contrato.<br><br>**4. Error de base de datos:** No se confirma el mensaje para permitir su reintento.<br><br>**5. Máximo de reintentos alcanzado:** El mensaje se envía a la cola de mensajes muertos. |
| **Requerimientos** | - Consumir eventos de forma asíncrona.<br>- Conservar identificador, tipo, fecha, payload y `correlationId`.<br>- Garantizar idempotencia.<br>- Permitir la trazabilidad de operaciones distribuidas.<br>- Aplicar reintentos y cola de mensajes muertos cuando corresponda. |

#### Diagrama de flujo — CDU-NOT-02

![Caso Expandido TRX-01](../Imagenes/Diagrama%20-%20NOT02.png)