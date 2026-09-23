# Horizontal Pod Autoscaler (HPA) — Bank USAC (Fase 2)

## Propósito

Este documento describe la configuración del Horizontal Pod Autoscaler (HPA) para cada microservicio de Bank USAC desplegado en Google Kubernetes Engine (GKE). El HPA garantiza que el sistema escale automáticamente ante incrementos de carga, manteniendo disponibilidad sin intervención manual.

---

## Principio de funcionamiento

El HPA monitorea el uso de CPU de los pods de cada Deployment. Cuando el uso promedio supera el **80 %** del límite configurado, Kubernetes crea réplicas adicionales del pod hasta el máximo permitido. Cuando la carga disminuye, reduce las réplicas de vuelta al mínimo.

```
Uso CPU > 80% ──→ HPA escala hacia arriba (más pods)
Uso CPU < 80% ──→ HPA escala hacia abajo (menos pods, tras cooldown)
```

El ciclo de evaluación del HPA es cada **15 segundos** (intervalo por defecto del metrics-server de Kubernetes).

---

## Metrics Server

El HPA requiere que `metrics-server` esté instalado en el clúster. En GKE esta extensión está habilitada por defecto. Se puede verificar con:

```bash
kubectl get deployment metrics-server -n kube-system
```

---

## Configuración por microservicio

### Recursos compartidos (Request / Limit por pod)

Todos los microservicios Go usan la misma base de recursos. El HPA calcula el porcentaje de CPU sobre el valor de **Request** (no del Limit).

| Recurso | Request | Limit |
|---|---|---|
| CPU | `100m` (0.1 vCPU) | `500m` (0.5 vCPU) |
| Memoria | `128Mi` | `256Mi` |

---

### API Gateway

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: hpa-api-gateway
  namespace: bank-usac-prod
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: api-gateway
  minReplicas: 2
  maxReplicas: 8
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 80
```

| Parámetro | Valor |
|---|---|
| `minReplicas` | 2 |
| `maxReplicas` | 8 |
| Umbral CPU | 80 % |
| Justificación | Punto de entrada único; alta concurrencia esperada |

---

### Customer Service

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: hpa-customer-service
  namespace: bank-usac-prod
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: customer-service
  minReplicas: 2
  maxReplicas: 6
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 80
```

| Parámetro | Valor |
|---|---|
| `minReplicas` | 2 |
| `maxReplicas` | 6 |
| Umbral CPU | 80 % |
| Justificación | Gestiona autenticación, JWT y validación KYC (Fase 2) |

---

### Account Service

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: hpa-account-service
  namespace: bank-usac-prod
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: account-service
  minReplicas: 2
  maxReplicas: 6
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 80
```

| Parámetro | Valor |
|---|---|
| `minReplicas` | 2 |
| `maxReplicas` | 6 |
| Umbral CPU | 80 % |
| Justificación | Operaciones de saldo, tipos de cuenta y comisiones (Fase 2) |

---

### Transaction Service

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: hpa-transaction-service
  namespace: bank-usac-prod
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: transaction-service
  minReplicas: 2
  maxReplicas: 8
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 80
```

| Parámetro | Valor |
|---|---|
| `minReplicas` | 2 |
| `maxReplicas` | 8 |
| Umbral CPU | 80 % |
| Justificación | Coordina Saga de transferencias; operación más intensiva en CPU |

---

### Payment Service

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: hpa-payment-service
  namespace: bank-usac-prod
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: payment-service
  minReplicas: 2
  maxReplicas: 6
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 80
```

| Parámetro | Valor |
|---|---|
| `minReplicas` | 2 |
| `maxReplicas` | 6 |
| Umbral CPU | 80 % |
| Justificación | Simula pagos externos con posibles timeouts; requiere resiliencia |

---

### Notification & Audit Service

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: hpa-notification-audit
  namespace: bank-usac-prod
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: notification-audit-service
  minReplicas: 2
  maxReplicas: 5
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 80
```

| Parámetro | Valor |
|---|---|
| `minReplicas` | 2 |
| `maxReplicas` | 5 |
| Umbral CPU | 80 % |
| Justificación | Consumidor pasivo de eventos; menor concurrencia directa |

---

## Resumen consolidado

| Microservicio | minReplicas | maxReplicas | CPU Umbral |
|---|---|---|---|
| API Gateway | 2 | 8 | 80 % |
| Customer Service | 2 | 6 | 80 % |
| Account Service | 2 | 6 | 80 % |
| Transaction Service | 2 | 8 | 80 % |
| Payment Service | 2 | 6 | 80 % |
| Notification & Audit | 2 | 5 | 80 % |

**Capacidad máxima total:** 39 pods distribuidos en el node pool GKE (2–6 nodos `e2-standard-2`).

---

## Entorno de desarrollo (`bank-usac-dev`)

En el namespace de desarrollo se usan valores reducidos para minimizar costos:

| Microservicio | minReplicas (dev) | maxReplicas (dev) |
|---|---|---|
| API Gateway | 1 | 3 |
| Customer Service | 1 | 2 |
| Account Service | 1 | 2 |
| Transaction Service | 1 | 3 |
| Payment Service | 1 | 2 |
| Notification & Audit | 1 | 2 |

---

## Comandos de operación

```bash
# Ver estado actual del HPA (producción)
kubectl get hpa -n bank-usac-prod

# Ver detalles de un HPA específico
kubectl describe hpa hpa-transaction-service -n bank-usac-prod

# Forzar escala manual (solo para pruebas, no en producción)
kubectl scale deployment transaction-service --replicas=4 -n bank-usac-dev

# Ver métricas de CPU de los pods en tiempo real
kubectl top pods -n bank-usac-prod

# Ver eventos de escalado
kubectl get events -n bank-usac-prod --field-selector reason=SuccessfulRescale
```

---

## Consideraciones importantes

### Cooldown de escala hacia abajo

Por defecto, Kubernetes espera **5 minutos** antes de reducir réplicas, evitando oscilaciones (flapping). Este valor se puede ajustar con `behavior.scaleDown.stabilizationWindowSeconds`.

### Límite de nodos (Cluster Autoscaler)

Si todos los pods al máximo superan la capacidad de los nodos actuales, el Cluster Autoscaler de GKE agrega nodos al node pool (máximo 6 configurado en Terraform). La combinación HPA + Cluster Autoscaler garantiza escalado en dos niveles.

### Idempotencia en réplicas múltiples

Todos los microservicios implementan idempotencia mediante `correlationId` y claves de idempotencia almacenadas en PostgreSQL. Tener múltiples réplicas procesando mensajes de RabbitMQ es seguro porque cada cola tiene un único consumidor activo por binding (configuración `prefetch_count = 1`).

---