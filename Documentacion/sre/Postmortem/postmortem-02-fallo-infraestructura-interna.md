# Postmortem Blameless — Terminación abrupta de pod y caída del broker durante una transferencia

| Campo | Valor |
|---|---|
| **ID del incidente** | PM-F3-03 |
| **Severidad** | SEV-2 (pérdida temporal de procesamiento asíncrono) |
| **Servicios involucrados** | Transaction Service, Account Service, Payment Service, Notification Service, RabbitMQ |
| **Componente fallido** | Cómputo (pod de GKE) y broker de mensajes |
| **Autores** | Equipo BANK03 |
| **Estado** |  Mitigado |

---

## 1. Resumen ejecutivo

Se inyectó de forma controlada la terminación abrupta del pod de `transaction-service` (y, en una segunda variante, la indisponibilidad del broker RabbitMQ) en medio de una transferencia activa, con el fin de verificar la resiliencia del flujo asíncrono basado en el patrón Outbox.

El objetivo fue comprobar que ninguna transacción se pierde, que no se producen débitos duplicados por reprocesamiento y que el sistema converge a un estado consistente al recuperarse la infraestructura.



---

## 2. Hipótesis a validar

| # | Hipótesis | Resultado esperado |
|---|---|---|
| H1 | El evento persistido en Outbox sobrevive a la muerte del pod | El mensaje se publica al reiniciar | 
| H2 | El consumidor es idempotente | Un solo débito por transferencia | 
| H3 | Kubernetes reprograma el pod automáticamente | Pod en `Running` con réplicas completas | 
| H4 | La caída de RabbitMQ no pierde mensajes ya confirmados | Cola drenada tras la recuperación |
| H5 | El estado de la transferencia converge a terminal | `COMPLETED` o `FAILED`, nunca `PROCESSING` perpetuo |
| H6 | La alerta de disponibilidad se dispara | Regla de `up` o de tasa de error activa | 
---



## 3. Cronología de los hechos (UTC)

 Evento | Fuente de evidencia |
|---|---|
| Inicio de la transferencia de prueba | `curl` / frontend |
| Inyección del fallo (pod eliminado o broker escalado a cero) | Comando ejecutado |
| Detección de la caída por Kubernetes | `kubectl get events` |
| Disparo de la alerta en Alertmanager | Captura de Alertmanager |
| Reprogramación del pod / recuperación del broker | `kubectl get pods` |
| Reprocesamiento del mensaje desde Outbox | Log de transaction-service |
| Estado final de la transferencia | Base de datos |
| Verificación de ausencia de duplicados | Consulta SQL |

---


## 4. Acciones correctivas

| # | Acción | Tipo | Responsable | Estado |
|---|---|---|---|---|
| 1 | Configurar `PodDisruptionBudget` para los servicios críticos | Mitigación | SRE | [COMPLETAR] |
| 2 | Añadir antiafinidad para distribuir réplicas entre nodos | Prevención | SRE | [COMPLETAR] |
| 3 | Verificar que las sondas `liveness`/`readiness` reflejen el estado real | Prevención | Backend | [COMPLETAR] |
| 4 | Confirmar `persistent` y `durable` en colas y mensajes de RabbitMQ | Prevención | Backend | [COMPLETAR] |
| 5 | Implementar clave de idempotencia persistida en base de datos | Prevención | Backend | [COMPLETAR] |
| 6 | Alerta sobre reinicios de pod y sobre profundidad de cola | Detección | SRE | [COMPLETAR] |
| 7 | Documentar el procedimiento de recuperación en el runbook | Mitigación | SRE | [COMPLETAR] |

---


