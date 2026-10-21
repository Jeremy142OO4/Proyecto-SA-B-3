# Transaction Service: rendimiento y estado

## Contrato implementado

- `bank_transfer_flow_duration_seconds_bucket`, `_sum`, `_count`: histograma real en segundos desde `fecha_creacion` hasta el resultado terminal confirmado. Incluye validaciones, esperas del broker, débito, crédito y compensación. Usa los buckets del contrato de telemetría, desde 5 ms hasta 24 h y +Inf.
- `bank_transfer_results_total{outcome="success|failure"}`: resultados terminales. COMPLETADA es éxito; RECHAZADA, COMPENSADA y COMPENSACION_FALLIDA son fallos del flujo original. Una compensación correcta devuelve dinero, pero no convierte la transferencia original en exitosa.
- Las etiquetas se limitan a servicio, entorno y resultado. No se usan IDs de clientes, cuentas, transacciones o trazas como etiquetas.
- Las métricas se registran después del commit. Los mensajes duplicados y los eventos incompatibles con el estado actual no vuelven a contarse. Los errores SQL y los rollbacks no generan resultados de negocio.

Los contadores e histogramas son telemetría en memoria, exportada por OTLP y disponible también por scrape. Prometheus trata los reinicios como resets; no se reimportan operaciones históricas. Existe una pequeña ventana entre el commit y el registro en memoria en la que una caída del proceso puede perder una muestra. No se deben interpretar como un libro contable ni como contadores históricos exactos. Las transferencias atascadas siguen PENDING y no entran en el denominador de resultados terminales.

## Estados y traza distribuida

Cada transición confirmada crea el span `transferencia.estado` y un evento PENDING, APPROVED o FAILED. Sus atributos incluyen `transaction.id`, `transaction.state` (estado operativo original), `transaction.phase2.state` y correlationId. Los pasos VALIDANDO_KYC, VALIDANDO_CUENTAS, PENDIENTE, PROCESANDO y COMPENSANDO son PENDING; COMPLETADA es APPROVED; los restantes resultados terminales son FAILED.

La migración aditiva y reejecutable `000005_traza_transferencia` guarda `trace_id` y el contexto W3C de creación. Las respuestas sin cabeceras o con otra traza recuperan el contexto persistido antes de generar los siguientes mensajes de Outbox. Los IDs de correlación existentes no se cambian. La consulta e historial autorizados a través del Gateway incluyen el `traceId` original. Las transacciones anteriores a esta migración no reciben trazas de creación inventadas y pueden tener ese campo vacío.

## Endpoints internos verificados

| Método y ruta | Resultado |
| --- | --- |
| GET `/salud` | 200, salud del proceso |
| GET `/metrics` | 200, exposición Prometheus con histogramas y contadores |
| GET `/observabilidad/transferencias/:idTransferencia` | 200, `id_transaccion`, `traceId`, `correlationId`, `estado`, `estadoFase2`, `trazaDisponible` |

La consulta de trazabilidad devuelve 400 para UUID inválido, 404 si no existe y 503 si falla la base. Son rutas del servicio dentro del clúster; no se añadieron rutas públicas al Gateway ni cambios para exponer el servicio a Internet. La consulta pública existente `GET /api/transferencias/:idTransferencia` mantiene la validación de propietario y agrega el `traceId` en su respuesta.

El flujo existente Collector → Prometheus recibe las métricas por OTLP. `/metrics` es una alternativa para inspección o scrape directo; usar una sola vía de ingestión para estas series para evitar duplicarlas. El dashboard de consulta se aprovisiona por separado; no se modifican las reglas de alerta.

## Consultas para Grafana

Percentiles p50, p95 y p99, sustituyendo Q por 0.50, 0.95 o 0.99:

```promql
histogram_quantile(Q,
  sum by (le) (rate(bank_transfer_flow_duration_seconds_bucket{service="transaction-service"}[5m]))
)
```

Tasa de fallos por segundo:

```promql
sum(rate(bank_transfer_results_total{service="transaction-service",outcome="failure"}[5m]))
```

Porcentaje de fallo entre transferencias terminadas:

