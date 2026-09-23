# Infraestructura como Código — Terraform (Fase 2)

## Propósito

Este documento describe el diseño y la estructura de los módulos Terraform utilizados para provisionar toda la infraestructura de Bank USAC en Google Cloud Platform (GCP). La infraestructura es completamente reproducible, versionada y automatizable: ningún recurso de red, cómputo ni clúster se crea manualmente.

---

## Proveedor y versiones

| Elemento | Valor |
|---|---|
| Proveedor | `hashicorp/google` (Google Cloud) |
| Versión Terraform | `>= 1.7.0` |
| Versión proveedor GCP | `~> 5.0` |
| Región principal | `us-central1` |
| Proyecto GCP | `bank-usac-prod` (configurable por workspace) |

---

## Estructura de directorios

```
infra/
├── terraform/
│   ├── main.tf                  # Raíz: invoca módulos
│   ├── variables.tf             # Variables globales
│   ├── outputs.tf               # Salidas exportadas
│   ├── terraform.tfvars         # Valores por entorno (no comitear si tiene secretos)
│   ├── backend.tf               # Estado remoto en GCS
│   └── modules/
│       ├── network/             # VPC, subnets, firewall
│       ├── gke/                 # Clúster GKE + node pools
│       ├── vm-database/         # VMs PostgreSQL por servicio
│       └── artifact-registry/   # Registro de imágenes Docker
```

---

## Backend de estado remoto

El estado de Terraform se almacena en un bucket de Google Cloud Storage (GCS) con versionado habilitado. Esto permite trabajo colaborativo y recuperación ante errores.

```hcl
# backend.tf
terraform {
  backend "gcs" {
    bucket = "bank-usac-terraform-state"
    prefix = "terraform/state"
  }
}
```

---

## Módulo `network` — Red VPC

Provisiona la red virtual privada donde residen todos los recursos del proyecto.

### Recursos creados

| Recurso | Nombre | Descripción |
|---|---|---|
| `google_compute_network` | `bank-usac-vpc` | VPC principal con enrutamiento personalizado |
| `google_compute_subnetwork` | `bank-usac-subnet-gke` | Subred `/20` para nodos GKE (`10.10.0.0/20`) |
| `google_compute_subnetwork` | `bank-usac-subnet-db` | Subred `/24` para VMs de base de datos (`10.10.16.0/24`) |
| `google_compute_firewall` | `allow-internal` | Tráfico interno entre subredes del proyecto |
| `google_compute_firewall` | `allow-ssh-iap` | SSH vía IAP (Identity-Aware Proxy) — sin IP pública en VMs |
| `google_compute_firewall` | `allow-postgres-gke` | Puerto 5432 desde subred GKE hacia subred DB |

### Variables

| Variable | Tipo | Valor por defecto |
|---|---|---|
| `region` | `string` | `"us-central1"` |
| `vpc_name` | `string` | `"bank-usac-vpc"` |
| `gke_subnet_cidr` | `string` | `"10.10.0.0/20"` |
| `db_subnet_cidr` | `string` | `"10.10.16.0/24"` |

---

## Módulo `gke` — Clúster Kubernetes

Provisiona el clúster GKE (zonal en `us-central1-a`) donde se ejecutan los microservicios.

### Recursos creados

| Recurso | Nombre | Descripción |
|---|---|---|
| `google_container_cluster` | `bank-usac-cluster` | Clúster GKE con control plane administrado |
| `google_container_node_pool` | `bank-usac-nodes` | Node pool principal con autoescalado |

### Configuración del clúster

```hcl
resource "google_container_cluster" "main" {
  name     = "bank-usac-cluster"
  location = var.zone  # "us-central1-a"

  remove_default_node_pool = true
  initial_node_count       = 1

  network    = module.network.vpc_name
  subnetwork = module.network.gke_subnet_name

  workload_identity_config {
    workload_pool = "${var.project_id}.svc.id.goog"
  }
}

resource "google_container_node_pool" "main" {
  name       = "bank-usac-nodes"
  cluster    = google_container_cluster.main.name
  location   = var.zone

  autoscaling {
    min_node_count = 2
    max_node_count = 6
  }

  node_config {
    machine_type = "e2-standard-2"   # 2 vCPU, 8 GB RAM
    disk_size_gb = 50
    disk_type    = "pd-standard"
    oauth_scopes = ["https://www.googleapis.com/auth/cloud-platform"]
  }
}
```

### Variables

