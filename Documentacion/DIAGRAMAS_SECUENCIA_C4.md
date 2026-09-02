# Diagramas de secuencia y C4 - Bank USAC

Los diagramas de este documento representan la arquitectura implementada y las restricciones del enunciado: cinco microservicios, API Gateway como punto de entrada, RabbitMQ para comunicación asíncrona, una base de datos independiente por servicio, Saga para transferencias y conservación del `correlationId`.

> Nota de nomenclatura: en el modelo C4, el diagrama de contexto corresponde al Nivel 1 y el diagrama de componentes al Nivel 3. Se incluyen ambos para cubrir la solicitud “Diagrama de Contexto (C4 - Nivel 3)”.

## Diagrama de contexto - C4 Nivel 1

```mermaid
flowchart LR
    admin["Administrador"]
    teller["Cajero Receptor"]
    client["Cliente"]
    bank["Bank USAC<br/>Sistema bancario distribuido"]
    email["Servicio de correo<br/>(simulado)"]
    external["Proveedor de pagos externo<br/>(simulado)"]

    admin -->|"Administra clientes y consulta auditoría"| bank
    teller -->|"Registra clientes y crea cuentas"| bank
    client -->|"Consulta cuentas, transfiere y paga"| bank
    bank -->|"Solicita envío de activaciones y alertas"| email
    bank -->|"Procesa pagos externos"| external
```

## Vista de componentes - C4 Nivel 3 consolidado

```mermaid
flowchart TB
    user["Administrador / Cajero / Cliente"]

    subgraph k8s["Cluster local de Kubernetes"]
        frontend["Frontend React + Nginx"]

        subgraph gateway["API Gateway - Fiber"]
            routes["Rutas y controladores"]
            auth["Middleware JWT y roles"]
            rpc["Solicitante asíncrono y gestor de respuestas"]
            operations["Seguimiento temporal de operaciones"]
        end

        rabbit["RabbitMQ<br/>banco.comandos<br/>banco.eventos<br/>banco.respuestas<br/>banco.fallidos"]

        subgraph customer["Customer Service"]
            customerConsumer["Consumidor de comandos"]
            customerDomain["Servicio de clientes, activación y JWT"]
            customerRepo["Repositorio y Outbox"]
            customerPublisher["Publicador de eventos"]
        end

        subgraph account["Account Service"]
            accountConsumer["Consumidor de cuentas"]
            accountDomain["Servicio de cuentas y saldos"]
            accountRepo["Repositorios e idempotencia"]
            accountPublisher["Publicador de eventos"]
        end

        subgraph transaction["Transaction Service"]
            transactionConsumer["Consumidor de transferencias"]
            saga["Orquestador Saga"]
            transactionRepo["Repositorio y Outbox"]
            transactionPublisher["Publicador de eventos"]
        end

        subgraph payment["Payment Service"]
            paymentConsumer["Consumidor de pagos"]
            paymentDomain["Servicio de pagos"]
            paymentRepo["Repositorio y Outbox"]
            paymentPublisher["Publicador de eventos"]
        end

        subgraph audit["Notification & Audit Service"]
            auditConsumer["Consumidor de eventos y consultas"]
            auditDomain["Auditoría y notificaciones"]
            auditRepo["Repositorios"]
        end
    end

    subgraph externalDB["Bases de datos externas al cluster - Docker"]
        customerDB[("customer_db")]
        accountDB[("cuentas_db")]
        transactionDB[("transacciones_db")]
        paymentDB[("pagos_db")]
        auditDB[("audit_db")]
    end

    user -->|"HTTPS/HTTP"| frontend
    frontend -->|"HTTP /api"| routes
    routes --> auth --> rpc
    routes --> operations
    rpc -->|"Publica comandos"| rabbit
    rabbit -->|"Entrega respuestas"| rpc

    rabbit --> customerConsumer --> customerDomain --> customerRepo --> customerDB
    customerRepo --> customerPublisher --> rabbit

    rabbit --> accountConsumer --> accountDomain --> accountRepo --> accountDB
    accountRepo --> accountPublisher --> rabbit

    rabbit --> transactionConsumer --> saga --> transactionRepo --> transactionDB
    transactionRepo --> transactionPublisher --> rabbit

    rabbit --> paymentConsumer --> paymentDomain --> paymentRepo --> paymentDB
    paymentRepo --> paymentPublisher --> rabbit

    rabbit --> auditConsumer --> auditDomain --> auditRepo --> auditDB
```

La comunicación HTTP termina en el API Gateway. Las flechas entre RabbitMQ y los microservicios representan comandos, eventos o respuestas asíncronas; no existen llamadas HTTP de negocio entre servicios.