```promql
100 * (
  sum(rate(bank_transfer_results_total{service="transaction-service",outcome="failure"}[5m]))
  or vector(0)
) / clamp_min(sum(rate(bank_transfer_results_total{service="transaction-service"}[5m])), 1e-9)
```

Para éxito, usar el mismo cociente con `outcome="success"`. Filtrar además el entorno cuando la fuente contenga varios. Agregar buckets antes de calcular los percentiles; no promediar latencias ni percentiles entre réplicas. Con poco tráfico se requieren muestras suficientes y una ventana adecuada; la ausencia de series no demuestra disponibilidad.

## Frontend y notificaciones

Solo las tarjetas de eventos/notificaciones de transferencia incorporan `id_transaccion`, usando el ID persistente `idTransferencia`. El comprobante de correo y los detalles de notificación guardados también incluyen ese identificador. Las demás pantallas y las notificaciones de cuentas conservan su comportamiento.

## CI/CD y verificación

La migración 000005 se ejecuta en el paso de esquemas antes del despliegue del backend/frontend. El job de Transaction Service ejecuta la nueva prueba con PostgreSQL desechable y detección de carreras. Se verificaron éxito, rechazo KYC, rechazo de cuentas, rechazo de débito, crédito fallido con compensación, fallo externo, timeout externo, fallo de compensación, rollback y mensajes duplicados. La prueba comprueba que el histograma mida el flujo completo, que exista una sola observación por resultado terminal y que Outbox conserve la traza original. Las pruebas HTTP verifican salud, métricas, consulta de traza y respuestas 400/404.

La compilación TypeScript/Vite y lint de la pantalla modificada también se verificaron. Las pruebas de notificaciones comprueban el ID en el correo y en sus detalles persistidos. Estos cambios locales no implican un despliegue ni un push al repositorio.

## Retiro de usuarios de prueba del servidor

Con autorización expresa se retiraron exactamente `telprueba924`, `cliflujo070`, `clinotificacion050`, `prukycfase2590` y `prukycexterno041` de la base de clientes en 201.216.128.230. Se comprobó previamente que no tenían cuentas, saldos, pagos ni transferencias. Permanecen `admin` (ADMIN) y `teller` (TELLER); los roles del sistema no se eliminaron. El respaldo previo está en `/home/servidor/bank-usac-test/customer-before-test-cleanup-20261018.dump`, con permisos privados. La prueba aislada anterior de Account Service ya se había eliminado; también se retiró su archivo temporal con credenciales ficticias.

## Dashboard Bank USAC-Prueba1: 03/Transaction_service

Una sola gráfica **Latencia del flujo de transferencia · p50, p95 y p99** muestra tres líneas y sus últimos valores, con datasource Prometheus. Usa `histogram_quantile(Q, sum by (le) (rate(bank_transfer_flow_duration_seconds_bucket{service="transaction-service",environment="$environment"}[5m])))` para Q=0.50, 0.95 y 0.99. Agrega réplicas y resultados antes de calcular los percentiles; no usa promedio ni promedia percentiles. Los valores se almacenan en segundos y Grafana adapta su presentación según magnitud.

La ventana móvil es de 5 minutos. Sin transferencias finalizadas observadas o sin suficientes muestras para rate puede aparecer sin datos; no equivale a cero latencia. La estimación depende de los buckets y del número de muestras. El rango temporal y el entorno son los del dashboard; el filtro de UUID de Account Service no aplica.

Para verificar manualmente, finalizar varias transferencias en el entorno seleccionado, esperar exportación y recolección y revisar las tres curvas. No se generan transferencias de prueba automáticamente. La sección incorpora también la tasa de error y la tabla de trazas Saga descritas a continuación. La publicación solo modifica el JSON aprovisionado por Helm y usa la instrumentación existente, sin migraciones ni cambios de backend. Pendiente comprobación de las consultas contra Prometheus desplegado.

### Tasa de error y traza Saga

