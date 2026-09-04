# API Gateway - Bank USAC

Único punto de entrada HTTP. El código, rutas y contratos de integración están en español y coinciden con Account, Payment y Transaction Service.

## Rutas autenticadas

- `GET|POST /api/cuentas`
- `GET /api/cuentas/:idCuenta`
- `GET /api/cuentas/:idCuenta/movimientos`
- `GET|POST /api/pagos`
- `GET /api/pagos/:idPago`
- `GET|POST /api/transferencias`
- `GET /api/transferencias/:idTransferencia`
- `GET /api/operaciones/:id`
- `POST /api/clientes/registro`
- `GET /api/clientes/activacion?token=...`
- `POST /api/clientes/login`
- `GET|PUT /api/clientes/perfil`

Las escrituras responden `202 Accepted`. Las consultas publican un comando en `banco.comandos` y esperan el evento correlacionado de `banco.eventos` hasta el timeout configurado. El Gateway valida JWT, propiedad del recurso, formato y `correlationId`; no contiene lógica financiera ni consulta bases de datos.

Las rutas de identidad traducen los contratos HTTP en español al contrato actual de Customer Service. Esta integración HTTP interna evita publicar contraseñas o tokens de activación en RabbitMQ; los flujos bancarios continúan siendo completamente asíncronos mediante el broker.

## Permisos por rol

- `TELLER`: registrar clientes, solicitar cuentas para un `idCliente` y consultar sus propias operaciones aceptadas.
- `CLIENTE`: perfil, cuentas propias, movimientos, pagos, transferencias y sus operaciones.
- `ADMIN`: listar clientes, cambiar su estado y consultar registros, trazas y notificaciones de auditoría.

La activación y el inicio de sesión son las únicas rutas públicas. Una ruta autenticada sin el rol exigido responde `403 Forbidden`.
