# Preparación de integración: BANK03 y BANK14

## Estado y fuente de verdad

La implementación de depósitos, reservas, callbacks, recuperación, reversos y conciliación se describe en [depósitos interbancarios](depositos.md). La recepción se habilita independientemente de la URL de BANK14. Su dirección ya está configurada en dev y en la variable del CI/CD `INTERBANK_PARTNER_BASE_URL`. Se verificaron HTTPS, salud, autenticación y consultas de solo lectura; también la conectividad desde el servicio de transacciones en GKE. No se ha desplegado el mock ni se ha probado una transferencia real con BANK14.

El [OpenAPI original](../../contracts/interbank/openapi.yaml), versión 1.0.0, es la fuente de verdad. La guía recibida de BANK14 es contexto de integración, no una modificación del contrato. Su archivo original contiene claves y no se incorpora al repositorio.

## Datos acordados y pendientes

| Dato | BANK03 | BANK14 |
| --- | --- | --- |
| Grupo | 3 | 14 |
| Código oficial | `BANK03` | `BANK14` |
| URL HTTPS interbancaria | `https://bank-usac-frontend-dev-116700127937.us-central1.run.app/interbank/v1` (Swagger: `/swagger/`) | `https://api.pitagoricos.tech/interbank/v1` |
| Credencial de entrada | Intercambiada por privado; configurada como Secret por CI, sin publicarla en Git | Configurada como Secret; verificada en consultas autenticadas de solo lectura |
| Cuentas de prueba | Pendientes de preparar/confirmar, sin asumir que un UUID interno sea número interbancario | Valores abajo; formato aclarado, todavía no verificados contra su API |

BANK14 confirmó retirar los dos últimos caracteres de sus identificadores originales:

| Cuenta BANK14 | `destinationAccount` de 34 caracteres |
| --- | --- |
| Monetaria | `6309429d-32d0-4db7-9b33-024ce45f8d` |
| Ahorro | `bbb4eca9-fc3f-415c-a317-d64e6e2aca` |

Son identificadores opacos publicados por el banco dueño de la cuenta, no UUID que debamos reconstruir. No recortar otros valores automáticamente. Falta verificar que BANK14 reconozca exactamente estos números cuando comparta su URL.

### Verificaciones del 19 de octubre de 2026

- `GET /interbank/v1/health`: `200`, banco `BANK14`.
- Consulta autenticada de una transferencia inexistente: `404 NOT_FOUND`, sin error de autenticación.
- Consulta autenticada de conciliación: `200`, pero devuelve `totalTransfers`, `totalAmount` y `transfers`, en lugar de la estructura esperada por nuestra implementación. Pendiente contrastar con el OpenAPI original y resolver la incompatibilidad antes de validar la conciliación conjunta.
- No se crearon depósitos ni se movieron fondos durante estas verificaciones. La activación de la configuración saliente permite al servicio reanudar sus trabajos automáticos pendientes.

## Dirección de las claves

- BANK03 → BANK14: enviar la clave que BANK14 valida.
- BANK14 → BANK03: enviar la clave que BANK03 valida.
- El header es `X-Api-Key`. Las claves se guardarán en Secrets y configuración privada del CI/CD, nunca en Git, ejemplos curl con valores reales, logs, imágenes o frontend.
- Las claves elegidas para la integración académica son predecibles; no reutilizarlas fuera de este entorno y rotarlas si se publican accidentalmente.
- HTTPS con validación de certificados sigue siendo parte del contrato. No usar `verify=False`, `curl -k`, HTTP público ni credenciales en la URL.

## Aclaraciones del contrato

1. Cuentas: caracteres alfanuméricos/guiones, entre 1 y 34; códigos `BANK` más dos dígitos.
2. Callback: responder `200` con `interbankTransferId` y `received: true`, no `acknowledged`.
3. Si el depósito ya llega vencido: `422`, sin procesarlo. Si vence antes de acreditar después de su recepción: callback `REJECTED` con `EXPIRED`.
4. La palabra informal “refund” utilizada en la conversación no cambia estas reglas. Se interpreta provisionalmente como rechazo cuando no hubo acreditación; un reverso es un procedimiento distinto. Confirmar en las pruebas con BANK14.
5. `Idempotency-Key` igual a `interbankTransferId` en `POST /transfers`. Los reintentos conservan id y contenido; un cambio de contenido con el mismo id devuelve `409`.
6. Todas las operaciones declaradas con `traceparent` lo requieren; `/health` es público. Importes como texto con dos decimales, GTQ, fechas UTC y vencimiento creación + 180 s.
7. TIMEOUT conserva la reserva; consulta/cancelación y margen adicional de 300 s siguen las reglas completas del OpenAPI. El mock de preparación no demuestra la seguridad financiera de nuestra futura implementación.

## Configuración preparada para los puntos 7 y 8

Variables: `INTERBANK_BANK_CODE=BANK03`, `INTERBANK_PARTNER_BANK_CODE=BANK14`, `INTERBANK_PUBLIC_BASE_URL`, `INTERBANK_PARTNER_BASE_URL`, `INTERBANK_API_KEY` y `INTERBANK_PARTNER_API_KEY`.

Las URLs base incluyen exactamente `/interbank/v1`. No agregarlo dos veces al construir los endpoints. `tools/interbank_mock/config.py` valida el acuerdo sin imprimir claves; el backend Go tiene su propia validación y queda desactivado mediante `INTERBANK_ENABLED=false`. No modificar la URL del frontend ni mover su IP para preparar este punto. La URL HTTPS del frontend, por sí sola, no demuestra que exista una API interbancaria detrás.

## Cierre pendiente con el equipo pareja

- Intercambiar y verificar las dos URLs HTTPS y las cuentas de BANK03.
- Configurar las claves en el entorno privado y verificar salud y autenticación.
- Acordar horarios y responsables de los escenarios de prueba; no inventar estados o formatos fuera del contrato.
- Confirmar que BANK14 publique `refundTransferId` desde PROCESSING, como BANK03, para identificar una devolución antes de su callback y evitar abonar de nuevo una reserva ya liberada.
- Probar depósitos reales en ambos sentidos, duplicados, vencimiento, cancelación, callbacks tardíos, reversos y conciliación después de implementar los puntos 7 y 8.
