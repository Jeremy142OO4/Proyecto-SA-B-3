# Depósitos y recuperación interbancaria — puntos 7 y 8

Implementación en Gateway, Transacciones y Cuentas. Está desactivada por defecto (`INTERBANK_ENABLED=false`). Antes de habilitarla hay que configurar los Secrets, confirmar URLs/cuentas con BANK14 y ejecutar las pruebas conjuntas. Las pruebas locales y del CI no equivalen a una conexión real con el equipo pareja.

## Flujo implementado

- Gateway autentica al banco con `X-Api-Key`, valida `traceparent` y envía el pedido por RabbitMQ. No espera una acreditación para responder `202`: primero persiste la recepción en Transacciones.
- Transacciones procesa los pendientes desde PostgreSQL con bloqueo entre réplicas, guardando el mismo id y contenido para cada reintento.
- Cuentas reserva monto más comisión al enviar; los débitos internos y pagos no pueden consumir esa reserva. El saldo contable no baja hasta confirmar el resultado. La respuesta de cuenta incluye saldo reservado y disponible (este último antes de aplicar el saldo mínimo).
- El receptor acredita una sola vez, valida cuenta/estado/límite y verifica el vencimiento después de bloquear la cuenta. No hay acreditación si ya venció.
- Un callback `ACCEPTED` confirma el débito reservado; `REJECTED` libera la reserva. Los callbacks se guardan antes del ACK `{interbankTransferId, received:true}`.
- Los callbacks salientes se reintentan con espera creciente durante un máximo de 24 h. El resultado y su fecha original sobreviven a reinicios.
- Cuentas confirma saldo, recibo idempotente, movimiento y evento de Outbox dentro de la misma transacción. Un fallo posterior en Transacciones vuelve a consultar ese recibo; no repite el movimiento.
- Transacciones guarda también sus cambios de estado en el Outbox de auditoría y conserva el contexto de traza durante los reintentos.

## Endpoints iniciales

Externos, bajo `/interbank/v1`: `GET /health` (sin autenticación), `POST /transfers`, `GET /transfers/{id}`, `POST /transfers/{id}/result`, `POST /transfers/{id}/cancel`, `POST /reversals`, `GET /reversals/{id}` y `GET /reconciliation?date=YYYY-MM-DD`. Usan el [contrato original](../../contracts/interbank/openapi.yaml), sin cambios. Solo se admiten operaciones con el banco pareja configurado.

Internos para clientes con JWT y rol CLIENTE:

- `POST /api/interbancarias/`: cuerpo basado en TransferRequest con `interbankTransferId`, `originAccount`, `destinationAccount`, `amount`, `currency` y descripción opcional. `Idempotency-Key` debe coincidir con el id. El servidor fija los códigos de banco y las fechas UTC (`createdAt` y `expiresAt` +180 s); no confiar en fechas del navegador.
- `GET /api/interbancarias/{id}`: consulta exclusivamente las operaciones del cliente autenticado.

El formulario y listado del frontend corresponden al punto 9. Nginx ya tiene una ruta para reenviar `/interbank/v1/` desde Cloud Run al Gateway; falta desplegarla/verificarla y compartir la URL real, no asumirla operativa por existir el frontend.

## Configuración

Gateway y Transacciones consumen opcionalmente el Secret separado `bank-usac-interbank`, sin valores reales en Git:

- `INTERBANK_API_KEY`: clave que valida BANK03 en las llamadas entrantes.
- `INTERBANK_PARTNER_API_KEY`: clave que valida BANK14 en nuestras llamadas salientes.

ConfigMap: `INTERBANK_BANK_CODE=BANK03`, `INTERBANK_PARTNER_BANK_CODE=BANK14`, `INTERBANK_PARTNER_BASE_URL` HTTPS terminada exactamente en `/interbank/v1`, y `INTERBANK_ENABLED`.

Habilitar el indicador en Gateway, Cuentas y Transacciones conjuntamente, con sus migraciones aplicadas previamente. La URL pública de BANK03 se acuerda con BANK14 (`INTERBANK_PUBLIC_BASE_URL` sirve para documentar/validar el acuerdo, no es un destino tomado de solicitudes entrantes). Los callbacks usan la URL fija de la pareja: no aceptar una URL arbitraria del cuerpo. No seguir redirecciones ni desactivar la validación TLS.

