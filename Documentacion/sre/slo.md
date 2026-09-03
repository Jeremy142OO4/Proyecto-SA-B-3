# Objetivos de nivel de servicio - SLO

Los **Service Level Objectives (SLO)** establecen los valores objetivo que Bank USAC espera alcanzar para los SLI definidos. Estos objetivos corresponden al ambiente local de demostración y deben evaluarse sobre una ventana móvil de 30 días o sobre una ejecución completa de pruebas cuando no exista operación continua.

## Objetivos generales

| ID | SLI relacionado | Objetivo | Ventana |
|---|---|---:|---|
| SLO-01 | SLI-01 Disponibilidad del punto de entrada | Mayor o igual a 99.0 % | 30 días |
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

Para el SLO de disponibilidad general de 99.0 %, el presupuesto de error es 1.0 % de la ventana observada.

```text
Presupuesto de error = 100 % - objetivo de disponibilidad
```

En una ventana de 30 días, un objetivo de 99.0 % admite aproximadamente 7 horas y 12 minutos de indisponibilidad acumulada. Este valor se usa como referencia de diseño; el proyecto académico se valida principalmente mediante periodos controlados de prueba.

## Condiciones de medición

- Se excluyen periodos de mantenimiento anunciados antes de iniciar la ventana.
- Se excluyen solicitudes inválidas, no autenticadas o no autorizadas rechazadas correctamente.
- Los fallos provocados deliberadamente durante pruebas de resiliencia deben reportarse por separado.
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

