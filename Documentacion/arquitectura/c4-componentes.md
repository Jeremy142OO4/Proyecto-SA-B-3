# C4 - Componentes
Este documento extiende la vista de componentes C4 (Nivel 3). La Fase 1 documentó la arquitectura interna de Transaction Service. La Fase 2 agrega la arquitectura interna de los cuatro servicios que incorporaron nuevas capacidades: Customer Service (KYC), Account Service (tipos de cuenta), Payment Service (pagos externos) y Notification & Audit Service (severidad). Transaction Service se muestra actualizado con los componentes de historial.

---

## Transaction Service — Componentes (actualizado Fase 2)

El Transaction Service coordina el flujo de transferencias bancarias de forma asíncrona mediante una Saga de coreografía. En la Fase 2 se extiende con componentes de historial y trazabilidad.


![Diagrama entidad-relación de Transaction Service](../Imagenes/componentes.drawio%20(1).png)

### Componentes de Transaction Service

| Componente | Responsabilidad |
|---|---|
| Transfer Event Handler | Consume mensajes AMQP; valida `correlationId`; enruta al Application Service |
| Transfer Application Service | Coordina el caso de uso; orquesta idempotencia, dominio y persistencia |
| Idempotency & Correlation | Valida claves de idempotencia; evita procesamiento duplicado |
| Transfer Domain Service | Estados de la Saga (`PENDING → APPROVED / FAILED`); lógica de compensación; **registro de historial ★** |
| Transaction Repository | CRUD sobre la tabla `transfers` en `transacciones_db` |
| History Repository ★ | Consultas de historial por cuenta; filtros por fecha y estado (`transfer_events`) |
| Outbox / Event Publisher | Publica eventos de dominio a RabbitMQ de forma confiable mediante patrón Outbox |

---
