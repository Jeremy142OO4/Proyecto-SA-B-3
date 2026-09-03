# Decisiones de arquitectura

Este documento registra las decisiones arquitectónicas adoptadas para **Bank USAC**, las alternativas consideradas y sus consecuencias. Su objetivo es explicar por qué la solución fue construida de esta manera y mantener coherencia entre los requisitos, el código, los diagramas y el despliegue.

## Resumen

| ID | Decisión | Estado |
|---|---|---|
| ADR-01 | Dividir el dominio en cinco microservicios | Aceptada |
| ADR-02 | Implementar los servicios en Go con Fiber | Aceptada |
| ADR-03 | Utilizar RabbitMQ para la comunicación asíncrona | Aceptada |
| ADR-04 | Utilizar un API Gateway como único punto de entrada | Aceptada |
| ADR-05 | Mantener una base de datos PostgreSQL por microservicio | Aceptada |
| ADR-06 | Aplicar consistencia eventual, Outbox e idempotencia | Aceptada |
| ADR-07 | Coordinar transferencias mediante una Saga | Aceptada |
| ADR-08 | Desplegar la aplicación en Kubernetes y las bases fuera del clúster | Aceptada |
| ADR-09 | Implementar el frontend con React y servirlo con Nginx | Aceptada |
| ADR-10 | Representar montos monetarios en centavos enteros | Aceptada |
| ADR-11 | Mantener trazabilidad mediante `correlationId` | Aceptada |
| ADR-12 | Organizar el proyecto como monorepo | Aceptada |

## ADR-01 — División del dominio en cinco microservicios

**Estado:** Aceptada.

### Contexto

El sistema reúne capacidades de clientes, cuentas, transferencias, pagos, notificaciones y auditoría. Concentrarlas en una sola aplicación aumentaría el acoplamiento y dificultaría asignar la propiedad de los datos.

### Decisión

Separar el dominio en los siguientes servicios:

- `customer-service`: clientes, usuarios, autenticación y estados de usuario.
- `account-service`: cuentas, balances, movimientos y control de inactividad.
- `transaction-service`: transferencias y coordinación de su Saga.
- `payment-service`: pagos internos y externos.
- `notification-audit-service`: notificaciones y auditoría de eventos.

Cada servicio contiene su propia lógica de dominio, persistencia y consumidores de mensajes.

### Alternativas consideradas

- Construir un monolito con todos los módulos.
- Dividir el sistema en una cantidad mayor de servicios técnicos.

### Consecuencias

- Los dominios pueden evolucionar y desplegarse de forma independiente.
- Cada equipo puede trabajar sobre un límite funcional definido.
- Se incrementa la complejidad operativa y de coordinación distribuida.
- Los cambios que afectan varios dominios requieren eventos y consistencia eventual.

## ADR-02 — Go con Fiber para los servicios

**Estado:** Aceptada.

### Contexto

Los componentes necesitan consumir mensajes, exponer comprobaciones de salud y ejecutar procesos concurrentes con un consumo moderado de recursos. El equipo también necesita una estructura uniforme entre servicios.

### Decisión

Implementar el API Gateway y los cinco microservicios en **Go**, utilizando **Fiber** para las interfaces HTTP requeridas, como el API externo y los endpoints de salud. La organización interna se divide en configuración, controladores, modelos, repositorios, rutas, servicios, mensajería y acceso a datos.

Los nombres de carpetas se conservan en inglés para seguir convenciones comunes del ecosistema, mientras las variables, funciones, entidades y campos del dominio se expresan en español.

### Alternativas consideradas

- Implementar distintos microservicios en lenguajes diferentes.
- Utilizar otro framework HTTP de Go.

### Consecuencias

- Se reduce la variedad tecnológica que el equipo debe mantener.
- Las imágenes de los servicios son pequeñas y reproducibles.
- La concurrencia de Go facilita consumidores, publicadores y procesos periódicos.
- El equipo queda ligado a las convenciones y librerías del ecosistema Go.

## ADR-03 — RabbitMQ como broker de mensajería

**Estado:** Aceptada.

### Contexto

La comunicación de negocio entre microservicios debe ser completamente asíncrona. Se necesita intercambiar comandos, eventos y respuestas sin realizar llamadas HTTP o RPC directas entre servicios.

### Decisión

Utilizar **RabbitMQ** y el protocolo **AMQP** para toda comunicación de negocio entre el API Gateway y los microservicios, así como entre los propios dominios. Se utilizan exchanges, colas durables, reintentos y colas de mensajes muertos.

Los endpoints HTTP internos se reservan para comprobaciones operativas y no representan comunicación de negocio entre microservicios.

### Alternativas consideradas

- Apache Kafka.
- NATS.
- Comunicación HTTP síncrona entre servicios.

### Consecuencias

