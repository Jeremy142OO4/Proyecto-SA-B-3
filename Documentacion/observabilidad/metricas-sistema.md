# Métricas sistema

Dashboard independiente, UID `bank-system-metrics`, carpeta Bank USAC, entorno dev.
Fuente versionada: `infrastructure/observability/system_dashboard.py`; JSON generado:
`chart/dashboards/sistema.json`. No sustituye los paneles por servicio.

## Contenido y fuentes

| Sección | Fuente / interpretación |
|---|---|
| Estado | Réplicas disponibles, scrape de Collector e infraestructura, disponibilidad HTTP y presupuesto de solicitudes. |
| Latencia | Histograma `bank_transfer_flow_duration_seconds_bucket`: p50, p95, p99; creación hasta resultado terminal, incluidas compensaciones. Sin terminales recientes no hay percentil. |
| Errores | 5xx HTTP separados de resultados de negocio fallidos/compensados. Un rechazo válido no es automáticamente un error técnico. |
| Throughput | `bank_messages_total` por servicio, acción y routing key del catálogo (`event_type`); publicaciones, entregas y ACK por cola desde RabbitMQ. Intentos, no movimientos financieros únicos. |
| Consumer lag | `rabbitmq_message_current`: ready y unacknowledged; consumidores por cola; ready/consumidores activos como promedio. RabbitMQ comparte pendientes entre consumidores: no se inventa una asignación individual. Colas sin consumidores se muestran aparte. |
| Infraestructura | Réplicas y HPA del receiver k8s_cluster. CPU en cores y memoria en bytes desde Metrics Server. |
| Errores en tiempo real | Loki, JSON original, WARNING/WARN/ERROR. Enlace de traceId a Tempo mediante derived field existente. |
| Trazas | Tempo, solicitudes de negocio del Gateway; abrir Trace ID permite recorrer los servicios y compensaciones. |
| Despliegues | Anotaciones por cambios de revisión de Deployment observados cada 30 s; no confundir con despliegue exitoso. |

El dashboard consulta métricas en Prometheus. Loki almacena logs y Tempo almacena trazas;
no se convierten sus UUID en etiquetas de métricas. Se conserva el contador agregado `other`
para routing keys ajenas al catálogo. Publicación y consumo se muestran separados para evitar
contabilizar ambos como eventos únicos.

## SLO y presupuesto: cobertura real

Se mantienen los objetivos de `Documentacion/sre/slo.md`, sin cambiarlos por metas más fáciles.
Ventana móvil de 30 días. `Sin datos` no equivale a 100% de cumplimiento. La retención máxima
es 30 días o 2 GB; con una historia menor o huecos no se certifica el mes completo.

| Objetivo | Medición / presupuesto disponible |
|---|---|
| SLO-01, Gateway ≥99.5% no 5xx | Sí. Incluye 4xx; excluye salud/readiness/métricas. Presupuesto = 0.005×solicitudes; restante = permitido−5xx. Se muestra porcentaje restante, unidades restantes y burn rate. Sin tráfico el cociente queda sin datos. |
| SLO-02, servicio Ready ≥98.5% | Proxy observado: al menos una réplica disponible por despliegue. Presupuesto 1.5% de los minutos observados. No mide funcionamiento financiero ni presume éxito durante ausencia de telemetría. |
| SLO-03, Gateway p95 ≤500 ms | Histograma de 30 días. Presupuesto de solicitudes lentas: hasta 5% >500 ms. No sustituye duración de Saga. |
| SLO-04, procesamiento completo p95 ≤10 s sin compensación | La latencia terminal de transferencias está disponible como SLI. Falta cobertura uniforme de todos los flujos y exclusión verificable de compensaciones para certificar este SLO general. El histograma del handler AMQP mide solo ejecución del consumidor. |
| SLO-05/06, éxito técnico ≥99% / error técnico ≤1% | Pendiente denominador de operaciones iniciadas y clasificación técnica uniforme. Los contadores actuales de resultado terminal mezclan rechazo de negocio y fallo: no se presentan como cumplimiento técnico. |
| SLO-07, idempotencia 100% | Requiere duplicados recibidos y duplicados sin efecto adicional; las pruebas funcionales no sustituyen una métrica mensual. |
| SLO-08/09, entrega ≥99.9% / DLQ ≤0.5% | Requiere seguimiento lógico de entrega y DLQ por mensaje, incluyendo pendientes y plazos. ACK/intentos y backlog no prueban entrega lógica única. |
| SLO-10, trazabilidad 100% | Consultar trazas es posible; falta denominador de operaciones auditadas y verificación automática de completitud. Un ID de log no prueba spans almacenados. |
| SLO-11, compensaciones ≥99% | Requiere contador de compensaciones iniciadas y completadas por flujo. No se infiere de la tasa de transferencias fallidas. |
| SLO-12, notificaciones enviadas ≥95% | Requiere contadores de notificaciones generadas/enviadas. Registrar una notificación o aceptar SMTP no prueba recepción en bandeja. |

Los objetivos pendientes quedan explícitos; este dashboard no declara implementados todos
los SLO del proyecto. No genera valores ficticios para llenar esas casillas.

## CPU, memoria y anotaciones

El sidecar `system-metrics` del Collector consulta solo dos endpoints con su ServiceAccount:
`metrics.k8s.io/v1beta1/namespaces/bank-usac-dev/pods` y
`apps/v1/namespaces/bank-usac-dev/deployments`. Reutiliza el Metrics Server existente.
RBAC adicional: get/list de pods en metrics.k8s.io, limitado al namespace; sin Secrets,
ClusterRole, nodos, acceso de escritura ni permisos sobre bases del banco.