## Pruebas reproducibles

El CI ejecuta `infrastructure/scripts/test-interbank.sh` para cada componente afectado: PostgreSQL efímero para reservas/depósitos, mock Python del contrato para aceptación/rechazo y callbacks HTTP reales de loopback, y RabbitMQ efímero para RPC entre réplicas/reinicio del consumidor. Los tests nunca leen las bases o claves reales del proyecto. Localmente se puede usar `TEST_DATABASE_URL` de una instancia desechable y `INTERBANK_MOCK_PYTHON` con PyYAML instalado para ejecutar `go test -count=1 ./internal/interbank` desde cada servicio. Sin esas variables los tests que requieren dependencias externas se omiten explícitamente.

Las migraciones aditivas están incluidas en el paso de preparación del despliegue. Se prueban la reejecución y los duplicados concurrentes. Sus migraciones de bajada se niegan a eliminar recibos financieros existentes; el rollback de imágenes no borra reservas ni datos.

## Recuperación, cancelación y devoluciones

TIMEOUT **conserva la reserva**. Se consulta el estado y se intenta cancelar. ACCEPTED/ALREADY_ACCEPTED confirma el débito; REJECTED/CANCELLED/ALREADY_REJECTED libera la reserva. Si no se obtiene respuesta fiable de ninguno de los dos endpoints, solo se libera después del margen adicional de 300 s y se genera `TIMEOUT_MARGIN_RELEASE`. Un fallo de autenticación o una respuesta PROCESSING no se convierte automáticamente en permiso para liberar: se conserva la reserva y se alerta para revisión.

La cancelación guarda un tombstone incluso si el depósito original nunca llegó. Una llegada posterior queda CANCELLED, no se acredita. Si Cuentas ya había acreditado pero Transacciones se reinició antes de guardar el resultado, la cancelación responde ALREADY_ACCEPTED a partir del recibo persistente. Los callbacks y sus fechas originales sobreviven a reintentos; cancelar de nuevo no reinicia el límite de 24 h.

ACCEPTED después de REVERSED se confirma con ACK 200, genera incidente ERROR y crea una sola solicitud de reverso con id determinista. Como receptor del reverso, se verifica el depósito original, se reserva el principal sin otra comisión y se envía un depósito normal de retorno con un id nuevo y persistente. COMPLETED requiere que ese retorno sea aceptado. Saldo insuficiente o retorno rechazado produce FAILED e incidente, nunca una devolución ficticia.

El emisor que ya liberó la reserva no debe abonar otra vez al cliente cuando regresa el dinero: el retorno se registra como **liquidación interbancaria**, con recibo idempotente asociado a la reserva RELEASED. Se verifica su id contra el `refundTransferId` publicado por el banco pareja; no se identifica por descripción o por monto solamente. BANK03 publica ese campo desde PROCESSING (el esquema lo permite). **Coordinar con BANK14 que también lo publique antes de esperar el callback del retorno**. Si la identidad del retorno no puede verificarse, se retiene su procesamiento y, al vencer, se rechaza y registra incidente; no se acredita a ciegas.

Un ACCEPTED tardío de un propio depósito de devolución ya liberado queda como incidente de revisión manual: no se inicia una cadena infinita de reversos de reversos. No forzar saldos negativos ni borrar recibos para resolver incidentes.

## Conciliación diaria

Un trabajo persistente dentro de Transacciones compara páginas de los depósitos recibidos por BANK14. Recupera días pendientes tras reinicios y espera hasta las 00:10 UTC para cerrar el día anterior. Se contrastan id, monto, moneda y estado; resultados finales pendientes alimentan la misma recuperación idempotente. Las operaciones junto a medianoche se consultan también por id para distinguir recepción en el día contiguo.

Se registran registros remotos desconocidos, montos distintos, estados contradictorios y depósitos aceptados ausentes. `conciliaciones_interbancarias` guarda fecha, página, estado y discrepancias; COMPLETED significa comparación terminada, no ausencia de incidentes. Una discrepancia no autoriza a modificar saldos por sí sola. Los incidentes se persisten, publican en Outbox y clasifican ERROR en Auditoría. Dashboards y reglas de Grafana corresponden al punto 10.
