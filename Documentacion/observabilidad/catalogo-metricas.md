# Catálogo de métricas — Fase 3

Estado: especificación de nombres exportados previstos, no métricas existentes. Referencias: [contrato](contrato-telemetria.md), [SLI](../sre/sli.md), [objetivos vigentes](../sre/slo.md).

## Reglas comunes

Duraciones en segundos; contadores acumulativos y histogramas. Etiquetas base: `service`, `environment`; versión en un indicador informativo, no en cada serie. Etiquetas adicionales solo de listas acotadas del código. Prohibidos IDs de usuario, cuenta, mensaje, correlación, traza, URLs completas, error libre y fecha como etiquetas. Rutas HTTP siempre plantilla; routing keys y colas solo del catálogo configurado, agrupando colas dinámicas por rol.

## Instrumentos

| Nombre previsto | Tipo | Etiquetas adicionales | Significado |
|---|---|---|---|
| bank_http_requests_total | Counter | method, route, status_class | Respuestas del Gateway, una por intento HTTP |
| bank_http_duration_seconds | Histogram | method, route | Latencia HTTP p50/p95/p99 |
| bank_operations_started_total | Counter | operation | Operaciones lógicas admitidas, no reentregas |
| bank_operations_resolved_total | Counter | operation, outcome | Resultado durable: success, business_rejected, technical_failed |
| bank_operation_duration_seconds | Histogram | operation, outcome | Desde admisión durable hasta estado terminal consultable |
| bank_operations_pending | Gauge | operation | Operaciones sin resolución |
| bank_operation_oldest_pending_seconds | Gauge | operation | Edad de la operación pendiente más antigua |
| bank_slo_evaluations_total | Counter | objective, result | Evaluación durable única por unidad madura; result=good o bad |
| bank_messages_total | Counter | event_type, queue_role, action | action=published, processed, retried, dead_lettered; intentos de transporte |
| bank_message_processing_seconds | Histogram | event_type, queue_role | Duración del procesamiento de cada intento |
| bank_outbox_pending | Gauge | — | Registros todavía no publicados |
| bank_outbox_oldest_pending_seconds | Gauge | — | Edad del Outbox pendiente más antiguo |
| bank_amqp_ready | Gauge | — | Conexión, canales y consumidor disponibles: 1/0 |
| bank_kyc_decisions_total | Counter | outcome | verified o rejected |
| bank_kyc_duration_seconds | Histogram | outcome | Creación a decisión KYC; separado de latencia de validación automática |
| bank_account_movements_total | Counter | account_type, direction | Movimientos confirmados; direction=debit o credit |
| bank_account_anomalies_total | Counter | account_type | Anomalías detectadas; umbral configurable N en M minutos |
| bank_compensations_total | Counter | outcome | started, completed o failed |
| bank_notifications_total | Counter | outcome | generated, sent o failed |
| bank_invariant_violations_total | Counter | invariant | Doble efecto, saldo inválido o reversión inconsistente |
| bank_trace_context_missing_total | Counter | transport | Operaciones que pierden contexto esperado; http o amqp |

Los gauges de pendientes se reconstruyen desde almacenamiento durable al reiniciar. Los contadores financieros y las evaluaciones SLO requieren deduplicación persistente: no basta incrementar memoria antes del commit. Una evaluación utiliza una clave única por objetivo y operación/mensaje; nunca exponer esa clave como etiqueta. Reintentos se cuentan como intentos de transporte, no nuevas operaciones.

Buckets propuestos de histogramas de duración: `0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5, 8, 10, 15, 20, 25, 30, 60, 120, 180, 300, 480, 900, 3600, 86400, +Inf`. Verificar coste y ajustar por instrumento sin eliminar umbrales SLO. KYC humano requiere buckets mayores si las decisiones exceden un día. Agregar buckets antes de calcular percentiles globales; no promediar percentiles de Pods.

## Medición sin ocultar operaciones atascadas

Los histogramas terminales no ven operaciones que nunca terminan. Un evaluador durable debe contar también vencimientos y pendientes fuera de plazo en `bank_slo_evaluations_total`, con una sola evaluación por unidad. Para ventanas por cohorte, evaluar solo operaciones cuyo plazo ya venció; las aún inmaduras se muestran aparte. Definir y versionar el plazo de cada objetivo antes de activarlo; hasta entonces el SLI correspondiente no está medible.

Entrega a DLQ satisface únicamente «entrega controlada», no éxito del negocio. Muestrear trazas no reduce el denominador de métricas. Idempotencia y ausencia de pérdida financiera se verifican además mediante consultas y pruebas de invariantes sobre registros persistidos.

## Broker e infraestructura

Recolectar mensajes ready y unacked por cola, consumidores activos, antigüedad del mensaje y tasas de publicación/ack. Ready + unacked es backlog, no una medida de tiempo de atraso. Completar con CPU, memoria, reinicios, réplicas disponibles/deseadas y HPA mediante exportadores de infraestructura; no inferir disponibilidad de usuario únicamente de Pod Ready. El SLI-02 se mantiene como señal diagnóstica histórica.

## Disponibilidad y presupuesto

Para Gateway: `1 - respuestas_5xx / respuestas_elegibles`, por entorno en 30 días móviles; salud y métricas quedan fuera. Incluir 4xx, mostrar errores de autorización separadamente. Objetivo **0.995**, presupuesto **0.005**. Sonda externa detecta caídas sin respuestas, incluido Cloud Run; una serie ausente es «sin datos» y requiere alarma.

Burn rate = proporción de errores de la ventana / 0.005. Se cuenta cada respuesta real; no sumar reintentos como operaciones de negocio. Los objetivos al 100 % (idempotencia y trazabilidad) tienen presupuesto cero: alertar por cualquier violación, no dividir entre cero.

## Alertas y runbooks por implementar

| Alerta prevista | Condición inicial de diseño | Acción del runbook |
|---|---|---|
| Consumo rápido de presupuesto | Burn rate >14.4 tanto en 1 h como en 5 min | Revisar despliegue, rutas 5xx, dependencias; mitigar o rollback y validar |
| Consumo sostenido de presupuesto | Burn rate >6 tanto en 6 h como en 30 min | Revisar tendencia, trazas y saturación; detener promociones si se agota presupuesto |
| Cola/Outbox atascado | Pendiente más antiguo >30 s durante 5 min en flujos internos | Revisar consumidor, binding, conexión y DB; recuperar sin purgar mensajes |
| Invariante financiera | Cualquier incremento de violaciones | Detener operación afectada, preservar evidencia, conciliar y reparar con aprobación |
| Servicio no observable/disponible | Sonda falla o serie esperada ausente por 2 min | Distinguir caída del servicio y del recolector; comprobar rutas y exportación |

Las alertas de burn rate necesitan muestras suficientes y señal separada para bajo tráfico; no declarar saludable un denominador cero. Los umbrales son diseño inicial, deben validarse con carga y capacidad reales. Falta convertirlos en reglas versionadas, runbooks completos y configurar/probar un canal de notificación; esta tabla no constituye alertas activas.

## Particularidades interbancarias

Separar operaciones internas de interbancarias. El plazo contractual de 180 s y la reserva hasta resolución o expiración más 300 s no son automáticamente un SLO de latencia. TIMEOUT no significa fracaso definitivo ni autoriza liberar fondos anticipadamente. Medir resolución, reservas pendientes y callbacks tardíos sin alterar esas reglas. El SLO interbancario debe acordarse y documentarse antes de anunciar cumplimiento.
