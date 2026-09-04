# Acuerdos de nivel de servicio - SLA

El **Service Level Agreement (SLA)** define los compromisos de servicio que Bank USAC presenta a sus usuarios durante el alcance académico del proyecto. Se deriva de los SLO, pero expresa únicamente los compromisos externos y las condiciones bajo las cuales se evalúan.

Este SLA corresponde a un sistema escolar ejecutado en infraestructura local. No representa un contrato bancario comercial, no contempla compensaciones económicas y no debe interpretarse como garantía para un ambiente productivo.

## Alcance

El acuerdo cubre:

- Acceso al frontend y API Gateway.
- Registro, actualización, activación y autenticación de clientes.
- Creación y consulta de cuentas.
- Actualización segura de balances.
- Transferencias y sus compensaciones.
- Pagos internos o externos simulados.
- Consulta del estado de operaciones asíncronas.
- Registro de auditoría y envío de notificaciones.

## Compromisos

| ID | Compromiso | Nivel acordado | Periodo de evaluación |
|---|---|---:|---|
| SLA-01 | Disponibilidad del punto de entrada | Mayor o igual a 98.0 % | Periodo de demostración o 30 días |
| SLA-02 | Respuesta inicial del API Gateway | 95 % en 1 segundo o menos | Periodo de demostración o 30 días |
| SLA-03 | Finalización de operaciones asíncronas normales | 95 % en 15 segundos o menos | Periodo de demostración o 30 días |
| SLA-04 | Transferencias que requieren compensación | 95 % con resultado en 30 segundos o menos | Periodo de demostración o 30 días |
| SLA-05 | Protección contra duplicados | Ningún efecto financiero duplicado conocido | Cada ejecución |
| SLA-06 | Trazabilidad | Toda operación financiera aceptada conserva `idCorrelacion` | Cada ejecución |
| SLA-07 | Integridad del balance | Ningún débito deja saldo negativo | Cada ejecución |
| SLA-08 | Recuperación de mensajes | Todo mensaje confirmado queda procesado o identificado en DLQ | Cada ejecución |

Los objetivos internos de los SLO son más estrictos que el SLA para proporcionar un margen de seguridad antes de incumplir un compromiso externo.

## Definición de disponibilidad

El sistema se considera disponible cuando el usuario puede cargar la interfaz y el API Gateway puede aceptar una solicitud válida o devolver una respuesta técnica controlada.

Una respuesta de negocio negativa, como credenciales inválidas, fondos insuficientes o cuenta bloqueada, no representa indisponibilidad si fue procesada correctamente y comunica un resultado claro.

## Inicio y final de una operación

- Una operación inicia cuando el API Gateway valida la solicitud y publica o registra correctamente el comando inicial.
- Una operación finaliza cuando existe un estado terminal consultable: completado, rechazado, compensado o compensación fallida.
- Un estado `PENDIENTE`, `PROCESANDO` o `COMPENSANDO` no se considera pérdida de la operación mientras se encuentre dentro del tiempo acordado.

## Exclusiones

El SLA no cubre:

- Mantenimiento local anunciado.
- Apagado del equipo anfitrión, Docker, Minikube o la red local por decisión del operador.
- Datos inválidos proporcionados por el usuario.
- Intentos sin autenticación o sin permisos.
- Indisponibilidad deliberada durante pruebas de fallos.
- Fallos de proveedores externos simulados cuando el sistema los maneja y registra correctamente.
- Uso distinto del alcance académico definido.

Las exclusiones no permiten ocultar pérdida de mensajes, duplicidad de operaciones, saldos inconsistentes o fallos no controlados del código.

## Severidad de incidentes

| Severidad | Descripción | Ejemplos | Tiempo objetivo de atención |
|---|---|---|---:|
| Crítica | Riesgo de pérdida, duplicidad o inconsistencia financiera. | Débito duplicado, saldo negativo, transferencia sin estado recuperable. | 1 hora durante la ventana de trabajo |
| Alta | Flujo principal indisponible sin corrupción conocida. | Gateway caído, RabbitMQ no disponible, varios servicios sin procesar. | 4 horas durante la ventana de trabajo |
| Media | Función degradada con alternativa o recuperación automática. | Retrasos elevados, reintentos frecuentes, notificaciones fallidas. | 1 día hábil |
| Baja | Problema visual o documental sin impacto en el procesamiento. | Texto incorrecto, problema menor de interfaz. | Antes de la siguiente entrega |

Los tiempos corresponden a objetivos de atención del equipo académico, no a soporte continuo 24/7.

## Evidencia de cumplimiento

El cumplimiento debe demostrarse mediante:

- Resultados de pruebas de integración.
- Estado de Pods y comprobaciones de salud.
- Logs estructurados del API Gateway y microservicios.
- Estados de RabbitMQ, reintentos y DLQ.
- Registros de base de datos y auditoría.
- Reconstrucción de operaciones mediante `idCorrelacion`.
- Cálculo de los SLI con periodo y cantidad de muestras.

## Consecuencia de incumplimiento

Debido al carácter académico del proyecto, no se ofrecen créditos ni compensaciones económicas. Ante un incumplimiento se debe registrar la evidencia, clasificar el incidente, identificar la causa, aplicar la corrección y repetir las pruebas relacionadas.

Un incidente crítico impide declarar el sistema confiable hasta demostrar que no existe pérdida o duplicidad de operaciones y que los balances permanecen consistentes.

## Relación entre SLI, SLO y SLA

- Los **SLI** indican lo que realmente ocurrió.
- Los **SLO** definen el objetivo técnico interno.
- El **SLA** expresa el compromiso ofrecido al usuario y las condiciones de evaluación.

Los valores de este acuerdo deben revisarse si cambia la infraestructura, el volumen de uso o el alcance funcional del sistema.