El panel **Tasa de error del flujo de transferencia (%)** divide la tasa de resultados `failure` entre todos los resultados terminales de `bank_transfer_results_total`, con servicio/entorno y ventana móvil de 5 min. Se suman las réplicas antes del cociente. Una serie de fallos ausente se reemplaza por cero solo si existe la tasa total; si el denominador no es positivo no se muestra porcentaje. Así el panel no representa ausencia de tráfico/telemetría como éxito. RECHAZADA, COMPENSADA y COMPENSACION_FALLIDA son fallos; las Sagas pendientes no se cuentan todavía. Es tasa de error del negocio, no porcentaje de respuestas HTTP 5xx.

La tabla **Trazas del flujo de transferencias · Saga** agrupa por Trace ID como Customer Service. Busca `transferencia.inicio` o `transferencia.estado` en el servicio y entorno seleccionados; se retiró el filtro sagaStates, que podía excluir trazas al exigir un estado adicional. No exige compensación: incluye éxito, fallo y flujos pendientes. Expandir cada fila muestra los spans coincidentes. Abrir Trace ID carga el flujo distribuido completo en Explore con validaciones, movimientos y compensaciones cuando existan. Los sondeos periódicos de outbox no son flujos de negocio y se excluyen de esta búsqueda.

Validación manual: ejecutar una transferencia exitosa y una con fallo posterior al débito que provoque compensación. Confirmar el resultado terminal en la tasa de error; abrir la segunda traza y comprobar mismo Trace ID/correlationId para la Saga, comando de compensación y su respuesta, COMPENSANDO y COMPENSADA (o COMPENSACION_FALLIDA si esa recuperación falla). Una compensación correcta sigue contando como fallo de la transferencia original. Solo aparecen trazas con spans de negocio dentro del rango y la retención de Tempo (48 h). Si no hay filas, comprobar primero la búsqueda amplia `{ resource.service.name = "transaction-service" }` en Explore, el entorno y las versiones desplegadas antes de asumir pérdida de datos.

Estos paneles reutilizan métricas/spans existentes: solo cambia el dashboard, sin migraciones ni modificaciones de backend. JSON y referencias comprobados localmente; pendiente validación con Prometheus/Tempo desplegados.

### Búsqueda amplia de Transaction Service

La tabla ahora se llama **Trazas de Transaction Service** y consulta únicamente `{ resource.service.name = "transaction-service" && resource.deployment.environment.name = "$environment" }`, sin exigir `transferencia.inicio`, `transferencia.estado` ni un estado de negocio. Agrupa por Trace ID y devuelve hasta 50 trazas. Esto permite ver la participación del servicio incluso si los spans de negocio específicos no coinciden; también incluye consultas y sondeos del outbox. No presentar una traza de sondeo como evidencia de una transferencia.

Para localizar una transferencia concreta, tomar el traceId de su consulta/historial autorizado (`GET /api/transferencias/:idTransferencia`) y abrirlo directamente en Tempo, o identificar transaction.id al inspeccionar los atributos. Si la búsqueda amplia sigue vacía, verificar el rango, entorno y llegada de spans en Tempo: no basta modificar otra vez el filtro. La consulta directa por Trace ID distingue falta de coincidencias en búsqueda de ausencia de la traza. Este cambio no confirma acceso ni ingesta del entorno remoto.

### Agrupación por identidad de transacción

La tabla filtra ahora `span.transaction.id =~ ".+"` además de servicio/entorno. Así las consultas ListarPendientes y los sondeos HTTP al partner dejan de ocupar filas. La transferencia interna ya incorpora transaction.id en inicio/estado y mantiene el contexto persistido. La creación y el procesamiento asíncrono interbancario también se anotan con transaction.id (ID del depósito); el worker retoma el trace_context guardado antes de crear su span. No se cambia el ID de negocio ni se crean etiquetas de métricas con UUID. Los spans previos sin ese atributo no se seleccionan retroactivamente.

transaction.id identifica la operación de negocio, Trace ID el recorrido y Span ID un paso. Se agrupa visualmente por Trace ID; el atributo transaction.id permite asociarlo a la transacción. Los sondeos periódicos siguen siendo observables en Explore, pero no son transacciones del usuario.

### Comparación de éxito y fallo

