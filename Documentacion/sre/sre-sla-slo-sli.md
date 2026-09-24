# SRE, SLA, SLI y SLO — Bank USAC (Fase 2)


> **Alcance de despliegue:** cinco microservicios en GKE y un frontend previsto para Cloud Run, según la decisión más reciente del equipo. El traslado del frontend desde Kubernetes debe validarse en el despliegue y el pipeline antes de considerarlo realizado. Las bases por servicio se proyectan en Cloud SQL.

## 1. Cómo se utilizan estos términos

- **SRE (Site Reliability Engineering):** prácticas de ingeniería para operar servicios confiables con monitoreo, automatización, respuesta a incidentes y mejora continua.
- **SLI (Service Level Indicator):** medición observada durante una ventana definida; ejemplo: proporción de solicitudes válidas atendidas correctamente.
- **SLO (Service Level Objective):** meta interna para un SLI, acordada por el equipo.
- **SLA (Service Level Agreement):** compromiso formal con un consumidor. **Aquí no se presume que exista un contrato ni penalización económica**; la sección «SLA propuesto» describe un acuerdo operativo que solo entraría en vigor tras su aprobación.
- **Presupuesto de error:** margen entre el 100 % y el SLO durante la ventana correspondiente; aplica solo si el SLI se mide efectivamente.

## 2. Reglas de medición comunes

**Entorno:** producción. No mezclar tráfico de `dev`, verificaciones sintéticas, cargas de prueba ni reintentos técnicos con la población principal sin identificarlos por separado. **Ventana propuesta:** mes calendario en zona horaria `America/Guatemala`. Para alertas operativas, revisar adicionalmente ventanas cortas; estas no sustituyen la evaluación mensual.

### Solicitudes HTTP

$$SLI_{éxito\ HTTP}=\frac{\text{solicitudes elegibles con respuesta correcta}}{\text{solicitudes HTTP elegibles}}\times 100\%$$

Definir «correcta» según el contrato funcional de cada operación. Las respuestas 4xx debidas a datos inválidos o restricciones de negocio esperadas se excluyen tanto del numerador como del denominador; los errores del servidor y respuestas incorrectas a solicitudes válidas cuentan como fallos. Separar `/health`, métricas y pruebas internas. La latencia se calcula desde el ingreso de una solicitud elegible hasta la respuesta, y se reporta como p95 o como porcentaje bajo un umbral, según se indique.

### Procesamiento asíncrono

$$SLI_{eventos}=\frac{\text{eventos elegibles procesados con resultado correcto dentro del plazo}}{\text{eventos elegibles recibidos}}\times 100\%$$

El plazo empieza cuando el mensaje queda disponible para el consumidor y termina cuando se confirma el efecto de negocio o el resultado esperado queda persistido. Agrupar reentregas por identificador lógico/idempotencia para evitar inflar el denominador. Los eventos inválidos conforme al contrato se contabilizan aparte; un fallo o timeout previsto por el simulador de pagos puede ser un **resultado de negocio correcto**, mientras que una pérdida, duplicación de efecto o ausencia de resultado terminal es un fallo técnico. Registrar atraso y mensajes sin resolver/DLQ como señales adicionales. Validar que la instrumentación realmente permita observar estas etapas.

### Disponibilidad y dependencias

Para componentes sin tráfico suficiente, medir con verificaciones sintéticas de operaciones de negocio representativas; no inferir disponibilidad del estado «pod listo» solamente. Una falla de Cloud SQL o RabbitMQ que impida completar una operación elegible afecta el SLI de extremo a extremo, aunque el proceso HTTP continúe respondiendo. Los indicadores de infraestructura ayudan al diagnóstico, no reemplazan el SLI del usuario.

**Presupuesto orientativo:** con SLO de disponibilidad de 99.5 %, se admite como máximo 0.5 % de fallos en las unidades medidas. Si se usa disponibilidad por minutos en un mes de 30 días, esto equivale a 216 minutos; no debe confundirse un presupuesto basado en minutos con uno basado en solicitudes o eventos.

## 3. Frontend — Cloud Run 


| Tipo | Propuesta |
|---|---|
| SLI | Porcentaje de cargas válidas de la aplicación servidas correctamente por HTTPS y p95 del tiempo de respuesta del documento/recursos esenciales. Separar errores de entrega de fallos de API de backend. |
| SLO | Éxito mensual ≥ **99.5 %**; p95 de entrega ≤ **2 s**. Umbrales **propuestos**, no verificados. |
| SLA propuesto | La interfaz debería estar accesible para los usuarios en producción con objetivo de éxito mensual ≥ **99.0 %**; incidentes confirmados se comunican a los responsables. Sin compensaciones definidas. |


