# Contratos de eventos

Los mensajes de **Bank USAC** utilizan un sobre común para intercambiar comandos, eventos y respuestas mediante RabbitMQ. El sobre aporta identificación, trazabilidad, versionado y metadatos; el campo `contenido` contiene el payload específico de cada routing key.

## Sobre común

```json
{
  "idMensaje": "7e214cec-baf4-4dc3-8b6d-fdcbb7150d8a",
  "idCorrelacion": "d11e65c6-69c2-4397-8630-020fde856f02",
  "idCausa": "dd828731-df44-4824-8540-adc5e04de519",
  "tipo": "cuenta.debitada",
  "version": 1,
  "ocurridoEn": "2026-09-03T14:30:00Z",
  "productor": "account-service",
  "contenido": {}
}
```

| Campo | Tipo | Obligatorio | Descripción |
|---|---|---:|---|
| `idMensaje` | UUID | Sí | Identifica de forma única el mensaje y permite detectar duplicados. |
| `idCorrelacion` | UUID | Sí | Relaciona todos los mensajes generados por la misma operación. |
| `idCausa` | UUID | No | Identifica el mensaje que causó la publicación actual. |
| `tipo` | String | Sí | Routing key y nombre contractual del mensaje. |
| `version` | Integer | Sí | Versión del contrato; actualmente se utiliza `1`. |
| `ocurridoEn` | ISO 8601 UTC | Sí | Fecha y hora en que se creó el mensaje. |
| `productor` | String | Sí | Componente que publica el mensaje. |
| `contenido` | JSON | Sí | Payload específico del comando o evento. |

## Reglas generales

1. `idMensaje` no se reutiliza en mensajes diferentes.
2. Los mensajes derivados de una misma operación conservan `idCorrelacion`.
3. Un mensaje derivado debe colocar el `idMensaje` anterior en `idCausa` cuando esté disponible.
4. Las fechas se expresan en UTC y formato ISO 8601.
5. Los montos se expresan como enteros en centavos.
6. La moneda inicial es `GTQ`.
7. Los campos desconocidos deben ignorarse cuando no alteren la interpretación del contrato.
8. Un cambio incompatible requiere incrementar `version`.
9. Los payloads no deben incluir contraseñas, hashes, JWT, secretos ni documentos completos innecesarios.
10. Los consumidores deben ser idempotentes utilizando `idMensaje` y su nombre de consumidor.

## Contratos de Customer Service

### Validar cliente

Routing key de comando: `cliente.validacion.solicitada`.

```json
{
  "idSolicitud": "91317212-237c-4c07-abf2-e507103c9e94",
  "idCliente": "ff46a46b-7de0-4683-b7ff-48034cc287e5"
}
```

Respuestas: `cliente.validado` o `cliente.rechazado`.

```json
{
  "idSolicitud": "91317212-237c-4c07-abf2-e507103c9e94",
  "idCliente": "ff46a46b-7de0-4683-b7ff-48034cc287e5",
  "activo": true,
  "motivo": ""
}
```

### Cliente creado

Routing key: `cliente.creado`.

```json
{
  "idCliente": "ff46a46b-7de0-4683-b7ff-48034cc287e5",
  "nombreCompleto": "Ana López",
  "correo": "ana@ejemplo.com",
  "usuario": "alopez",
  "documento": "IDENTIFICADOR_CONTROLADO",
  "rol": "CLIENTE",
  "estado": "PENDIENTE_ACTIVACION"
}
```

### Solicitud de correo de activación

Routing key: `notificacion.correo-activacion.solicitado`.

```json
{
  "idCliente": "ff46a46b-7de0-4683-b7ff-48034cc287e5",
  "correo": "ana@ejemplo.com",
  "nombreCompleto": "Ana López",
  "enlaceActivacion": "http://localhost:30080/activar?token=TOKEN",
  "expiraEn": "2026-09-03T15:30:00Z"
}
```

### Cliente activado