El panel de tasa incluye dos líneas: **Exitosas (%)** (verde) y **Fallidas (%)** (rojo), con la misma tasa total de resultados terminales como denominador y ventana de 5 min. Para muestras válidas con tráfico, suman 100%; pendientes quedan fuera. Los huecos fuera de actividad no son pérdida automática de datos: el panel no muestra porcentajes si la tasa total no es positiva. El último valor visible puede corresponder al último intervalo con operaciones.

Una captura del visor de Customer Service obtenida por Trace ID no es equivalente a una búsqueda filtrada de Transaction Service: se debe tomar el traceId real de la transferencia y abrirlo directamente en Tempo para distinguir ausencia de indexación/atributos de pérdida de spans. No se considera resuelto un No rows sin revisar esa operación contra el entorno desplegado. Se solicitó el ID/traceId y URL de Grafana para esa comprobación.

## Diagnóstico real de traza ausente (21/10/2026)

Se consultó la transferencia `d9948c11-3057-479f-bfbd-9552b494f38a` en el entorno dev. Loki conserva cinco cambios confirmados (VALIDANDO_KYC, VALIDANDO_CUENTAS, PENDIENTE, PROCESANDO, COMPLETADA), todos con Trace ID `8ed62e5570fa9fbe5a43f8a62ac16bb2` y correlación `b833d014-c8da-4e4d-87d3-85cd10830c6a`. Los logs de servicios participantes también comparten el Trace ID. Tempo devuelve 404 al consultarlo por `/api/traces/{id}` y no encuentra datos en su API v2.

La consulta SQL de solo lectura de `trace_id,trace_context` para esa fila devolvió un traceparent terminado en `-00`. Es un contexto válido **no muestreado**: tener traceId en logs no demuestra que existan spans exportados. El sampler ParentBased anterior respeta esa decisión heredada, incluso con ratio raíz igual a 1. Esta evidencia explica por qué no bastaba cambiar filtros de Grafana. No se modificaron la transferencia, sus saldos ni su historial.

Corrección: en Gateway, para `dev` y `local`, un padre remoto no muestreado se evalúa con el ratio de muestreo configurado del banco (por defecto 1). Se conserva el Trace ID original y se propaga la decisión resultante a RabbitMQ y downstream. En `prod` se conserva ParentBased normal; `OTEL_SDK_DISABLED=true` sigue deshabilitando grabación. Una prueba reproduce el padre remoto `-00`, comprueba exportación, conservación de ID y propagación; también verifica ratio 0 y comportamiento prod.

La corrección requiere publicar la imagen actualizada de API Gateway por CI/CD y repetir una transferencia después del despliegue. No se pueden reconstruir retroactivamente los spans no grabados de esta transferencia. La política de sampling de desarrollo aumenta cobertura respecto a padres externos no muestreados y puede aumentar la ingesta de Tempo.

Durante este diagnóstico se identificó el clúster activo como `bank-usac-gke`, proyecto `bank-usac`, zona `us-central1-a`; el contexto local anterior apuntaba a un proyecto sin clúster. Se usó un kubeconfig temporal separado, sin cambiar el contexto predeterminado. La consulta SQL temporal fue de solo lectura; el pod de diagnóstico y el túnel local se retiraron al finalizar.

### Diagnóstico real del 0% de éxito (21/10/2026)

La lectura de Prometheus mostró la serie `bank_transfer_results_total{outcome="success"}` constante en 1, con su primera muestra disponible ya en 1. No había una muestra cero previa; los incrementos iniciales y reinicios que vuelven al mismo valor no se pueden deducir de esa secuencia. Por eso la tasa de incremento puede resultar cero aunque auditoría confirme una transferencia completada. No es correcto inventar el incremento a partir del valor actual.

Transaction Service inicializa ahora success y failure en cero al configurar el MeterProvider. La prueba local comprueba línea base, exclusión de PENDING y primer éxito. Publicar la imagen por CI/CD y esperar al menos una exportación/recolección antes de probar; después del resultado esperar otra recolección. El gráfico conserva ventana móvil de 5 min, por lo que no equivale a todos los éxitos históricos. No modifica ni restaura retrospectivamente los contadores previos.