## 4. Customer Service

**Función:** gestión de clientes y estado KYC (`PENDING`, `VERIFIED`, `REJECTED`), incluyendo validación de clientes para operaciones.

| Tipo | Propuesta |
|---|---|
| SLI | Éxito de solicitudes válidas de consulta/actualización KYC; porcentaje de validaciones KYC resueltas correctamente dentro del plazo, si se implementan mediante eventos. |
| SLO | Solicitudes correctas ≥ **99.5 %** mensual; validaciones elegibles completas ≤ **5 s** en ≥ **95 %** de casos. |
| SLA propuesto | Los consumidores internos deberían obtener un resultado KYC correcto y trazable para operaciones válidas; objetivo operativo mensual ≥ **99.0 %** de resultados correctos. |


## 5. Account Service

**Función:** gestión de cuentas y aplicación de reglas para tipos de cuenta, saldo mínimo y comisiones cuando correspondan.

| Tipo | Propuesta |
|---|---|
| SLI | Porcentaje de operaciones válidas de cuenta procesadas con resultado correcto, sin doble efecto; tiempo desde recepción de la solicitud/evento hasta persistencia del resultado. |
| SLO | Resultado correcto ≥ **99.5 %** mensual; ≥ **95 %** de operaciones elegibles resueltas ≤ **5 s**. |
| SLA propuesto | Los servicios consumidores deberían recibir resultados de débito/crédito correctos o rechazos de negocio explícitos y trazables; objetivo operativo de resultados correctos ≥ **99.0 %** mensual. |


## 6. Transaction Service

**Función:** transferencias, seguimiento de estados (`PENDING`, `APPROVED`, `FAILED`) e historial dentro del flujo Saga.

| Tipo | Propuesta |
|---|---|
| SLI | Porcentaje de transferencias elegibles que alcanzan un estado terminal coherente (`APPROVED` o `FAILED`) dentro del plazo; porcentaje de consultas válidas de historial respondidas correctamente. |
| SLO | ≥ **99.5 %** de transferencias elegibles con estado terminal ≤ **30 s**; consultas de historial correctas ≥ **99.5 %** mensual. |
| SLA propuesto | Cada transferencia aceptada debería quedar trazable y obtener confirmación o fallo de negocio explícito; objetivo operativo de resolución en plazo ≥ **99.0 %** mensual. |


## 7. Payment Service

**Función:** simulación de pagos externos con respuestas de éxito, fallo y timeout.

| Tipo | Propuesta |
|---|---|
| SLI | Porcentaje de intentos elegibles con resultado de negocio persistido y publicado de forma coherente dentro del plazo, incluido un resultado de timeout correctamente manejado. |
| SLO | Resultado correcto y trazable ≥ **99.5 %** mensual; ≥ **95 %** de intentos resueltos ≤ **30 s**. |
| SLA propuesto | Los consumidores internos deberían recibir resultado explícito o timeout gestionado, sin cobro o efecto duplicado; objetivo operativo ≥ **99.0 %** mensual. |


## 8. Notification & Audit Service

**Función:** recepción y clasificación de eventos (`INFO`, `WARNING`, `ERROR`), notificaciones y persistencia del historial de auditoría.

| Tipo | Propuesta |
|---|---|
| SLI | Porcentaje de eventos elegibles registrados una sola vez con clasificación y `correlationId` correctos dentro del plazo. Medir la entrega de notificaciones separadamente si existe canal de envío verificable. |
| SLO | Eventos registrados correctamente ≥ **99.5 %** mensual; ≥ **95 %** registrados ≤ **10 s** desde su disponibilidad para consumo. |
| SLA propuesto | Los servicios consumidores deberían disponer de una bitácora de eventos coherente y consultable; objetivo operativo de registro correcto ≥ **99.0 %** mensual. No prometer entrega de correos sin validar proveedor e instrumentación. |


## 9. Respuesta ante incumplimientos (propuesta SRE)

1. **Detectar:** alertar por degradación sostenida de SLIs y revisar errores HTTP, eventos atrasados, DLQ, estado de RabbitMQ, bases y despliegues.
2. **Diagnosticar:** correlacionar solicitud/evento y cambio de versión; distinguir fallo de negocio esperado de fallo técnico.
3. **Mitigar:** detener promociones de versiones problemáticas y, si corresponde, revertir el despliegue afectado mediante el procedimiento del pipeline; comprobar funcionamiento después de la reversión.
4. **Documentar:** registrar incidente, impacto, período, SLI afectado, causa conocida y acciones preventivas. Si se consume el presupuesto de error, priorizar estabilidad hasta acordar reanudación de cambios.


