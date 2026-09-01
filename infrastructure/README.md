# Infraestructura local

El archivo `docker-compose.yml` ubicado en la raíz levanta el entorno local compartido.

## Preparación

Copiar `.env.docker.example` como `.env` si se necesitan cambiar puertos o credenciales.
Las credenciales incluidas son únicamente para desarrollo local.

## Entorno completo disponible

```bash
docker compose up --build
```

Este comando levanta automáticamente toda la infraestructura y los componentes que ya tienen implementación ejecutable:

- RabbitMQ.
- PostgreSQL exclusivo de Account Service.
- PostgreSQL exclusivo de Payment Service.
- Migraciones de Account Service.
- Account Service.
- Migraciones de Payment Service.
- Payment Service.
- Frontend React.

Servicios disponibles:

- Account Service: `http://localhost:8082/salud`
- RabbitMQ Management: `http://localhost:15672`
- PostgreSQL Account: `localhost:5433`
- PostgreSQL Payment: `localhost:5434`
- Payment Service: `http://localhost:8084/salud`
- Frontend: `http://localhost:3000`

El frontend requiere que el API Gateway exponga los endpoints `/api/cuentas` y `/api/pagos` para ejecutar operaciones de negocio.

## Limpieza

Detener contenedores conservando los datos:

```bash
docker compose down
```

Eliminar también los volúmenes locales y comenzar con bases vacías:

```bash
docker compose down --volumes
```

La eliminación de volúmenes borra los datos locales de desarrollo.
