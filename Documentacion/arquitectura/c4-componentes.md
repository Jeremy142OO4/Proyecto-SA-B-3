# C4 - Componentes

El diagrama de componentes C4 Nivel 3 muestra la estructura interna del microservicio Transaction Service, implementado en Go, encargado de coordinar el flujo de transferencias bancarias de forma asíncrona.


![Diagrama entidad-relación de Transaction Service](../Imagenes/Diagrama%20-%20C4-Nivel-3-Componentes.png)


El Transfer Event Handler consume los eventos provenientes del broker RabbitMQ y valida la información de correlación. Posteriormente, envía la operación al Transfer Application Service, encargado de coordinar el caso de uso de transferencia.

El Transfer Application Service utiliza el componente Idempotency & Correlation para validar claves de idempotencia y trazabilidad, evitando el procesamiento duplicado de mensajes. Además, delega las reglas de negocio al Transfer Domain Service, donde se administran los estados y la lógica asociada al flujo Saga.

La persistencia de las transferencias es gestionada por Transaction Repository, el cual es el único componente que accede a la base de datos Transaction DB PostgreSQL para consultar y almacenar la información correspondiente.

Para la comunicación con otros microservicios, el Outbox / Event Publisher se encarga de publicar de forma confiable los eventos generados por el servicio hacia RabbitMQ, manteniendo una arquitectura desacoplada y basada en eventos.