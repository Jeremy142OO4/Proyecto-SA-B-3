# API Gateway: instrumentación y dashboard

## 06/API_gateway en Bank USAC-Prueba1

- Tráfico HTTP por clase de respuesta: solicitudes por segundo en ventana de 5 min.
- Latencia HTTP p50/p95/p99 desde bank_http_duration_seconds_bucket; suma buckets entre réplicas antes del percentil. Mide respuesta HTTP, no toda la Saga asíncrona.
- Porcentajes 4xx y 5xx sobre respuestas observadas. 4xx no se presenta como error de negocio. Sin denominador positivo no se inventa éxito.
- Intentos AMQP por acción: published, publish_failed, processed, failed. No son eventos de negocio únicos.
- Latencia p95 del consumidor desde bank_message_processing_seconds_bucket; no incluye espera en cola.
- Trazas de entrada: spans de servidor HTTP /api/ del Gateway, agrupados por Trace ID y enlazados al flujo completo de Tempo.
- Logs HTTP/broker con traceId, spanId y correlationId cuando existe contexto. Reutilizan filtros de IDs/nivel de auditoría; servicio fijado al Gateway. IDs se extraen del JSON al consultar Loki, no se añaden a métricas o índices.

Se reutilizan environment/rango del dashboard y datasources existentes. Sin actividad suficiente, rate y percentiles pueden carecer de datos. Las consultas de salud/métricas se excluyen. No se agregan paneles que afirmen cobertura W3C al 100% a partir de volumen de mensajes: esa propiedad requiere revisar los headers y las relaciones de los spans.

## Contexto, OTLP y convenciones

Publish inyecta traceparent/tracestate en propiedades AMQP. Consume extrae ese contexto antes de crear el span de consumidor. correlationId sigue siendo atributo del span y campo de log; traceId/spanId son los identificadores técnicos. El código conserva los nombres de métricas bank_* existentes y sus etiquetas acotadas para no romper paneles.

Se completaron atributos del Gateway: http.request.method, http.route, http.response.status_code, messaging.system=rabbitmq, messaging.operation.type=send/process, messaging.message.id y messaging.rabbitmq.destination.routing_key. En consumo se añade messaging.destination.name cuando se conoce el exchange. Las claves y la operación se ajustan a las convenciones RabbitMQ: https://opentelemetry.io/docs/specs/semconv/messaging/rabbitmq/ . No se registran cuerpos ni credenciales.

La configuración existente exporta OTLP por HTTP al Collector en el clúster. Collector envía métricas a Prometheus, trazas a Tempo y logs a Loki; los logs stdout los recoge el agente. La raíz propia del portal y el enlace a padres externos siguen aplicándose. Las rutas interbancarias conservan propagación externa.

## Validación y despliegue

El conjunto de pruebas del Gateway pasó, incluida una prueba de publicación/consumo con W3C: mismo Trace ID, consumidor hijo del productor, correlación preservada y atributos semánticos. Se consultaron los ocho targets PromQL contra Prometheus desplegado: todos fueron válidos y devolvieron series; se confirmaron los cuatro instrumentos base HTTP/AMQP. La visualización de los paneles nuevos queda pendiente de aprovisionar.

Publicar el JSON y la imagen del Gateway mediante CI/CD. No requiere migraciones, secretos nuevos ni cambios de rutas públicas. El resto de microservicios mantiene su instrumentación existente; este cambio de atributos corresponde al Gateway.
