# Instrumentación común — Punto 4

Estado al 18 de octubre de 2026: **implementada, publicada y desplegada en dev; pruebas aisladas y verificaciones en la nube completadas**. La instrumentación se publicó en `1e28285`; la plataforma y la recuperación de mensajería, en `22eef62`, con CI/CD exitoso. No certifica el SLO de 99.5 %. La plataforma corresponde al [punto 3](plataforma.md).

## Alcance implementado

- SDK OpenTelemetry Go v1.38.0 en Gateway y los cinco microservicios.
- Spans de ingreso HTTP, publicación/consumo AMQP y operaciones de repositorio, sin texto SQL ni parámetros. Las compensaciones se identifican por el tipo de evento.
- Propagación W3C en headers AMQP y transporte HTTP reutilizable para futuras llamadas interbancarias. Se conserva la correlación de negocio; no se acepta baggage externo.
- Columna `trace_context` en los cuatro Outbox, escrita en la transacción de negocio y restaurada al publicar. Así se conserva la traza al reanudar la Saga mediante eventos; un futuro temporizador autónomo interbancario deberá persistir también su contexto de reanudación.
- Logs JSON con servicio, entorno, versión, correlación y contexto de span. Errores libres y atributos sensibles redactados. Los mensajes heredados de `log.Printf` se convierten deliberadamente en una señal genérica segura, sin publicar su texto arbitrario.
- Contadores e histogramas HTTP/AMQP. No incluye todavía métricas de dominio KYC/anomalías ni evaluadores durables de SLO. El tablero inicial de infraestructura y telemetría común está disponible mediante la plataforma del punto 3; no sustituye los dashboards y alertas completos del punto 10.

El código común reside en `internal/telemetry` de cada módulo porque los contextos Docker son independientes. Mantener sincronizadas sus implementaciones y pruebas; extraer un módulo compartido requiere cambiar contextos y filtros.

## Configuración

| Variable | Uso |
|---|---|
| OTEL_EXPORTER_OTLP_ENDPOINT | URL base del Collector **OTLP/HTTP**, por ejemplo http://otel-collector:4318; no usar el puerto gRPC 4317 |
| OTEL_EXPORTER_OTLP_TRACES_ENDPOINT | URL completa opcional para trazas |
| OTEL_EXPORTER_OTLP_METRICS_ENDPOINT | URL completa opcional para métricas |
| DEPLOYMENT_ENVIRONMENT / APP_ENV / ENTORNO | Entorno, en ese orden; desarrollo y produccion se normalizan a dev y prod |
| SERVICE_VERSION / IMAGE_TAG | Versión; como respaldo se usa la incrustada durante el build |
| OTEL_TRACES_SAMPLER_ARG | Proporción de raíces entre 0 y 1; se respeta la decisión del padre. Predeterminado 1 |
| OTEL_SDK_DISABLED=true | Desactiva grabación/exportación; conserva logs JSON |

Sin endpoint no se exporta, pero funcionan el contexto y los logs. No se configura un Collector inexistente en Kubernetes. Exportación asíncrona acotada y con plazos limitados: su caída no determina el resultado bancario. Se mantienen las comprobaciones TLS.

CI pasa el tag de imagen al build de los seis componentes. Clientes y Auditoría ahora leen también el ConfigMap común para recibir entorno y futura configuración OTLP.

## Migraciones y despliegue

Clientes utiliza migración 000006, Cuentas 000005, Transacciones 000004 y Pagos 000004. El script `infrastructure/scripts/prepare-telemetry-schema.sh` prepara estas columnas con Jobs temporales antes del rollout dev/prod. Solo aplica ampliaciones idempotentes: **no repite las migraciones iniciales**, pues las bases existentes se inicializaron fuera de un historial de migrate. Las credenciales provienen del Secret. Un fallo detiene el rollout.

Los eventos anteriores conservan contexto vacío. No eliminar columnas durante un rollback de aplicación. El publicador antiguo de Clientes usa SELECT * con sqlx y puede rechazar temporalmente la columna nueva mientras se sustituye el Pod: no se promete compatibilidad perfecta de esa versión antigua. Comprobar el Outbox pendiente después del rollout. Evitar consumidores dev/prod concurrentes sobre bases compartidas.

## Validación del 18 de octubre de 2026

- go test ./... y go vet ./... en los seis módulos.
- 36 pruebas comunes nuevas: continuidad Outbox/reintento, contexto inválido, estados HTTP/métricas, logs seguros, OTLP/Collector caído y transporte HTTP a un compañero simulado.
- 6 pruebas contra RabbitMQ real y 4 de persistencia Outbox contra PostgreSQL real, con migraciones aplicadas.
- Pruebas existentes de recuperación. Se corrigió una suposición de entrega única del test: el cierre antes del ACK puede producir una reentrega; solo se aceptan mensajes ya conocidos.
- Seis ejecutables reales: login, cuentas, transferencia exitosa y compensada, pago exitoso y compensado por timeout. Comprobación de saldos y trazas de cinco saltos con el mismo traceId, presentes en OTLP.
- Bash/YAML y simulación del gate de migraciones con éxito/fallo. Actionlint sin errores nuevos, excluyendo advertencias históricas SC2129 y SC2034.

Las pruebas comunes entran en los jobs CI existentes por componente. Las reales necesitan TEST_RABBITMQ_URL y TEST_DATABASE_URL; sin ellas se omiten expresamente. Las pruebas previas de repositorio necesitan además URL_BASE_DATOS_PRUEBAS sobre una base migrada. Nunca utilizar bases con clientes reales.

Las dependencias temporales y los ejecutables se eliminan al terminar. Posteriormente se desplegó el commit `1e28285` en dev y se verificaron las cuatro columnas Outbox y los seis componentes. El punto 3 conecta Collector/backends y el tablero inicial; consultar [plataforma](plataforma.md). Tras desplegar `22eef62`, se reinició RabbitMQ en dev: los seis servicios recuperaron la conexión conservando sus pods y sin reinicios adicionales de los procesos. Después se verificaron consultas, métricas, logs y trazas. Siguen pendientes métricas de dominio y alertas. La definición del SLO no demuestra su cumplimiento.

Referencias: [OpenTelemetry Go](https://opentelemetry.io/docs/languages/go/getting-started/) y [exportadores](https://opentelemetry.io/docs/languages/go/exporters/).
