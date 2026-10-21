# Customer Service: ciclo de vida y KYC

La migración `000007_customer_lifecycle` agrega la correlación y el contexto W3C de la creación, las fechas `kyc_updated_at` y `kyc_verified_at`, y el historial `customer_lifecycle_events`. Aplicar las migraciones antes de desplegar esta versión del servicio. La instrumentación no requiere cambios en frontend ni reglas de alerta. El dashboard de consulta se aprovisiona por separado con Helm.

## Eventos y consistencia

- Creación: `cliente.creado`.
- Decisión KYC: `cliente.kyc.estado.actualizado`, con `estadoKyc=VERIFIED`, `REJECTED` o `PENDING`. Verificación y rechazo quedan registrados como resultados explícitos del cambio de estado.
- Acceso: `cliente.activado` y `cliente.estado.actualizado`.
- Perfil: `cliente.actualizado`.

Se conserva el contrato existente del broker. Los eventos de ciclo de vida usan el `correlationId` original del registro tanto en el sobre como en el outbox. Las respuestas RPC conservan la correlación de su solicitud. `cliente.kyc.verificado` y `cliente.kyc.rechazado` siguen siendo respuestas a validaciones de una Saga; no son nuevas decisiones documentales y no incrementan las métricas KYC.

El estado, el historial sin datos personales y el evento del outbox se confirman en una sola transacción. La fila del cliente se bloquea durante el cambio; repetir el mismo estado no vuelve a publicar ni contar la decisión. El token de activación se consume de forma atómica y no puede activar una cuenta bloqueada. Una edición de perfil no sobrescribe el estado de acceso/KYC.

## Logs y trazas

Cada cambio confirmado emite un log JSON con `timestamp`, `serviceName`, `environment`, `version`, `level`, `message`, `correlationId`, `traceId`, `spanId`, `customerId`, `eventType`, `stateKind`, `previousState` y `state`. Un rechazo KYC o bloqueo usa WARNING; los demás cambios, INFO. No se registra el payload, documento, correo, fotografía, dirección, contraseña ni tokens. La redacción de claves contempla nombres en español e inglés.

Los spans `customer.lifecycle.created`, `customer.lifecycle.activated`, `customer.lifecycle.access`, `customer.lifecycle.updated` y `customer.lifecycle.kyc` retoman el contexto persistido del registro. Una solicitud posterior queda enlazada mediante un span link. El contexto viaja también en el outbox para continuar la traza por RabbitMQ. Los atributos/eventos de estos spans solo incorporan correlación y estados controlados, nunca campos personales o credenciales. No se mantiene un span abierto durante la espera humana del KYC.

Conservar el mismo traceId no amplía la retención del backend de trazas: para visualizar creaciones antiguas, la retención deberá cubrir la espera del proceso KYC. El historial SQL persiste independientemente de esa retención. El muestreo de trazas no condiciona la medición KYC.

## Métricas implementadas

Los nombres de Prometheus incluyen los sufijos que agrega la exportación del Collector:

| Instrumento OTel | Serie Prometheus | Tipo y significado |
|---|---|---|
| `bank_kyc_customers` | `bank_kyc_customers` | Gauge: clientes con rol CLIENTE por `status=PENDING/VERIFIED/REJECTED`, consultados desde la base con timeout de 2 s. Se reconstruye después de reinicios; incluye clientes anteriores. |
| `bank_kyc_decisions` | `bank_kyc_decisions_total` | Counter: transiciones KYC confirmadas por `outcome=verified/rejected`. Una nueva transición posterior cuenta como otra decisión; una solicitud idéntica repetida no. |
| `bank_kyc_duration` | `bank_kyc_duration_seconds_bucket`, `_sum`, `_count` | Histogram: segundos entre la creación y la primera verificación observada. No mide el tiempo de una consulta KYC de la Saga. |

Etiquetas base `service` y `environment`; nunca IDs o datos personales como etiquetas. Los buckets de KYC cubren desde 1 segundo hasta 30 días, además de +Inf. Las decisiones/histogramas se registran después del commit; un fallo de transacción no produce una observación de éxito. Como las demás métricas OTel del servicio, estos instrumentos son señales operativas en memoria: una caída entre el commit y la emisión/exportación puede perder una muestra. El historial SQL y el outbox son la evidencia durable; no se presentan los counters como un conteo contable exacto.

Cada réplica observa el mismo total de la base compartida. Para contar clientes, deduplicar los gauges por entorno/estado con `max by (environment, status) (bank_kyc_customers{service="customer-service"})`; no sumar las réplicas. Los counters e histogramas sí se agregan entre réplicas.

