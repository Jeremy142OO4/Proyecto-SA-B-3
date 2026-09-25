# Diagrama de Secuencia – Procesamiento de Pago Externo

![Diagrama de secuencia pago externo](../Imagenes/pago%20externo.drawio.png)

## 1. Descripción general

Este diagrama describe el flujo de **procesamiento de pagos hacia sistemas externos** (simulados), incluyendo los tres escenarios posibles: **éxito**, **fallo** (rechazo del sistema externo) y **timeout** (sin respuesta dentro del tiempo límite). En caso de que el débito haya sido aplicado antes del fallo o timeout, se ejecuta una compensación para devolver el monto al cliente.

El **Payment Service** es el coordinador exclusivo de este flujo y actúa como intermediario entre el ecosistema interno (Account Service) y el sistema de pago externo simulado.

---

## 2. Participantes

| Actor / Sistema                    | Rol                                                                                             |
|------------------------------------|-------------------------------------------------------------------------------------------------|
| **Cliente**                        | Usuario final que inicia el pago externo                                                        |
| **Frontend**                       | Aplicación web/móvil que recoge la solicitud                                                    |
| **API Gateway**                    | Punto de entrada HTTP; valida JWT y publica el evento en el Broker                              |
| **Broker (RabbitMQ)**              | Bus de mensajes asíncrono; coordina los eventos entre servicios                                 |
| **Payment Service**                | Orquesta el flujo de pago externo; gestiona estados, reintentos y compensación                  |
| **Account Service**                | Ejecuta el débito preventivo y la compensación (reverso) sobre la cuenta del cliente            |
| **Account DB**                     | Base de datos de cuentas y saldos                                                               |
| **Sistema de Pago Externo (sim.)** | Sistema externo simulado que procesa o rechaza el pago                                          |
| **Notification & Audit**           | Registra auditoría de todos los eventos y envía notificaciones al cliente                       |
| **Notification DB**                | Almacenamiento de notificaciones y registros de auditoría                                       |

---

## 3. Estados del pago

```
PENDIENTE
   → DEBITANDO        (débito preventivo en proceso)
       → PROCESANDO   (débito aplicado; esperando respuesta del sistema externo)
           → COMPLETADO         (pago.completado recibido)
           → FALLIDO            (rechazo sin débito previo)
           → COMPENSANDO        (fallo/timeout con débito previo → revertir)
               → COMPENSADO     (débito revertido con éxito)
               → COMPENSACION_FALLIDA (error al revertir; requiere intervención manual)
```

---

## 4. Flujo principal

### 4.1 Inicio del pago externo

1. El **Cliente** inicia el pago externo desde el **Frontend** (monto, beneficiario, referencia).
2. El **Frontend** envía `POST /payments/external` al **API Gateway** con JWT y `idempotencyKey`.
3. El **API Gateway** valida el JWT y publica `ExternalPaymentRequested` en el **Broker**:
   - `paymentId` (UUID)
   - `customerId`, `sourceAccountId`
   - `amount`, `currency`
   - `externalReference` (referencia del beneficiario externo)
   - `idempotencyKey`
4. El **Payment Service** consume `ExternalPaymentRequested`:
   - Verifica idempotencia.
   - Crea el registro del pago con estado `PENDIENTE`.
5. El **API Gateway** devuelve `202 Accepted` al **Frontend**.

---

### 4.2 Débito preventivo (antes de contactar al sistema externo)

6. El **Payment Service** publica `DebitRequested` en el **Broker** con `{sourceAccountId, amount, paymentId}`.
7. Actualiza estado a `DEBITANDO`.
8. El **Account Service** consume `DebitRequested`:
   - Valida fondos disponibles y saldo mínimo en **Account DB**.
   - Si fondos insuficientes → publica `DebitFailed`; Payment Service registra `FALLIDO` y notifica.
   - Si OK → aplica débito: `balance(origen) -= amount`.
   - Persiste en **Account DB** y publica `SourceDebited`.
9. El **Payment Service** consume `SourceDebited` y actualiza estado a `PROCESANDO`.

---

### 4.3 Procesamiento en el sistema externo

10. El **Payment Service** envía la solicitud de pago al **Sistema de Pago Externo (simulado)** y aguarda respuesta dentro del tiempo límite configurado.

A partir de aquí, el flujo diverge en tres escenarios (fragmento alternativo):

---

## 5. Escenarios alternativos

### 5.1 Escenario: ÉXITO (`pago.completado`)

**Condición:** El sistema externo responde con confirmación dentro del tiempo límite.

11a. El **Sistema de Pago Externo** devuelve respuesta `APROBADO` → **Payment Service** recibe `pago.completado`.  
12a. **Payment Service** actualiza estado a `COMPLETADO`.  
13a. Publica `ExternalPaymentCompleted` en el **Broker** con `{paymentId, externalConfirmationId}`.  
14a. **Notification & Audit** consume el evento:
  - Registra auditoría (tipo: `INFO`) en **Notification DB**.
  - Envía notificación: "Pago externo de Q{amount} completado. Ref: {externalReference}."  
