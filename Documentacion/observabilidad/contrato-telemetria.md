# Contrato común de telemetría — Fase 3

Estado: **diseño aprobado para implementación, todavía no instrumentado ni desplegado**. Alcance: Gateway y cinco microservicios. No modifica el contrato OpenAPI ni los payloads de negocio.

## Identidad

| Carpeta | serviceName estable |
|---|---|
| gateway/api-gateway | api-gateway |
| services/service-customer | customer-service |
| services/account-service | account-service |
| services/transaction-service | transaction-service |
| services/payment-service | payment-service |
| services/service-notification-audit | notification-audit-service |

Todos los instrumentos llevan servicio, entorno (`dev`, `prod`, `local` o `test`) y versión inmutable de imagen/commit. No mezclar entornos. El frontend se observa mediante HTTP y sondas; no sustituye la instrumentación del backend.

## Logs

Una línea JSON por registro, enviada a stdout. Campos obligatorios: `timestamp` (UTC RFC3339), `serviceName`, `environment`, `version`, `level` (`DEBUG`, `INFO`, `WARNING`, `ERROR`), `message`, `correlationId`, `traceId`, `spanId`. Para arranque o tareas sin operación activa, los tres identificadores pueden ser nulos; nunca inventar un contexto para aparentar trazabilidad.

`correlationId` conserva el `idCorrelacion` de negocio; no es el traceId. Añadir cuando corresponda `messageId`, `operation`, `eventType`, `errorCode` y `durationMs`. Los identificadores de cuenta y operación solo se admiten en logs/trazas con acceso restringido, nunca como etiquetas de métricas.

No registrar cuerpos completos, contraseñas, hashes, JWT, API keys, cadenas de conexión, documentos, fotografías, correos ni enlaces de activación. Sanear también errores de librerías, URLs y consultas SQL. Los logs técnicos irán al recolector de logs; la auditoría de negocio mantiene su persistencia y flujo de eventos independientes.

## Trazas y transporte

1. Extraer contexto W3C `traceparent` y `tracestate` de HTTP; validar el formato. Sin contexto válido, iniciar una traza nueva. No confiar en baggage externo ni copiar secretos.
2. Crear spans para ingreso HTTP, publicación, procesamiento del consumidor, persistencia, llamada externa y compensación. Usar rutas plantilla y nombres de operación estables, no IDs en nombres.
3. Inyectar el contexto del span productor en headers AMQP `traceparent`/`tracestate`. El consumidor lo extrae antes de crear su span. El sobre existente conserva `idMensaje`, `idCorrelacion`, `idCausa` y `version`.
4. Persistir contexto de traza con el registro Outbox **en la misma transacción** que el cambio de negocio. Requiere migraciones compatibles en cada servicio. El publicador restaura ese contexto después de reinicios; no depende de una variable en memoria.
5. Cada reentrega crea un span de intento con el contexto original, conservando mensaje, correlación y clave de idempotencia. No reutilizar spanId ni repetir efectos financieros. Registrar redelivery e intento sin alterar el contrato.
6. Persistir contexto de Saga para que continuaciones y compensaciones conserven traceId aunque cambie el proceso. Evitar spans abiertos esperando horas: spans breves relacionados mediante contexto persistido. Operaciones humanas posteriores de KYC pueden tener trazas nuevas enlazadas a la inicial y la misma correlación de ciclo de vida.
7. En integración interbancaria propagar `traceparent` en solicitudes, callbacks, consultas y devoluciones según OpenAPI. Persistir contexto por transferencia. Una consulta independiente mantiene su propia traza y enlaza a la operación original; los callbacks derivados continúan la traza de la operación. No derivar IDs de trazas del UUID de negocio.

## Exportación y seguridad operativa

SDK OpenTelemetry en los seis componentes, exportación OTLP al Collector interno. Destino previsto: métricas a Prometheus, logs a Loki y trazas a Tempo, visualizados en Grafana. Versiones y despliegue reproducibles se resolverán en el paso de infraestructura; este documento no crea recursos.

Inicialmente conservar el 100 % de trazas en dev durante validación; producción tendrá muestreo explícito y documentado según capacidad. Métricas SLI no se calculan de trazas muestreadas. Colas de exportación acotadas, envío asíncrono y cierre con plazo limitado: la caída del Collector no debe bloquear una transacción ni cambiar su resultado. Medir pérdida de telemetría. No exponer Collector, métricas ni Grafana públicamente sin control de acceso.

## Pruebas de aceptación pendientes

- Registro, cuenta, transferencia, pago y compensación visibles con correlación y traceId entre servicios.
- Header AMQP y contexto Outbox sobreviven a reentrega, reconexión y reinicio; no hay doble débito/crédito.
- Un mock del banco compañero conserva traceId en recepción y callback; timeout y reverso siguen siendo trazables.
- Logs JSON válidos y sin datos sensibles, incluidos errores de autenticación y base de datos.
- Collector inaccesible: operación bancaria sigue funcionando y la pérdida de exportación es observable.
- Métricas sin IDs ni PII en etiquetas; pruebas verifican el catálogo y los límites de cardinalidad.

Estas pruebas deben incorporarse al CI al implementar la instrumentación. La documentación por sí sola no las satisface.
