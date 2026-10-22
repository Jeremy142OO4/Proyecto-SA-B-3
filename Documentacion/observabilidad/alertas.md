# Alertas y operación — entorno dev

Responsable: equipo técnico. Destino de correo: `3141193530501@ingenieria.usac.edu.gt`.
Grafana → Alerting → Alert rules muestra estados Pending/Firing/Normal/Error. No son notificaciones del panel de clientes ni del administrador bancario.
El canal de correo usa el SMTP existente mediante un Secret separado, STARTTLS obligatorio y validación de certificados. Nunca versionar credenciales.
Agrupación por alerta y ambiente: espera inicial 30 s, agrupación 5 min, recordatorio 4 h (críticas 1 h); se envía recuperación.

## SLO y límites de interpretación

Disponibilidad del Gateway: 99.5% en ventana móvil de 30 días, respuestas no 5xx / respuestas de negocio. Los 4xx cuentan como respuestas válidas; se excluyen health (incluidos sus subpaths), healthz, salud, ready, readyz y metrics.
Presupuesto restante = `0.005 * solicitudes - respuestas_5xx`. Puede ser negativo. Sin tráfico no hay datos, no se asume 100%.
Burn rate = fracción de 5xx / 0.005. Se requieren al menos 20 solicitudes en la ventana corta para evitar alarmas por una única prueba.
Retención: máximo 30 días **o 2 GB**, lo que se alcance primero, conservando el disco existente de 3 GiB. El panel muestra historia TSDB disponible; esto no acredita cobertura continua del Gateway. No certificar un mes completo hasta reunir evidencia suficiente.
El p95 HTTP de 500 ms mide aceptación/consulta, no finalización de transferencias interbancarias. Los plazos interbancarios no se reemplazan por el SLO interno de 10 segundos.
Los receivers existentes del Collector recogen despliegues/HPA del namespace dev y métricas reales del management API de RabbitMQ cada 30 s. No se crean nodos, discos, balanceadores ni servicios públicos adicionales. El acceso Kubernetes es únicamente get/list/watch sin permisos de Secrets ni escritura.

## Runbooks

Convenciones de esta sección:

- Namespace: `bank-usac-dev`.
- `<deploy>` = nombre del Deployment afectado (`kubectl -n bank-usac-dev get deploy`). Los servicios son `api-gateway`, `customer-service`, `account-service`, `transaction-service`, `payment-service` y `notification-audit-service`.
- Las consultas PromQL se ejecutan en Grafana → Explore → Prometheus; LogQL en Loki; las trazas en Tempo. Los nombres de etiquetas de Loki (`service`, `level`, `traceId`) 
- Regla general de todas las alertas: **no reiniciar, purgar colas ni compensar transferencias automáticamente**. Toda acción correctiva la decide una persona del equipo técnico.
- Niveles de escalamiento: **N1** = integrante de guardia del equipo técnico (atiende la alerta); **N2** = integrante responsable del servicio afectado; **N3** = tutor/docente, solo si hay impacto sobre datos financieros o el entorno no se recupera.
- Cada ejecución real de un runbook debe dejar evidencia (captura de la alerta, consulta usada, hora de inicio/recuperación) y, si hubo impacto, un postmortem en `sre/postmortems/`.

### Matriz resumen

| Alerta (uid) | Severidad | Dispara tras | Impacto principal | Primer paso |
|---|---|---|---|---|
| `bank-burn-fast` | critical | 2 min | Presupuesto de error agotándose rápido; usuarios reciben 5xx | Panel SLO + errores por ruta |
| `bank-burn-slow` | warning | 5 min | Degradación sostenida del SLO 99.5% | Identificar ruta/servicio con 5xx |
| `bank-http-latency` | warning | 5 min | Lentitud percibida en el Gateway | Trazas lentas en Tempo |
| `bank-collector-down` | critical | 2 min | Se pierde observabilidad completa | Pod y logs del Collector |
| `bank-infra-telemetry` | warning | 2 min | Sin visibilidad de despliegues/HPA | RBAC del receiver k8s_cluster |
| `bank-deployment-unavailable` | critical | 3 min | Servicio caído (0 réplicas disponibles) | Pods Pending/CrashLoop y eventos |
| `bank-rabbitmq-backlog` | warning | 5 min | Mensajes sin procesar; operaciones asíncronas retrasadas | Consumidores de la cola |
| `bank-rabbitmq-telemetry` | warning | 2 min | Sin visibilidad de colas | Management API y pod del broker |

