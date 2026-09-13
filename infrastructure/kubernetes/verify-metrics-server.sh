#!/usr/bin/env sh
set -eu

if kubectl get --raw "/apis/metrics.k8s.io/v1beta1" >/dev/null 2>&1; then
  echo "Metrics Server disponible en el contexto actual."
  kubectl top pods -n bank-usac 2>/dev/null || true
  exit 0
fi

if [ "$(kubectl config current-context 2>/dev/null || true)" = "minikube" ]; then
  echo "Habilitando Metrics Server en Minikube..."
  minikube addons enable metrics-server
  kubectl -n kube-system rollout status deployment/metrics-server --timeout=120s
  intentos=0
  until kubectl get --raw "/apis/metrics.k8s.io/v1beta1" >/dev/null 2>&1; do
    intentos=$((intentos + 1))
    [ "$intentos" -ge 12 ] && {
      echo "Metrics Server inició, pero su API todavía no responde." >&2
      exit 1
    }
    sleep 5
  done
  echo "Metrics Server habilitado."
  exit 0
fi

echo "Metrics Server no está disponible. Habilítalo en el clúster GKE antes de usar HPA." >&2
exit 1
