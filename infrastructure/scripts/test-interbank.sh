#!/usr/bin/env bash
# Disposable local test dependencies only. Never reads application DB secrets.
set -euo pipefail
component="${1:?account-service, transaction-service payment-service or api-gateway required}"
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
case "$component" in
  account-service|transaction-service) module="$root/services/$component"; image=postgres:17-alpine ;;
  api-gateway) module="$root/gateway/api-gateway"; image=rabbitmq:3.13-alpine ;;
  payment-service) module="$root/services/payment-service"; image=rabbitmq:3.13-alpine ;;
  *) echo 'Unknown test component' >&2; exit 1 ;;
esac
name="bank-interbank-${component}-${GITHUB_RUN_ID:-local}-${GITHUB_RUN_ATTEMPT:-1}-$$"
cleanup() {
  local test_status=$?
  if [[ "$test_status" != 0 ]]; then docker logs "$name" || true; fi
  docker rm -f -v "$name" >/dev/null 2>&1 || true
  return "$test_status"
}
trap cleanup EXIT
unset TEST_DATABASE_URL TEST_AMQP_URL INTERBANK_MOCK_PYTHON
if [[ "$component" == api-gateway || "$component" == payment-service ]]; then
  # Erlang otherwise sizes server AND diagnostic CLI schedulers from the host
  # CPU count, which can exhaust runner threads before AMQP starts. Keep the
  # container after a failed boot until cleanup has captured its logs.
  docker run -d --name "$name" --cpus=1 -e ERL_FLAGS='+S 2:2 +A 4' -e RABBITMQ_DEFAULT_USER=interbank_test -e RABBITMQ_DEFAULT_PASS=local-interbank-only -p 127.0.0.1::5672 "$image"
  ready=false
  for attempt in {1..60}; do
    if [[ "$(docker inspect --format '{{.State.Running}}' "$name")" != true ]]; then echo 'Test broker exited before readiness' >&2; exit 1; fi
    # A root CLI started before the server can create a root-owned cookie and
    # make the service fail with .erlang.cookie: eacces. Use its actual user.
    if docker exec --user rabbitmq "$name" rabbitmq-diagnostics -q check_port_connectivity >/dev/null 2>&1; then ready=true; break; fi
    sleep 1
  done
  if [[ "$ready" != true ]]; then docker logs "$name"; exit 1; fi
  port="$(docker port "$name" 5672/tcp | cut -d: -f2)"
  export TEST_AMQP_URL="amqp://interbank_test:local-interbank-only@127.0.0.1:${port}/"
  cd "$module"
  if [[ "$component" == payment-service ]]; then
    go test -race -count=1 -v ./internal/interbank
  else
    go test -race -count=1 -v ./internal/bankrpc
  fi
else
  docker run -d --rm --name "$name" -e POSTGRES_HOST_AUTH_METHOD=trust -p 127.0.0.1::5432 "$image"
  ready=false
  for attempt in {1..60}; do
    if docker exec "$name" pg_isready -h 127.0.0.1 -U postgres; then ready=true; break; fi
    sleep 1
  done
  if [[ "$ready" != true ]]; then docker logs "$name"; exit 1; fi
  port="$(docker port "$name" 5432/tcp | cut -d: -f2)"
  export TEST_DATABASE_URL="postgres://postgres@127.0.0.1:${port}/postgres?sslmode=disable"
  if [[ "$component" == transaction-service ]]; then
    python3 -m venv /tmp/"${name}-python"
    export INTERBANK_MOCK_PYTHON="/tmp/${name}-python/bin/python"
    "$INTERBANK_MOCK_PYTHON" -m pip install --disable-pip-version-check -r "$root/tools/interbank_mock/requirements.txt"
  fi
  cd "$module"
  go test -race -count=1 -v ./internal/interbank
fi
