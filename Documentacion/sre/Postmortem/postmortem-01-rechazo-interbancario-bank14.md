# Postmortem Blameless — Rechazo de transferencia interbancaria por cuenta destino inválida

| Campo | Valor |
|---|---|
| **ID del incidente** | PM-F3-01 |
| **Fecha** | 2026-10-20 |
| **Hora de inicio** | 17:48:32 UTC (11:48:32 UTC-6) |
| **Hora de resolución** | 19:41:55 UTC (13:41:55 UTC-6) |
| **Duración del impacto** | 3 segundos en la contraparte  |
| **Severidad** | SEV-3 (fallo de negocio aislado, sin pérdida de datos) |
| **Servicios involucrados** | Frontend (Cloud Run), API Gateway, Transaction Service, Account Service, Payment Service |
| **Banco pareja** | BANK14 |
| **Autores** | Equipo BANK03 |
| **Estado** | Resuelto en la contraparte / Pendiente de compensación interna |

---

## 1. Resumen ejecutivo

Durante una prueba de envío de depósito interbancario desde BANK03 hacia BANK14, la solicitud fue aceptada localmente con estado `PROCESSING` y rechazada tres segundos después por el banco pareja con el código `ACCOUNT_NOT_FOUND`.

La causa fue el envío de un identificador de cuenta destino con formato UUID inválido (último segmento de 10 caracteres hexadecimales en lugar de 12), que BANK14 no pudo resolver. El incidente expuso una ausencia de validación de formato del identificador de cuenta destino antes de emitir la solicitud al banco pareja, así como la inexistencia de una capa de traducción entre esquemas de identificador de ambos bancos.

**Escenario controlado:** el fallo es reproducible a voluntad enviando una transferencia hacia una cuenta inexistente en BANK14, por lo que se documenta como prueba de inyección de fallo del entregable de Fase 3.

---

## 2. Impacto

| Dimensión | Valor |
|---|---|
| Transferencias afectadas | 1 (`ec028648-36b3-4402-9431-dbcffe853cbc`) |
| Monto | GTQ 100.00 |
| Cuenta origen | `107102621189` (cuenta de prueba) |
| Cuenta destino | `b29d1efe-e59d-4749-a127-7d5d311f06` (UUID malformado) |
| Usuarios afectados | 1 |
| Fondos retenidos |  ¿se liberó la reserva? Sí  |
| Impacto en SLI de éxito interbancario | 75% sobre 5 intentos del período |
| Consumo de error budget |  0.01% del presupuesto mensual |
| Pérdida de datos | Ninguna |

---

## 3. Cronología de los hechos (UTC)

| Hora | Evento | Fuente de evidencia |
|---|---|---|
| 17:48:32.282 | `POST /api/interbancarias/` aceptado. Respuesta con `status: PROCESSING` y `receivedAt` | ![rechazo](../imgF3/request-transferencia.jpeg)|
| 17:48:32 | Transaction Service registra la transferencia y solicita la reserva de fondos a Account Service |  |
| 17:48:35.276 | BANK14 resuelve la transferencia como `REJECTED` con `reasonCode: ACCOUNT_NOT_FOUND` | ![rechazo0](../imgF3/rechazado.jpeg) |
|  17:48:33 | BANK03 recibe y procesa el rechazo | |




---

## 4. Detección y diagnóstico

- **Método de detección:** consulta manual del estado en el endpoint del banco pareja. No hubo alerta automática.
- **Alerta disparada:** Alertmanager

- **Consulta en Loki:**

```logql
{k8s_container_name="transaction-service"} |= "ec028648-36b3-4402-9431-dbcffe853cbc"
```

- **Consulta de spans en Tempo:** buscar por `interbankTransferId` o por el `traceId` anterior y verificar si el span de salida HTTP registra el código de respuesta de BANK14.
- **Verificación en base de datos:**

```sql
-- Estado final de la transferencia
SELECT id, status, reason_code, created_at, updated_at
FROM interbank_transfers
WHERE id = 'ec028648-36b3-4402-9431-dbcffe853cbc';

-- Movimientos de la cuenta origen para confirmar la liberación de la reserva
SELECT id, type, amount, status, created_at
FROM account_movements
WHERE account_id = '107102621189'
ORDER BY created_at DESC
LIMIT 10;
```

---

## 5. Causa raíz (análisis de los 5 porqués)

1. **¿Por qué se rechazó la transferencia?** BANK14 no encontró la cuenta destino indicada.
2. **¿Por qué no la encontró?** El identificador enviado no es un UUID válido: `b29d1efe-e59d-4749-a127-7d5d311f06` tiene 10 caracteres en el último segmento cuando el formato exige 12.
3. **¿Por qué se envió un identificador inválido?** No existe validación de formato del campo `destinationAccount` antes de emitir la solicitud al banco pareja.
4. **¿Por qué no existe esa validación?** El contrato interbancario define el campo como cadena libre y no se implementó normalización por esquema de banco.
5. **Causa fundamental:** el acuerdo técnico entre BANK03 y BANK14 no especifica el formato canónico del identificador de cuenta, y el sistema confía en que el llamador envía datos válidos.

**Factores contribuyentes:**

- Los esquemas de identificador son distintos entre bancos: el origen es numérico de 12 dígitos (`107102621189`) y el destino es UUID, sin capa de traducción.
- El estado `PROCESSING` se devuelve al cliente antes de conocer el resultado de la contraparte, lo que puede percibirse como éxito prematuro.
- No existe alerta sobre tasa de rechazos por `reasonCode`.

---


## 6. Lecciones aprendidas

- Un identificador malformado no debe llegar nunca al banco pareja: la validación de contrato es responsabilidad del emisor.
- El estado `PROCESSING` debe acompañarse de un mecanismo visible de resolución final para el usuario.
- La ausencia de alertas obligó a detectar el fallo por consulta manual; la observabilidad existía pero no estaba accionada sobre este caso de negocio.

---


