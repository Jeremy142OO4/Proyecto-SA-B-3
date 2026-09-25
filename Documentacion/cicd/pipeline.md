# Pipeline CI/CD — Bank USAC (Fase 2)

## Propósito

Este documento describe el diseño, las etapas y las reglas del pipeline de Integración Continua y Entrega/Despliegue Continuo (CI/CD) implementado para Bank USAC en la Fase 2. El pipeline automatiza el ciclo completo desde la escritura del código hasta el despliegue en producción en Google Kubernetes Engine (GKE), garantizando calidad, trazabilidad y alta disponibilidad.

---

## Herramienta y repositorio

| Elemento | Valor |
|---|---|
| Herramienta CI/CD | GitHub Actions |
| Repositorio | Monorepo con subdirectorios por microservicio |
| Registry de imágenes | Google Artifact Registry (GAR) |
| Orquestador | GKE (Google Kubernetes Engine) |
| Namespaces | `bank-usac-dev` (desarrollo), `bank-usac-prod` (producción) |

---

## Estrategia de ramas (Git Flow)

```
main ──────────────────────────────────────────── producción (CD)
  └── release/vX.X.X ──────────────────────────── entrega / versionado
        └── develop ──────────────────────────── integración (CI + deploy dev)
              └── feature/<nombre-func> ──────── desarrollo individual (CI)
```

### Reglas por rama

| Rama | Acción que dispara el pipeline | Resultado esperado |
|---|---|---|
| `feature/*` | Push | CI: Build + Test + Validate |
| `develop` | Merge de feature (PR aprobado) | CI completo + Build Docker + Deploy a `bank-usac-dev` |
| `release/vX.X.X` o tag `vX.X.X` | Push / tag | Versionado + Build imagen final + Push a GAR |
| `main` | Merge de release (PR aprobado) | CD: Rolling update en `bank-usac-prod` |

---

## Etapas del pipeline

### Fase 1 — Integración Continua (rama `feature/*` y `develop`)

```
┌──────────┐    ┌──────────┐    ┌────────────┐
│  Build   │ →  │   Test   │ →  │  Validate  │
└──────────┘    └──────────┘    └────────────┘
```

#### 1.1 Build

- Compila cada microservicio Go con `go build ./...`.
- Falla inmediatamente si hay error de compilación.
- Se ejecuta en paralelo para todos los microservicios modificados (detección por paths).

```yaml
# Ejemplo — .github/workflows/ci.yml
- name: Build
  run: |
    go build ./...
  working-directory: services/${{ matrix.service }}
```

#### 1.2 Test

- Ejecuta pruebas unitarias: `go test ./... -v -cover`.
- Requiere cobertura mínima del 70 % (configurable por servicio).
- Genera reporte de cobertura como artefacto del workflow.

```yaml
- name: Test
  run: |
    go test ./... -v -coverprofile=coverage.out
    go tool cover -html=coverage.out -o coverage.html
  working-directory: services/${{ matrix.service }}
```

#### 1.3 Validate (Sanity Check)

- Construye la imagen Docker del servicio en modo `--no-push` para verificar que el Dockerfile es válido.
- Ejecuta `docker run --rm <imagen> --version` o endpoint `/health` si aplica.
- Verifica que el binario arranca sin errores de configuración.

---

### Fase 2 — Build y despliegue en `develop`

Después de un merge a `develop`, el pipeline ejecuta adicionalmente:

```
┌─────────────────┐    ┌───────────────┐    ┌──────────────────────┐
│ Docker Build    │ →  │  Push a GAR   │ →  │ kubectl apply (dev)  │
│ (imagen :dev)   │    │ :dev-<sha>    │    │ namespace: dev       │
└─────────────────┘    └───────────────┘    └──────────────────────┘
```

#### 2.1 Docker Build

- Imagen etiquetada con SHA corto del commit: `bank-usac/<servicio>:dev-<SHA7>`.
- Se usa Dockerfile multi-etapa (builder + imagen mínima distroless).
- Prohíbe la etiqueta `latest`.

```yaml
- name: Docker Build & Push
  uses: docker/build-push-action@v5
  with:
    context: services/${{ matrix.service }}
    push: true
    tags: |
      ${{ env.REGISTRY }}/${{ matrix.service }}:dev-${{ github.sha | cut -c1-7 }}
```

#### 2.2 Despliegue en desarrollo

- Actualiza el Deployment de Kubernetes en el namespace `bank-usac-dev` usando `kubectl set image`.
- Espera rollout completo: `kubectl rollout status deployment/<servicio> -n bank-usac-dev`.
- Antes del despliegue se guardan las revisiones estables de todos los Deployments y de Cloud Run.
- Si el rollout falla, se ejecuta automáticamente `kubectl rollout undo --to-revision` para todos los Deployments y se restaura el tráfico de Cloud Run a la revisión anterior.