## Secuencia 1 - Autenticación y emisión de JWT

```mermaid
sequenceDiagram
    autonumber
    actor Usuario
    participant FE as Frontend
    participant GW as API Gateway
    participant MQ as RabbitMQ
    participant CUS as Customer Service
    participant DB as customer_db

    Usuario->>FE: Ingresa correo/usuario y contraseña
    FE->>GW: POST /api/clientes/login
    GW->>MQ: cliente.login.solicitado (correlationId)
    MQ-->>CUS: Entrega comando
    CUS->>DB: Buscar usuario y hash
    DB-->>CUS: Cliente, rol y estado
    CUS->>CUS: Validar bcrypt y generar JWT
    CUS->>MQ: cliente.login.respondido (correlationId)
    MQ-->>GW: Entrega respuesta
    GW-->>FE: 200 + JWT + perfil
    FE->>FE: Guardar JWT en sessionStorage
    FE-->>Usuario: Mostrar pantalla según rol
```

## Secuencia 2 - Registro y activación de cliente

```mermaid
sequenceDiagram
    autonumber
    actor Cajero
    actor Cliente
    participant GW as API Gateway
    participant MQ as RabbitMQ
    participant CUS as Customer Service
    participant CDB as customer_db
    participant AUD as Notification & Audit Service
    participant ADB as audit_db

    Cajero->>GW: POST /api/clientes/registro + JWT
    GW->>GW: Validar rol TELLER
    GW->>MQ: cliente.registro.solicitado (correlationId)
    MQ-->>CUS: Entrega comando
    CUS->>CUS: Validar datos, mayoría de edad y unicidad
    CUS->>CDB: Guardar cliente PENDIENTE_ACTIVACION, token y Outbox
    CUS->>MQ: cliente.creado
    CUS->>MQ: notificacion.correo-activacion.solicitado
    CUS->>MQ: cliente.registro.respondido
    MQ-->>GW: Resultado del registro
    GW-->>Cajero: 201 Cliente registrado
    MQ-->>AUD: Eventos del cliente y notificación
    AUD->>ADB: Guardar auditoría y notificación simulada

    Cliente->>GW: GET /api/clientes/activacion?token=...
    GW->>MQ: cliente.activacion.solicitada (nuevo correlationId)
    MQ-->>CUS: Entrega comando
    CUS->>CDB: Validar token, marcarlo usado y activar cliente
    CUS->>MQ: cliente.activado
    CUS->>MQ: cliente.activacion.respondida
    MQ-->>GW: Resultado de activación
    GW-->>Cliente: Cuenta activada
```

## Secuencia 3 - Creación de cuenta bancaria

```mermaid
sequenceDiagram
    autonumber
    actor Cajero
    participant GW as API Gateway
    participant MQ as RabbitMQ
    participant ACC as Account Service
    participant CUS as Customer Service
    participant ADB as cuentas_db
    participant AUD as Notification & Audit Service

    Cajero->>GW: POST /api/cuentas + JWT
    GW->>GW: Validar rol TELLER
    GW->>MQ: cuenta.creacion.solicitada (correlationId)
    GW-->>Cajero: 202 PENDIENTE + operationId
    MQ-->>ACC: Entrega comando
    ACC->>MQ: cliente.validacion.solicitada (mismo correlationId)
    MQ-->>CUS: Validar existencia y estado del cliente
    CUS->>MQ: cliente.validado
    MQ-->>ACC: Entrega validación

    alt Cliente activo
        ACC->>ADB: Crear cuenta MONETARIA o AHORRO
        ACC->>MQ: cuenta.creada
        MQ-->>GW: Actualizar operación a COMPLETADO
        MQ-->>AUD: Registrar evento
    else Cliente inexistente o inactivo
        ACC->>MQ: cuenta.creacion.rechazada
        MQ-->>GW: Actualizar operación a RECHAZADO
        MQ-->>AUD: Registrar rechazo
    end
```

## Secuencia 4 - Transferencia bancaria con Saga