Routing key: `cliente.activado`.

```json
{
  "idCliente": "ff46a46b-7de0-4683-b7ff-48034cc287e5",
  "activadoEn": "2026-09-03T14:40:00Z"
}
```

### Cliente actualizado

El payload de actualización y su respuesta conservan como mínimo los siguientes campos:

```json
{
  "idCliente": "ff46a46b-7de0-4683-b7ff-48034cc287e5",
  "direccion": "Ciudad de Guatemala",
  "correo": "nuevo@ejemplo.com"
}
```

## Contratos de Account Service

### Crear cuenta

Routing key: `cuenta.creacion.solicitada`.

```json
{
  "idSolicitud": "0e917de7-56ad-4533-b8bb-1c6536e9af65",
  "idCliente": "ff46a46b-7de0-4683-b7ff-48034cc287e5",
  "tipoCuenta": "AHORRO"
}
```

Account Service publica `cliente.validacion.solicitada` antes de completar el proceso. El resultado final se comunica mediante `cuenta.creada` o `cuenta.creacion.rechazada`.

### Consultar cuenta

Routing key: `cuenta.consulta.solicitada`.

```json
{
  "idCuenta": "dbd100f4-a377-460d-a707-88be2b17ca73"
}
```

La respuesta utiliza `cuenta.consultada` y contiene la representación de la cuenta autorizada.

### Listar cuentas

Routing key: `cuenta.historial.solicitado`.

```json
{
  "idCliente": "ff46a46b-7de0-4683-b7ff-48034cc287e5"
}
```

La respuesta `cuenta.historial.consultado` utiliza esta estructura:

```json
{
  "idCliente": "ff46a46b-7de0-4683-b7ff-48034cc287e5",
  "cuentas": []
}
```

### Listar movimientos

Routing key: `cuenta.movimientos.solicitados`.

```json
{
  "idCuenta": "dbd100f4-a377-460d-a707-88be2b17ca73",
  "limite": 20,
  "desplazamiento": 0
}
```

La respuesta `cuenta.movimientos.consultados` contiene `idCuenta` y una colección `movimientos`.

### Débito, crédito y compensación

Routing keys:

- `cuenta.debito.solicitado`.
- `cuenta.credito.solicitado`.
- `cuenta.compensacion.solicitada`.

```json
{
  "idCuenta": "dbd100f4-a377-460d-a707-88be2b17ca73",
  "idOperacion": "18236dfb-c887-48af-9780-82270b09a166",
  "montoCentavos": 12500
}
```

El resultado utiliza una de las routing keys `cuenta.debitada`, `cuenta.debito.rechazado`, `cuenta.acreditada`, `cuenta.credito.rechazado`, `cuenta.compensada` o `cuenta.compensacion.rechazada`.

```json
{
  "idOperacion": "18236dfb-c887-48af-9780-82270b09a166",
  "idCuenta": "dbd100f4-a377-460d-a707-88be2b17ca73",
  "montoCentavos": 12500,
  "codigo": "",
  "mensaje": ""
}
```

## Contratos de Transaction Service

### Solicitar transferencia

Routing key principal: `transferencia.solicitada`.

```json
{
  "idTransferencia": "18236dfb-c887-48af-9780-82270b09a166",
  "idCliente": "ff46a46b-7de0-4683-b7ff-48034cc287e5",
  "idCuentaOrigen": "dbd100f4-a377-460d-a707-88be2b17ca73",
  "idCuentaDestino": "8963e68e-1478-405d-ad2d-a4a1971d75aa",
  "montoCentavos": 12500,
  "descripcion": "Transferencia de ejemplo"
}
```

Durante la integración también se acepta temporalmente `transfer.requested`, cuyo sobre y payload utilizan nombres en inglés. No debe utilizarse para contratos nuevos.

### Resultado de movimiento para la Saga

