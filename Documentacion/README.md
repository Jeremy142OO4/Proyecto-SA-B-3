# Bank USAC - Documentación del proyecto

Documento maestro de análisis, arquitectura, implementación, pruebas y operación de **Bank USAC**. Este archivo se actualizará durante el proyecto para mantener consistencia entre los requisitos, los diagramas y el código.



## Índice

- [Manual técnico](manual-tecnico.md)

- [Análisis](#análisis)
  - [Requisitos funcionales](#requisitos-funcionales)
  - [Matriz básica de trazabilidad](#matriz-básica-de-trazabilidad)
  - [Requisitos no funcionales y restricciones](#requisitos-no-funcionales-y-restricciones)
  - [Funcionalidades complementarias](#funcionalidades-complementarias)
- [Definición del dominio](#definición-del-dominio)
  - [Definición de los cinco microservicios](#definición-de-los-cinco-microservicios)
  - [Responsabilidades de los microservicios](#responsabilidades-de-los-microservicios)
  - [Entidades y datos](#entidades-y-datos)
  - [Bases de datos por microservicio](#bases-de-datos-por-microservicio)
  - [Acciones por tipo de usuario](#acciones-por-tipo-de-usuario)
- [Comunicación entre microservicios](#comunicación-entre-microservicios)
  - [Mecanismo de mensajería](#mecanismo-de-mensajería)
  - [Modelo de comunicación](#modelo-de-comunicación)
  - [Catálogo de eventos](#catálogo-de-eventos)
  - [Topics, queues o subjects](#topics-queues-o-subjects)
- [Flujos del sistema](#flujos-del-sistema)
  - [Creación de cliente](#flujo-de-creación-de-cliente)
  - [Activación por correo](#activación-del-cliente-por-correo)
  - [Autenticación JWT](#autenticación-mediante-jwt)
  - [Creación de cuentas](#flujo-de-creación-de-cuentas)
  - [Consulta de saldo](#flujo-de-consulta-de-saldo)
  - [Transferencias](#flujo-de-transferencia-entre-cuentas)
  - [Pagos](#flujo-de-procesamiento-de-pagos)
  - [Notificaciones y auditoría](#flujo-de-notificaciones-y-auditoría)
  - [Desactivación de cuentas](#desactivación-automática-de-cuentas-inactivas)
- [Saga](#saga)
  - [Saga de transferencia bancaria](#saga-de-transferencia-bancaria)
- [Diagramas](#diagramas)
  - [Diagramas generales del sistema](#diagramas-generales-del-sistema)
  - [Diagramas por microservicio](#diagramas-por-microservicio)
  - [Catálogo de diagramas de casos de uso](#catálogo-de-diagramas-de-casos-de-uso)
  - [Distribución entre tres integrantes](#distribución-entre-tres-integrantes)
  - [Matriz de diagramas](#matriz-de-diagramas)
- [SRE](#sre)
  - [SLI](#sli)
  - [SLO](#slo)
  - [SLA](#sla)
  - [Idempotencia](#manejo-de-idempotencia)
  - [Retries](#estrategia-de-retries)
  - [Errores](#manejo-de-errores)
  - [Trazabilidad](#trazabilidad-mediante-correlationid)
- [Preparación del código](#preparación-del-código)
  - [Monorepo](#estructura-del-monorepo)
  - [Proyectos base](#proyectos-base-de-los-microservicios)
  - [API Gateway](#api-gateway)
  - [Conexión con el broker](#conexión-con-el-broker)
  - [Bases de datos](#bases-de-datos-independientes)
- [Desarrollo](#desarrollo)
  - [Customer Service](#customer-service)
  - [Account Service](#account-service)
  - [Transaction Service](#transaction-service)
  - [Payment Service](#payment-service)
  - [Notification & Audit Service](#notification--audit-service)
  - [API Gateway](#implementación-del-api-gateway)
  - [Publicación de eventos](#publicación-de-eventos)
  - [Consumo de eventos](#consumo-de-eventos)
  - [Saga](#implementación-de-saga)
  - [Frontend](#frontend)
- [Contenedores](#contenedores)
  - [Dockerfiles](#dockerfiles)
  - [Validación de componentes](#validación-de-componentes)
- [Kubernetes](#kubernetes)
  - [Cluster local](#cluster-local)
  - [Configuraciones](#configuraciones-de-kubernetes)
  - [Microservicios](#despliegue-de-los-microservicios)
  - [API Gateway](#despliegue-del-api-gateway)
  - [Broker](#despliegue-del-broker)
  - [Bases de datos externas](#bases-de-datos-fuera-del-cluster)
- [Estrategia de pruebas](#estrategia-de-pruebas)
- [Cierre documental](#cierre-documental)


## Análisis

### Requisitos funcionales

Los requisitos RF-01 a RF-30 traducen las capacidades y restricciones funcionales principales del enunciado. Los requisitos RF-31 a RF-40 corresponden a las funciones complementarias seleccionadas para cerrar los flujos bancarios. Cada fila representa un comportamiento verificable e independiente; la justificación de las funciones añadidas se presenta en [Funcionalidades complementarias](#funcionalidades-complementarias).

| ID | Requisito funcional |
|---|---|
| **RF-01** | El sistema debe permitir **registrar clientes**. |
| **RF-02** | El sistema debe permitir **validar la identidad del cliente mediante una fotografía de su documento de identificación**. |
| **RF-03** | El sistema debe permitir **actualizar los datos del cliente**. |
| **RF-04** | El sistema debe permitir **activar al usuario durante su primer inicio de sesión mediante un enlace enviado por correo electrónico**. |
| **RF-05** | El sistema debe permitir **autenticar y validar usuarios mediante JWT**. |
| **RF-06** | El sistema debe **validar la coherencia de la fecha de nacimiento del cliente**. |
| **RF-07** | El sistema debe **generar un `username` a partir del nombre del cliente**. |
| **RF-08** | El sistema debe permitir **crear cuentas bancarias asociadas a un cliente**. |
| **RF-09** | El sistema debe permitir **crear cuentas monetarias y cuentas de ahorro**. |
| **RF-10** | El sistema debe permitir **consultar el saldo de una cuenta bancaria**. |
| **RF-11** | El sistema debe permitir **actualizar el balance de una cuenta como resultado de una operación financiera válida**. |
| **RF-12** | El sistema debe **desactivar automáticamente las cuentas con balance menor a Q50.00 después de seis meses de inactividad**. |
| **RF-13** | El sistema debe permitir **realizar transferencias entre cuentas**. |
| **RF-14** | El sistema debe **validar que la cuenta de origen posea fondos suficientes antes de realizar una transferencia**. |
| **RF-15** | El sistema debe permitir **registrar las transacciones financieras y su estado**. |
| **RF-16** | El sistema debe **ejecutar las transferencias mediante una Saga que contemple el flujo exitoso, los fallos y las compensaciones**. |
| **RF-17** | El sistema debe permitir **procesar pagos internos o externos**. |
| **RF-18** | El sistema debe **validar una operación financiera antes de confirmar el pago**. |
| **RF-19** | El sistema debe permitir **registrar cada pago y su estado**. |
| **RF-20** | El sistema debe **manejar fallos durante la interacción con sistemas de pago externos**. |
| **RF-21** | El sistema debe permitir **enviar notificaciones relacionadas con las operaciones bancarias**. |
| **RF-22** | El sistema debe permitir **registrar eventos para fines de auditoría**. |
| **RF-23** | El sistema debe conservar en cada registro de auditoría **el identificador, tipo de evento, fecha y hora, y payload**. |
| **RF-24** | El sistema debe permitir **relacionar los eventos de una operación mediante `correlationId`**. |
| **RF-25** | El sistema debe reconocer los roles **Administrador, Cajero Receptor y Cliente**. |
| **RF-26** | El sistema debe aplicar **las funciones y permisos correspondientes al rol del usuario autenticado**. |
| **RF-27** | El sistema debe recibir **todas las solicitudes externas mediante el API Gateway**. |
| **RF-28** | El API Gateway debe permitir **iniciar flujos de negocio mediante el mecanismo de mensajería asíncrona**. |
| **RF-29** | El sistema debe permitir **consultar el resultado de las operaciones procesadas de forma asíncrona**. |
| **RF-30** | El sistema debe proporcionar **una interfaz intuitiva, agradable y funcional para validar las funcionalidades de los cinco microservicios**. |
| **RF-31** | El sistema debe **impedir el registro duplicado de documento de identificación, correo electrónico y `username`**. |
| **RF-32** | El enlace de activación debe ser **de un solo uso, tener expiración y permitir un reenvío controlado mientras el usuario siga pendiente**. |
| **RF-33** | El sistema debe manejar los estados de usuario **pendiente de activación, activo y bloqueado**. |
| **RF-34** | El sistema debe **validar que el cliente exista y esté activo antes de crearle una cuenta bancaria**. |
| **RF-35** | El sistema debe manejar los estados de cuenta **activa, inactiva, bloqueada y cerrada**, y registrar su última actividad financiera. |
| **RF-36** | El sistema debe **rechazar débitos sobre cuentas no activas e impedir que el balance quede negativo**. |
| **RF-37** | El sistema debe validar que una transferencia tenga **monto mayor que cero, cuentas diferentes y cuentas habilitadas**. |
| **RF-38** | El sistema debe manejar los estados de transacción **pendiente, procesando, completada, rechazada y compensada**. |
| **RF-39** | El sistema debe registrar para cada pago **beneficiario, concepto, monto, tipo y estado**, y permitir consultar su resultado. |
| **RF-40** | El cliente debe poder **consultar el historial de sus propias transacciones y pagos**. |

#### Matriz básica de trazabilidad

Esta primera versión asigna cada requisito a un responsable, un actor y un caso de uso. Las entidades, eventos, diagramas y pruebas se incorporarán cuando esas partes del diseño estén definidas.

| Requisito | Componente responsable | Actor principal | Caso de uso relacionado |
|---|---|---|---|
| **RF-01** | Customer Service | Cajero Receptor | Registrar cliente |
| **RF-02** | Customer Service | Cajero Receptor | Validar identidad del cliente |
| **RF-03** | Customer Service | Cliente / Cajero Receptor | Actualizar datos del cliente |
| **RF-04** | Customer Service | Cliente | Activar usuario |
| **RF-05** | Customer Service | Administrador / Cajero Receptor / Cliente | Iniciar sesión |
| **RF-06** | Customer Service | Cajero Receptor | Registrar cliente |
| **RF-07** | Customer Service | Sistema | Generar nombre de usuario |
| **RF-08** | Account Service | Cajero Receptor | Crear cuenta bancaria |
| **RF-09** | Account Service | Cajero Receptor | Seleccionar tipo de cuenta |
| **RF-10** | Account Service | Cliente / Cajero Receptor | Consultar saldo |
| **RF-11** | Account Service | Sistema | Actualizar balance |
| **RF-12** | Account Service | Sistema | Desactivar cuenta inactiva |
| **RF-13** | Transaction Service | Cliente / Cajero Receptor | Realizar transferencia |
| **RF-14** | Transaction Service | Sistema | Validar fondos |
| **RF-15** | Transaction Service | Sistema | Registrar transacción |
| **RF-16** | Transaction Service | Sistema | Ejecutar Saga de transferencia |
| **RF-17** | Payment Service | Cliente / Cajero Receptor | Procesar pago |
| **RF-18** | Payment Service | Sistema | Validar pago |
| **RF-19** | Payment Service | Sistema | Registrar pago |
| **RF-20** | Payment Service | Sistema externo / Sistema | Manejar fallo de pago externo |
| **RF-21** | Notification & Audit Service | Sistema | Enviar notificación |
| **RF-22** | Notification & Audit Service | Administrador | Registrar eventos de auditoría |
| **RF-23** | Notification & Audit Service | Sistema | Conservar detalle del evento |
| **RF-24** | Todos los microservicios | Sistema | Trazar una operación distribuida |
| **RF-25** | Customer Service | Administrador / Cajero Receptor / Cliente | Identificar rol del usuario |
| **RF-26** | API Gateway y servicio propietario | Administrador / Cajero Receptor / Cliente | Autorizar operación |
| **RF-27** | API Gateway | Administrador / Cajero Receptor / Cliente | Acceder al sistema |
| **RF-28** | API Gateway | Administrador / Cajero Receptor / Cliente | Iniciar operación asíncrona |
| **RF-29** | API Gateway y servicio propietario | Administrador / Cajero Receptor / Cliente | Consultar estado de operación |
| **RF-30** | Frontend | Administrador / Cajero Receptor / Cliente | Utilizar funciones del sistema |
| **RF-31** | Customer Service | Cajero Receptor | Validar unicidad del cliente |
| **RF-32** | Customer Service y Notification & Audit Service | Cliente | Activar usuario / Reenviar activación |
| **RF-33** | Customer Service | Administrador / Cliente | Gestionar estado del usuario |
| **RF-34** | Account Service | Cajero Receptor | Validar cliente para crear cuenta |
| **RF-35** | Account Service | Sistema / Administrador | Gestionar estado y actividad de cuenta |
| **RF-36** | Account Service | Sistema | Validar débito y proteger balance |
| **RF-37** | Transaction Service | Cliente / Cajero Receptor | Validar transferencia |
| **RF-38** | Transaction Service | Sistema | Actualizar estado de transacción |
| **RF-39** | Payment Service | Cliente / Cajero Receptor | Registrar y consultar pago |
| **RF-40** | Transaction Service y Payment Service | Cliente | Consultar historial financiero |




### Requisitos no funcionales y restricciones

| ID | Requisito no funcional o restricción |
|---|---|
| **RNF-01** | El sistema debe estar compuesto por **exactamente cinco microservicios autónomos**, con responsabilidades claramente separadas. |
| **RNF-02** | La comunicación entre microservicios debe ser **100 % asíncrona**; no se permiten llamadas HTTP, RPC síncrono ni comunicación directa entre servicios. |
| **RNF-03** | El sistema debe utilizar **Kafka, RabbitMQ, NATS o gRPC en modalidad asíncrona** como mecanismo de mensajería. |
| **RNF-04** | Cada microservicio debe poseer **su propia base de datos y acceder únicamente a ella**. |
| **RNF-05** | Las bases de datos deben ejecutarse **fuera del cluster de Kubernetes**, como contenedores Docker o servicios locales. |
| **RNF-06** | Los cambios distribuidos deben manejarse mediante **consistencia eventual**, sin depender de transacciones ACID entre microservicios. |
| **RNF-07** | Los consumidores deben ser **idempotentes** para que un evento repetido no duplique operaciones ni altere dos veces el balance. |
| **RNF-08** | Los fallos transitorios deben manejarse mediante **retries limitados y backoff**, trasladando los mensajes no recuperables a una dead-letter queue o mecanismo equivalente. |
| **RNF-09** | Todo comando, evento y operación derivada debe conservar el mismo **`correlationId`** durante el flujo completo. |
| **RNF-10** | Los flujos deben diferenciar **errores de validación, fallos transitorios, fallos permanentes y timeouts**, y aplicar compensaciones cuando existan cambios parciales. |
| **RNF-11** | El fallo de un consumidor no debe bloquear al productor ni detener servicios no relacionados; los componentes deben **recuperarse sin perder ni duplicar operaciones financieras**. |
| **RNF-12** | Los microservicios deben permitir **escalamiento horizontal en Kubernetes** y no depender de estado de sesión almacenado únicamente en una instancia. |
| **RNF-13** | Las operaciones protegidas deben exigir **JWT y autorización por rol y propiedad del recurso**; las contraseñas deben almacenarse mediante hashing seguro. |
| **RNF-14** | Los logs, eventos y respuestas no deben exponer **contraseñas, tokens, secretos ni documentos de identificación completos**. |
| **RNF-15** | Cada microservicio y el API Gateway deben contar con **una imagen Docker reproducible y configurable por entorno**. |
| **RNF-16** | Los cinco microservicios, API Gateway y broker deben desplegarse en **un cluster local de Kubernetes** con networking, configuración y comprobaciones de salud. |
| **RNF-17** | Los servicios deben producir **logs estructurados y datos para medir SLI, SLO y SLA**, permitiendo reconstruir una operación mediante `correlationId`. |
| **RNF-18** | El código debe organizarse como **monorepo** y mantener consistencia entre contratos de eventos, diagramas C4 y UML, casos de uso, despliegue e implementación. |

### Funcionalidades complementarias

El enunciado solicita completar las capacidades necesarias para que el flujo bancario tenga sentido. Después de analizar su impacto, se incorporaron únicamente las funciones indispensables para preservar identidad, autorización, integridad de saldos y seguimiento de operaciones.

#### Funciones incorporadas al alcance

| Área | Decisión incorporada | Requisitos relacionados | Justificación |
|---|---|---|---|
| Identidad | Unicidad de documento, correo y `username` | RF-31 | Evita clientes y credenciales ambiguas. |
| Activación | Enlace de un solo uso, expiración y reenvío controlado | RF-32 | Evita activaciones indefinidas o reutilizadas. |
| Usuarios | Estados pendiente, activo y bloqueado | RF-33 | Permite controlar activación y acceso. |
| Autorización | Permisos según rol y propiedad del recurso | RF-25 y RF-26 | Evita que un cliente opere cuentas ajenas. |
| Cuentas | Validación del cliente antes de crear una cuenta | RF-34 | Evita cuentas sin propietario válido. |
| Cuentas | Estados, última actividad y protección del balance | RF-35 y RF-36 | Hace posible la desactivación y evita saldos negativos. |
| Transferencias | Validaciones mínimas y estados del flujo | RF-37 y RF-38 | Define cuándo una transferencia es válida y trazable. |
| Pagos | Datos mínimos, estados y consulta del resultado | RF-39 | Permite procesar y rastrear pagos asíncronos. |
| Historial | Consulta de transacciones y pagos propios | RF-40 | Permite al cliente verificar sus operaciones. |

Los montos se manejarán inicialmente en **GTQ** mediante un tipo decimal exacto. Esta es una regla del modelo de datos y no requiere un requisito funcional adicional.

#### Funciones fuera del alcance inicial

La recuperación de contraseña, las búsquedas avanzadas de auditoría, la administración manual de intentos de notificación, las políticas configurables de retención y el soporte multimoneda se consideran mejoras posteriores. No son necesarias para demostrar el flujo principal exigido y añadirlas aumentaría el dominio, los eventos y las pruebas sin aportar puntuación directa a los criterios obligatorios.


## Definición del dominio

### Definición de los cinco microservicios


### Responsabilidades de los microservicios


### Entidades y datos


### Bases de datos por microservicio


### Acciones por tipo de usuario



## Comunicación entre microservicios

### Mecanismo de mensajería


### Modelo de comunicación


### Catálogo de eventos


### Topics, queues o subjects



## Flujos del sistema

### Flujo de creación de cliente


### Activación del cliente por correo


### Autenticación mediante JWT


### Flujo de creación de cuentas


### Flujo de consulta de saldo


### Flujo de transferencia entre cuentas


### Flujo de procesamiento de pagos


### Flujo de notificaciones y auditoría


### Desactivación automática de cuentas inactivas



## Saga

### Saga de transferencia bancaria



## Diagramas

Los diagramas Mermaid de contexto C4, componentes y secuencias end-to-end se encuentran en [DIAGRAMAS_SECUENCIA_C4.md](DIAGRAMAS_SECUENCIA_C4.md).

Los diagramas se dividirán entre vistas generales del sistema y vistas particulares de cada microservicio. Todos deberán utilizar los mismos nombres de componentes, actores, eventos, entidades y relaciones definidos en esta documentación.

### Diagramas generales del sistema

Estos diagramas representan Bank USAC como un sistema distribuido completo y las interacciones end-to-end que atraviesan varios microservicios.

#### Diagrama de arquitectura general


Debe mostrar el frontend, API Gateway, broker, cinco microservicios, bases de datos independientes, servicios externos y límites del cluster de Kubernetes.

#### Diagrama C4 - Contexto


Debe representar el sistema como una caja negra, sus tres tipos de usuario, los sistemas externos y el API Gateway como punto de entrada.

#### Diagrama C4 - Contenedores


Debe mostrar los cinco microservicios, API Gateway, broker, bases de datos externas y la comunicación asíncrona.

#### Diagrama UML - Secuencia de creación de cuenta


Será un diagrama general end-to-end porque el flujo involucra al API Gateway, broker y más de un microservicio.

#### Diagrama UML - Secuencia de transferencia bancaria


Será un diagrama general end-to-end e incluirá la Saga, los eventos, `correlationId`, el flujo exitoso y las compensaciones.

#### Diagrama UML - Despliegue


Debe mostrar el cluster local, pods, API Gateway, broker y bases de datos fuera del cluster.

### Diagramas por microservicio

Cada uno de los cinco microservicios tendrá su propia documentación visual:

- **C4 de componentes:** handlers, servicios de dominio, publicadores, consumidores, repositorios y adaptadores.
- **Diagrama de clases:** estructura interna, responsabilidades y relaciones del modelo de dominio.
- **Diagrama entidad-relación:** únicamente las tablas o colecciones pertenecientes a la base de datos del microservicio.
- **Diagrama de casos de uso:** se elaborará uno por cada funcionalidad identificada, dentro del límite del microservicio responsable.
- **Diagramas de secuencia internos:** procesamiento de comandos y eventos relevantes dentro del límite del servicio.

Se elaborará un CDU independiente por funcionalidad y se organizará dentro del microservicio responsable. No se elaborará un CDU general del sistema. Los requisitos que sean validaciones o pasos obligatorios de una misma funcionalidad podrán representarse mediante relaciones `include` o dentro del flujo del caso de uso, sin crear diagramas duplicados.

#### Catálogo de diagramas de casos de uso

El alcance contempla **18 CDU**. Las validaciones, el registro interno de estados y el manejo de errores se representarán como relaciones `include`, `extend` o flujos alternos del CDU principal cuando no constituyan un objetivo independiente para el actor.

| ID | Microservicio | Funcionalidad del CDU | Actor principal | RF cubiertos |
|---|---|---|---|---|
| **CDU-CUS-01** | Customer Service | Registrar cliente | Cajero Receptor | RF-01, RF-02, RF-06, RF-07 y RF-31 |
| **CDU-CUS-02** | Customer Service | Actualizar datos del cliente | Cliente / Cajero Receptor | RF-03 |
| **CDU-CUS-03** | Customer Service | Activar usuario | Cliente | RF-04, RF-32 y RF-33 |
| **CDU-CUS-04** | Customer Service | Iniciar sesión | Administrador / Cajero Receptor / Cliente | RF-05, RF-25 y RF-26 |
| **CDU-CUS-05** | Customer Service | Gestionar estado del usuario | Administrador | RF-33 |
| **CDU-ACC-01** | Account Service | Crear cuenta bancaria | Cajero Receptor | RF-08, RF-09 y RF-34 |
| **CDU-ACC-02** | Account Service | Consultar saldo | Cliente / Cajero Receptor | RF-10 |
| **CDU-ACC-03** | Account Service | Actualizar balance | Sistema | RF-11 y RF-36 |
| **CDU-ACC-04** | Account Service | Desactivar cuenta inactiva | Sistema | RF-12 y RF-35 |
| **CDU-ACC-05** | Account Service | Consultar estado de cuenta | Cliente / Cajero Receptor / Administrador | RF-35 |
| **CDU-TRX-01** | Transaction Service | Realizar transferencia | Cliente / Cajero Receptor | RF-13, RF-14, RF-15 y RF-37 |
| **CDU-TRX-02** | Transaction Service | Ejecutar Saga de transferencia | Sistema | RF-16 y RF-38 |
| **CDU-TRX-03** | Transaction Service | Consultar estado de transferencia | Cliente / Cajero Receptor | RF-29 y RF-38 |
| **CDU-TRX-04** | Transaction Service | Consultar historial de transacciones | Cliente | RF-40 |
| **CDU-PAY-01** | Payment Service | Procesar pago | Cliente / Cajero Receptor | RF-17, RF-18, RF-19, RF-20 y RF-39 |
| **CDU-PAY-02** | Payment Service | Consultar pagos | Cliente / Cajero Receptor | RF-29, RF-39 y RF-40 |
| **CDU-NOT-01** | Notification & Audit Service | Enviar notificación | Sistema | RF-21 y RF-32 |
| **CDU-NOT-02** | Notification & Audit Service | Registrar evento de auditoría | Sistema | RF-22, RF-23 y RF-24 |

#### Distribución entre tres integrantes

La distribución mezcla microservicios para mantener una carga de seis CDU por persona. Cada archivo deberá conservar el ID indicado para facilitar integración y revisión.

| Integrante | CDU asignados | Total |
|---|---|---:|
| **Integrante 1** | CDU-CUS-01, CDU-CUS-03, CDU-CUS-05, CDU-ACC-01, CDU-TRX-01 y CDU-NOT-01 | 6 |
| **Integrante 2** | CDU-CUS-02, CDU-CUS-04, CDU-ACC-02, CDU-ACC-03, CDU-ACC-04 y CDU-NOT-02 | 6 |
| **Integrante 3** | CDU-ACC-05, CDU-TRX-02, CDU-TRX-03, CDU-TRX-04, CDU-PAY-01 y CDU-PAY-02 | 6 |

Antes de repartir el trabajo, el equipo deberá acordar una misma plantilla, nombres de actores y estilo visual. Las relaciones con otros microservicios se mostrarán como sistemas externos al límite del CDU correspondiente, sin representar llamadas síncronas entre ellos.

No se dibujarán relaciones directas entre las bases de datos de distintos microservicios. La relación entre dominios se expresará mediante identificadores y eventos, no mediante llaves foráneas entre bases independientes.

Los CDU se desarrollarán según el catálogo anterior. Los diagramas de componentes, clases y entidad-relación se organizarán por microservicio.

#### Customer Service


Diagramas requeridos: C4 de componentes, clases, entidad-relación, casos de uso y secuencias internas de registro, activación y autenticación.

#### Account Service


Diagramas requeridos: C4 de componentes, clases, entidad-relación, casos de uso y secuencias internas de creación, consulta de saldo, actualización de balance y desactivación.

#### Transaction Service


Diagramas requeridos: C4 de componentes, clases, entidad-relación, casos de uso y secuencias internas de validación y coordinación de transferencias.

#### Payment Service


Diagramas requeridos: C4 de componentes, clases, entidad-relación, casos de uso y secuencias internas de procesamiento, confirmación y rechazo de pagos.

#### Notification & Audit Service


Diagramas requeridos: C4 de componentes, clases, entidad-relación, casos de uso y secuencias internas de notificación y registro de auditoría.

### Matriz de diagramas

| Alcance | Arquitectura / C4 | Clases | Entidad-relación | Casos de uso | Secuencia | Despliegue |
|---|---:|---:|---:|---:|---:|---:|
| Sistema completo | Sí | No | No | No | Sí, flujos end-to-end | Sí |
| Customer Service | Componentes | Sí | Sí | Uno por funcionalidad | Sí | Incluido en el general |
| Account Service | Componentes | Sí | Sí | Uno por funcionalidad | Sí | Incluido en el general |
| Transaction Service | Componentes | Sí | Sí | Uno por funcionalidad | Sí | Incluido en el general |
| Payment Service | Componentes | Sí | Sí | Uno por funcionalidad | Sí | Incluido en el general |
| Notification & Audit Service | Componentes | Sí | Sí | Uno por funcionalidad | Sí | Incluido en el general |


## SRE

### SLI


### SLO


### SLA


### Manejo de idempotencia


### Estrategia de retries


### Manejo de errores


### Trazabilidad mediante `correlationId`



## Preparación del código

### Estructura del monorepo


### Proyectos base de los microservicios


### API Gateway


### Conexión con el broker


### Bases de datos independientes



## Desarrollo

### Customer Service


### Account Service


### Transaction Service


### Payment Service


### Notification & Audit Service


### Implementación del API Gateway


### Publicación de eventos


### Consumo de eventos


### Implementación de Saga


### Frontend



## Contenedores

### Dockerfiles


- Customer Service
- Account Service
- Transaction Service
- Payment Service
- Notification & Audit Service
- API Gateway

### Validación de componentes



## Kubernetes

### Cluster local


### Configuraciones de Kubernetes


### Despliegue de los microservicios


### Despliegue del API Gateway


### Despliegue del broker


### Bases de datos fuera del cluster



## Estrategia de pruebas

La validación del sistema cubrirá los flujos funcionales, la comunicación asíncrona, la resiliencia y el despliegue completo.

### Pruebas funcionales

Se verificarán el registro y la activación de clientes, la creación de cuentas, la consulta de saldo, las transferencias exitosas y el rechazo de transferencias sin fondos suficientes.

### Pruebas de mensajería y resiliencia

Se enviarán eventos duplicados para comprobar la idempotencia, se simularán fallos para validar las compensaciones de la Saga y se comprobará que la estrategia de retries no produzca operaciones duplicadas.

### Pruebas de trazabilidad

Cada flujo será inspeccionado de extremo a extremo para confirmar que conserva el mismo `correlationId` en los mensajes, registros de auditoría y respuestas relacionadas.

### Pruebas de despliegue

El flujo bancario completo se ejecutará en Kubernetes local. La validación incluirá el frontend, API Gateway, broker, cinco microservicios y bases de datos externas al cluster.


## Cierre documental

La versión final de la documentación integrará la arquitectura, los diagramas generales y por microservicio, los casos de uso, el catálogo de eventos y las definiciones de SLI, SLO y SLA. También incluirá la justificación de las decisiones arquitectónicas y un manual de usuario.

Antes de entregar el proyecto se realizará una revisión integral del Markdown, los enlaces y la legibilidad de los diagramas. Los nombres de servicios, eventos, entidades y flujos deberán coincidir con la implementación desplegada; cualquier cambio en el código tendrá que reflejarse también en esta documentación.
