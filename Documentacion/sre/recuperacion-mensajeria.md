# Recuperación automática de RabbitMQ

Gateway y Clientes ya supervisaban sus sesiones. Cuentas, Transferencias, Pagos y Auditoría ahora también reconstruyen conexión, topología, canales, consumidores y publicadores después de una caída. Se detectan tanto el cierre de conexión como el de un canal aislado o del flujo de entregas.

Cada sesión cancela y espera sus trabajadores antes de iniciar la siguiente. Se cierra el transporte para desbloquear confirmaciones pendientes. El intervalo de reconexión aumenta de 1 hasta 15 segundos; el intento de conexión tiene un límite de 5 segundos y heartbeat de 10 segundos. Si el broker no está disponible al iniciar, el supervisor continúa intentando conectar.

La supervisión **no vuelve a emitir operaciones bancarias**. Los mensajes del Outbox siguen pendientes si no se confirma la publicación, y se conservan sus identificadores y contexto de traza. Los mensajes sin ACK pueden ser entregados nuevamente por RabbitMQ: las reglas de idempotencia del servicio siguen siendo necesarias. No se promete entrega exactamente una vez ni cero pérdida de telemetría durante una caída prolongada.

Auditoría reconstruye también su resolvedor de clientes; su sustitución está sincronizada para no competir con la atención de eventos. Transferencias espera la finalización de sus dos consumidores antes de reconstruir la sesión.

## Pruebas

- Pruebas unitarias en `internal/recovery`: cancelación y espera de trabajadores, cierre de consumidor sin error y reintentos de inicio.
- Con `TEST_RABBITMQ_URL` apuntando a un broker aislado: cierre de un canal publicador y de toda la conexión, seguido de una entrega real en cada sesión reconstruida. Sin la variable, esta prueba se omite explícitamente.
- Prueba aislada de los seis ejecutables con PostgreSQL y RabbitMQ: reiniciar el broker manteniendo los procesos vivos, consultar cuentas/transferencias/pagos y comprobar transferencias y pagos exitosos y compensados, con validación de saldos y trazas.
- Comprobación posterior al despliegue: reiniciar únicamente RabbitMQ en dev, comprobar las consultas y contrastar UID y contador de reinicios de los seis servicios. No ejecutar esta prueba mientras alguien esté realizando operaciones reales.

El CI ejecuta las pruebas unitarias junto a las existentes de cada componente. Las pruebas con dependencias reales requieren la configuración aislada indicada; nunca utilizar bases con clientes reales para las pruebas financieras.
