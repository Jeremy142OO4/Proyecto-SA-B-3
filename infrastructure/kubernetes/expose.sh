#!/usr/bin/env sh
set -eu
DIRECCION_ACCESO=${DIRECCION_ACCESO:-127.0.0.1}

iniciar_tunel() {
  nombre=$1
  puerto_local=$2
  puerto_servicio=$3
  archivo_pid="/tmp/bank-usac-${nombre}.pid"

  if [ -f "$archivo_pid" ] && kill -0 "$(cat "$archivo_pid")" 2>/dev/null; then
    return
  fi

  nohup sh -c 'hijo=""
  finalizar() { [ -n "$hijo" ] && kill "$hijo" 2>/dev/null || true; }
  trap finalizar TERM INT EXIT
  while :; do
    kubectl -n bank-usac port-forward --address="$4" "service/$1" "$2:$3" &
    hijo=$!
    wait "$hijo" || true
    hijo=""
    sleep 2
  done' sh "$nombre" "$puerto_local" "$puerto_servicio" "$DIRECCION_ACCESO" \
    >"/tmp/bank-usac-${nombre}.log" 2>&1 </dev/null &
  echo $! > "$archivo_pid"
}

iniciar_tunel frontend 3000 80
iniciar_tunel account-service 8082 8082
iniciar_tunel payment-service 8084 8084
iniciar_tunel rabbitmq 15672 15672

echo "Frontend: http://localhost:3000"
echo "Account Service: http://localhost:8082/salud"
echo "Payment Service: http://localhost:8084/salud"
echo "RabbitMQ: http://localhost:15672"
