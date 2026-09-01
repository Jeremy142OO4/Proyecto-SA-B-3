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

Las escrituras responden `202 Accepted`. Las consultas publican un comando en `banco.comandos` y esperan el evento correlacionado de `banco.eventos` hasta el timeout configurado. El Gateway valida JWT, propiedad del recurso, formato y `correlationId`; no contiene lógica financiera ni consulta bases de datos.

Customer Service todavía no está presente en el repositorio. Login, activación y administración de clientes deben conectarse cuando se incorporen sus contratos reales; no se inventaron endpoints incompatibles.
