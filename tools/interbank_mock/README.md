# Mock local del contrato interbancario — punto 6

Sustituto HTTP de prueba implementado con Python, biblioteca estándar y PyYAML. No usa WireMock, no conecta a bases del proyecto ni mueve dinero real. La integración real debe llamar al banco pareja, no a este proceso. No se incluye en Kubernetes, Cloud Run ni las imágenes del backend.

## Ejecución desde la raíz del repositorio

```sh
python -m pip install -r tools/interbank_mock/requirements.txt
# Generar una clave LOCAL de prueba, distinta de las claves reales.
export MOCK_API_KEY="$(python -c 'import secrets; print(secrets.token_urlsafe(32))')"
python -m tools.interbank_mock.app --port 8099
```

En PowerShell:

```powershell
$env:MOCK_API_KEY = python -c "import secrets; print(secrets.token_urlsafe(32))"
python -m tools.interbank_mock.app --port 8099
```

Salud: `http://127.0.0.1:8099/interbank/v1/health`. Es HTTP solamente para pruebas en loopback, no una URL válida de integración real. El servidor siempre se enlaza a `127.0.0.1`; no exponerlo con un túnel público. Para TLS local se pueden proporcionar `--cert` y `--key` con certificados confiables para el cliente; no se desactiva su validación.

Por defecto representa a BANK14 y acepta la cuenta **ficticia** `TEST-BANK14-001`. `MOCK_ACCOUNTS` permite indicar otras cuentas separadas por comas; no se considera una comprobación de cuentas del banco real. `--bank` y `--partner` permiten invertir los roles.

Configuración opcional de callbacks: `MOCK_PARTNER_BASE_URL` y `MOCK_PARTNER_API_KEY`. La URL es fija, no proviene del cuerpo de la solicitud, y debe terminar en `/interbank/v1`. El mock solo admite destinos de callback en loopback, también con TLS; no permite llamar accidentalmente al banco real. Las redirecciones no reciben la credencial. Sin URL de callback no se hace ninguna llamada saliente: el resultado sigue consultable por GET.

## Escenarios

- `--scenario accept`: procesa después de `--delay` segundos y registra una acreditación ficticia única. Rechaza cuentas no listadas, importes fuera del límite y operaciones vencidas.
- `--scenario reject`: genera `REJECTED/PROCESSING_ERROR`.
- `--scenario no_callback`: procesa pero no envía callbacks, para probar la resolución por consulta.
- `--scenario unavailable`: `/health` y nuevas solicitudes de depósito devuelven `503`.
- `--delay 181`: una solicitud recibida a tiempo vence antes de procesarse; nunca se registra acreditación después de `expiresAt`.

La configuración de escenarios es local al mock. No modifica el sistema de simulación de Pagos ni decide cómo responderá BANK14 realmente.

## Panel de administración aislado

El panel **Pruebas interbancarias** permite elegir escenarios sin reiniciar procesos.
El entorno completo y sus fixtures se encuentran en
[`infrastructure/interbank-sandbox`](../../infrastructure/interbank-sandbox/README.md).
No cambia automáticamente la URL de BANK14 del banco en GCP.

La API privada `/__mock/control` admite GET y PUT con `{"scenario":"accept"}`
(también `reject`, `unavailable`, `no_callback`, `delayed`). Está deshabilitada por
defecto; se habilita con `MOCK_CONTROL_KEY`, diferente de `MOCK_API_KEY`, y requiere
el header `X-Mock-Control-Key`. El Gateway valida ADMIN y nunca entrega esa clave
al navegador. No es parte del OpenAPI interbancario. Escenario y demora quedan
capturados por depósito; las decisiones ya iniciadas no cambian al restablecer
el comportamiento normal. La auditoría registra actor/fecha/escenario sin claves.

## Alcance y límites

El contrato original se verifica por SHA-256; el validador implementa el subconjunto de esquemas utilizado por ese archivo, no es un validador OpenAPI de propósito general. Se verifican tipos, campos requeridos, campos extra prohibidos, enums, patrones, fechas, UUIDs y respuestas exitosas contra los esquemas originales.

Hay estado persistente SQLite de prueba, idempotencia/conflictos, procesamiento diferido, cancelación con tombstone, resultados consultables, callbacks con espera creciente hasta 24 h y conciliación por fecha UTC con paginación. Reiniciar el proceso retoma depósitos pendientes y callbacks, pero no crea nuevas acreditaciones para IDs ya procesados.

Los reversos publican un `refundTransferId` estable desde PROCESSING y, si se configuró una URL de pareja en loopback, envían un nuevo depósito de retorno por HTTP. COMPLETED requiere su aceptación y registra un débito ficticio único en `refund_debits`; un retorno rechazado queda FAILED. Sin URL permanece PROCESSING: no interpretar ese estado como dinero devuelto. El receptor de callbacks reconoce fixtures de `State.register_outbound()` y los retornos creados por reversos válidos, no solicitudes arbitrarias. El mock sigue sin manejar saldos reales y no sustituye las pruebas entre bancos.

No se modelan reservas, saldos reales, débito del emisor, reconciliación de libros contables ni alertas de producción. Las pruebas verdes del mock no cierran los puntos 7, 8 u 11.

## Pruebas y CI/CD

```sh
python -m unittest discover -s tools/interbank_mock -t . -p 'test_*.py' -v
```

No se requiere la laptop, Docker, GCP, claves reales ni servicios externos. Incluye dos servidores HTTP locales para verificar callback, autenticación, reintento y ACK. Prueba también concurrencia, persistencia, expiración sin acreditar, cancelación previa a llegada, formatos y paginación.

El job `Validar contrato interbancario y mock` se activa por cambios en `contracts/interbank/` o `tools/interbank_mock/`. Esas rutas por sí solas no reconstruyen ni despliegan microservicios. Cambios globales del workflow siguen activando la validación completa; un fallo de este job impide publicar imágenes mediante el gate del sistema. Documentación sola conserva el comportamiento de omitir builds.

Los archivos `.state/` son datos locales de prueba ignorados por Git. No almacenar allí claves ni datos de clientes reales.