<a id="bank-burn-fast"></a>
### Consumo rápido del presupuesto

- **Severidad / impacto:** critical. Al ritmo actual el presupuesto de error mensual (0.5%) se consumiría en pocas horas. Los clientes reciben respuestas 5xx en el Gateway.
- **Condición:** burn >14.4 en 1 h **y** 5 min, durante 2 min; mínimo 20 solicitudes/5 min.
- **Detección:** correo al equipo técnico (las críticas se repiten cada 1 h) y estado Firing en Grafana → Alerting.

**Diagnóstico**

1. Abrir el dashboard `Bank USAC — SLO 99.5% y servicios`: paneles *Consumo de presupuesto · 1 h*, *Presupuesto disponible* y *Errores del servidor · 5xx*.
2. Ubicar ruta y servicio con 5xx:
   ```promql
   topk(10, sum by (service, route) (increase(bank_http_requests_total{environment="dev",status_class="5xx"}[15m])))
   ```
3. Correlacionar con la hora del último despliegue y con los pods:
   ```bash
   kubectl -n bank-usac-dev get pods -o wide
   kubectl -n bank-usac-dev rollout history deploy/<deploy>
   kubectl -n bank-usac-dev logs deploy/<deploy> --since=15m | grep -i error
   ```
4. En Loki filtrar errores del servicio y tomar un `traceId`; abrirlo en Tempo para ver en qué salto falla (Gateway → servicio → DB/RabbitMQ).
5. Comprobar dependencias: Cloud SQL (conectividad y conexiones), RabbitMQ (alertas `bank-rabbitmq-*`) y réplicas (`bank-deployment-unavailable`).

**Mitigación**

- Si el inicio coincide con un despliegue defectuoso: aplicar el procedimiento de rollback del pipeline (`kubectl -n bank-usac-dev rollout undo deploy/<deploy>`), previa autorización de N2. No ejecutarlo a ciegas.
- Si la causa es una dependencia (DB, broker), restaurar la dependencia; no reiniciar servicios financieros sin necesidad.
- No compensar ni reintentar transferencias manualmente; revisar primero su estado en la auditoría.

**Verificación de recuperación**

- Ambas ventanas (1 h y 5 min) dejan de satisfacer la condición y la alerta pasa a Normal (llega correo de recuperación).
- `Presupuesto disponible` deja de descender. Registrar cuánto presupuesto se consumió.

**Escalamiento:** N2 si no se identifica la causa en 15 min; N3 si hay transferencias en estado inconsistente.
**Evidencia / relacionado:** captura de la alerta y del panel; postmortem `PM-F3-01` o `PM-F3-04` si el origen fue lógica de negocio o contrato interbancario.

<a id="bank-burn-slow"></a>
### Consumo sostenido del presupuesto

- **Severidad / impacto:** warning. Degradación menor pero persistente; si no se corrige agota el presupuesto del mes.
- **Condición:** burn >6 en 6 h y 30 min, durante 5 min; mínimo 20 solicitudes/30 min.

**Diagnóstico**

1. Identificar si el error es de una ruta concreta o general:
   ```promql
   topk(10, sum by (service, route) (increase(bank_http_requests_total{environment="dev",status_class="5xx"}[6h])))
   ```
2. Comparar volumen de solicitudes y versiones desplegadas en las últimas 6 h (`kubectl -n bank-usac-dev rollout history deploy/<deploy>`).
3. Revisar trazas con error en Tempo y logs de nivel error en Loki para la ruta degradada.
4. Verificar dependencias (DB, broker) y recursos de los pods (`kubectl -n bank-usac-dev top pods`).

**Mitigación:** corregir la causa raíz (fix, configuración, rollback autorizado). Si es un error esperado de una prueba controlada, registrarlo como tal.

**Verificación de recuperación:** el burn cae por debajo de 6 en ambas ventanas y la alerta se resuelve. Documentar el seguimiento aunque se recupere sola.

