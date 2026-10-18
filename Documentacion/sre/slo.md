# Objetivos de nivel de servicio - SLO

Los **Service Level Objectives (SLO)** establecen los valores objetivo que Bank USAC espera alcanzar para los SLI definidos. Para la fase 3, el objetivo oficial de disponibilidad es **99.5 %**, acordado por el equipo. Se evalúa por entorno, sobre una ventana móvil de 30 días. Una prueba corta se reporta como evidencia de esa ejecución, no como cumplimiento mensual.

Este documento es la referencia vigente de objetivos. Las propuestas de fase 2 se conservan como antecedentes; no sustituyen estos umbrales. La instrumentación común ya está implementada y probada en aislamiento; faltan el despliegue, las métricas de dominio y la medición real. Ver [implementación](../observabilidad/implementacion.md) y [contrato de telemetría](../observabilidad/contrato-telemetria.md).

## Objetivos generales

| ID | SLI relacionado | Objetivo | Ventana |
|---|---|---:|---|
| SLO-01 | SLI-01 Disponibilidad del punto de entrada | Mayor o igual a 99.5 % | 30 días |
| SLO-02 | SLI-02 Disponibilidad por microservicio | Mayor o igual a 98.5 % por servicio | 30 días |
| SLO-03 | SLI-03 Latencia del API Gateway | `p95` menor o igual a 500 ms para aceptar o consultar una solicitud | 30 días |
| SLO-04 | SLI-04 Procesamiento asíncrono | `p95` menor o igual a 10 s para operaciones sin compensación | 30 días |
| SLO-05 | SLI-05 Operaciones técnicamente exitosas | Mayor o igual a 99.0 % | 30 días |
| SLO-06 | SLI-06 Errores técnicos | Menor o igual a 1.0 % | 30 días |
| SLO-07 | SLI-07 Idempotencia | 100 % de duplicados sin efecto adicional | Cada prueba y 30 días |
| SLO-08 | SLI-08 Entrega controlada | Mayor o igual a 99.9 % procesados o ubicados en DLQ | 30 días |
| SLO-09 | SLI-10 Mensajes en DLQ | Menor o igual a 0.5 % | 30 días |
| SLO-10 | SLI-11 Trazabilidad | 100 % de operaciones con traza por `idCorrelacion` | 30 días |
| SLO-11 | SLI-12 Compensaciones | Mayor o igual a 99.0 % de compensaciones completadas | 30 días |
| SLO-12 | SLI-13 Notificaciones | Mayor o igual a 95.0 % enviadas | 30 días |

## Objetivos por flujo

| Flujo | Resultado esperado | Tiempo objetivo p95 | Confiabilidad objetivo |
|---|---|---:|---:|
| Registro de cliente | Cliente almacenado o rechazo explícito | 5 s | 99.0 % |
| Creación de cuenta | Cuenta creada o solicitud rechazada | 8 s | 99.0 % |
| Consulta de saldo | Respuesta asíncrona disponible | 5 s | 99.0 % |
| Transferencia sin compensación | Estado terminal consultable | 10 s | 99.0 % |
| Transferencia con compensación | Estado compensado o fallo explícito | 20 s | 99.0 % |
| Procesamiento de pago | Estado terminal consultable | 10 s | 99.0 % |
| Registro de auditoría | Evento persistido | 5 s | 99.9 % |
| Notificación | Estado de envío registrado | 15 s | 95.0 % |

Los rechazos por reglas válidas del negocio cuentan como respuestas correctas si el sistema conserva el estado, motivo y trazabilidad. Por ejemplo, rechazar una transferencia por fondos insuficientes no incumple el SLO.

## Presupuesto de error

Para el SLO de disponibilidad general de 99.5 %, el presupuesto de error es 0.5 % de las solicitudes elegibles observadas.

```text
Presupuesto de error = 100 % - objetivo de disponibilidad
```

Para N solicitudes elegibles, el presupuesto es `0.005 × N` fallos. El presupuesto restante es `0.005 × N - fallos observados`; puede ser negativo. Sin tráfico se informa «sin datos», no 100 %. Si se mide por tiempo mediante una sonda independiente, 30 días equivalen a 216 minutos (3 h 36 min) permitidos; ese presupuesto temporal no sustituye el basado en solicitudes.

El 99.5 % aplica a disponibilidad: no cambia automáticamente los objetivos independientes de entrega, compensación, notificaciones o latencia de las tablas anteriores.

## Condiciones de medición

- Se cuentan todas las respuestas HTTP de negocio del Gateway; una respuesta no 5xx cuenta como disponible, incluidos los rechazos 4xx. Esto no demuestra que el resultado de negocio sea correcto: SLI-05 lo mide por separado.
- Se excluyen únicamente endpoints de salud, readiness y métricas. El mantenimiento y los fallos reales no se eliminan retrospectivamente.
- Dev y prod se evalúan por separado. Las pruebas de carga y resiliencia se identifican y se muestran también en su ventana completa, incluidos los fallos; no se oculta su consumo de presupuesto.
- Se incluyen fallos internos, timeouts, pérdida de mensajes y errores inesperados.
- Todo valor debe indicar periodo, cantidad de muestras y fuente de medición.
- No se declara cumplimiento cuando no existen muestras suficientes.

## Respuesta ante incumplimientos

Cuando un SLO no se cumple, el equipo debe:

1. Identificar los servicios, routing keys y operaciones afectadas.
2. Reconstruir una muestra de incidentes mediante `idCorrelacion`.
3. Revisar reintentos, DLQ, latencia del broker y errores de persistencia.
4. Corregir primero pérdida de datos, duplicidad o balances inconsistentes.
5. Registrar la causa y repetir la prueba afectada.

El incumplimiento de idempotencia, trazabilidad o consistencia financiera se considera crítico aunque los demás porcentajes permanezcan dentro del objetivo.

