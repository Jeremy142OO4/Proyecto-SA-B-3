# Prueba de carga real de transferencias

Ejecutar sólo con autorización sobre el ambiente objetivo. No se realiza ninguna
llamada a BANK14 ni se usan cuentas personales. Las transferencias pasan por el
Gateway público, broker, KYC, cuentas y Saga reales; `EXITO` selecciona el resultado
del proveedor simulado del flujo interno, no inventa métricas ni resultados.

## Preparación (una vez por conjunto de cuentas)

Requisitos: Python 3.10+, k6 **1.6.0**, acceso del operador como ADMIN y TELLER.
Los tokens se entregan en las variables `ADMIN_TOKEN` y `TELLER_TOKEN`; no se guardan
en el repositorio. No hace falta modificar IAM, HPA ni recursos del clúster.

```powershell
python tools/load/prepare_fixture.py --base-url https://HOST-PRODUCCION --output tmp/carga-fixture.json --pairs 8 --confirm create-new-test-fixtures
```

Crea un cliente nuevo `carga-*`, con correo reservado `.invalid`, KYC verificado,
16 cuentas corrientes sin comisión y Q1,000 de fondos estudiantiles en cada una.
ADMIN activa/verifica; TELLER registra y abre cuentas. El cliente hace login normal.
Nunca sobrescribe un fixture existente. Conserva las identidades antes de cada
escritura: si se interrumpe, **conciliar antes de repetir**. No se reintentan POST.
No se cambia la configuración de monitoreo ni se borran operaciones históricas.

`tmp/carga-fixture.json` contiene credenciales: mantener privado, no adjuntar a
evidencia ni subir a Git. El correo `.invalid` no entrega mensajes a personas:
errores de entrega de ese cliente no representan disponibilidad real de SMTP.

## Ejecución

Primero inicia la observación en otra terminal (kubectl autenticado al clúster):

```powershell
python tools/load/observe_hpa.py --namespace bank-usac --output tmp/hpa-carga.jsonl --seconds 900
```

Después ejecuta la comprobación mínima:

```powershell
$env:FIXTURE_FILE = 'tmp/carga-fixture.json'
$env:CONFIRM_LOAD = 'dedicated-test-accounts'
$env:PROFILE = 'smoke'
$env:SUMMARY_FILE = 'tmp/carga-smoke.json'
k6 run tools/load/transferencias.js
```

Si las tres operaciones terminan COMPLETADA, no hay reservas y el saldo conjunto
se conserva, ejecutar el perfil de carga:

```powershell
$env:PROFILE = 'load'
$env:SUMMARY_FILE = 'tmp/carga-resumen.json'
k6 run tools/load/transferencias.js
```

Rampa de seis minutos: 1→2→4→8 usuarios, tres minutos sostenidos y descenso.
Cada usuario tiene un par propio y alterna dirección, sin competencia artificial
por una única cuenta. Máximo 200 transferencias de **Q0.01** por usuario. No usar
`--vus`, `--iterations` ni otras opciones para sobrepasar esos límites.

El script valida titularidad, KYC, estado, comisiones y fondos antes de empezar.
Consulta el estado durable `/api/transferencias/:id`, no `/api/operaciones`.
Ante un resultado rechazado, timeout o respuesta ambigua detiene nuevas operaciones;
conservar el ID y conciliar, no reenviar. Un 404 inicial sólo se tolera al consultar
el registro recién aceptado, por consistencia eventual.

Al terminar cada perfil, incluso si falla, realizar esta conciliación de sólo lectura:

```powershell
python tools/load/reconcile_fixture.py --fixture tmp/carga-fixture.json --output tmp/carga-conciliacion.json
```

Además de `consistent=true`, comparar `transfers` con el total acumulado de
`transfers_accepted` de los perfiles ejecutados. Un saldo conservado por sí solo
no prueba que todas las operaciones aceptadas hayan terminado.

### Capacidad acotada (sólo después de la rampa normal)

Si el perfil normal es estable pero no supera la utilización objetivo del HPA,
se puede medir mayor concurrencia **sin alterar** requests/limits ni el HPA:

```powershell
$env:PROFILE = 'capacity'
$env:CONFIRM_CAPACITY = 'bounded-64'
$env:SUMMARY_FILE = 'tmp/carga-capacidad.json'
k6 run tools/load/transferencias.js
```

Sube 1→8→16→32→64 usuarios durante siete minutos (máximo 64, o menos si hay pocos
pares), con hasta ocho usuarios por par dedicado. Esta variante **sí incluye
contención sobre las cuentas** y debe interpretarse como una prueba de capacidad
concentrada, no como 64 clientes independientes. Máximo global 1,600 operaciones
por perfil; con 64 usuarios son 25 por usuario. Se detiene ante el primer fallo
confirmado o resultado incierto y también requiere conciliación al terminar.

## Evidencia y criterio de aceptación

### Ejecución real en producción

Se completaron 156 transferencias (3 iniciales y 153 de carga), con cero fallos
y saldo total conservado. Notification & Audit escaló automáticamente de una a
dos réplicas. **El p95 HTTP de aceptación fue 3.216 s y no pasó el objetivo de
500 ms**. El escalamiento y la latencia son criterios distintos.

Resultados, conciliación y archivos originales en el
[informe de la prueba](../../Documentacion/sre/pruebas-carga.md).

#### Latencia y resultados en Grafana

![Latencia y resultados durante la rampa de transferencias](../../Documentacion/sre/evidencia-carga/ejecucion-produccion/grafana-latencia-rampa.png)

#### CPU y memoria durante la carga

![CPU y memoria por pod durante la prueba](../../Documentacion/sre/evidencia-carga/ejecucion-produccion/grafana-cpu-hpa.png)

#### Evidencia de escalamiento automático

![HPA de una a dos réplicas y aumento de nueve a diez pods](../../Documentacion/sre/evidencia-carga/ejecucion-produccion/grafana-hpa-listo.png)

El servicio que escaló fue Notification & Audit; Transaction y Payment quedaron
en una réplica. No se escaló manualmente ni se alteró el objetivo de CPU del HPA.

### Criterios para interpretar las mediciones

- `transfers_attempted`, `transfers_accepted` y `transfers_completed` son distintos.
- Latencia de aceptación: `http_req_duration{name:POST /api/transferencias}`.
  Objetivo p95 <500 ms, medido desde el generador incluyendo red.
- Latencia completa: `transfer_end_to_end_ms`, hasta estado terminal, p50/p95/p99.
  Incluye hasta 500 ms de resolución del sondeo. Límite operativo de prueba: 60 s;
  no sustituye los SLO del enunciado.
- Throughput: `transfers_completed.rate` (operaciones terminadas/segundo).
- Errores de flujo: `transfer_failure_rate`; pendientes: `transfers_unresolved`.
  Las respuestas 404 de sondeo no equivalen a transferencias fallidas.
- Revisar saldos y reservas de **todas** las cuentas del fixture y conciliar el
  historial por cliente y descripción `runId` al finalizar, incluso si k6 falla.
  Con comisión cero, el total final debe igualar al inicial.
- Evidencia HPA: conservar JSONL con condiciones/CPU observada, desired/current,
  disponibilidad y creación de pods; capturar también Grafana durante el pico.
  **HPA configurado no demuestra escalamiento**: debe observarse un aumento real
  automático de réplicas y nuevos pods listos. Si no ocurre, declararlo pendiente.

En Grafana, dashboard **Bank USAC — Métricas del sistema**, ajustar el intervalo
al inicio/fin UTC de la prueba. Consultas útiles (sin mezclar el laboratorio):

```promql
sum(rate(bank_http_requests_total{environment="prod",service="api-gateway",route="/api/transferencias",method="POST"}[1m]))
```

```promql
histogram_quantile(0.95, sum by (le) (rate(bank_transfer_flow_duration_seconds_bucket{environment="prod",service="transaction-service"}[1m])))
```

```promql
{__name__=~"k8s_hpa_(current|desired)_replicas",job="bank-infra",k8s_namespace_name="bank-usac"}
```

No bajar umbrales del HPA, escalar manualmente ni producir métricas sintéticas
para aparentar cumplimiento. Una prueba de capacidad limitada no prueba 30 días
de SLO, entrega SMTP, fallos controlados ni conformidad del banco pareja.
