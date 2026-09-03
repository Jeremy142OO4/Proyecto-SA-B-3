# Diagrama de despliegue

El diagrama representa la distribución física y lógica del sistema bancario **Bank USAC** en el ambiente local del proyecto. La aplicación se divide en dos zonas principales: los componentes de aplicación desplegados en un clúster de Kubernetes y las bases de datos PostgreSQL ejecutadas como contenedores independientes mediante Docker Compose.

![Diagrama de despliegue de Bank USAC](../Imagenes/Diagrama%20-%20Despliegue.png)

## Vista general

El cliente utiliza un navegador web para acceder al frontend. El frontend, el API Gateway, RabbitMQ y los cinco microservicios se ejecutan dentro de un clúster local de **Kubernetes con Minikube**, bajo el namespace `bank-usac`. Las bases de datos permanecen fuera del clúster y son administradas mediante **Docker Compose**.

Esta separación permite que Kubernetes administre el ciclo de vida de los componentes de aplicación, mientras Docker Compose proporciona un entorno sencillo y persistente para las bases de datos durante el desarrollo y la demostración del proyecto.

## Punto de entrada al sistema

El navegador es el único cliente externo representado. El acceso se realiza mediante HTTP al puerto `30080`, publicado por Kubernetes como un servicio de tipo `NodePort`.

La solicitud llega al contenedor del frontend, compuesto por una aplicación **React** servida mediante **Nginx**. Las operaciones realizadas desde la interfaz se envían al **API Gateway** utilizando la ruta HTTP `/api`.

El API Gateway está desarrollado en **Go con Fiber** y actúa como la frontera de entrada del backend. Entre sus responsabilidades se encuentran recibir las solicitudes externas, aplicar las validaciones transversales correspondientes e iniciar los flujos de negocio. El frontend no accede directamente a los microservicios.

## Comunicación asíncrona

Después de recibir una operación, el API Gateway se comunica con **RabbitMQ** mediante el protocolo AMQP en el puerto `5672`. RabbitMQ funciona como intermediario de mensajes y administra los exchanges, colas, respuestas y colas de mensajes muertos o **DLQ**.

Las líneas discontinuas del diagrama representan la comunicación asíncrona. Los comandos y eventos son enviados a RabbitMQ y consumidos por el microservicio responsable. De esta manera, los microservicios no dependen de llamadas HTTP directas entre ellos y permanecen desacoplados.

RabbitMQ utiliza el volumen persistente `rabbitmq-datos`. Este volumen permite conservar la información del broker que haya sido configurada como durable incluso si su Pod es reiniciado.

## Microservicios desplegados

Dentro del clúster se ejecutan cinco microservicios independientes:

| Microservicio | Responsabilidad principal | Componentes destacados |
|---|---|---|
| `customer-service` | Administración de clientes, usuarios, autenticación y estados de usuario. | Consumidores AMQP, lógica de dominio y patrón Outbox. |
| `account-service` | Creación y administración de cuentas, balances, movimientos e inactividad. | Consumidores AMQP, lógica de dominio y patrón Outbox. |
| `transaction-service` | Registro y coordinación de transferencias entre cuentas. | Saga, consumidores AMQP, lógica de dominio y patrón Outbox. |
| `payment-service` | Procesamiento y seguimiento de pagos internos y externos. | Consumidores AMQP, lógica de dominio y patrón Outbox. |
| `notification-audit-service` | Envío de notificaciones y conservación de eventos de auditoría. | Consumidores AMQP, auditoría y notificaciones. |

Cada microservicio se empaqueta en su propia imagen de contenedor y se ejecuta en un Pod de Kubernetes. Esto permite desplegar, reiniciar y escalar cada componente sin tener que modificar los demás servicios.

Aunque los servicios pueden exponer un puerto HTTP para comprobaciones operativas como el endpoint de salud, la comunicación de negocio mostrada en el diagrama se realiza mediante RabbitMQ.

## Persistencia de datos

Cada microservicio es propietario de su propia base de datos PostgreSQL. Ningún servicio debe consultar directamente la base de datos perteneciente a otro dominio.

| Microservicio propietario | Base de datos | Puerto del host | Esquema o dominio |
|---|---|---:|---|
| `customer-service` | `customer-data` | `5432` | Clientes y usuarios. |
| `account-service` | `account-data` | `5433` | Cuentas y movimientos. |
| `transaction-service` | `transaction-data` | `5435` | Transferencias y estado de la Saga. |
| `payment-service` | `payment-data` | `5434` | Pagos e intentos de pago. |
| `notification-audit-service` | `audit-data` | `5436` | Notificaciones y eventos de auditoría. |

Las conexiones entre los microservicios y PostgreSQL utilizan TCP. Los puertos diferentes permiten ejecutar las cinco instancias en el mismo host local sin provocar conflictos. La separación de bases de datos mantiene la autonomía de cada microservicio y evita el acoplamiento por medio de tablas compartidas.

Los identificadores pertenecientes a otros dominios, como `id_cliente`, `id_cuenta` o `correlationId`, se conservan como referencias lógicas. No se crean claves foráneas entre bases de datos de microservicios diferentes.

## Recorrido de una operación

El flujo general de una solicitud puede explicarse de la siguiente manera:

1. El usuario realiza una acción desde el navegador.
2. La solicitud ingresa al frontend por el puerto `30080`.
3. React envía la operación al API Gateway mediante `/api`.
4. El API Gateway publica el comando correspondiente en RabbitMQ.
5. El microservicio responsable consume y procesa el mensaje.
6. El microservicio modifica exclusivamente su propia base de datos.
7. Los eventos generados se publican mediante RabbitMQ utilizando el patrón Outbox.
8. Otros componentes interesados, como el servicio de auditoría y notificaciones, consumen esos eventos sin establecer una dependencia directa con el servicio emisor.

## Alcance del despliegue

El diagrama describe el ambiente local utilizado para desarrollar, integrar y demostrar el proyecto. No representa todavía un ambiente productivo en la nube. En una instalación de producción sería necesario sustituir Minikube por un clúster administrado, utilizar almacenamiento persistente de alta disponibilidad, gestionar secretos mediante un servicio seguro y colocar un Ingress o balanceador de carga frente al sistema.
