#!/usr/bin/env sh
set -eu

RAIZ_PROYECTO=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
PERFIL_MINIKUBE=${PERFIL_MINIKUBE:-minikube}
if [ -z "${DRIVER_MINIKUBE:-}" ]; then
  if command -v podman >/dev/null 2>&1; then
    DRIVER_MINIKUBE=podman
  else
    DRIVER_MINIKUBE=docker
  fi
fi

cd "$RAIZ_PROYECTO"
if command -v podman >/dev/null 2>&1; then
  podman rm -f bank-usac-rabbitmq bank-usac-api-gateway bank-usac-customer-service bank-usac-account-service bank-usac-payment-service bank-usac-transaction-service bank-usac-frontend 2>/dev/null || true
fi
docker compose up -d

if ! minikube status -p "$PERFIL_MINIKUBE" >/dev/null 2>&1; then
  minikube start -p "$PERFIL_MINIKUBE" --driver="$DRIVER_MINIKUBE"
fi

minikube image build -p "$PERFIL_MINIKUBE" -t bank-usac/account-service:local services/account-service
minikube image build -p "$PERFIL_MINIKUBE" -t bank-usac/customer-service:local services/service-customer
minikube image build -p "$PERFIL_MINIKUBE" -t bank-usac/payment-service:local services/payment-service
minikube image build -p "$PERFIL_MINIKUBE" -t bank-usac/transaction-service:local services/transaction-service
minikube image build -p "$PERFIL_MINIKUBE" -t bank-usac/api-gateway:local gateway/api-gateway
minikube image build -p "$PERFIL_MINIKUBE" -t bank-usac/notification-audit-service:local services/service-notification-audit
minikube image build -p "$PERFIL_MINIKUBE" -t bank-usac/frontend:local frontend/bank-usac-web

kubectl apply -f infrastructure/kubernetes/namespace.yaml
kubectl -n bank-usac create secret generic bank-usac-secrets \
  --from-literal=RABBITMQ_USUARIO="${RABBITMQ_USUARIO:-bank_usac}" \
  --from-literal=RABBITMQ_CLAVE="${RABBITMQ_CLAVE:-bank_usac_local}" \
  --from-literal=JWT_SECRET="${JWT_SECRET:-cambie-este-secreto-local-de-al-menos-32-caracteres}" \
  --from-literal=URL_RABBITMQ="amqp://${RABBITMQ_USUARIO:-bank_usac}:${RABBITMQ_CLAVE:-bank_usac_local}@rabbitmq:5672/" \
  --from-literal=URL_BASE_DATOS_CUENTAS="postgres://${CUENTAS_USUARIO:-cuentas_usuario}:${CUENTAS_CLAVE:-cuentas_local}@host.minikube.internal:${CUENTAS_PUERTO_BD:-5433}/${CUENTAS_BD:-cuentas_db}?sslmode=disable" \
  --from-literal=URL_BASE_DATOS_PAGOS="postgres://${PAGOS_USUARIO:-pagos_usuario}:${PAGOS_CLAVE:-pagos_local}@host.minikube.internal:${PAGOS_PUERTO_BD:-5434}/${PAGOS_BD:-pagos_db}?sslmode=disable" \
  --from-literal=URL_BASE_DATOS_TRANSACCIONES="postgres://${TRANSACCIONES_USUARIO:-transacciones_usuario}:${TRANSACCIONES_CLAVE:-transacciones_local}@host.minikube.internal:${TRANSACCIONES_PUERTO_BD:-5435}/${TRANSACCIONES_BD:-transacciones_db}?sslmode=disable" \
  --from-literal=URL_BASE_DATOS_CLIENTES="postgres://${CLIENTES_USUARIO:-customer_user}:${CLIENTES_CLAVE:-customer_password}@host.minikube.internal:${CLIENTES_PUERTO_BD:-5432}/${CLIENTES_BD:-customer_db}?sslmode=disable" \
  --from-literal=URL_BASE_DATOS_AUDITORIA="postgres://${AUDITORIA_USUARIO:-audit_user}:${AUDITORIA_CLAVE:-audit_password}@host.minikube.internal:${AUDITORIA_PUERTO_BD:-5436}/${AUDITORIA_BD:-audit_db}?sslmode=disable" \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl apply -k infrastructure/kubernetes
for despliegue in rabbitmq api-gateway customer-service account-service payment-service transaction-service notification-audit-service frontend; do
  kubectl -n bank-usac rollout restart "deployment/$despliegue"
done
for despliegue in rabbitmq api-gateway customer-service account-service payment-service transaction-service notification-audit-service frontend; do
  kubectl -n bank-usac rollout status "deployment/$despliegue" --timeout=180s
done
kubectl -n bank-usac get pods,services
"$RAIZ_PROYECTO/infrastructure/kubernetes/expose.sh"