| Variable | Tipo | Valor por defecto |
|---|---|---|
| `gke_machine_type` | `string` | `"e2-standard-2"` |
| `gke_min_nodes` | `number` | `2` |
| `gke_max_nodes` | `number` | `6` |
| `gke_disk_gb` | `number` | `50` |

---

## Módulo `vm-database` — VMs PostgreSQL

Cada microservicio tiene su propia base de datos PostgreSQL ejecutada en una VM de Compute Engine independiente. Esto cumple el requisito del proyecto: **bases de datos fuera de Kubernetes, en máquinas virtuales**.

### Bases de datos y VMs provisionadas

| VM | Base de datos | Puerto | Servicio propietario |
|---|---|---|---|
| `vm-db-customer` | `customer_db` | 5432 | Customer Service |
| `vm-db-account` | `cuentas_db` | 5432 | Account Service |
| `vm-db-transaction` | `transacciones_db` | 5432 | Transaction Service |
| `vm-db-payment` | `pagos_db` | 5432 | Payment Service |
| `vm-db-audit` | `audit_db` | 5432 | Notification & Audit Service |

### Configuración de cada VM

```hcl
resource "google_compute_instance" "db" {
  for_each     = var.databases
  name         = "vm-db-${each.key}"
  machine_type = "e2-small"      # 2 vCPU, 2 GB RAM
  zone         = var.zone

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-12"
      size  = 30
      type  = "pd-ssd"
    }
  }

  network_interface {
    subnetwork = module.network.db_subnet_name
    # Sin IP pública — acceso solo desde VPC interna
  }

  metadata_startup_script = templatefile("${path.module}/scripts/install-postgres.sh", {
    db_name     = each.value.db_name
    db_user     = each.value.db_user
    db_password = each.value.db_password
  })

  tags = ["db-server"]
}
```

### Script de arranque (`install-postgres.sh`)

El script de inicio instala PostgreSQL 17, crea la base de datos y el usuario, configura `pg_hba.conf` para aceptar conexiones desde la subred GKE y habilita el inicio automático del servicio.

### Variables del módulo

| Variable | Tipo | Descripción |
|---|---|---|
| `databases` | `map(object)` | Mapa de nombre → {db_name, db_user, db_password} |
| `db_machine_type` | `string` | Tipo de VM (default: `"e2-small"`) |
| `db_disk_gb` | `number` | Tamaño del disco (default: `30`) |

### Nota de seguridad

Las contraseñas de las bases de datos se pasan como variables sensibles (`sensitive = true`). En producción se recomienda usar Google Secret Manager e inyectarlas en el startup script en tiempo de apply.

---

## Módulo `artifact-registry` — Registro de imágenes

```hcl
resource "google_artifact_registry_repository" "bank_usac" {
  location      = var.region
  repository_id = "bank-usac"
  format        = "DOCKER"
  description   = "Imágenes Docker de microservicios Bank USAC"
}
```

---

## Workspaces de Terraform (entornos)

Se usan Terraform Workspaces para separar los entornos `dev` y `prod` con el mismo código base:

| Workspace | Proyecto GCP | Clúster GKE | Nº nodos mín/máx |
|---|---|---|---|
| `dev` | `bank-usac-dev` | `bank-usac-cluster-dev` | 1 / 3 |
| `prod` | `bank-usac-prod` | `bank-usac-cluster` | 2 / 6 |

```bash
# Cambiar de entorno
terraform workspace select prod

# Aplicar infraestructura
terraform apply -var-file="environments/prod.tfvars"
```

---

## Salidas exportadas (`outputs.tf`)

| Output | Descripción |
|---|---|
| `gke_cluster_name` | Nombre del clúster GKE |
| `gke_endpoint` | Endpoint del API server de Kubernetes |
| `db_internal_ips` | Mapa de nombre → IP interna de cada VM de base de datos |
| `artifact_registry_url` | URL del registry Docker |

---

## Comandos de operación

```bash
# Inicializar (primera vez o después de cambiar módulos)
terraform init

# Ver plan de cambios sin aplicar
terraform plan -var-file="environments/prod.tfvars"

# Aplicar infraestructura
terraform apply -var-file="environments/prod.tfvars"

# Destruir infraestructura (solo en dev/testing)
terraform destroy -var-file="environments/dev.tfvars"

# Ver estado actual
terraform show

# Importar recurso existente
terraform import google_compute_instance.db["customer"] projects/bank-usac-prod/zones/us-central1-a/instances/vm-db-customer
```

---