Muestreo cada 30 s, peticiones con timeout de 8 s, token rotatorio y CA verificada.
Muestras de consumo con más de 120 s se omiten; un fallo de API elimina las series
de esa fuente y publica `bank_system_source_up=0`, no CPU ficticia de cero.
No almacena payloads ni credenciales en Prometheus. Añade 5m CPU/32Mi de requests
al pod existente, sin crear nodos, discos o endpoints públicos. Preflight incluye ese coste.

Las anotaciones detectan cambios del número de revisión, incluidos rollbacks. Solo cubren
cambios observados después del despliegue del sidecar. No reconstruyen fechas antiguas y
pueden omitir cambios muy rápidos entre scrapes; no constituyen auditoría del CI.

Fuente de CPU/memoria: [Resource Metrics API de Kubernetes](https://kubernetes.io/docs/tasks/debug/debug-cluster/resource-metrics-pipeline/).

## Alertas, logs y operación

Se reutilizan las ocho reglas existentes y su canal visible de correo técnico;
se añade `bank-system-source` para fallos de Metrics Server/consulta de revisiones.
Incluyen burn rápido y sostenido, latencia HTTP, Collector caído, infraestructura ausente,
despliegue no disponible y acumulación/ausencia de RabbitMQ.
Cada regla enlaza su runbook en [alertas.md](alertas.md). No se envían correos de prueba
ni se generan transacciones automáticamente con este cambio.

Los seis servicios ya usan el handler JSON común: timestamp, serviceName, environment,
version, level, correlationId, traceId y spanId; los ID de traza se completan cuando existe
contexto activo (arranque y tareas fuera de una traza no inventan IDs).
La recolección existente preserva el cuerpo JSON y los enlaces a Tempo.
El dashboard no añade datos personales, JWT ni contraseñas y no sustituye una revisión de
todos los mensajes de negocio para detectar datos sensibles en texto libre.

## Validación y despliegue

```sh
python infrastructure/observability/system_dashboard.py
python infrastructure/observability/system_dashboard.py --check
python infrastructure/observability/test_system_dashboard.py
python infrastructure/observability/alerting.py --check
```

El CI ejecuta los controles anteriores dentro de `validate.py`, además de Helm y las
pruebas existentes. El despliegue se realiza con el pipeline de observabilidad existente;
no requiere migraciones SQL. Luego abrir `/d/bank-system-metrics` y comprobar
`bank_system_source_up=1`, CPU/memoria, paneles y estados de las nueve reglas.
Los percentiles/rates necesitan al menos dos muestras y actividad en su ventana.

### Verificación de RabbitMQ

En la revisión del 21 de octubre de 2026 se compararon las consultas del JSON provisionado
con el generador y con RabbitMQ directamente (`rabbitmqctl list_queues`, solo metadatos).
Prometheus recibió ready/unacknowledged y consumidores para las 18 colas persistentes:
14 con consumidor y cuatro colas de fallidos vacías sin consumidor. Los valores actuales
coincidieron con el broker: cero mensajes pendientes y cero sin confirmar.

Las tres consultas de throughput devolvieron 13 series por cola. Los incrementos de
publicación y ACK en las últimas 24 horas fueron distintos de cero en varias colas,
incluidas pagos, cuentas y transacciones. Las colas sin contadores de message_stats no se
rellenan con movimientos ficticios. Estos incrementos son estimaciones de Prometheus,
no cantidades contables de operaciones únicas. Las consultas de throughput de aplicación
por servicio/routing key son otra fuente, separada de los contadores del broker.

El panel «Colas con pendientes y sin consumidores» ahora cuenta las colas afectadas:
**0** indica que ninguna cola observada cumple ready > 0 y consumidores = 0.
Si faltan métricas de colas/consumidores o falla el scrape bank-infra, conserva **Sin datos**.
No confunde una DLQ vacía sin consumidor con una cola bloqueada.
Siete escenarios de esa consulta se ejecutaron con promtool, sin publicar mensajes,
detener consumidores ni modificar operaciones del banco; quedan incluidos en el CI.

El receiver mide el estado del broker cada 30 s y Prometheus recolecta cada 15 s.
Los gauges ready/unacked pueden permanecer en cero cuando los mensajes se procesan
entre mediciones; el throughput y los contadores históricos permiten verificar actividad.
`up{job="bank-infra"}=1` solo prueba el scrape del Collector; también se verificó la
presencia de las métricas específicas de RabbitMQ y sus valores contra el broker.

### Bootstrap de permisos del Collector

La identidad CI puede desplegar workloads, pero no necesariamente modificar Roles.
El cambio que incorpora lectura de `metrics.k8s.io/pods` requiere una sincronización
administrativa inicial del Role `bank-observability`, solo en `bank-usac-dev`.
No requiere conceder `cluster-admin` ni administración de RBAC al pipeline.

El 21 de octubre de 2026 se sincronizaron las reglas del Role existente con el chart,
incluido su orden, conservando las anotaciones de propiedad de Helm. La única capacidad
añadida fue `get/list` de métricas de pods. Se debe reintentar el job fallido; el rollout
anterior fue revertido por `--atomic` y no se considera desplegado.

`rbac_preflight.py` se ejecuta antes de sincronizar Secrets o iniciar Helm: si el Role
ya coincide, CI no necesita modificarlo. Si cambia y la identidad no puede hacer patch,
falla con un diagnóstico antes de iniciar el upgrade. Un administrador debe revisar
y sincronizar el Role desde `chart/templates/rbac.yaml`, manteniendo exactamente las
reglas lectoras del chart y su propiedad Helm. Los cambios futuros de permisos requieren
esa misma revisión; no se aplica una ampliación automática de privilegios al CI.