```json
{
  "idOperacion": "18236dfb-c887-48af-9780-82270b09a166",
  "idCuenta": "dbd100f4-a377-460d-a707-88be2b17ca73",
  "montoCentavos": 12500,
  "codigo": "",
  "mensaje": ""
}
```

Según el resultado, Transaction Service publica eventos de estado como `transferencia.procesando`, `transferencia.completada`, `transferencia.rechazada`, `transferencia.compensando`, `transferencia.compensada` o `transferencia.compensacion.fallida`.

### Consultar transferencia

Routing key: `transferencia.consulta.solicitada`.

```json
{
  "idTransferencia": "18236dfb-c887-48af-9780-82270b09a166"
}
```

### Consultar historial

Routing key: `transferencia.historial.solicitado`.

```json
{
  "idCliente": "ff46a46b-7de0-4683-b7ff-48034cc287e5",
  "limite": 20,
  "desplazamiento": 0
}
```

La respuesta `transferencia.historial.consultado` contiene `idCliente` y una colección `transferencias`.

## Contratos de Payment Service

### Procesar pago

Routing key: `pago.procesamiento.solicitado`.

```json
{
  "idPago": "61aa78ac-e92f-48d8-89b1-151f331fd1a0",
  "idCliente": "ff46a46b-7de0-4683-b7ff-48034cc287e5",
  "idCuentaOrigen": "dbd100f4-a377-460d-a707-88be2b17ca73",
  "beneficiario": "Empresa de ejemplo",
  "concepto": "Pago de servicio",
  "montoCentavos": 7500,
  "tipoPago": "EXTERNO"
}
```

Payment Service solicita el débito a Account Service. Después publica `pago.completado` o `pago.rechazado`.

### Consultar pago

Routing key: `pago.consulta.solicitada`.

```json
{
  "idPago": "61aa78ac-e92f-48d8-89b1-151f331fd1a0"
}
```

La respuesta utiliza `pago.consultado` y contiene la representación del pago autorizado.

### Consultar historial de pagos

Routing key: `pago.historial.solicitado`.

```json
{
  "idCliente": "ff46a46b-7de0-4683-b7ff-48034cc287e5",
  "limite": 20,
  "desplazamiento": 0
}
```

La respuesta `pago.historial.consultado` contiene `idCliente` y una colección `pagos`.

## Contratos de Notification & Audit Service

### Consultar registros

Routing key: `auditoria.registros.solicitados`.

El payload contiene los filtros admitidos por la consulta. La respuesta incluye los registros de auditoría encontrados sin modificar los eventos originales.

### Consultar traza

Routing key: `auditoria.traza.solicitada`.

```json
{
  "idCorrelacion": "d11e65c6-69c2-4397-8630-020fde856f02"
}
```

La respuesta reúne cronológicamente los eventos almacenados para esa operación.

### Consultar notificaciones

Routing key: `auditoria.notificaciones.solicitadas`.

El payload contiene los filtros de destinatario, estado o correlación que soporte la consulta. La respuesta entrega únicamente resúmenes seguros de las notificaciones.

## Compatibilidad y versionado

Los consumidores deben seleccionar la estructura del payload según `tipo` y `version`. Agregar un campo opcional mantiene la compatibilidad. Eliminar un campo, cambiar su significado o modificar su tipo requiere una nueva versión.

La compatibilidad temporal con `transfer.requested` y con campos de transferencia en inglés existe para integrar componentes desarrollados en paralelo. El contrato canónico del proyecto utiliza el sobre y los nombres de campos en español.

## Manejo de errores

- Un mensaje con JSON inválido no debe procesarse como operación de negocio.
- Un contrato incompleto debe rechazarse indicando el motivo sin exponer información sensible.
- Los fallos temporales utilizan reintentos controlados.
- Al agotar los reintentos, el mensaje se dirige a la DLQ correspondiente.
- Un mensaje duplicado se reconoce mediante `idMensaje` y no vuelve a aplicar efectos.
- El productor no debe quedar bloqueado mientras un consumidor se recupera.