- Los productores quedan desacoplados de los consumidores.
- Un servicio puede procesar mensajes cuando vuelve a estar disponible.
- Es necesario manejar duplicados, reintentos, confirmaciones y mensajes fallidos.
- La respuesta al usuario puede ser diferida y requiere consultar el estado de la operación.

## ADR-04 — API Gateway como único punto de entrada

**Estado:** Aceptada.

### Contexto

Exponer todos los microservicios directamente al frontend trasladaría al cliente la responsabilidad de conocer direcciones, protocolos y contratos internos.

### Decisión

Utilizar un **API Gateway** como frontera del backend. El frontend envía sus solicitudes HTTP únicamente al Gateway. Este valida la solicitud, aplica autenticación y autorización cuando corresponde, publica el comando en RabbitMQ y permite consultar el resultado de las operaciones asíncronas.

### Alternativas consideradas

- Permitir que el frontend invoque directamente cada servicio.
- Crear un backend específico por cada tipo de cliente.

### Consecuencias

- El frontend utiliza una interfaz externa unificada.
- Los microservicios no necesitan exponerse públicamente.
- Las políticas transversales pueden aplicarse en un solo punto.
- El Gateway debe mantenerse sin lógica propia de los dominios.
- Su disponibilidad influye en todas las solicitudes externas.

## ADR-05 — Base de datos PostgreSQL por microservicio

**Estado:** Aceptada.

### Contexto

Compartir una sola base de datos permitiría que un servicio consulte o modifique tablas de otro, generando acoplamiento y dificultando la evolución independiente.

### Decisión

Asignar una instancia o base PostgreSQL independiente a cada microservicio. Cada servicio puede acceder únicamente a sus propias tablas:

| Servicio | Base o dominio de datos |
|---|---|
| `customer-service` | Clientes y usuarios |
| `account-service` | Cuentas y movimientos |
| `transaction-service` | Transferencias y Saga |
| `payment-service` | Pagos e intentos |
| `notification-audit-service` | Notificaciones y auditoría |

Los identificadores externos se conservan como referencias lógicas y no como claves foráneas entre bases de datos.

### Alternativas consideradas

- Una base de datos compartida por todos los servicios.
- Un esquema diferente por servicio dentro de la misma base compartida.

### Consecuencias

- Cada servicio es propietario de sus datos y su esquema.
- Se evita el acoplamiento mediante consultas o transacciones compartidas.
- No pueden utilizarse `JOIN` ni claves foráneas entre dominios.
- La información distribuida se sincroniza mediante eventos.

## ADR-06 — Consistencia eventual, Outbox e idempotencia

**Estado:** Aceptada.

### Contexto

Una operación distribuida puede modificar datos y publicar eventos. Si ambas acciones se realizan por separado, un fallo intermedio puede guardar el cambio sin publicar el evento o publicar un evento sin completar la operación.

### Decisión

Aplicar el patrón **Transactional Outbox**. Cada servicio guarda el cambio de negocio y el mensaje de salida dentro de la misma transacción local. Un publicador procesa posteriormente la tabla `mensajes_salida` y entrega el evento a RabbitMQ.

Los consumidores registran los mensajes atendidos en `mensajes_procesados`. La combinación del identificador del mensaje y el consumidor permite reconocer entregas repetidas y evitar efectos duplicados.

### Alternativas consideradas

- Publicar directamente en RabbitMQ después de guardar en la base de datos.
- Utilizar transacciones distribuidas de dos fases.
- Suponer que cada mensaje será entregado una sola vez.

### Consecuencias

- Los cambios locales y la intención de publicar permanecen consistentes.
- Los mensajes pueden reintentarse sin duplicar movimientos o pagos.
- La información entre servicios converge de manera eventual, no inmediata.
- Se requieren tablas técnicas, publicadores periódicos y limpieza controlada de registros.

## ADR-07 — Saga para transferencias bancarias

**Estado:** Aceptada.

### Contexto

Una transferencia afecta más de una cuenta y puede involucrar varios mensajes. No existe una transacción ACID que abarque las bases independientes de los microservicios.

### Decisión

Utilizar una **Saga orquestada** por `transaction-service`. El orquestador mantiene el estado de la transferencia, solicita el débito y el crédito, procesa las respuestas y ejecuta una compensación cuando una etapa posterior falla después de haberse aplicado un cambio previo.

### Alternativas consideradas

- Transacción distribuida entre bases de datos.
- Saga coreografiada exclusivamente mediante eventos.
- Llamadas síncronas entre los servicios participantes.

### Consecuencias

- El flujo exitoso, los rechazos y las compensaciones quedan explícitos.
- `transaction-service` proporciona un punto central para consultar el estado.
- Las operaciones pueden permanecer temporalmente en procesamiento.
- El orquestador y las acciones compensatorias aumentan la complejidad del dominio.

## ADR-08 — Kubernetes para la aplicación y bases externas al clúster

**Estado:** Aceptada.

### Contexto

