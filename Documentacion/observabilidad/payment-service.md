# Payment Service: simulación avanzada de fallos

## Administración en ejecución

El API Gateway protege ambas rutas con JWT y rol ADMIN:

| Método | Ruta | Propósito |
| --- | --- | --- |
| GET | `/api/administracion/pagos/simulacion` | Consultar parámetros y versión actuales |
| PUT | `/api/administracion/pagos/simulacion` | Reemplazar parámetros sin redesplegar |

Ejemplo de cuerpo para PUT:

```json
{
  "habilitada": true,
  "tasaFallosPorcentaje": 25.5,
  "latenciaMs": 250,
  "timeoutMs": 1000,
  "forzarTimeout": false
}
```

- `tasaFallosPorcentaje`: 0–100, admite decimales.
- `latenciaMs`: 0–30000, entero.
- `timeoutMs`: 1–30000, entero.
- `forzarTimeout`: provoca un timeout al alcanzar `timeoutMs`, aun con latencia menor.
- `habilitada=false`: conserva la selección EXITO/FALLO/TIMEOUT por pago de la Fase 2, sin demora adicional. Es el estado inicial para evitar fallos inesperados al desplegar.

Cuando se habilita, la configuración administrativa tiene prioridad sobre `resultadoSimulado` enviado por el cliente. Si se fuerza el timeout, o la latencia es mayor o igual al timeout, se obtiene TIMEOUT y se espera el plazo de timeout. De otro modo se espera la latencia y se aplica la probabilidad de fallo. Los pagos INTERNOS nunca pasan por el proveedor simulado.

El administrador se obtiene del JWT; no se acepta su identidad desde el cuerpo HTTP. Las rutas rechazan solicitudes sin autenticación (401), roles CLIENTE/TELLER (403), JSON inválido (400) y valores fuera de rango (422). El Gateway espera exclusivamente `pago.simulacion.respondida`, por lo que eventos de negocio con la misma correlación no sustituyen la respuesta administrativa. Ante 504, consultar la configuración antes de reintentar, porque la escritura puede haber ocurrido.

La configuración vive en PostgreSQL y se comparte entre réplicas y reinicios. El cambio, su versión, el evento `pago.simulacion.configuracion.actualizada` y la respuesta de Outbox son atómicos. Un mensaje administrativo duplicado no cambia de nuevo la versión. La ruta HTTP de escritura no se publica directamente en Payment Service.

## Simulación estable y compensaciones

Después del débito confirmado, se guarda una única simulación por pago: configuración exacta, versión, resultado, espera e inicio. La selección usa una muestra uniforme derivada del UUID del pago; el resultado es determinista para reintentos y también queda persistido. Una tasa del 25 % describe una probabilidad, no garantiza exactamente uno de cada cuatro pagos.

Las esperas ocurren fuera de las transacciones de negocio y respetan la cancelación. El plazo restante se calcula con el reloj de PostgreSQL para evitar diferencias de reloj entre el servicio y su base. Otro administrador puede cambiar los parámetros mientras un pago espera; el pago en curso conserva su decisión y versión, y los pagos externos que aún no hayan iniciado la simulación toman la configuración nueva.

Un fallo o timeout crea, en la misma transacción que el estado COMPENSANDO y la finalización de la simulación:

1. Un comando `cuenta.compensacion.solicitada`, con la misma cuenta, operación y monto debitados.
2. Un evento `pago.proveedor.fallido` o `pago.timeout`, con `simulado=true`, código y versión.
3. El evento `pago.compensando`.

El pago termina RECHAZADO cuando Account Service confirma la compensación. El evento `cuenta.compensacion.rechazada` lleva el pago a COMPENSACION_FALLIDA y emite `pago.compensacion.fallida`, con notificación de error para revisión. No se presenta como fondos devueltos. El Gateway expone estos estados en su seguimiento de operación.

Los controles de estado evitan que mensajes tardíos, duplicados con IDs nuevos o reintentos de validación repitan el débito, cambien un resultado final o creen varias compensaciones. El consumidor reintenta los conflictos transaccionales; los efectos financieros siguen siendo responsabilidad del Account Service y su idempotencia por operación.

## Trazas y logs

`pagos.trace_context` conserva el contexto W3C de creación. Las validaciones, el proveedor simulado y las compensaciones recuperan esa traza aunque llegue un evento sin cabeceras. El span CLIENT `pago.proveedor.simulado` registra resultado, versión, demora y marca de simulación; los fallos tienen estado de error. Los spans `pago.estado` incluyen COMPENSANDO, RECHAZADO, COMPENSATED y COMPENSACION_FALLIDA cuando corresponde.

