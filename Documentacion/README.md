# Bank USAC — Índice de documentación

Este archivo es el índice maestro de la documentación. El contenido detallado se encuentra en el documento correspondiente a cada área; así se evita duplicar requisitos, diagramas y decisiones.

## Guías de uso y operación

- [Manual de usuario](manual-usuario.md): acceso, roles y flujos del frontend.
- [Manual técnico](manual-tecnico.md): configuración, ejecución local, Kubernetes, API y diagnóstico.
- [Estrategia de pruebas](pruebas/estrategia-pruebas.md): pruebas funcionales, mensajería, trazabilidad y despliegue.

## Análisis y requisitos

- [Requisitos funcionales](analisis/requisitos-funcionales.md): RF-01 a RF-40 y matriz de trazabilidad.
- [Requisitos no funcionales](analisis/requisitos-no-funcionales.md): RNF-01 a RNF-18 y restricciones técnicas.
- [Funcionalidades complementarias](analisis/funcionalidades-complementarias.md): alcance incorporado y funciones fuera del alcance inicial.

## Dominio

- [Modelo de dominio](dominio/modelo-dominio.md): entidades, atributos, estados y responsabilidades.
- [Contextos delimitados](dominio/contextos-delimitados.md): límites de los cinco microservicios y propiedad de datos.
- [ER de Customer Service](dominio/entidad-relacion/er-customer-service.md)
- [ER de Account Service](dominio/entidad-relacion/er-account-service.md)
- [ER de Transaction Service](dominio/entidad-relacion/er-transaction-service.md)
- [ER de Payment Service](dominio/entidad-relacion/er-payment-service.md)
- [ER de Notification & Audit Service](dominio/entidad-relacion/er-notification-audit.md)

## Comunicación y eventos

- [Catálogo de eventos](eventos/catalogo-eventos.md): routing keys, productores, consumidores y propósito.
- [Contratos de eventos](eventos/contratos-eventos.md): sobre común y payloads versionados.
- [Saga de transferencia](saga/saga-transferencia.md): estados, comandos, eventos y compensaciones.

## Arquitectura y despliegue

- [C4 — Contexto](arquitectura/c4-contexto.md)
- [C4 — Contenedores](arquitectura/c4-contenedores.md)
- [C4 — Componentes](arquitectura/c4-componentes.md)
- [Diagrama de despliegue](arquitectura/despliegue.md)
- [Decisiones arquitectónicas](decisiones/decisiones-arquitectura.md)

## UML y casos de uso

- [Casos de uso (CDU)](uml/casos-de-uso.md): catálogo, distribución, diagramas y especificaciones de los 18 CDU.
- [Matriz de diagramas](uml/matriz-diagramas.md): vistas requeridas por sistema y microservicio.
- [Secuencia de creación de cuenta](uml/secuencia-creacion-cuenta.md)
- [Secuencia de transferencia](uml/secuencia-transferencia.md)

El catálogo de CDU se mantiene dentro de `uml/casos-de-uso.md`. Cada caso conserva su ID, microservicio responsable, actor, RF cubiertos, diagrama de caso de uso y flujo expandido. Las validaciones que no son objetivos independientes se documentan como `include`, `extend` o escenarios alternativos.

## SRE y operación

- [SLI](sre/sli.md): indicadores medibles.
- [SLO](sre/slo.md): objetivos para los indicadores.
- [SLA](sre/sla.md): compromisos del entorno académico.

Las reglas de idempotencia, retries, errores y trazabilidad se describen en los documentos SRE y en el [Manual técnico](manual-tecnico.md), junto con su aplicación en RabbitMQ y los microservicios.

## Correspondencia con el código

| Área | Ubicación principal |
|---|---|
| API Gateway | `gateway/api-gateway/` |
| Customer Service | `services/service-customer/` |
| Account Service | `services/account-service/` |
| Transaction Service | `services/transaction-service/` |
| Payment Service | `services/payment-service/` |
| Notification & Audit Service | `services/service-notification-audit/` |
| Frontend | `frontend/bank-usac-web/` |
| Manifiestos Kubernetes | `infrastructure/kubernetes/` |
| Bases de datos y RabbitMQ | `docker-compose.yml` |

## Regla de mantenimiento

Cuando cambie un requisito, evento, entidad, diagrama o endpoint, se actualiza primero su archivo especializado y luego se verifica este índice. No se deben volver a copiar bloques completos al README; los enlaces anteriores son la fuente de navegación oficial.