Consulta p95 (incluida en el dashboard de prueba con filtros de servicio y entorno):

```promql
histogram_quantile(
  0.95,
  sum by (le) (rate(bank_kyc_duration_seconds_bucket[24h]))
)
```

Agregar filtros de entorno/servicio según el despliegue. Sin verificaciones en la ventana el resultado es sin datos. Las verificaciones que excedan el último bucket finito requieren ampliar los límites para estimar correctamente ese percentil; no promediar percentiles por Pod.

## Clientes existentes y pruebas

No se inventan fechas de verificación ni eventos de creación anteriores a la migración. Un cliente anterior adquiere correlación/contexto en su primer cambio observado y aparece en el gauge; queda fuera del histograma de primera verificación. `lifecycle_observed_from_creation` distingue estos registros. Para nuevos clientes, `kyc_verified_at` se conserva aunque se reabra/rechace y después se apruebe nuevamente.

Pruebas del módulo:

```sh
go test ./...
TEST_DATABASE_URL='postgres://usuario:clave@localhost:5432/base_aislada?sslmode=disable' go test ./repositories -run CustomerLifecycle -count=1
```

Las pruebas de ciclo de vida crean y eliminan un esquema con nombre aleatorio dentro de la base aislada indicada. Validan migraciones, commit/rollback, concurrencia, repetición, activación, continuidad del traceId/correlationId, privacidad, conteos y unidades del histograma. La prueba existente `TestTelemetryOutboxDatabase` requiere que las migraciones también estén aplicadas en el esquema predeterminado de esa base. RabbitMQ se prueba por separado con `TEST_RABBITMQ_URL`.

Validación realizada en el servidor de pruebas: `go test -race ./... -count=1` y `go vet ./...` terminaron correctamente con PostgreSQL 17 y RabbitMQ 4 temporales. También se verificó el endpoint HTTP KYC con autorización de administrador. Los contenedores temporales se retiraron; las bases existentes de Bank USAC y su despliegue no se modificaron.

## Dashboard Bank USAC-Prueba1

En la carpeta **Bank USAC**, abrir **Bank USAC-Prueba1** → fila **01/Customer_Service** → **Clientes verificados vs rechazados**. Elegir el entorno en el selector superior. El panel muestra el estado actual, no el número de decisiones realizadas en el rango de tiempo. Se actualiza cada 15 segundos, sujeto al intervalo de exportación y recolección.

Fuente: `infrastructure/observability/chart/dashboards/prueba1.json`. La plantilla `configmaps.yaml` incluye el JSON y `workloads.yaml` lo monta en Grafana y calcula su checksum para reiniciar el pod al cambiar. El workflow existente ejecuta `infrastructure/observability/deploy.sh`; no hay que modificar ese script ni agregar migraciones.

Consulta PromQL:

```promql
max by (status) (
  bank_kyc_customers{service="customer-service",environment="$environment",status=~"VERIFIED|REJECTED"}
)
```

El `max` evita duplicar el conteo de la base compartida entre réplicas. Los estados sin clientes tienen valor cero cuando el servicio exporta correctamente; una ausencia de telemetría aparece como **Sin datos**. Para probar, verificar un cliente y rechazar otro desde administración; el panel debe coincidir con sus estados actuales. No se necesita crear usuarios sintéticos. Los logs, trazas, correlación y redacción de datos siguen usando la instrumentación existente; este panel muestra solamente el conteo agregado solicitado.

Como alternativa al despliegue, importar el JSON desde Grafana → Dashboards → New → Import y seleccionar la carpeta Bank USAC. Debe existir el datasource Prometheus con UID `prometheus`. El dashboard aprovisionado se mantiene desde el repositorio.

### Tiempo KYC y trazas por Trace ID

La sección incluye el p95 KYC y una tabla en formato `traces`: una fila por Trace ID y sus spans de creación/decisión KYC al expandirla. Abrir el Trace ID permite inspeccionar el recorrido en Explore. Se retiraron la variable traceId y el panel de traza seleccionada. El visor independiente ya no genera errores por un Trace ID vacío. Puede persistir el aviso de raíz ausente en metadatos de búsqueda de Tempo aunque se encuentre la raíz al abrir la traza; no se presenta como una corrección de la ingesta.

Validar creación, decisión con estado VERIFIED y correlationId compartido en el recorrido. Tempo conserva trazas durante 48 h; la configuración del dashboard no reconstruye datos eliminados. El p95 usa verificaciones observadas durante las últimas 24 h y buckets de histograma, no la duración de la consulta SQL.