15a. **Frontend** recibe confirmación: estado `COMPLETADO` + ID de confirmación.

---

### 5.2 Escenario: FALLO (rechazo del sistema externo)

**Condición:** El sistema externo responde con rechazo explícito (fondos externos insuficientes, beneficiario inválido, etc.).

11b. El **Sistema de Pago Externo** devuelve `RECHAZADO` con código de error.  
12b. **Payment Service** determina que el débito preventivo ya fue aplicado → inicia compensación.  
13b. Publica `CompensationRequested` en el **Broker** con `{sourceAccountId, amount, paymentId}`.  
14b. **Account Service** consume `CompensationRequested`:
  - Revierte el débito: `balance(origen) += amount`.
  - Persiste en **Account DB** y publica `SourceRefunded`.  
15b. **Payment Service** consume `SourceRefunded`:
  - Actualiza estado a `COMPENSADO`.
  - Publica `ExternalPaymentFailed` (causa: `RECHAZADO_EXTERNO`) en el **Broker**.  
16b. **Notification & Audit** consume el evento:
  - Registra auditoría (tipo: `WARNING`) en **Notification DB**.
  - Envía notificación: "Pago externo rechazado. El monto ha sido devuelto a su cuenta."  
17b. **Frontend** recibe: estado `COMPENSADO` + motivo de rechazo.

---

### 5.3 Escenario: TIMEOUT (sin respuesta del sistema externo)

**Condición:** El sistema externo no responde dentro del tiempo límite configurado.

11c. El tiempo límite expira → **Payment Service** detecta timeout.  
12c. **Payment Service** intenta un **reintento temporal** (máx. N intentos configurables):
  - Re-envía la solicitud al **Sistema de Pago Externo**.  
13c. **Si el reintento tiene éxito** → flujo retoma el escenario ÉXITO (§5.1).  
14c. **Si el reintento agota los intentos** sin respuesta:
  - Si el débito fue aplicado → inicia compensación (igual que §5.2, pasos 13b–17b).
  - Publica `ExternalPaymentFailed` (causa: `TIMEOUT_AGOTADO`).  
15c. **Notification & Audit** consume el evento:
  - Registra auditoría (tipo: `ERROR`) en **Notification DB**.
  - Envía notificación: "Error en pago externo por timeout. El monto ha sido devuelto si ya fue debitado."  
16c. **Frontend** recibe: estado `COMPENSADO` o `FALLIDO` + causa `TIMEOUT_AGOTADO`.

---

## 6. Clasificación de eventos de auditoría

| Escenario                             | Estado final          | Tipo de evento  |
|---------------------------------------|-----------------------|-----------------|
| Pago completado exitosamente          | `COMPLETADO`          | `INFO`          |
| Rechazo por sistema externo           | `COMPENSADO`          | `WARNING`       |
| Timeout con compensación exitosa      | `COMPENSADO`          | `ERROR`         |
| Timeout / fallo sin compensación      | `FALLIDO`             | `ERROR`         |
| Compensación fallida                  | `COMPENSACION_FALLIDA`| `ERROR` crítico |

---

## 7. Eventos publicados

| Evento                        | Publicado por       | Consumido por             |
|-------------------------------|---------------------|---------------------------|
| `ExternalPaymentRequested`    | API Gateway         | Payment Service           |
| `DebitRequested`              | Payment Service     | Account Service           |
| `SourceDebited`               | Account Service     | Payment Service           |
| `DebitFailed`                 | Account Service     | Payment Service           |
| `CompensationRequested`       | Payment Service     | Account Service           |
| `SourceRefunded`              | Account Service     | Payment Service           |
| `CompensationFailed`          | Account Service     | Payment Service           |
| `ExternalPaymentCompleted`    | Payment Service     | Notification & Audit      |
| `ExternalPaymentFailed`       | Payment Service     | Notification & Audit      |

---

## 8. Notas de implementación

- **Débito preventivo:** El débito se aplica antes de contactar al sistema externo para garantizar que los fondos existen al momento de iniciar la transacción. Si el pago falla, se revierte mediante compensación.
- **Sistema externo simulado:** En entornos de desarrollo y pruebas, el sistema externo es simulado con respuestas configurables (éxito, rechazo, timeout) para cubrir todos los escenarios.
- **Reintentos:** El número máximo de reintentos y el tiempo de espera entre ellos son configurables por entorno (variables de entorno en GKE).
- **Idempotencia:** El `paymentId` garantiza que reintentos del cliente no generen pagos duplicados.
- **Compensación:** Si la compensación también falla, el estado queda en `COMPENSACION_FALLIDA` y se genera una alerta para intervención manual del equipo de operaciones.
- **Clasificación de eventos:** Todos los eventos se clasifican en INFO, WARNING o ERROR para facilitar el monitoreo y las alertas en el sistema de auditoría.
- **Payment Service vs Transaction Service:** El Payment Service es exclusivo del flujo de pago externo. Las transferencias internas usan Transaction Service y no pasan por Payment Service.