```mermaid
sequenceDiagram
    autonumber
    actor Cliente
    participant GW as API Gateway
    participant MQ as RabbitMQ
    participant TRX as Transaction Service
    participant TDB as transacciones_db
    participant ACC as Account Service
    participant CDB as cuentas_db
    participant AUD as Notification & Audit Service

    Cliente->>GW: POST /api/transferencias + JWT
    GW->>MQ: transferencia.solicitada (correlationId)
    GW-->>Cliente: 202 PENDIENTE + operationId
    MQ-->>TRX: Entrega comando
    TRX->>TDB: Crear transferencia PENDIENTE
    TRX->>MQ: cuenta.debito.solicitado
    MQ-->>ACC: Solicitar débito
    ACC->>CDB: Validar estado, fondos e idempotencia

    alt Fondos suficientes
        ACC->>CDB: Debitar cuenta origen
        ACC->>MQ: cuenta.debitada
        MQ-->>TRX: Confirmar débito
        TRX->>MQ: cuenta.credito.solicitado
        MQ-->>ACC: Solicitar crédito

        alt Crédito exitoso
            ACC->>CDB: Acreditar cuenta destino
            ACC->>MQ: cuenta.acreditada
            MQ-->>TRX: Confirmar crédito
            TRX->>TDB: Marcar COMPLETADA
            TRX->>MQ: transferencia.completada
            MQ-->>GW: Operación COMPLETADA
            MQ-->>AUD: Auditoría y alerta de transferencia
        else Crédito rechazado
            ACC->>MQ: cuenta.credito.rechazado
            MQ-->>TRX: Iniciar compensación
            TRX->>TDB: Marcar COMPENSANDO
            TRX->>MQ: cuenta.compensacion.solicitada
            MQ-->>ACC: Reintegrar monto a origen
            ACC->>CDB: Aplicar compensación idempotente
            ACC->>MQ: cuenta.compensada
            MQ-->>TRX: Confirmar compensación
            TRX->>TDB: Marcar COMPENSADA
            TRX->>MQ: transferencia.compensada
            MQ-->>GW: Operación COMPENSADA
            MQ-->>AUD: Registrar fallo y compensación
        end
    else Fondos insuficientes o cuenta inválida
        ACC->>MQ: cuenta.debito.rechazado
        MQ-->>TRX: Rechazo de débito
        TRX->>TDB: Marcar RECHAZADA
        TRX->>MQ: transferencia.rechazada
        MQ-->>GW: Operación RECHAZADA
        MQ-->>AUD: Registrar rechazo
    end
```

## Secuencia 5 - Procesamiento de pago

```mermaid
sequenceDiagram
    autonumber
    actor Cliente
    participant GW as API Gateway
    participant MQ as RabbitMQ
    participant PAY as Payment Service
    participant PDB as pagos_db
    participant ACC as Account Service
    participant CDB as cuentas_db
    participant AUD as Notification & Audit Service

    Cliente->>GW: POST /api/pagos + JWT
    GW->>MQ: pago.procesamiento.solicitado (correlationId)
    GW-->>Cliente: 202 PENDIENTE + operationId
    MQ-->>PAY: Entrega comando
    PAY->>PDB: Registrar pago PROCESANDO y Outbox
    PAY->>MQ: cuenta.debito.solicitado
    MQ-->>ACC: Solicitar débito
    ACC->>CDB: Validar cuenta, fondos e idempotencia

    alt Débito aprobado
        ACC->>CDB: Actualizar saldo
        ACC->>MQ: cuenta.debitada
        MQ-->>PAY: Confirmar débito
        PAY->>PAY: Procesar pago interno o simular proveedor externo
        alt Pago confirmado
            PAY->>PDB: Marcar COMPLETADO
            PAY->>MQ: pago.completado
            MQ-->>GW: Operación COMPLETADA
            MQ-->>AUD: Registrar pago y notificación
        else Proveedor externo falla
            PAY->>PDB: Marcar COMPENSANDO
            PAY->>MQ: cuenta.compensacion.solicitada
            MQ-->>ACC: Reintegrar monto
            ACC->>CDB: Aplicar compensación
            ACC->>MQ: cuenta.compensada
            MQ-->>PAY: Confirmar compensación
            PAY->>PDB: Marcar RECHAZADO
            PAY->>MQ: pago.rechazado
            MQ-->>GW: Operación RECHAZADA
            MQ-->>AUD: Registrar fallo
        end
    else Débito rechazado
        ACC->>MQ: cuenta.debito.rechazado
        MQ-->>PAY: Entrega rechazo
        PAY->>PDB: Marcar RECHAZADO
        PAY->>MQ: pago.rechazado
        MQ-->>GW: Operación RECHAZADA
        MQ-->>AUD: Registrar rechazo
    end
```

## Convenciones

- Todas las publicaciones conservan el mismo `correlationId` durante un flujo distribuido.
- Los comandos se publican en `banco.comandos`.
- Los eventos de negocio se publican en `banco.eventos`.
- Las respuestas asíncronas de Customer y Audit se publican en `banco.respuestas`.
- Los mensajes no recuperables se trasladan a `banco.fallidos` y sus colas DLQ.
- Cada consumidor aplica idempotencia antes de modificar su propia base de datos.
- Ningún microservicio consulta la base de datos de otro servicio.
