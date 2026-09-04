# Infraestructura local

La ejecución está separada según el requisito del proyecto:

- **Docker/Podman Compose:** las cinco bases PostgreSQL (clientes, cuentas, transacciones, pagos y auditoría) y sus migraciones.
- **Kubernetes/Minikube:** RabbitMQ, API Gateway, los cinco microservicios y el frontend.

Así, las bases de datos permanecen fuera del clúster de Kubernetes.

## Preparación

Se puede copiar `.env.docker.example` como `.env` para cambiar puertos y credenciales locales. Nunca se debe confirmar el archivo `.env` en Git.

## Bases de datos externas

Desde la raíz del repositorio:

```bash
docker compose up -d
```

Quedan disponibles PostgreSQL de Customer Service en el puerto `5432`, Account Service en `5433`, Payment Service en `5434`, Transaction Service en `5435` y Notification & Audit Service en `5436`. Los contenedores de migración terminan con código cero después de aplicar cada esquema.

## Despliegue Kubernetes

En Linux con Minikube, Podman, `kubectl` y Docker Compose:

```bash
chmod +x infrastructure/kubernetes/deploy.sh
./infrastructure/kubernetes/deploy.sh
```

El script inicia Minikube, construye y carga las imágenes locales, crea el Secret sin guardarlo en Git, aplica los manifiestos y espera que todos los despliegues estén disponibles.

Para exponer temporalmente los componentes en la máquina anfitriona:

```bash
./infrastructure/kubernetes/expose.sh
```

El script de despliegue ya ejecuta este paso automáticamente. Los componentes quedan disponibles únicamente en `localhost`, mediante los puertos `3000`, `8082`, `8084` y `15672`.

El frontend reserva el nombre interno `api-gateway:8080` para el proxy hacia el Gateway. Las solicitudes de negocio externas entran por el frontend; los servicios de aplicación no se exponen como puntos de entrada públicos.

## Limpieza

Eliminar los recursos de aplicación sin borrar las bases:

```bash
kubectl delete namespace bank-usac
```

Detener las bases conservando los datos:

```bash
docker compose down
```

Usar `docker compose down --volumes` únicamente si se desea borrar definitivamente la información local.