El proyecto debe demostrar contenerización, orquestación, networking, configuración y comprobaciones de salud. También exige que las bases de datos se ejecuten fuera del clúster de Kubernetes.

### Decisión

Desplegar el frontend, API Gateway, RabbitMQ y los cinco microservicios en un clúster local **Minikube**, dentro del namespace `bank-usac`. Ejecutar las cinco bases PostgreSQL mediante **Docker Compose** fuera del clúster.

Cada componente de aplicación posee su propio Dockerfile y manifiesto de Kubernetes. La configuración se proporciona mediante ConfigMaps y Secrets, y se definen comprobaciones de salud para los componentes desplegados.

### Alternativas consideradas

- Ejecutar toda la solución únicamente con Docker Compose.
- Ejecutar también las bases de datos dentro de Kubernetes.
- Utilizar directamente un proveedor de nube.

### Consecuencias

- El ambiente local reproduce conceptos de despliegue usados en producción.
- Los componentes pueden reiniciarse y escalarse de forma independiente.
- Las bases conservan un ciclo de vida separado del clúster.
- La ejecución requiere coordinar Docker Compose, Minikube y la conectividad hacia el host.

## ADR-09 — React con Nginx para el frontend

**Estado:** Aceptada.

### Contexto

El proyecto necesita una interfaz web que permita demostrar las funcionalidades de los cinco microservicios sin exponer al usuario la comunicación asíncrona interna.

### Decisión

Construir una aplicación de página única con **React** y servir sus archivos estáticos mediante **Nginx** dentro de un contenedor. El frontend se organiza por funcionalidades y consume únicamente la interfaz HTTP del API Gateway.

### Alternativas consideradas

- Interfaz desarrollada sin framework.
- Renderizado de páginas desde el backend.
- Otro framework de frontend.

### Consecuencias

- La interfaz se organiza mediante componentes reutilizables.
- Nginx entrega eficientemente los recursos compilados.
- Se necesita gestionar el estado de autenticación y el JWT en el navegador.
- El frontend debe contemplar que las operaciones asíncronas pueden tardar en completarse.

## ADR-10 — Montos representados en centavos enteros

**Estado:** Aceptada.

### Contexto

Los cálculos financieros no deben depender de números de punto flotante porque pueden producir errores de precisión.

### Decisión

Representar los montos y balances mediante enteros de 64 bits expresados en centavos, por ejemplo `Q50.00` como `5000`. La moneda inicial del sistema es `GTQ`.

### Alternativas consideradas

- Utilizar números de punto flotante.
- Utilizar tipos decimales en todas las capas.

### Consecuencias

- Las operaciones aritméticas son exactas para la unidad mínima definida.
- Se evitan errores de redondeo binario.
- La API y el frontend deben convertir correctamente entre valores visibles y centavos.
- Una futura compatibilidad con monedas de distinta cantidad de decimales requeriría ampliar el modelo.

## ADR-11 — Trazabilidad mediante correlationId

**Estado:** Aceptada.

### Contexto

Una operación atraviesa el API Gateway, RabbitMQ y varios consumidores. Los registros aislados de cada componente no son suficientes para reconstruir el flujo completo.

### Decisión

Propagar un `correlationId` en los comandos, eventos, respuestas, registros de auditoría y logs relacionados con una misma operación. Cada mensaje también conserva su propio identificador para idempotencia.

### Alternativas consideradas

- Rastrear únicamente mediante el identificador propio de cada entidad.
- Depender de la hora de los logs para relacionar mensajes.

### Consecuencias

- Es posible reconstruir una operación distribuida de extremo a extremo.
- Facilita el diagnóstico, la auditoría y la medición de indicadores.
- Todos los productores y consumidores deben conservar el identificador sin reemplazarlo.

## ADR-12 — Organización como monorepo

**Estado:** Aceptada.

### Contexto

El frontend, API Gateway, microservicios, infraestructura y documentación evolucionan de manera coordinada durante el proyecto académico.

### Decisión

Mantener todos los componentes en un único repositorio, separados en directorios para servicios, frontend, infraestructura y documentación. Los contratos y diagramas se actualizan junto con la implementación que describen.

### Alternativas consideradas

- Un repositorio independiente por microservicio.
- Un repositorio para código y otro para documentación e infraestructura.

### Consecuencias

- La entrega y revisión integral del proyecto son más sencillas.
- Los cambios que afectan varios componentes pueden coordinarse en una sola versión.
- Es necesario mantener límites claros para evitar dependencias accidentales entre servicios.
- El repositorio aumenta de tamaño conforme se incorporan imágenes y documentación.

## Criterio de actualización

Una decisión deberá modificarse o sustituirse cuando cambie un requisito, cuando la implementación deje de coincidir con lo documentado o cuando aparezca una alternativa con beneficios suficientes para justificar el costo de migración. Si una decisión es reemplazada, debe conservarse su registro histórico indicando el nuevo estado y la decisión que la sustituye.
