# Diagrama entidad-relación — Payment Service

El diagrama representa la estructura persistente administrada exclusivamente por **Payment Service**. Incluye los pagos, sus intentos de procesamiento y las tablas técnicas utilizadas para idempotencia y publicación confiable de eventos.

![Diagrama entidad-relación de Payment Service](../../Imagenes/ER_Payment.png)

## Relaciones principales

- Un pago puede registrar múltiples intentos de procesamiento.
- Cada intento pertenece a un único pago.
- `id_cliente` e `id_cuenta_origen` son referencias lógicas a otros contextos y no claves foráneas entre bases de datos.
- `mensajes_procesados` evita procesar dos veces el mismo mensaje.
- `mensajes_salida` implementa el patrón Transactional Outbox.

