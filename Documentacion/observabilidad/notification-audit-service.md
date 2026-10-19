# Notification & Audit Service: auditoría consultable y logs

## Responsabilidades

Los eventos de negocio llegan por RabbitMQ y se conservan en PostgreSQL. Los logs operativos se escriben en stdout como JSON; un DaemonSet de OpenTelemetry Collector lee los archivos de los seis servicios instrumentados y envía OTLP al Collector central, que los almacena en Loki. Notification & Audit no consulta ni recibe esos archivos de log, y no necesita acceder a los procesos de otros servicios.

El Gateway dejó de publicar cada error HTTP genérico como `auditoria.error.http`: ahora genera un log JSON, sin depender del broker para devolver el error. Sus eventos de negocio siguen su flujo normal. Los registros históricos del tipo anterior no se borran. Las anomalías de cuenta y las alertas SLO son señales de negocio/operación auditables, no transporte de logs.

El agente normaliza INFO, WARNING (incluido WARN) y ERROR (incluido FATAL), y asigna la severidad numérica OTLP. Conserva correlationId, traceId y spanId como metadatos y rellena los campos nativos de traza/span cuando los IDs son válidos. Los IDs no se convierten en etiquetas de índice de Loki. Se conserva el cuerpo JSON para búsqueda y para el vínculo Loki → Tempo ya configurado en la fuente de datos.

El contrato actual recolecta logs JSON de Gateway, Customer, Account, Transaction, Payment y Notification & Audit. Los mensajes no JSON se descartan según la política previa del receptor; no se añadieron colectores para aplicaciones ajenas al banco. La retención de Loki continúa en 48 horas y los offsets del agente siguen persistidos.

## Endpoints del API Gateway

Todas estas rutas exigen JWT y rol ADMIN:

| Método y ruta | Uso |
| --- | --- |
| GET `/api/auditoria/registros` | Consulta histórica con filtros combinables |
| GET `/api/auditoria/trazas/:traceId` | Consulta por traza distribuida, con los mismos filtros adicionales |
| GET `/api/auditoria/traza/:idCorrelacion` | Consulta existente por correlación |
| GET `/api/auditoria/notificaciones?tipo=ACCOUNT_ANOMALY_DETECTED&nivel=WARNING` | Alertas de cuentas |
| GET `/api/auditoria/notificaciones?tipo=SLO_ALERT` | Notificaciones SLO |
| POST `/api/administracion/alertas/slo` | Registrar una alerta SLO y generar la notificación |

Filtros de `/registros` y `/trazas/:traceId`:

- `correlationId`, con alias anterior `idCorrelacion`: UUID no nulo.
- `traceId`: 32 caracteres hexadecimales, distinto de cero. No es un UUID de correlación.
- `servicio`, alias `service`: productor del evento, por ejemplo `payment-service`; en alertas SLO también coincide con el servicio afectado.
- `nivel`, alias `level`/`severity`: INFO, WARNING o ERROR.
- `desde`/`hasta`: RFC3339, inicio inclusivo y fin exclusivo.
- `limite`: 1–100, por defecto 50; `desplazamiento`: 0–100000, por defecto 0.

Los filtros se combinan con AND. Los resultados usan orden descendente por `occurred_at` e `id`, con paginación estable. Si no hay coincidencias, la respuesta es `[]`. Los alias contradictorios y parámetros inválidos devuelven 400, sin enviar una consulta al broker. Los errores internos de base no exponen cadenas de conexión ni SQL al cliente.

Ejemplo:

```text
GET /api/auditoria/registros?correlationId=<UUID>&traceId=<32-hex>&servicio=payment-service&nivel=ERROR&limite=25&desplazamiento=0
```

La consulta HTTP interna existente `/api/v1/audit/logs` también soporta `correlationId`, `traceId`, `service`, `severity`, `from`, `to`, `limit` y `offset`, con JWT ADMIN. La entrada pública sigue siendo el Gateway; no se añadieron servicios públicos de infraestructura.

## TraceId durable

La migración `000004_consulta_auditoria_trazas` agrega `audit_logs.trace_id` e índices por traza, correlación, productor/severidad y tiempo. El dato proviene del contexto W3C del productor recibido por AMQP. Un span nuevo del consumidor no se usa para inventar la traza de un evento sin contexto. Los registros anteriores conservan traceId vacío y siguen consultables por los demás campos; no se realiza un backfill especulativo.

El servicio guarda la auditoría de negocio en PostgreSQL; este endpoint no es un proxy de Loki. La búsqueda de logs operativos y la navegación hacia Tempo se realizarán desde Grafana.

## Alertas SLO y notificaciones

El consumidor acepta el evento exacto `SLOAlertTriggered` en `banco.eventos`, además de `AccountAnomalyDetected`. El endpoint administrativo recibe este formato canónico:

```json
{
  "idAlerta": "payment-error-budget",
  "servicio": "payment-service",
  "nivel": "ERROR",
  "resumen": "Se agotó el presupuesto de errores del pago externo",
  "inicio": "2026-10-19T01:00:00Z",
  "correlationId": "11111111-1111-4111-8111-111111111111",
  "traceId": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
}
```

`correlationId` y `traceId` son opcionales; los demás campos son obligatorios, salvo `nivel`, cuyo valor inicial es WARNING. Solo se aceptan WARNING y ERROR para SLO. `inicio` identifica el inicio real del episodio. El productor normalizado de auditoría es `slo-monitor`; el servicio afectado permanece en el payload `servicio` y también participa en el filtro `servicio` de consulta.

El ID de evento se deriva de servicio afectado, idAlerta, severidad e inicio UTC. Repeticiones del mismo episodio/severidad no crean otra notificación; un cambio de severidad tiene identidad propia. La correlación recibida se conserva, o se deriva una correlación estable del episodio. Si se proporciona una traza de la señal original, se usa como referencia en la fila de auditoría.

Se registra una notificación `SLO_ALERT` para ADMIN, con severidad, resumen y detalles. Sin correo configurado queda PENDING en la bandeja consultable. `SLO_NOTIFICATION_RECIPIENT` permite indicar un correo operativo y utilizar el SMTP existente; su valor inicial en ConfigMap es vacío. Un fallo SMTP o una dirección inválida conserva la notificación con estado FAILED. Los mensajes de correo se escapan antes de generar HTML. La entrega de correo usa el mecanismo existente del servicio; no se promete exactamente una entrega SMTP ante caídas entre envío y registro.

`AccountAnomalyDetected` mantiene su notificación WARNING, identificación de cuenta y detalles ya implementados. No se añadieron usuarios ni roles de prueba.

El POST canónico permite registrar/probar alertas con autorización ADMIN. No recibe directamente el formato de webhooks de Grafana/Alertmanager ni configura sus reglas: un productor o adaptador autorizado debe convertir sus alertas al contrato canónico. Esa integración visual/operativa queda para la etapa de Grafana.

## Consultas futuras en Grafana

Buscar un correlationId en logs de Loki:

```logql
{deployment_environment_name="dev"} | json | correlationId="<UUID>"
```

Buscar una traza:

```logql
{service_name="payment-service"} | json | traceId="<32-hex>"
```

Buscar errores de un servicio:

```logql
{service_name="payment-service"} | json | level="ERROR"
```

El campo derivado TraceID de Loki ya apunta a la fuente Tempo. No se crearon dashboards, reglas de alertas ni cambios visuales en Grafana en esta tarea. Referencia de los campos nativos y severidades: [OpenTelemetry Log Context](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/contexts/ottllog/README.md).

## Verificación y despliegue

Pruebas locales verifican filtros individuales/combinados y temporales, paginación, índices, datos sin traza, deduplicación, notificaciones reales en PostgreSQL para anomalías y SLO, correo opcional simulado, escape HTML, roles del Gateway, parámetros inválidos y respuestas 401/403/400/422/502/503. La prueba del consumidor confirma la traducción RPC de todos los filtros y la recepción de SLO. Una prueba del Gateway verifica errores HTTP sin broker ni filtración del error técnico.

La migración 000004 se incluyó en el paso de esquemas anterior al despliegue. El CI/CD de Notification & Audit ejecuta las pruebas de repositorios y servicio con PostgreSQL temporal y `go test -race`. Se validaron localmente el YAML del agente/Collector/Loki, el workflow, el ConfigMap, sintaxis Bash y los tests de scripts de observabilidad. Tras recuperar SSH se ejecutaron en Linux el render Helm, la validación binaria del agente y el Collector 0.163.1 y una prueba de ingestión con logs CRI sintéticos. La prueba verificó INFO/WARNING/ERROR, correlationId y los campos nativos trace/span. Esa comprobación reproducible se añadió al validador de observabilidad del CI/CD. El validador completo también terminó correctamente en el servidor, incluyendo Helm, configuraciones binarias de Collector/Prometheus/Loki y la nueva prueba de los tres niveles de log. También pasaron el nuevo paso de PostgreSQL del CI y las pruebas con detección de carreras del Gateway y de los paquetes de repositorios/servicios de auditoría.

Los primeros intentos de SSH a 201.216.128.230 agotaron el tiempo de espera, pero el último permitió conectarse y ejecutar las verificaciones descritas en contenedores y directorios temporales. No se modificaron las bases desplegadas ni el clúster de Google Cloud. La prueba del agente utilizó una salida de depuración aislada; no constituye una prueba de ingestión del Loki desplegado ni una búsqueda visual en Grafana. Los cambios siguen locales hasta su publicación y ejecución del CI/CD.