---

### Fase 3 — Release / Versionado (rama `release/*` o tag `vX.X.X`)

```
┌────────────────────┐    ┌──────────────────┐    ┌────────────────┐
│ Generar versión    │ →  │ Docker Build      │ →  │  Push a GAR   │
│ (CHANGELOG, tag)  │    │ (imagen :vX.X.X)  │    │  :vX.X.X      │
└────────────────────┘    └──────────────────┘    └────────────────┘
```

#### 3.1 Generación de versión

- Crea o valida el tag semántico `vX.X.X` (Semantic Versioning).
- Genera o actualiza `CHANGELOG.md` con los commits incluidos.
- El número de versión se inyecta como variable de entorno `APP_VERSION`.

#### 3.2 Build de imagen de producción

- Imagen etiquetada `bank-usac/<servicio>:vX.X.X`.
- Ejecuta escaneo de vulnerabilidades con Trivy antes de publicar.
- Si Trivy reporta vulnerabilidades CRÍTICAS, el pipeline falla.

```yaml
- name: Trivy scan
  uses: aquasecurity/trivy-action@master
  with:
    image-ref: ${{ env.REGISTRY }}/${{ matrix.service }}:${{ env.APP_VERSION }}
    exit-code: '1'
    severity: CRITICAL
```

---

### Fase 4 — Despliegue Continuo en producción (merge a `main`)

```
┌──────────────────────────────────────────────────────────────────┐
│  kubectl set image ... :<version>  →  Rolling Update  →  Verify │
└──────────────────────────────────────────────────────────────────┘
```

#### 4.1 Rolling Update

- Usa la imagen versionada producida en la Fase 3.
- Kubernetes aplica el rolling update con la estrategia:
  - `maxSurge: 1` — un pod adicional al máximo configurado.
  - `maxUnavailable: 0` — sin pods no disponibles durante la actualización.
- Garantiza cero tiempo de inactividad.

```yaml
strategy:
  type: RollingUpdate
  rollingUpdate:
    maxSurge: 1
    maxUnavailable: 0
```

#### 4.2 Verificación post-despliegue

- `kubectl rollout status` espera hasta 5 minutos.
- Prueba de humo: solicitud HTTP al endpoint `/health` del API Gateway.
- Si falla cualquier rollout o la verificación de exposición, revierte todos los Deployments a sus revisiones anteriores y devuelve el tráfico de Cloud Run a la revisión estable previa. El workflow conserva el estado fallido para detener la promoción.

---

## Restricciones obligatorias del pipeline

| Restricción | Descripción |
|---|---|
| Sin etiqueta `latest` | Todas las imágenes deben llevar versión o SHA |
| Sin despliegue manual en producción | Solo el pipeline puede aplicar cambios en `bank-usac-prod` |
| Sin omisión de etapas | Cada fase debe completarse antes de continuar |
| Fallo = detención | Cualquier error en Build, Test o Validate detiene el pipeline |
| Separación de entornos | Dev y Prod son namespaces independientes en GKE |

---

## Variables de entorno y secretos

Los secretos se almacenan en GitHub Secrets y se inyectan en tiempo de ejecución. Nunca se hardcodean en el código o en los YAML de workflow.

| Secreto / Variable | Uso |
|---|---|
| `GCP_SA_KEY` | Autenticación con GCP (JSON de Service Account) |
| `GAR_REGISTRY` | URL del Artifact Registry (`us-central1-docker.pkg.dev/...`) |
| `GKE_CLUSTER` | Nombre del clúster GKE |
| `GKE_ZONE` | Zona del clúster (ej. `us-central1-a`) |
| `KUBECONFIG` | Generado dinámicamente por `gke-gcloud-auth-plugin` |

---

## Matriz de microservicios

El pipeline usa una estrategia de matrix para ejecutar jobs en paralelo por servicio:

| Servicio | Directorio | Puerto interno |
|---|---|---|
| API Gateway | `services/api-gateway` | 8080 |
| Customer Service | `services/service-customer` | 8081 |
| Account Service | `services/account-service` | 8082 |
| Transaction Service | `services/transaction-service` | 8083 |
| Payment Service | `services/payment-service` | 8084 |
| Notification & Audit | `services/service-notification-audit` | 8085 |

---

## Diagrama de flujo del pipeline

![Diagrama de flujo pipeline](../Imagenes/Diagrama%20de%20flujo%20del%20Pipeline%20CI_CD%20—%20Bank%20USAC.drawio.png)
