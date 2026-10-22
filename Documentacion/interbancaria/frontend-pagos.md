# Interbancaria desde Pagos

El portal de clientes ofrece «Realizar pago interbancario» y «Escenarios» junto a «Realizar pago». Se retiró el acceso de Administración a las pruebas aisladas. Las rutas de API administrativas existentes no se cambiaron ni se abrieron a clientes.

## Banco pareja

`/pagos/interbancario` consulta las cuentas del cliente y su perfil actual. Antes de enviar vuelve a comprobar cuenta activa, titularidad y KYC VERIFIED. Esta comprobación del navegador es de usabilidad: no sustituye la autorización y validación KYC obligatoria dentro de la Saga del servidor.

Envía `POST /api/interbancarias/` con `Idempotency-Key` igual al UUID `interbankTransferId`, cuenta de origen numérica, cuenta destino del banco pareja, monto con dos decimales y moneda GTQ. Los códigos de banco y fechas los asigna el servidor; no se modificó el contrato OpenAPI del banco pareja.

El último ID se conserva en sessionStorage por cliente. Un envío incierto no se reintenta automáticamente. Se consulta `GET /api/interbancarias/{id}` cada cinco segundos mientras la página está visible; también se puede introducir manualmente un ID anterior. Los estados terminales permiten iniciar otro depósito. Se muestra el saldo total, sin presentarlo como saldo disponible.

## Escenarios con saldo real: pendiente de backend

`/pagos/escenarios` muestra aceptación, rechazo, 503, ausencia de callback y demora. La ejecución permanece deshabilitada: el backend actual solo ofrece un mock ADMIN sobre cuenta ficticia en el laboratorio aislado. Ningún escenario se redirige al banco pareja, usa la cuenta de laboratorio como si fuera la del cliente ni simula un éxito en el navegador.

Para ejecutar escenarios con saldo real se necesita un endpoint CLIENTE con KYC dentro de la Saga, selección de mock por operación persistida y aislada de otros clientes, reservas/reintentos/callbacks/compensaciones y autorización de consulta por titular. Este cambio es exclusivamente de frontend y no afirma haber implementado ese backend.

## Verificación

TypeScript y compilación Vite; pruebas del portal más regresiones de bloqueo por cambio de KYC, doble envío concurrente, conservación del ID ante POST incierto y bloqueo de envíos desde Escenarios. La vista previa usa cuentas ficticias y no realiza operaciones bancarias.