**Escalamiento:** si se mantiene más de 1 h tras la mitigación, pasar a N2; considerar que puede escalar a `bank-burn-fast`.
**Evidencia / relacionado:** consulta usada y ruta degradada anotadas en el registro de incidentes.

<a id="bank-http-latency"></a>
### Latencia HTTP

- **Severidad / impacto:** warning. p95 de aceptación/consulta del Gateway >500 ms. Este valor **no** mide la finalización de transferencias interbancarias.
- **Condición:** p95 >500 ms durante 5 min, mínimo 20 solicitudes/5 min.

**Diagnóstico**

1. Ver p95 por servicio (panel *HTTP p95 por servicio*) o:
   ```promql
   histogram_quantile(0.95, sum by (le, service) (rate(bank_http_duration_seconds_bucket{environment="dev"}[5m])))
   ```
2. Abrir en Tempo las trazas más lentas y separar el tiempo en: espera de RabbitMQ, consulta a DB, procesamiento del servicio.
3. Revisar réplicas y HPA:
   ```bash
   kubectl -n bank-usac-dev get hpa
   kubectl -n bank-usac-dev top pods
   ```
4. Revisar el panel *RabbitMQ: mensajes por cola* por si hay acumulación que retrase respuestas.

**Mitigación:** corregir el cuello de botella (consulta lenta, consumidor lento, límite de recursos) o revertir un cambio autorizado. **No** aumentar nodos ni costos de forma automática; cualquier ajuste de capacidad requiere aprobación.

**Verificación de recuperación:** p95 por debajo de 500 ms sostenido y alerta en Normal. Confirmar con una consulta de lectura (sin generar transferencias de prueba).

**Escalamiento:** N2 si el p95 sigue alto tras 30 min.
**Evidencia / relacionado:** captura de la traza lenta y del panel; resultados de `pruebas-carga.md` si coincide con una prueba de carga.

<a id="bank-collector-down"></a>
### Recolección caída

- **Severidad / impacto:** critical. Sin el Collector no llegan métricas, logs ni trazas; las demás alertas dejan de ser fiables.
- **Condición:** target `bank-otel` caído o ausente durante 2 min.

**Diagnóstico**

1. Estado del target en Prometheus: `up{job="bank-otel"}`.
2. Estado del pod y eventos del Collector (ajustar el nombre real del Deployment/DaemonSet):
   ```bash
   kubectl -n bank-usac-dev get pods | grep -i otel
   kubectl -n bank-usac-dev describe pod <pod-collector>
   kubectl -n bank-usac-dev logs <pod-collector> --tail=100
   ```
3. Buscar `OOMKilled`, errores de configuración o rechazo de exporters. Revisar su Service y NetworkPolicy.
4. Comprobar que Prometheus esté sano (Grafana → Connections → Data sources → Test).

**Mitigación:** recuperar solo el componente afectado (corregir configuración, ajustar memoria autorizada o reiniciar el pod del Collector). No reiniciar servicios financieros.

**Verificación de recuperación:** `up{job="bank-otel"} == 1` y muestras recientes en paneles (el dashboard vuelve a actualizarse cada 30 s). Los datos del período caído se perdieron; anotarlo en la evidencia.

**Limitación conocida:** si Grafana o Prometheus están caídos, esta alerta no se entrega; se requiere comprobación externa (revisión manual periódica del dashboard).
**Escalamiento:** N2 si no se restablece en 15 min.
**Evidencia / relacionado:** `PM-F3-03` (fallo de infraestructura) si se inyectó el fallo.

<a id="bank-infra-telemetry"></a>
### Métricas Kubernetes ausentes

- **Severidad / impacto:** warning. Los servicios pueden estar bien, pero se pierde la visibilidad de réplicas y HPA, y `bank-deployment-unavailable` no puede evaluarse.
- **Condición:** sin muestras de réplicas deseadas por 5 min, sostenido 2 min.

**Diagnóstico**

1. Confirmar la ausencia: `k8s_deployment_desired{job="bank-infra"}` sin resultados.
2. Revisar logs del Collector buscando errores `403 Forbidden` del receiver `k8s_cluster`.
3. Verificar permisos (solo get/list/watch):
   ```bash
   kubectl -n bank-usac-dev get serviceaccount,role,rolebinding
   kubectl auth can-i list deployments --as=system:serviceaccount:bank-usac-dev:<sa-collector> -n bank-usac-dev
   ```

