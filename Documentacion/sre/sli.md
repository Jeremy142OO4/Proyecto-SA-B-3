# Indicadores de nivel de servicio - SLI

Los **Service Level Indicators (SLI)** son mediciones cuantitativas del comportamiento real de Bank USAC. Permiten determinar si el sistema cumple los objetivos de confiabilidad definidos en los SLO.

Las mediciones se realizan sobre ventanas móviles de 30 días. Durante la evaluación académica también pueden calcularse sobre la duración completa de una prueba de integración, indicando claramente el periodo observado.

## Fuentes de medición

- Respuestas del frontend y API Gateway.
- Endpoints de salud de los Pods.
- Logs estructurados de los servicios.
- Timestamps de comandos y eventos RabbitMQ.
- Estados de las operaciones en las bases de datos.
- Registros de mensajes procesados, reintentos y DLQ.
- Registros de auditoría relacionados mediante `idCorrelacion`.

## SLI-01 - Disponibilidad del punto de entrada

Mide la proporción de solicitudes válidas al frontend y API Gateway que reciben una respuesta técnica correcta. Una operación aceptada con estado asíncrono cuenta como disponible aunque el procesamiento continúe después.

```text
Disponibilidad (%) = solicitudes atendidas / solicitudes válidas totales × 100
```

Se excluyen solicitudes rechazadas correctamente por datos inválidos, falta de autenticación o permisos insuficientes.

## SLI-02 - Disponibilidad de los microservicios

Mide el tiempo durante el cual cada Pod está listo para recibir o procesar trabajo según sus comprobaciones de salud.

```text
Disponibilidad del servicio (%) = tiempo Ready / tiempo total observado × 100
```

Debe calcularse individualmente para Customer, Account, Transaction, Payment y Notification & Audit Service.

## SLI-03 - Latencia del API Gateway

Mide el tiempo desde que el Gateway recibe una solicitud hasta que devuelve la aceptación, rechazo o resultado disponible. No incluye necesariamente la finalización de la operación asíncrona.

Se reportan los percentiles `p50`, `p95` y `p99` en milisegundos.

## SLI-04 - Tiempo de procesamiento asíncrono

Mide el tiempo desde la publicación inicial del comando hasta la aparición de un evento terminal consultable.

```text
Tiempo de procesamiento = fecha del evento terminal - fecha del comando inicial
```

Se calcula por tipo de operación: creación de cliente, creación de cuenta, transferencia y pago. Los estados terminales incluyen completado, rechazado, compensado o compensación fallida, según el flujo.

## SLI-05 - Tasa de operaciones exitosas

Mide la proporción de operaciones técnicamente válidas que llegan al resultado de negocio esperado.

```text
Éxito (%) = operaciones completadas / operaciones válidas iniciadas × 100
```

Los rechazos esperados por reglas de negocio se registran separadamente y no deben confundirse con errores técnicos.

## SLI-06 - Tasa de errores técnicos

Mide las operaciones que no pueden completarse por errores internos, pérdida de conexión, timeout, mensaje malformado generado por el sistema o fallo de infraestructura.

```text
Errores técnicos (%) = operaciones con fallo técnico / operaciones iniciadas × 100
```

No incluye rechazos correctos por fondos insuficientes, cuentas bloqueadas, credenciales inválidas o datos incorrectos proporcionados por el usuario.

## SLI-07 - Idempotencia

Mide la capacidad de procesar mensajes duplicados sin repetir el efecto de negocio.

```text
Idempotencia (%) = duplicados sin efecto adicional / duplicados recibidos × 100
```

La verificación se realiza comparando movimientos, pagos, transferencias y registros de mensajes procesados antes y después de reenviar el mismo `idMensaje`.

## SLI-08 - Entrega de mensajes

Mide la proporción de mensajes publicados que son consumidos o enviados explícitamente a una DLQ dentro del tiempo máximo definido.

```text
Entrega controlada (%) = mensajes procesados o enviados a DLQ / mensajes publicados × 100
```

Un mensaje perdido, sin confirmación y sin registro en DLQ representa un incumplimiento.

## SLI-09 - Tasa de reintentos

Mide qué proporción de mensajes requiere más de un intento de procesamiento.

```text
Reintentos (%) = mensajes reintentados / mensajes consumidos × 100
```

Este indicador permite detectar inestabilidad aunque la operación finalmente termine correctamente.

## SLI-10 - Mensajes en DLQ

Mide la proporción de mensajes que agotan sus reintentos y terminan en una cola de mensajes muertos.

```text
DLQ (%) = mensajes enviados a DLQ / mensajes publicados × 100
```

También debe reportarse el número absoluto por servicio y routing key.

## SLI-11 - Trazabilidad completa

Mide la proporción de operaciones para las que es posible reconstruir el flujo utilizando un único `idCorrelacion`.

```text
Trazabilidad (%) = operaciones con traza completa / operaciones auditadas × 100
```

Una traza completa contiene el comando inicial, los eventos intermedios requeridos y el evento terminal.

## SLI-12 - Compensaciones de transferencias

Mide el resultado de las Sagas que necesitaron revertir un débito.

```text
Compensación exitosa (%) = compensaciones completadas / compensaciones iniciadas × 100
```

Toda transferencia en `COMPENSACION_FALLIDA` debe contarse como incidente y conservar su traza completa.

## SLI-13 - Notificaciones enviadas

Mide la proporción de notificaciones válidas que alcanzan estado `SENT`.

```text
Envío de notificaciones (%) = notificaciones enviadas / notificaciones válidas generadas × 100
```

El fallo de una notificación no convierte automáticamente en fallida la operación bancaria que la originó.

## Resumen de indicadores

| ID | Indicador | Unidad | Alcance |
|---|---|---|---|
| SLI-01 | Disponibilidad del punto de entrada | Porcentaje | Frontend y API Gateway |
| SLI-02 | Disponibilidad de microservicios | Porcentaje | Cada microservicio |
| SLI-03 | Latencia del Gateway | Milisegundos, percentiles | API Gateway |
| SLI-04 | Procesamiento asíncrono | Segundos, percentiles | Flujos de negocio |
| SLI-05 | Operaciones exitosas | Porcentaje | Flujos de negocio |
| SLI-06 | Errores técnicos | Porcentaje | Sistema completo |
| SLI-07 | Idempotencia | Porcentaje | Consumidores |
| SLI-08 | Entrega controlada | Porcentaje | RabbitMQ y consumidores |
| SLI-09 | Reintentos | Porcentaje | Consumidores |
| SLI-10 | Mensajes en DLQ | Cantidad y porcentaje | Cada servicio |
| SLI-11 | Trazabilidad completa | Porcentaje | Auditoría distribuida |
| SLI-12 | Compensaciones exitosas | Porcentaje | Transferencias |
| SLI-13 | Notificaciones enviadas | Porcentaje | Notification & Audit Service |

## Periodicidad del reporte

Para cada ejecución de validación se debe registrar el periodo, cantidad de muestras, valor observado y resultado frente al SLO. Una medición sin periodo ni número de muestras no se considera suficiente para evaluar confiabilidad.

