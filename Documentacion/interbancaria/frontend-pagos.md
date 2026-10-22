# Pagos interbancarios y escenarios

Pagos muestra depósitos reales y escenarios desde el historial del servidor, filtrado por cliente (últimos 50, actualización cada cinco segundos). El formulario redirige al historial tras el envío y no muestra tarjeta de seguimiento ni botón de actualización manual de KYC.

Escenarios ofrece cuatro casillas de selección única: rechazo, 503, sin callback y respuesta lenta. Usa cuenta y saldo reales con una respuesta externa simulada. Solo está habilitado en entorno dev. No se cambia la URL ni el escenario global del banco pareja.

La Saga persiste el escenario, reserva mediante Account y consulta el KYC actual en Customer antes de reservar; si Customer no responde se reintenta sin reservar. Payment mantiene un mock durable por depósito, emite la respuesta de callback por el broker, permite consultar/cancelar y conserva el resultado idempotente. Una respuesta sin callback puede recuperarse consultando al mock: no implica necesariamente fracaso. Una demora de 181 segundos vence la ventana de 180 segundos y permite cancelar antes de liberar la reserva. Los reintentos mantienen ID y escenario.

Migraciones nuevas: Transaction 000008_interbank_scenario y Payment 000006_mock_interbank, incluidas en prepare-telemetry-schema.sh. El contrato OpenAPI entre bancos no cambia. Los UUID permanecen en trazas/logs, no en etiquetas de métricas.

Validación local: TypeScript, build Vite y pruebas Go/Node. Las pruebas con PostgreSQL se ejecutan en CI con una base desechable; no se realizaron pagos reales desde el agente.

## Restablecer una prueba

El historial muestra «Restablecer proceso normal» para escenarios pendientes de banco no disponible, respuesta lenta y ausencia de callback. POST `/api/interbancarias/{id}/restablecer` comprueba titularidad y que sea una simulación; jamás cambia el banco real ni otros depósitos. Payment guarda la recuperación y adelanta el trabajo pendiente. Se conserva el ID y la fecha de vencimiento original: después del vencimiento se resuelve como rechazo/compensación, sin autorizar un débito tardío. Un estado final no se reabre.

El historial usa GET `/api/interbancarias/historial`. El error 405 observado correspondía a un Gateway desplegado anterior a esta ruta; se comprobó la versión 8e9dd9c frente al frontend más reciente. Deben desplegarse Gateway, Customer, Transaction y Payment antes de verificar el flujo completo. Migración nueva: Payment 000007_mock_recovery.
