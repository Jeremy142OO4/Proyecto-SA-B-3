# Matriz de diagramas

Esta matriz conserva la planificación visual del proyecto y sirve como lista de verificación para la entrega.

| Alcance | Arquitectura / C4 | Clases | Entidad-relación | Casos de uso | Secuencia | Despliegue |
|---|---:|---:|---:|---:|---:|---:|
| Sistema completo | Sí | No | No | No | Sí, flujos end-to-end | Sí |
| Customer Service | Componentes | Sí | Sí | Uno por funcionalidad | Sí | Incluido en el general |
| Account Service | Componentes | Sí | Sí | Uno por funcionalidad | Sí | Incluido en el general |
| Transaction Service | Componentes | Sí | Sí | Uno por funcionalidad | Sí | Incluido en el general |
| Payment Service | Componentes | Sí | Sí | Uno por funcionalidad | Sí | Incluido en el general |
| Notification & Audit Service | Componentes | Sí | Sí | Uno por funcionalidad | Sí | Incluido en el general |

## Criterios de elaboración

- Los diagramas generales muestran las interacciones que atraviesan varios microservicios.
- Cada microservicio conserva sus diagramas de componentes, clases y entidad-relación.
- Los CDU se documentan en [casos-de-uso.md](casos-de-uso.md), con un caso por funcionalidad del catálogo.
- Las secuencias generales requeridas son creación de cuenta y transferencia bancaria.
- El despliegue muestra Kubernetes para la aplicación y Docker Compose para las bases de datos.

