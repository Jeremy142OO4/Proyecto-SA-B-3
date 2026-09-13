# Despliegues Kubernetes

La configuración está separada en una base común y dos overlays:

- `overlays/dev`: Minikube, imágenes locales con etiqueta `dev` y bases de datos Docker.
- `overlays/prod`: GKE, imágenes versionadas de Artifact Registry y secretos proporcionados por el entorno.

## Desarrollo local

Desde la raíz del proyecto:

```sh
sh infrastructure/kubernetes/deploy.sh
sh infrastructure/kubernetes/verify-metrics-server.sh
```

El script construye las imágenes `bank-usac/*:dev`, levanta las bases de datos Docker y aplica el overlay de desarrollo.

## Producción en GKE

Las imágenes deben publicarse en Artifact Registry, por ejemplo:

```sh
gcloud auth configure-docker us-central1-docker.pkg.dev
gcloud builds submit services/transaction-service \
  --tag us-central1-docker.pkg.dev/PROJECT_ID/bank-usac/transaction-service:v2.0.0
```

Se repite el comando para cada servicio y para el frontend. Luego se definen las variables de conexión (URLs de las bases, RabbitMQ, JWT y SMTP) y se ejecuta:

```sh
GCP_PROJECT_ID=mi-proyecto \
GKE_CLUSTER=bank-usac \
GKE_ZONE=us-central1-a \
PUBLIC_HOST=bank-usac.example.com \
IMAGE_TAG=v2.0.0 \
URL_BASE_DATOS_CUENTAS='postgres://...' \
URL_BASE_DATOS_PAGOS='postgres://...' \
URL_BASE_DATOS_TRANSACCIONES='postgres://...' \
URL_BASE_DATOS_CLIENTES='postgres://...' \
URL_BASE_DATOS_AUDITORIA='postgres://...' \
URL_RABBITMQ='amqp://...' \
JWT_SECRET='secreto-produccion' \
SMTP_USERNAME='...' SMTP_APP_PASSWORD='...' SMTP_FROM='...' \
sh infrastructure/kubernetes/deploy-gke.sh
```

El overlay de producción reemplaza `PROJECT_ID` y la etiqueta de imagen, utiliza `IfNotPresent` y nunca usa `imagePullPolicy: Never`. Las bases de datos no se despliegan dentro de GKE: sus URLs se inyectan como secretos, de modo que pueden apuntar a Cloud SQL u otro PostgreSQL administrado.

## HPA y métricas

`hpa-transaction.yaml` y `hpa-payment.yaml` escalan entre 1 y 4 réplicas cuando el uso promedio de CPU alcanza el 80 %. Ambos servicios ya declaran `requests` de CPU, requisito para calcular la utilización.

GKE proporciona la API de métricas para HPA. En Minikube, `verify-metrics-server.sh` habilita el addon `metrics-server` si todavía no está instalado.