Los logs JSON de error simulado incluyen `errorCode`, `paymentId`, `simulationVersion`, `outcome`, `correlationId`, `traceId` y `spanId`. No incluyen beneficiario, concepto, documentos, correo ni JWT. Los eventos controlados pasan al servicio de auditoría mediante su suscripción existente `pago.#`.

## Métricas para Grafana

| Serie Prometheus | Tipo | Etiquetas de negocio |
| --- | --- | --- |
| `bank_payment_external_results_total` | Counter | `outcome=success|failure|timeout` |
| `bank_payment_external_duration_seconds_bucket` (con `_sum`, `_count`) | Histogram | `outcome=success|failure|timeout` |

Se agregan servicio y entorno; nunca se usan IDs como etiquetas. Los resultados se contabilizan después del commit que confirma la decisión, una sola vez por pago. El histograma mide el intervalo lógico desde el inicio persistido de la llamada hasta su confirmación, incluida la espera; si hubo una caída y reanudación, también incluye ese intervalo. Los pagos internos, llamadas canceladas antes de confirmar y rollbacks no cuentan. Como la telemetría es en memoria, una caída entre commit y registro puede perder una muestra; no sustituye los registros contables persistidos.

La vía existente es OTLP → Collector → Prometheus. `GET /metrics` en Payment Service permite inspección o scrape interno alternativo. No ingerir las mismas métricas por ambas vías, para no duplicarlas. El dashboard se aprovisiona por separado con Helm según la sección descrita al final.

Fallos externos por segundo, incluyendo timeouts:

```promql
sum(rate(bank_payment_external_results_total{service="payment-service",outcome=~"failure|timeout"}[5m]))
```

Porcentaje de fallo externo:

```promql
100 * (
  sum(rate(bank_payment_external_results_total{service="payment-service",outcome=~"failure|timeout"}[5m]))
  or vector(0)
) / clamp_min(sum(rate(bank_payment_external_results_total{service="payment-service"}[5m])), 1e-9)
```

Latencia p95 en segundos:

```promql
histogram_quantile(0.95,
  sum by (le) (rate(bank_payment_external_duration_seconds_bucket{service="payment-service"}[5m]))
)
```

Filtrar también por entorno si la fuente contiene varios. Agregar buckets antes de calcular percentiles; no calcular p95 a partir de promedios.

## Migración y pruebas

`000005_simulacion_avanzada` agrega el contexto de traza de pagos, la configuración singleton y las simulaciones por pago; amplía el estado de compensación fallida. Es aditiva y reejecutable, y el paso de esquemas del CI/CD la aplica antes del despliegue. No modifica saldos ni crea usuarios.

Pruebas ejecutadas con PostgreSQL local temporal: éxito lento, fallo al 100 %, timeout forzado y por latencia, duplicados con el mismo ID y con IDs nuevos, concurrencia entre repositorios, cambios administrativos durante una espera, cancelación, reanudación con configuración nueva, pago interno excluido, rechazo de configuración, fallo de compensación y rollback con reintento. Se verificaron comandos de compensación únicos, cuenta/monto/operación correctos, finalización por evento de Account Service, la traza de Outbox y las métricas.

Las pruebas HTTP verifican las rutas reales del Gateway, autenticación, roles, identidad del administrador, parámetros inválidos y errores de respuesta/timeout. Otras pruebas verifican `/metrics`, el histograma, las etiquetas y los logs JSON de error. El workflow de Payment Service incluye la prueba de PostgreSQL desechable con `go test -race`; en Windows se ejecutaron las pruebas normales y `go vet`, sin afirmar una ejecución local con detección de carreras.

El servidor 201.216.128.230 no respondió por SSH durante esta tarea. Por eso no se probó un flujo desplegado con RabbitMQ ni se inspeccionaron trazas en el Grafana remoto. Las verificaciones descritas son locales y de contratos/eventos; la publicación y el despliegue quedan a cargo del CI/CD al subir los cambios por el flujo de ramas correspondiente.

## Grafana: 04/Payment_service

En `Bank USAC-Prueba1`, la sección `04/Payment_service` contiene:

- **Tasa de fallos externos (%)**: failure + timeout sobre todos los resultados confirmados de `bank_payment_external_results_total`, con rate de 5 min y filtros service/environment. No confundir con el porcentaje configurado de simulación: es el resultado observado. Una serie de fallos ausente solo se reemplaza por cero cuando existe el total; si no hay resultados, no se muestra 0% inventado.
- **Latencia del proveedor externo simulado · p95**: `histogram_quantile(0.95, sum by (le) (rate(bank_payment_external_duration_seconds_bucket{service="payment-service",environment="$environment"}[5m])))`. Agrega réplicas/resultados antes del percentil. No usa promedios ni la duración de toda la Saga.
- **Logs de errores simulados**: Loki filtra message `simulated external error` y códigos `SIMULATED_EXTERNAL_FALLO` / `SIMULATED_EXTERNAL_TIMEOUT`. Son WARNING por ser fallos controlados, no necesariamente ERROR. Se conserva JSON original para detalles y enlace a Tempo. No se guarda ningún JWT en el dashboard.

Los paneles respetan el entorno y rango temporal; el UUID de Account Service no aplica. Los fallos/timeouts solo aparecen al confirmar intentos externos; pagos internos y rollbacks no cuentan. La latencia incluye la espera y una eventual reanudación del intervalo lógico persistido. Los logs se limitan a 100 líneas por consulta y 48 h de retención. No sustituir ausencia de datos por cero.

Validación manual pendiente tras despliegue: configurar la simulación mediante el endpoint ADMIN existente, ejecutar pagos externos de éxito lento, fallo y timeout, esperar exportación/recolección y comprobar tasa, p95 y logs. Las Sagas con compensación se inspeccionan desde sus trazas; los paneles no prueban por sí solos que se restauraron fondos. El cambio reutiliza instrumentación existente: JSON de dashboard y documentación, sin nuevas migraciones ni cambios de backend.

### Diagnóstico: primer fallo, niveles y Sagas de pago

Payment Service inicializa `bank_payment_external_results` con cero para success/failure/timeout al configurar el MeterProvider. Evita que el primer resultado cree una serie ya incrementada sin línea base previa. Prometheus todavía necesita recolectar la línea base antes del resultado y al menos dos muestras para `rate`; no recupera incrementos anteriores a la primera muestra. Un 0% en la captura por sí solo no permite distinguir este caso de fallos fuera de la ventana o problemas de ingesta. Comprobar las series crudas por outcome y rango.

Los nuevos errores simulados se emiten con ERROR; también los estados confirmados COMPENSANDO, RECHAZADO y COMPENSACION_FALLIDA. Los logs anteriores WARNING permanecen sin cambios y la consulta no los excluye. El panel incluye ambos mensajes de negocio: simulated external error y payment state committed (estados fallidos). Varias líneas del mismo pago no representan varios fallos externos: el contador externo se incrementa una vez después del commit correspondiente, mientras auditoría registra cada evento del ciclo.

Se añadió **Trazas de pagos · Saga y compensaciones** en 04/Payment_service. Filtra spans pago.estado/pago.proveedor.simulado y agrupa Trace ID. Los pagos no generan por sí mismos spans transferencia.inicio/estado de Transaction Service: la tabla 03 solo muestra transferencias. La traza completa del pago incluye Account Service y la compensación al abrir su Trace ID. Para validar Transaction Service ejecutar una transferencia, no únicamente un pago.

Requiere desplegar la imagen de Payment Service y el dashboard. JSON y pruebas unitarias locales verificadas; pendiente comprobar las series y las trazas del entorno remoto. Tras el despliegue esperar la primera recolección de ceros antes de repetir el fallo.

### Traza localizable desde el inicio del pago

La tabla de pagos filtra por `span.payment.id =~ ".+"` y agrupa por Trace ID. La versión actual ya tiene `pago.inicio` con payment.id y correlación en Iniciar, antes de guardar el contexto W3C en SQL/outbox. El filtro anterior solo buscaba pago.estado/proveedor.simulado y excluía ese inicio. El nuevo filtro incluye cualquier etapa con payment.id, sin cambiar IDs ni etiquetas de métricas; no requiere duplicar spans ni otra modificación de backend.

La consulta de pago contra Tempo fue válida pero no encontró trazas en los datos existentes. El contexto no muestreado (-00) investigado anteriormente no se corrige retroactivamente: desplegar la corrección de muestreo de Gateway y la instrumentación de Payment Service, luego generar un pago nuevo. La consulta por identificador no reconstruye spans nunca exportados. La tabla conserva el acceso a Explore y no añade un visor vacío al dashboard.
