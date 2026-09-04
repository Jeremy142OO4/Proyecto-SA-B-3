# Funcionalidades complementarias y restricciones de alcance

Estas funciones se incorporaron para conservar la identidad, autorización, integridad de saldos y trazabilidad del flujo principal.

| Área | Función | RF relacionados | Justificación |
|---|---|---|---|
| Identidad | Unicidad de documento, correo y `username` | RF-31 | Evita clientes y credenciales ambiguas. |
| Activación | Enlace de un solo uso, expiración y reenvío | RF-32 | Evita activaciones reutilizadas. |
| Usuarios | Estados pendiente, activo y bloqueado | RF-33 | Controla activación y acceso. |
| Autorización | Permisos por rol y propiedad | RF-25, RF-26 | Impide operar cuentas ajenas. |
| Cuentas | Validación del cliente y estados de cuenta | RF-34, RF-35 | Evita cuentas sin propietario y permite inactividad. |
| Saldos | Protección contra débitos inválidos o balances negativos | RF-36 | Preserva la integridad financiera. |
| Transferencias | Validaciones y estados del flujo | RF-37, RF-38 | Define cuándo una transferencia es válida y trazable. |
| Pagos | Datos mínimos, estados y consulta | RF-39 | Permite rastrear pagos asíncronos. |
| Historial | Consulta de transacciones y pagos propios | RF-40 | Permite verificar operaciones. |

Los montos se manejan en GTQ mediante un tipo decimal exacto.

## Fuera del alcance inicial

La recuperación de contraseña, búsquedas avanzadas de auditoría, administración manual de notificaciones, políticas configurables de retención y soporte multimoneda quedan como mejoras posteriores. No son necesarias para demostrar el flujo exigido.

