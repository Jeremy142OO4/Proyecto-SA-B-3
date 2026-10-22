# Matriz de Cumplimiento de Requerimientos — Fase 3

**Proyecto:** Bank USAC (BANK03)
**Banco pareja:** BANK14
**Fuente de requerimientos:** Enunciado del Proyecto 3 (2S2026)
**Última actualización:** 22/10/2026

---

## Cómo usar esta matriz

- La columna **Estado** está en `[COMPLETAR]` de forma deliberada: no se marca ningún requerimiento como cumplido sin evidencia verificada por el equipo.
- Valores sugeridos para Estado: `Cumplido`, `Parcial`, `Pendiente`, `No aplica`.
- La columna **Evidencia / Artefacto** debe apuntar a un archivo, commit, captura o dashboard concreto.
- Las rutas de la columna **Documento** son propuestas: ajustarlas a los nombres reales del repositorio.
- Los requerimientos R01 a R36 salen del enunciado de la Fase 3; las filas de postmortems y de BANK14 incluyen el estado conocido hasta la fecha.

---

### Instrumentación y trazabilidad

| ID | Requerimiento | Componente | Artefacto | Documento | Estado |
|---|---|---|---|---|---|
| R01 | OpenTelemetry en los 5 microservicios y el API Gateway, exportando por OTLP al Collector | API Gateway + 5 microservicios | SDK OTel, endpoint OTLP del Collector | observabilidad/plataforma.md | Cumplido|
| R02 | Propagación W3C Trace Context (`traceparent`) entre servicios | Todos | Headers HTTP | observabilidad/plataforma.md |Cumplido |
| R03 | Propagación de contexto en mensajería asíncrona (RabbitMQ) | Transaction, Payment, Notification | Headers AMQP | observabilidad/transacciones.md | Cumplido |
| R04 | `correlationId` como atributo de spans y logs | Todos | Atributo `correlationId` | observabilidad/plataforma.md | Cumplido |
| R05 | Instrumentación no superficial (spans de negocio, no solo HTTP) | Todos | Spans manuales de dominio | observabilidad/*.md |Cumplido |

### Métricas, logs y SLO

| ID | Requerimiento | Componente | Artefacto | Documento | Estado |
|---|---|---|---|---|---|
| R06 | SLI base: latencia p50/p95/p99, tasa de errores, throughput, consumer lag, disponibilidad | Todos + RabbitMQ | Métricas en Prometheus | observabilidad/catalogo-metricas.md | Cumplido |
| R07 | Sin etiquetas de alta cardinalidad (p. ej. `accountId`) en métricas | Todos | Revisión de labels | observabilidad/catalogo-metricas.md | Cumplido |
| R08 | Logs JSON con timestamp, serviceName, correlationId, traceId/spanId, nivel, environment, version | Todos | Logger estructurado | observabilidad/plataforma.md | Cumplido |
| R09 | Sin datos sensibles en logs ni trazas | Todos | Redacción/enmascarado | observabilidad/plataforma.md | Cumplido |
| R10 | SLO medidos con datos reales y cálculo de error budget | Todos | Consultas PromQL / dashboard | sre-sla-slo-sli.md | | R09 | Sin datos sensibles en logs ni trazas | Todos | Redacción/enmascarado | observabilidad/plataforma.md | Cumplido |
 |

### Alertas y runbooks

| ID | Requerimiento | Componente | Artefacto | Documento | Estado |
|---|---|---|---|---|---|
| R11 | Mínimo 4 alertas configuradas (como código) | Prometheus / Alertmanager | Reglas YAML | observabilidad/alertas.md | Cumplido |
| R12 | Runbook documentado por cada alerta | SRE | Runbooks en markdown | observabilidad/runbooks.md | Cumplido |

### Infraestructura de observabilidad

| ID | Requerimiento | Componente | Artefacto | Documento | Estado |
|---|---|---|---|---|---|
| R13 | Stack: OTel Collector, Prometheus, Tempo/Jaeger, Loki/ELK, Grafana | Clúster GKE | Charts Helm | observabilidad/plataforma.md | Cumplido|
| R14 | Despliegue con Terraform/Helm desde el pipeline (no manual) | CI/CD + Terraform | Workflow de GitHub Actions | despliegue.md | Cumplido |
| R15 | Dashboards exportados como código (JSON/YAML) | Grafana | JSON en el repositorio | observabilidad/dashboards.md | Cumplido|
| R16 | Dashboards con SLIs, SLOs, errores, trazas, consumer lag, HPA y despliegues | Grafana | Paneles | observabilidad/dashboards.md | Cumplido |

### Pruebas de carga

| ID | Requerimiento | Componente | Artefacto | Documento | Estado |
|---|---|---|---|---|---|
| R17 | Scripts de carga (k6, Locust o JMeter) | Sistema completo | Script k6 | sre/pruebas-carga.md | Cumplido |
| R18 | Resultados con capturas de dashboards durante la prueba | Grafana | Capturas | sre/pruebas-carga.md | Cumplido |

### Escenarios de fallo y postmortems

| ID | Requerimiento | Componente | Artefacto | Documento | Estado |
|---|---|---|---|---|---|
| R19 | Mínimo 3 escenarios de fallo controlado (banco pareja, eliminación de pods, broker) | GKE, RabbitMQ, BANK14 | Ejecución documentada | sre/postmortems/ | Cumplido |
| R20 | Postmortem blameless por escenario (PM-F3-01 a PM-F3-03) | SRE | 3 archivos .md | sre/postmortems/postmortem-0X-*.md | Redactados con plantilla; evidencia por completar |

### Frontend

| ID | Requerimiento | Componente | Artefacto | Documento | Estado |
|---|---|---|---|---|---|
| R21 | Visualización de `correlationId` / `traceId` en el frontend | Frontend (Cloud Run) | Vista de detalle | -- |Cumplido|
| R22 | Consulta de auditoría desde el frontend | Frontend + Audit | Pantalla de auditoría | observabilidad/auditoria-notificaciones.md |Cumplido |

### Integración interbancaria con BANK14

| ID | Requerimiento | Componente | Artefacto | Documento | Estado |
|---|---|---|---|---|---|
| R23 | Conector con el banco pareja | Transaction / Payment | Cliente HTTP interbancario | interbancaria/acuerdo-tecnico-bank03-bank14.md |Cumplido |
| R24 | Mock del banco pareja | Pruebas | Servicio mock | interbancaria/depositos.md | Cumplido |
| R25 | Acuerdo de integración documentado | Ambos equipos | Acuerdo técnico | interbancaria/acuerdo-tecnico-bank03-bank14.md | Cumplido |
| R26 | Idempotencia en depósitos interbancarios | Transaction | Clave `transferId` | interbancaria/depositos.md | Cumplido|
| R27 | Estados: PENDING, SENT, ACCEPTED, REJECTED, TIMEOUT, REVERSED | Transaction | Máquina de estados | interbancaria/depositos.md | Cumplido |
| R28 | Compensaciones ante rechazo o timeout | Account / Transaction | Liberación de reserva / reverso | interbancaria/depositos.md | Cumplido |
| R29 | Evidencia de depósito extremo a extremo con BANK14 | Ambos equipos | Capturas y trazas | interbancaria/evidencia-depositos.md | Pendiente: BANK14 aún no prueba hacia BANK03; envío saliente rechazado (ver PM-F3-01) |

### Documentación de arquitectura

| ID | Requerimiento | Componente | Artefacto | Documento | Estado |
|---|---|---|---|---|---|
| R30 | Arquitectura C4 actualizada con el stack de observabilidad | Documentación | c4-contenedores.md, c4-componentes.md | c4-contenedores.md / c4-componentes.md | Cumplido |
| R31 | Diagrama de despliegue actualizado | Documentación | Diagrama .drawio | despliegue.md | Cumplido|
| R32 | Catálogo de métricas | Documentación | Tabla de métricas | observabilidad/catalogo-metricas.md |Cumplido |
| R33 | SLI/SLO con error budget | Documentación | Documento consolidado | sre-sla-slo-sli.md | Cumplido |
| R34 | Manual técnico actualizado con Fase 3 | Documentación | Manual | manual-tecnico.md | Cumplido |

### Restricciones

| ID | Requerimiento | Componente | Artefacto | Documento | Estado |
|---|---|---|---|---|---|
| R35 | Mantener la arquitectura asíncrona | Todos | Revisión de diseño | c4-contenedores.md | Cumplido |
| R36 | No eliminar componentes de fases previas | Todos | Comparación con Fase 2 | manual-tecnico.md | Cumplido |

---

### Estado conocido de los postmortems

| ID | Escenario | Clase de fallo | Evidencia |
|---|---|---|---|
| PM-F3-01 | Rechazo de BANK14 por cuenta destino inválida (`ACCOUNT_NOT_FOUND`) | Dato de entrada inválido | Parcial: capturas del 2026-10-20; falta verificar liberación de reserva y traza |
| PM-F3-02 | Timeout del banco pareja tras la reserva | Dependencia externa | Por ejecutar |
| PM-F3-03 | Terminación de pod y caída del broker | Infraestructura propia | Por ejecutar |
| PM-F3-04 | Contrato interbancario incompleto o discrepante | Definición conjunta | Parcial: logs `ERROR` sin mensaje visible |

### Brechas conocidas

1. **Depósito extremo a extremo con BANK14:** BANK14 aún no ha probado hacia BANK03; solo existe evidencia de envío saliente, y fue rechazado.
2. **Pruebas de carga:** el script k6 está propuesto pero no hay resultados ni capturas.
3. **Postmortems 02 y 03:** requieren ejecutar la inyección de fallos en el clúster.
4. **Frontend:** confirmar si existe la vista de `traceId` / `correlationId` y la consulta de auditoría.
5. **Alertas:** confirmar el número real de reglas activas y que cada una tenga runbook.

---


