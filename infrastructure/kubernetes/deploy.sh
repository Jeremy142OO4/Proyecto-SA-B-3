#!/usr/bin/env sh
set -eu

RAIZ_PROYECTO=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
PERFIL_MINIKUBE=${PERFIL_MINIKUBE:-minikube}

cd "$RAIZ_PROYECTO"
# Retira los contenedores de la arquitectura anterior sin borrar sus volúmenes.
podman rm -f bank-usac-rabbitmq bank-usac-account-service bank-usac-payment-service bank-usac-frontend 2>/dev/null || true
docker compose up -d

if ! minikube status -p "$PERFIL_MINIKUBE" >/dev/null 2>&1; then
  minikube start -p "$PERFIL_MINIKUBE" --driver=podman
fi

minikube image build -p "$PERFIL_MINIKUBE" -t bank-usac/account-service:local services/account-service
minikube image build -p "$PERFIL_MINIKUBE" -t bank-usac/payment-service:local services/payment-service
minikube image build -p "$PERFIL_MINIKUBE" -t bank-usac/frontend:local frontend/bank-usac-web

kubectl apply -f infrastructure/kubernetes/namespace.yaml
kubectl -n bank-usac create secret generic bank-usac-secrets \
  --from-literal=RABBITMQ_USUARIO="${RABBITMQ_USUARIO:-bank_usac}" \
  --from-literal=RABBITMQ_CLAVE="${RABBITMQ_CLAVE:-bank_usac_local}" \
  --from-literal=URL_RABBITMQ="amqp://${RABBITMQ_USUARIO:-bank_usac}:${RABBITMQ_CLAVE:-bank_usac_local}@rabbitmq:5672/" \
  --from-literal=URL_BASE_DATOS_CUENTAS="postgres://${CUENTAS_USUARIO:-cuentas_usuario}:${CUENTAS_CLAVE:-cuentas_local}@host.minikube.internal:${CUENTAS_PUERTO_BD:-5433}/${CUENTAS_BD:-cuentas_db}?sslmode=disable" \
  --from-literal=URL_BASE_DATOS_PAGOS="postgres://${PAGOS_USUARIO:-pagos_usuario}:${PAGOS_CLAVE:-pagos_local}@host.minikube.internal:${PAGOS_PUERTO_BD:-5434}/${PAGOS_BD:-pagos_db}?sslmode=disable" \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl apply -k infrastructure/kubernetes
kubectl -n bank-usac rollout status deployment/rabbitmq --timeout=180s
kubectl -n bank-usac rollout status deployment/account-service --timeout=180s
kubectl -n bank-usac rollout status deployment/payment-service --timeout=180s
kubectl -n bank-usac rollout status deployment/frontend --timeout=180s
kubectl -n bank-usac get pods,services
"$RAIZ_PROYECTO/infrastructure/kubernetes/expose.sh"
