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
| **Postcondiciones** | - La solicitud de transferencia queda registrada.<br>- Se genera un `transactionId` para identificar la operación.<br>- Se conserva un `correlationId` para mantener la trazabilidad del flujo.<br>- La transferencia queda con el estado correspondiente según el resultado del procesamiento. |
| **Escenario Principal** | 1. El Cliente o Cajero Receptor selecciona **Realizar transferencia**.<br>2. El sistema solicita la cuenta origen, la cuenta destino y el monto.<br>3. El actor ingresa la información solicitada.<br>4. El sistema valida el formato de los datos.<br>5. Transaction Service genera un `transactionId` y un `correlationId`.<br>6. Transaction Service registra la solicitud de transferencia.<br>7. Se incluye el `CDU-TRX-02 - Ejecutar Saga de transferencia` para realizar el procesamiento distribuido.<br>8. Transaction Service recibe el resultado correspondiente al flujo.<br>9. El estado de la transferencia es actualizado.<br>10. El sistema informa el resultado de la operación al actor. |
| **Escenario Alternativo** | **1. Monto inválido:** El sistema solicita corregir el monto antes de iniciar la operación.<br><br>**2. Datos incompletos:** El sistema indica qué información debe completarse.<br><br>**3. Transferencia rechazada:** Si el CDU-TRX-02 determina que la operación no puede realizarse, Transaction Service registra el estado correspondiente.<br><br>**4. Fallo durante el procesamiento:** El manejo distribuido del fallo y sus compensaciones corresponde al `CDU-TRX-02 - Ejecutar Saga de transferencia`.<br><br>**5. Operación duplicada:** El sistema debe evitar procesar nuevamente una transferencia ya registrada. |
| **Requerimientos** | - Transaction Service debe permitir iniciar transferencias entre cuentas.<br>- Cada operación debe contar con un `transactionId` único.<br>- Debe mantenerse un `correlationId` durante el flujo.<br>- La transferencia debe registrar cuenta origen, cuenta destino, monto y estado.<br>- La ejecución distribuida de la transferencia debe delegarse al `CDU-TRX-02`.<br>- La comunicación entre microservicios debe realizarse mediante eventos asíncronos.<br>- Transaction Service no debe consultar directamente las bases de datos de otros servicios.<br>- El procesamiento debe contemplar idempotencia. |

![Caso Expandido TRX-01](../Imagenes/Diagrama%20-%20TRX01.png)

---
## Casos de Uso de Payment Service



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
