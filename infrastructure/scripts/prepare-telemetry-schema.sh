#!/usr/bin/env bash
# Existing schemas were initialized manually: apply only additive telemetry DDL.
set -euo pipefail
: "${KUBE_NAMESPACE:?KUBE_NAMESPACE is required}"
suffix="${GITHUB_RUN_ID:-manual}-${GITHUB_RUN_ATTEMPT:-1}"
case "$suffix" in (*[!0-9a-z-]*) echo "Invalid run suffix" >&2; exit 1;; esac
resources=()
cleanup() {
  for resource in "${resources[@]}"; do
    kubectl -n "$KUBE_NAMESPACE" delete "$resource" --ignore-not-found --wait=false >/dev/null || true
  done
}
trap cleanup EXIT
while read -r service key version; do
  name="${service}-trace-${suffix}"
  migration="services/$service/database/migrations/${version}_persistir_contexto_traza.up.sql"
  test -f "$migration"
  kubectl -n "$KUBE_NAMESPACE" create configmap "$name" --from-file="migration.sql=$migration" --dry-run=client -o yaml | kubectl apply -f -
  resources+=("configmap/$name")
  kubectl -n "$KUBE_NAMESPACE" apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: $name
spec:
  backoffLimit: 0
  activeDeadlineSeconds: 90
  ttlSecondsAfterFinished: 300
  template:
    spec:
      restartPolicy: Never
      automountServiceAccountToken: false
      containers:
        - name: migrate
          image: postgres:17-alpine
          command: [sh, -c, 'psql --no-psqlrc --set=ON_ERROR_STOP=1 --single-transaction --dbname="\$DATABASE_URL" --file=/migration/migration.sql']
          env:
            - name: DATABASE_URL
              valueFrom:
                secretKeyRef:
                  name: bank-usac-secrets
                  key: $key
            - name: PGCONNECT_TIMEOUT
              value: "10"
            - name: PGOPTIONS
              value: "-c lock_timeout=5000 -c statement_timeout=30000"
          resources:
            requests: {cpu: 25m, memory: 32Mi}
            limits: {cpu: 100m, memory: 64Mi}
          securityContext:
            allowPrivilegeEscalation: false
            capabilities: {drop: [ALL]}
          volumeMounts:
            - {name: migration, mountPath: /migration, readOnly: true}
      volumes:
        - name: migration
          configMap:
            name: $name
EOF
  resources+=("job/$name")
done <<'SERVICES'
service-customer URL_BASE_DATOS_CLIENTES 000006
account-service URL_BASE_DATOS_CUENTAS 000005
transaction-service URL_BASE_DATOS_TRANSACCIONES 000004
payment-service URL_BASE_DATOS_PAGOS 000004
SERVICES
for resource in "${resources[@]}"; do
  if [[ "$resource" == job/* ]]; then
    if ! kubectl -n "$KUBE_NAMESPACE" wait --for=condition=complete "$resource" --timeout=120s; then
      echo "Migration failed: $resource; rollout stopped" >&2
      kubectl -n "$KUBE_NAMESPACE" logs "$resource" || true
      exit 1
    fi
  fi
done
