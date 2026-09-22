variable "project_id" {
  description = "ID del proyecto de Google Cloud."
  type        = string
  default     = "bank-usac"
}

variable "region" {
  description = "Region de GCP donde se creara la VM."
  type        = string
  default     = "us-central1"
}

variable "zone" {
  description = "Zona de GCP donde se creara la VM."
  type        = string
  default     = "us-central1-a"
}

variable "environment" {
  description = "Entorno de despliegue."
  type        = string
  default     = "dev"

  validation {
    condition     = contains(["dev", "prod"], var.environment)
    error_message = "environment debe ser dev o prod."
  }
}

variable "cloud_sql_tier" {
  description = "Tier de cada instancia Cloud SQL."
  type        = string
  default     = "db-f1-micro"
}

variable "cloud_sql_storage_gb" {
  description = "Almacenamiento inicial de cada instancia Cloud SQL en GB."
  type        = number
  default     = 10
}

variable "cloud_sql_authorized_networks" {
  description = "CIDR autorizados para acceder a Cloud SQL. Restringir al rango de GKE cuando este definido."
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

variable "gke_cluster_name" {
  description = "Nombre del cluster GKE administrado por Terraform."
  type        = string
  default     = "bank-usac-gke"
}

variable "gke_node_count" {
  description = "Cantidad inicial de nodos del pool principal de GKE."
  type        = number
  default     = 2
}

variable "gke_max_node_count" {
  description = "Cantidad maxima de nodos del pool principal cuando el cluster necesita capacidad adicional."
  type        = number
  default     = 4
}

variable "gke_machine_type" {
  description = "Tipo de maquina para los nodos de GKE."
  type        = string
  default     = "e2-medium"
}

variable "gke_disk_size_gb" {
  description = "Tamano del disco de cada nodo de GKE en GB."
  type        = number
  default     = 30
}
