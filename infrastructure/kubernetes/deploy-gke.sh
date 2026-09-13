#!/usr/bin/env sh
set -eu

: "${GCP_PROJECT_ID:?Define GCP_PROJECT_ID con el ID de tu proyecto GCP}"
: "${GKE_CLUSTER:?Define GKE_CLUSTER con el nombre del clúster}"
: "${GKE_ZONE:?Define GKE_ZONE con la zona del clúster}"
IMAGE_TAG=${IMAGE_TAG:-v2.0.0}
REGION=${REGION:-us-central1}
PUBLIC_HOST=${PUBLIC_HOST:-bank-usac.example.com}

gcloud container clusters get-credentials "$GKE_CLUSTER" --zone "$GKE_ZONE" --project "$GCP_PROJECT_ID"
kubectl apply -f infrastructure/kubernetes/base/namespace.yaml

: "${URL_BASE_DATOS_CUENTAS:?Define URL_BASE_DATOS_CUENTAS}"
: "${URL_BASE_DATOS_PAGOS:?Define URL_BASE_DATOS_PAGOS}"
: "${URL_BASE_DATOS_TRANSACCIONES:?Define URL_BASE_DATOS_TRANSACCIONES}"
: "${URL_BASE_DATOS_CLIENTES:?Define URL_BASE_DATOS_CLIENTES}"
: "${URL_BASE_DATOS_AUDITORIA:?Define URL_BASE_DATOS_AUDITORIA}"
: "${JWT_SECRET:?Define JWT_SECRET}"
: "${URL_RABBITMQ:?Define URL_RABBITMQ}"
: "${SMTP_USERNAME:?Define SMTP_USERNAME}"
: "${SMTP_APP_PASSWORD:?Define SMTP_APP_PASSWORD}"
: "${SMTP_FROM:?Define SMTP_FROM}"

kubectl -n bank-usac create secret generic bank-usac-secrets \
  --from-literal=RABBITMQ_USUARIO="${RABBITMQ_USUARIO:-bank_usac}" \
  --from-literal=RABBITMQ_CLAVE="${RABBITMQ_CLAVE:-bank_usac_local}" \
  --from-literal=JWT_SECRET="$JWT_SECRET" \
  --from-literal=URL_RABBITMQ="$URL_RABBITMQ" \
  --from-literal=URL_BASE_DATOS_CUENTAS="$URL_BASE_DATOS_CUENTAS" \
  --from-literal=URL_BASE_DATOS_PAGOS="$URL_BASE_DATOS_PAGOS" \
  --from-literal=URL_BASE_DATOS_TRANSACCIONES="$URL_BASE_DATOS_TRANSACCIONES" \
  --from-literal=URL_BASE_DATOS_CLIENTES="$URL_BASE_DATOS_CLIENTES" \
  --from-literal=URL_BASE_DATOS_AUDITORIA="$URL_BASE_DATOS_AUDITORIA" \
  --from-literal=SMTP_HOST="${SMTP_HOST:-smtp.gmail.com}" \
  --from-literal=SMTP_PORT="${SMTP_PORT:-587}" \
  --from-literal=SMTP_USERNAME="$SMTP_USERNAME" \
  --from-literal=SMTP_APP_PASSWORD="$SMTP_APP_PASSWORD" \
  --from-literal=SMTP_FROM="$SMTP_FROM" \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl kustomize infrastructure/kubernetes/overlays/prod \
  | sed "s/PROJECT_ID/${GCP_PROJECT_ID}/g; s/v2.0.0/${IMAGE_TAG}/g; s/bank-usac.example.com/${PUBLIC_HOST}/g" \
  | kubectl apply -f -

sh infrastructure/kubernetes/verify-metrics-server.sh
for despliegue in rabbitmq api-gateway customer-service account-service payment-service transaction-service notification-audit-service frontend; do
  kubectl -n bank-usac rollout status "deployment/$despliegue" --timeout=180s
done
kubectl -n bank-usac get deployments,services,hpa
