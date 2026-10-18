# Matriz de medición y cierre del diseño

Esta matriz completa el punto 2: definición de telemetría y revisión de SLO. No certifica instrumentación, pruebas de carga ni cumplimiento. Los objetivos oficiales están en [SLO](../sre/slo.md); el [contrato JSON](telemetria.json) contiene las constantes comunes para futuras implementaciones. Si cambian, actualizar también los documentos y pruebas que los consuman.

## Fuentes y denominadores

| SLI | Fuente prevista | Regla de evaluación |
|---|---|---|
| 01 Disponibilidad | bank_http_requests_total + sonda externa | No 5xx / respuestas elegibles; 99.5 %, ventana móvil 30 días, entorno separado. La sonda muestra caídas sin respuestas como indicador independiente. |
| 02 Microservicios | Estado Ready y réplicas disponibles | Tiempo con al menos una réplica Ready / tiempo observado por servicio; diagnóstico, no disponibilidad funcional. No sumar tiempos de Pods como si fueran servicios independientes. |
| 03 Latencia HTTP | bank_http_duration_seconds | p50/p95/p99 agregados por buckets; separar ruta y entorno. Objetivo p95 0.5 s para aceptación/consulta. |
| 04 Latencia asíncrona | bank_operation_duration_seconds + pendientes + evaluador durable | Por operación lógica, desde admisión hasta terminal consultable; umbrales por flujo de SLO. Mostrar incumplimientos de plazo aunque nunca llegue terminal. |
| 05 Éxito técnico | Evaluaciones durables de resultado | Resultado coherente / cohorte admitida cuyo plazo venció; rechazo de negocio correcto es resultado coherente. Distinguir fallo del proveedor, compensación correcta y éxito financiero. |
| 06 Errores técnicos | Evaluaciones durables de resultado | Fallos técnicos / misma cohorte del SLI-05. No sumar timeout y compensación de la misma operación como dos operaciones fallidas. |
| 07 Idempotencia | Tabla de deduplicación + movimientos persistidos | Duplicados sin efecto adicional / duplicados recibidos; sin duplicados, sin datos. Cualquier doble efecto es incidente, aunque la proporción agregada sea alta. |
| 08 Entrega controlada | Registro durable de mensajes confirmados y consumidores/DLQ | Una unidad por mensaje y consumidor esperado. Fan-out no es duplicidad: cada destino esperado debe resolverse. Procesado o DLQ dentro del plazo / unidades maduras esperadas. |
| 09 Reintentos | bank_messages_total | Intentos adicionales / intentos consumidos; publicar también mensajes lógicos afectados para evitar interpretar intentos como usuarios u operaciones. |
| 10 DLQ | Registro de destino final + broker | Unidades únicas en DLQ / unidades de entrega maduras esperadas; separar servicio y tipo. La DLQ no equivale a éxito de negocio. |
| 11 Trazabilidad | Validador de evidencia de operación + contexto persistido | Operaciones con etapas esperadas, correlación y propagación completas / operaciones maduras evaluadas; no usar solo las trazas que llegaron al backend de trazas como denominador. |
| 12 Compensaciones | Estado durable de Saga | Completadas / iniciadas maduras, con pendientes vencidas como incumplimiento. Fallo de compensación siempre incidente. |
| 13 Notificaciones | Estado durable del envío | SENT / generadas maduras; reintentar el envío no crea una notificación nueva. Separar fallo de correo del resultado financiero. |

Los contadores Prometheus de intentos son útiles para tasas operativas, pero no bastan para reconstruir estas cohortes. El evaluador debe persistir unidades, plazo, resultado y deduplicación antes de exportar agregados. Reinicios, pérdida de scrape o publicación repetida no deben cambiar el resultado histórico. El diseño físico y migraciones pertenecen a la implementación.

## Plazos de evaluación

Para objetivos por flujo se usa el tiempo de la tabla SLO: registro 5 s, cuenta 8 s, saldo 5 s, transferencia normal 10 s, compensada 20 s, pago 10 s, auditoría 5 s y notificación 15 s. El percentil y la confiabilidad se muestran separados: un resultado tardío puede ser técnicamente coherente y, aun así, incumplir latencia.

Para SLI-08/10 se propone un plazo inicial de entrega controlada de **60 s** en flujos internos. Es un parámetro de diseño que debe contrastarse con la política real de reintentos antes de activar esos SLO; no modifica timeouts ni autoriza descartar mensajes. La evaluación de trazabilidad interna espera ese mismo plazo para reunir evidencia. Fuera del plazo, conservar evidencia tardía sin reescribir silenciosamente el resultado de una evaluación ya cerrada.

KYC humano se mide desde creación hasta decisión, separado de las validaciones automáticas de KYC. Se reporta su p95 y número de decisiones; aún no hay un objetivo temporal humano acordado. Las operaciones interbancarias necesitan objetivos propios y no heredan los 10 s internos.

## Presupuesto y criterios operativos

- Disponibilidad oficial: 99.5 %, presupuesto de solicitudes fallidas 0.5 %. Una ventana de pruebas no prueba un mes completo.
- No mezclar disponibilidad del Gateway, frontend, sondeo externo, éxito financiero y estado Ready en un único porcentaje.
- Las pruebas provocadas se identifican y se presentan con sus fallos; no excluir retrospectivamente tráfico real ni mantenimiento.
- Una muestra ausente es «sin datos». Si se esperaba tráfico o scrape, alertar por ausencia.
- Mantener los demás objetivos de fase 1 salvo acuerdo explícito: el cambio de disponibilidad no los reemplaza por 99.5 %.

## Secuencia de implementación posterior

1. Biblioteca/convenios comunes, logger JSON y SDK en los seis componentes, con versiones fijadas.
2. Propagación HTTP/AMQP y migraciones Outbox/Saga compatibles, probadas con reinicios.
3. Instrumentación por dominio y evaluador durable, sin efectos financieros duplicados.
4. Collector y backends mediante infraestructura versionada; dashboards, reglas y canal de alertas.
5. Pruebas de carga/fallos, verificación de cardinalidad y datos sensibles, evidencia de trazas y presupuestos reales.

El cierre del punto 2 significa que está definido **qué medir y cómo correlacionarlo**. No significa que estén instalados Collector, Grafana o los SDK, ni que el 99.5 % se haya alcanzado.
