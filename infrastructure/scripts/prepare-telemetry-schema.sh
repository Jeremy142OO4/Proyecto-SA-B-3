#!/usr/bin/env bash
# Existing schemas may be initialized manually: apply additive, repeatable DDL.
# Keep this gate before publishing the new frontend or rolling backend pods.
set -euo pipefail
: "${KUBE_NAMESPACE:?KUBE_NAMESPACE is required}"
suffix="${GITHUB_RUN_ID:-manual}-${GITHUB_RUN_ATTEMPT:-1}"
case "$suffix" in (*[!0-9a-z-]*) echo "Invalid run suffix" >&2; exit 1;; esac
resources=()
sql_dir=$(mktemp -d)
cleanup() {
  rm -rf "$sql_dir"
  for resource in "${resources[@]}"; do
    kubectl -n "$KUBE_NAMESPACE" delete "$resource" --ignore-not-found --wait=false >/dev/null || true
  done
}
trap cleanup EXIT
while read -r service key migrations; do
  name="${service}-schema-${suffix}"
  migration="$sql_dir/$service.sql"
  # Serialize deployments sharing the same database, including dev/prod.
  printf "SELECT pg_advisory_xact_lock(hashtext('bank-usac-schema'));\n" > "$migration"
  for filename in $migrations; do
    source="services/$service/database/migrations/$filename.up.sql"
    test -f "$source"
    cat "$source" >> "$migration"
    printf '\n' >> "$migration"
  done
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
service-customer URL_BASE_DATOS_CLIENTES 000006_persistir_contexto_traza 000007_customer_lifecycle
account-service URL_BASE_DATOS_CUENTAS 000005_persistir_contexto_traza 000006_monitoreo_cuentas 000007_reservas_interbancarias 000008_recuperacion_interbancaria
transaction-service URL_BASE_DATOS_TRANSACCIONES 000004_persistir_contexto_traza 000005_traza_transferencia 000006_interbancaria 000007_recuperacion_interbancaria
payment-service URL_BASE_DATOS_PAGOS 000004_persistir_contexto_traza 000005_simulacion_avanzada
service-notification-audit URL_BASE_DATOS_AUDITORIA 000002_add_event_severity 000003_nivel_notificaciones 000004_consulta_auditoria_trazas
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