**Mitigación:** corregir ServiceAccount, Role o RoleBinding del namespace. **No** ampliar permisos a administrador ni dar acceso a Secrets.

**Verificación de recuperación:** réplicas y HPA reaparecen con etiquetas de despliegue y la alerta se resuelve.

**Escalamiento:** N2 si hay cambios de RBAC pendientes de aprobación.
**Evidencia / relacionado:** captura del error 403 y de la corrección.

<a id="bank-deployment-unavailable"></a>
### Despliegue sin disponibilidad

- **Severidad / impacto:** critical. Un servicio no tiene ninguna réplica Ready: la funcionalidad correspondiente está caída. La alerta ignora despliegues apagados intencionalmente (deseadas = 0).
- **Condición:** disponibles = 0 y deseadas > 0 durante 3 min.
- **Nota:** con 2 réplicas, la caída de un solo pod **no** dispara esta alerta.

**Diagnóstico**

1. Identificar el despliegue: `k8s_deployment_available{job="bank-infra"} == 0`.
2. Estado de pods y eventos:
   ```bash
   kubectl -n bank-usac-dev get pods -l app=<deploy>
   kubectl -n bank-usac-dev describe pod <pod>
   kubectl -n bank-usac-dev get events --sort-by=.lastTimestamp | tail -20
   kubectl -n bank-usac-dev logs <pod> --previous
   ```
3. Causas frecuentes: `ImagePullBackOff`, `CrashLoopBackOff`, probes fallando, falta de recursos, Secret/ConfigMap ausente, DB (Cloud SQL) o RabbitMQ inaccesibles.

**Mitigación:** aplicar la corrección (imagen, configuración, recursos) o un rollback autorizado (`kubectl -n bank-usac-dev rollout undo deploy/<deploy>`). Si la causa es DB o broker, restaurar primero esa dependencia.

**Verificación de recuperación:** réplicas Ready (`kubectl -n bank-usac-dev get deploy <deploy>`), alerta en Normal y una operación de **lectura** exitosa. No generar transferencias como prueba automática.

**Escalamiento:** N2 inmediato por ser crítica; N3 si el servicio afectado es `transaction-service` o `payment-service` y hay operaciones en vuelo.
**Evidencia / relacionado:** `PM-F3-03` (infraestructura); registrar tiempo de indisponibilidad para el cálculo del SLO de disponibilidad Ready (objetivo 98.5%).

<a id="bank-rabbitmq-backlog"></a>
### Acumulación RabbitMQ

- **Severidad / impacto:** warning. Una cola acumula >100 mensajes `ready`: las operaciones asíncronas (Saga, notificaciones, auditoría) se retrasan. Se excluyen colas RPC temporales `amq.gen-*`.
- **Condición:** >100 mensajes ready por 5 min.
- **Límite:** mide volumen, no antigüedad ni Saga detenida.

**Diagnóstico**

1. Identificar la cola: `topk(10, rabbitmq_message_current{job="bank-infra",state="ready"})` y el panel *RabbitMQ: consumidores*.
2. Verificar que la cola tenga consumidores y que el servicio consumidor esté Ready (`kubectl -n bank-usac-dev get deploy`).
3. Revisar logs del consumidor y errores de procesamiento; comparar tasa de llegada vs. procesamiento.
4. Ver desconexiones recientes en el Management UI de RabbitMQ (acceso privado, sin exponer credenciales).

**Mitigación:** recuperar el consumidor afectado (corregir error, escalar réplicas con aprobación). **No** purgar colas ni reencolar pagos indiscriminadamente: riesgo de duplicar o perder operaciones financieras.

**Verificación de recuperación:** el backlog desciende de forma sostenida bajo 100, hay consumidores activos y no se observan operaciones duplicadas (revisar la auditoría).

**Escalamiento:** N2 si el backlog crece 15 min tras la mitigación; N3 si hay mensajes de pago afectados.
**Evidencia / relacionado:** `PM-F3-03` si el backlog se debió a caída de consumidor o broker; gráfica del backlog antes/después.

<a id="bank-rabbitmq-telemetry"></a>
### RabbitMQ sin observación

