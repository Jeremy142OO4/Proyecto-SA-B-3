#!/usr/bin/env bash
set -euo pipefail
export OBSERVABILITY_NAMESPACE="${OBSERVABILITY_NAMESPACE:-bank-usac-observability}"
: "${KUBE_NAMESPACE:=bank-usac-dev}"
if [[ "$KUBE_NAMESPACE" != bank-usac-dev ]]; then
  echo "This profile is for dev only; production remains disabled" >&2
  exit 1
fi
directory="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
python3 "$directory/preflight.py"
kubectl create namespace "$OBSERVABILITY_NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
if ! kubectl -n "$OBSERVABILITY_NAMESPACE" get secret observability-grafana >/dev/null 2>&1; then
  kubectl -n "$OBSERVABILITY_NAMESPACE" create secret generic observability-grafana --from-literal="password=$(openssl rand -hex 24)"
fi
helm upgrade --install bank-observability "$directory/chart" \
  --namespace "$OBSERVABILITY_NAMESPACE" \
  --set "applicationNamespace=$KUBE_NAMESPACE" \
  --atomic --wait --timeout 8m --history-max 5

# The same desired value is in the cloud-dev overlay. Only restart if it changed.
if ! kubectl -n "$KUBE_NAMESPACE" get configmap bank-usac-config >/dev/null 2>&1; then
  echo "Application configuration will be created by the cloud-dev overlay"
  exit 0
fi
endpoint="http://otel-collector.${OBSERVABILITY_NAMESPACE}.svc.cluster.local:4318"
previous="$(kubectl -n "$KUBE_NAMESPACE" get configmap bank-usac-config -o jsonpath='{.data.OTEL_EXPORTER_OTLP_ENDPOINT}' 2>/dev/null || true)"
kubectl -n "$KUBE_NAMESPACE" patch configmap bank-usac-config --type merge \
  -p "{\"data\":{\"OTEL_EXPORTER_OTLP_ENDPOINT\":\"$endpoint\"}}"
if [[ "$previous" != "$endpoint" ]]; then
  python3 "$directory/restart.py"
fi
