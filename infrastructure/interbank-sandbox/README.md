# Laboratorio interbancario del panel admin

## Uso principal: nube GCP

El pipeline de `develop` publica el mock y ejecuta `deploy-cloud.py`. El panel se
abre en el portal HTTPS habitual, entrando como ADMIN y seleccionando
**Pruebas interbancarias**. No requiere laptop encendida, túnel ni login de demo.

- Servicio privado `interbank-scenarios` en GKE, con Gateway, Cuentas,
  Transacciones, RabbitMQ y mock en un pod. No crea IP pública ni balanceador.
- PostgreSQL permanece **fuera de Kubernetes**: usa el servidor existente, con
  esquema y rol `bank_scenarios`, credenciales generadas y almacenadas en Secret.
  Las conexiones de los servicios apuntan únicamente a ese esquema, nunca a `public`.
  El despliegue inicial necesita permiso para crear rol/esquema en esas bases.
- La cuenta ficticia comienza con Q100,000. Migraciones posteriores no reinician
  su saldo. RabbitMQ es independiente y el PVC de 2 GiB conserva su estado y SQLite.
- El Gateway habitual verifica JWT ADMIN y reenvía solo las rutas del panel al
  servicio privado fijo. El mock escucha en loopback. Su control exige otra clave.
  Las cuentas normales y la configuración/credenciales de BANK14 no se cambian.
- El pod recibe telemetría OTLP del proyecto con entorno `interbank-cloud-tests`.
  En Grafana Explore filtrar ese entorno y buscar el ID de la prueba. No usar
  únicamente los paneles filtrados a `dev` para demostrar pruebas del laboratorio.
- Recursos solicitados: 150m CPU / 304 MiB RAM; límites agregados 1.4 CPU /
  1,056 MiB RAM. PVC 2 GiB. Revisar capacidad antes de ampliar escenarios.
- El escenario seleccionado es compartido entre administradores; coordinar la
  demostración para no cambiarlo antes de que el mock reciba una solicitud.
- No adapta el contrato a las incompatibilidades de BANK14. La conciliación real
  sigue pendiente hasta que el banco pareja respete el contrato.

Al seleccionar un escenario **no se envía dinero**. Usar el formulario para enviar
una prueba y conservar el ID hasta un resultado definitivo. Para mostrar recuperación
de un 503, restablecer normal mientras el mismo depósito sigue pendiente.

## Alternativa de desarrollo local (opcional)

La siguiente configuración local **no cambia BANK14 ni usa las cuentas, secretos, RabbitMQ o bases de GCP**.
Ejecuta Gateway, Clientes, Cuentas y Transacciones del proyecto, con sus migraciones
y almacenamiento independiente, contra el mock Python del contrato original.
El mock no sustituye a WireMock si el auxiliar exige específicamente esa herramienta.
No se modificó `contracts/interbank/openapi.yaml`.

## Arranque (raíz del repositorio, Docker con Compose)

```sh
docker compose -f infrastructure/interbank-sandbox/compose.yaml up --build -d
```

Abrir `http://127.0.0.1:8180`, entrar con **sandbox-admin / SandboxDemo2026!** y
seleccionar **Pruebas interbancarias**. Son credenciales ficticias y locales, no
usarlas en el banco público. El broker y PostgreSQL no publican puertos.

1. Elegir un escenario y esperar confirmación. Cada solicitud conserva el escenario
   que tenía al entrar; cambiarlo no reescribe depósitos pendientes.
2. Usar el formulario explícito **Enviar nuevo depósito de prueba**. No se envía
   dinero al seleccionar un escenario. El Gateway fija cliente y cuenta del sandbox:
   origen `900000000003`, destino `TEST-BANK14-001`. No admite otra URL ni cliente.
3. Conservar el ID, observar estado y abrir **Administrar cuentas** para ver el saldo
   y la reserva. Saldo inicial Q100,000, comisión ficticia Q0.
4. Antes de otra prueba, esperar un resultado terminal. Si se pierde la respuesta,
   consultar el mismo ID; no pulsar Send repetidamente ni borrar el almacenamiento.

## Escenarios

| Botón | Resultado a observar |
| --- | --- |
| Operación normal | Callback, ACCEPTED y un solo débito. |
| Rechazo | REJECTED remoto, liberación del saldo; el coordinador usa REVERSED al liberar una reserva. |
| Error 503 | Reintentos conservando ID. Restablecer normal antes de 180 s para mostrar recuperación; si vence, mostrar la resolución/cancelación. |
| Sin callback | El mock acepta, no notifica. El banco recupera el resultado mediante un reintento idempotente o una consulta tras TIMEOUT, sin duplicar. No obliga a llegar al timeout. |
| Respuesta demorada | Procesa a los 181 s: cancelación o EXPIRED; no acredita fuera de plazo. No equivale a un callback tardío ACCEPTED. |

En timeout la reserva puede permanecer hasta comprobar el resultado/cancelación.
No reducir los 180 s del contrato para hacer la exposición más rápida.

Verificación automática opcional (crea cinco depósitos en las cuentas ficticias):

```sh
python infrastructure/interbank-sandbox/smoke.py --execute
```

Sin `--execute` solo comprueba login, aislamiento, permisos y configuración.

## Aislamiento y alcance

- Red Docker interna, claves solo de fixtures locales, SQLite del mock y PostgreSQL
  independientes. La API de control escucha solo en loopback; requiere una clave
  distinta de la del contrato. Las rutas del panel requieren JWT ADMIN en servidor.
- El control directo del mock permanece **deshabilitado** en el banco dev/prod. Para habilitarlo en la instancia aislada deben coincidir
  `ENTORNO=interbank-sandbox`, `INTERBANK_TEST_MODE=true`, la URL de control loopback y
  el destino interbancario local. Las rutas no permiten cambiar el destino.
- El Compose local no se despliega por push; GCP usa el script anterior. No expone públicamente
  PostgreSQL, RabbitMQ o el control del mock. Las claves demo no son secretos reales.
- El Compose no incluye Grafana/Collector: el panel muestra el enlace a Grafana solo
  si se configura `INTERBANK_TEST_GRAFANA_URL` en el Gateway. Un enlace no configura
  por sí solo telemetría: debe conectarse un Collector del sandbox a los servicios
  mediante OTLP. Las consultas deben filtrar `interbank-sandbox`, no `dev`.
- Auditoría de cambios: tabla SQLite `control_audit` con actor, instante y escenario.
  Un reinicio restaura el escenario de arranque normal; las operaciones conservan
  el escenario persistido con el que se recibieron. Consultar el panel tras reiniciar.

## Parada recuperable

```sh
docker compose -f infrastructure/interbank-sandbox/compose.yaml down
```

No elimina los volúmenes. No borrar datos mientras haya operaciones pendientes.
Para conservar evidencia, guardar IDs, estados, saldo/reservas y logs antes de detener.