- **Severidad / impacto:** warning. No se conoce el estado de las colas; **ausencia de métricas no equivale a cola vacía (0)**. Si el broker está caído, `bank-rabbitmq-backlog` tampoco puede dispararse.
- **Condición:** sin métricas de colas durante 5 min, sostenido 2 min.

**Diagnóstico**

1. Confirmar: `absent_over_time(rabbitmq_message_current{job="bank-infra"}[5m])`.
2. Estado del broker (ajustar nombre real):
   ```bash
   kubectl -n bank-usac-dev get pods | grep -i rabbit
   kubectl -n bank-usac-dev describe pod <pod-rabbitmq>
   kubectl -n bank-usac-dev get pvc
   ```
3. Si el broker está sano, revisar la conectividad privada hacia el management API y el Secret del receiver (verificar su existencia, **nunca** imprimir contraseñas).
4. Revisar logs del Collector en busca de errores 401/timeout hacia el management API.

**Mitigación:** restaurar el pod/PVC del broker o corregir la conectividad/Secret del receiver. Tras recuperar el broker, revisar mensajes pendientes antes de reanudar consumidores.

**Verificación de recuperación:** retornan las métricas de colas y los consumidores en el panel; alerta en Normal.

**Escalamiento:** N2 si el broker no recupera en 15 min; N3 si se perdió el PVC.
**Evidencia / relacionado:** `PM-F3-03` (infraestructura) cuando se inyecte la caída de RabbitMQ.

### Brechas de detección conocidas

Las 8 reglas actuales no detectan por sí solas:

- Transferencias interbancarias `REJECTED` o `TIMEOUT` (relacionado con `PM-F3-02` y `PM-F3-04`).
- Transferencias estancadas en estado intermedio (Saga detenida con colas vacías).
- Caída de un solo pod con réplicas restantes disponibles.

Estas brechas deben documentarse en los postmortems correspondientes como acción correctiva (nueva alerta basada en las métricas de transiciones interbancarias).

## Verificación

<a id="bank-system-source"></a>
### Consumo de recursos o revisiones sin medición

- Condición: fuente resources/deployments con `bank_system_source_up=0`, scrape bank-system caído o ausente durante 5 min.
- Revisar el sidecar system-metrics del Collector, Metrics Server y el Role del namespace: solo requiere get/list de pods en metrics.k8s.io y lectura de deployments en apps. No ampliar a administrador ni leer Secrets.
- Confirmar `bank_system_source_up=1` y muestras recientes. Un fallo no se presenta como consumo CPU/memoria cero. Las anotaciones necesitan observar un cambio real de revisión; no ejecutar un despliegue financiero solo para provocar una marca.

`python infrastructure/observability/alerting.py --check` valida que los artefactos generados coincidan con su fuente. `validate.py` valida Helm, seguridad, componentes y regresiones antes del despliegue.
Cambios en reglas/dashboard se realizan en `alerting.py`, regenerando sus JSON; las modificaciones desde la UI no son la fuente de verdad.
Tras desplegar: comprobar estado y consultas de cada regla, paneles, datos de RabbitMQ/HPA y enviar una prueba del contacto al correo técnico. Confirmar recepción con el equipo; aceptación SMTP por sí sola no prueba que el mensaje llegó a la bandeja.
No inyectar fallos en el banco ni hacer transferencias reales para probar notificaciones. Las pruebas de fallos y sus postmortems pertenecen a la etapa de evidencia.

## Comprobación inicial, 20 de octubre de 2026

Las ocho reglas se evaluaron en Grafana 13.2.4 con `health=ok`, sin errores de consulta. Prometheus recibió métricas de ocho despliegues, cinco HPA y colas reales de RabbitMQ. El contacto técnico respondió `status=success` a la prueba de correo desde Grafana; falta la confirmación humana de recepción en bandeja.
La validación incluye arranque real de la imagen de Grafana, provisión de reglas/contacto y expresiones ejecutadas con promtool en 16 escenarios sintéticos, sin modificar saldos ni enviar transferencias. Las pruebas del CI usan SMTP deshabilitado y no envían correos reales.
Las métricas de dominio basadas en eventos pueden mostrar «sin datos» si no hubo actividad reciente. Esto no demuestra una falla ni certifica que todos los casos funcionales hayan sido probados.
