# Requisitos no funcionales y restricciones

| ID | Requisito |
|---|---|
| RNF-01 | Exactamente cinco microservicios autónomos y separados por responsabilidad. |
| RNF-02 | Comunicación entre microservicios 100 % asíncrona; sin HTTP, RPC síncrono ni acceso directo entre servicios. |
| RNF-03 | Utilizar RabbitMQ, Kafka, NATS o gRPC asíncrono como mecanismo de mensajería. |
| RNF-04 | Cada microservicio posee y accede únicamente a su propia base de datos. |
| RNF-05 | Las bases de datos se ejecutan fuera de Kubernetes como contenedores Docker o servicios locales. |
| RNF-06 | Consistencia eventual, sin transacciones ACID distribuidas. |
| RNF-07 | Consumidores idempotentes para evitar duplicados. |
| RNF-08 | Retries limitados con backoff y dead-letter queue para mensajes no recuperables. |
| RNF-09 | Mantener el mismo `correlationId` durante todo el flujo. |
| RNF-10 | Diferenciar validaciones, fallos transitorios, fallos permanentes y timeouts; compensar cambios parciales. |
| RNF-11 | Un consumidor fallido no debe bloquear productores ni servicios no relacionados. |
| RNF-12 | Permitir escalamiento horizontal en Kubernetes sin depender de sesión local. |
| RNF-13 | Exigir JWT, autorización por rol y propiedad del recurso; almacenar contraseñas con hashing. |
| RNF-14 | No exponer contraseñas, tokens, secretos ni DPI completos en logs o respuestas. |
| RNF-15 | Cada servicio y el Gateway deben tener una imagen Docker reproducible y configurable. |
| RNF-16 | Desplegar microservicios, Gateway y broker en Kubernetes con configuración y health checks. |
| RNF-17 | Producir logs estructurados y datos para SLI, SLO y SLA. |
| RNF-18 | Mantener un monorepo consistente entre código, contratos, diagramas, casos de uso y despliegue. |

