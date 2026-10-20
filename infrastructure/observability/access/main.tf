terraform {
  required_version = ">= 1.6.0"
  backend "local" {}
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

variable "project_id" {
  type    = string
  default = "bank-usac"
}

variable "region" {
  type    = string
  default = "us-central1"
}

data "google_compute_network" "bank" {
  name = "bank-usac-vpc"
}

data "google_compute_subnetwork" "bank" {
  name   = "bank-usac-database-subnet"
  region = var.region
}

resource "google_compute_address" "grafana" {
  name         = "bank-usac-grafana-internal"
  region       = var.region
  address_type = "INTERNAL"
  subnetwork   = data.google_compute_subnetwork.bank.id
}

resource "google_service_account" "proxy" {
  account_id   = "bank-grafana-proxy"
  display_name = "Grafana HTTPS proxy - no application permissions"
}

resource "google_cloud_run_v2_service" "grafana" {
  name                = "bank-usac-grafana"
  location            = var.region
  deletion_protection = false
  ingress             = "INGRESS_TRAFFIC_ALL"

  template {
    service_account                  = google_service_account.proxy.email
    timeout                          = "300s"
    max_instance_request_concurrency = 40
    scaling {
      min_instance_count = 0
      max_instance_count = 2
    }
    vpc_access {
      egress = "PRIVATE_RANGES_ONLY"
      network_interfaces {
        network    = data.google_compute_network.bank.name
        subnetwork = data.google_compute_subnetwork.bank.name
        tags       = ["bank-grafana-proxy"]
      }
    }
    containers {
      image   = "docker.io/library/nginx:stable-alpine"
      command = ["/bin/sh", "-c"]
      args    = ["printf '%s' \"$NGINX_CONFIGURATION\" > /etc/nginx/conf.d/default.conf && exec nginx -g 'daemon off;'"]
      ports {
        container_port = 8080
      }
      resources {
        limits = {
          cpu    = "1"
          memory = "128Mi"
        }
        cpu_idle          = true
        startup_cpu_boost = false
      }
      env {
        name = "NGINX_CONFIGURATION"
        value = templatefile("${path.module}/nginx.conf.tftpl", {
          grafana_ip = google_compute_address.grafana.address
        })
      }
      startup_probe {
        http_get {
          path = "/api/health"
          port = 8080
        }
        timeout_seconds   = 5
        period_seconds    = 10
        failure_threshold = 24
      }
    }
  }
}

# Transport is public HTTPS; authentication remains enforced by Grafana.
resource "google_cloud_run_v2_service_iam_member" "https" {
  project  = var.project_id
  location = var.region
  name     = google_cloud_run_v2_service.grafana.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}

output "grafana_url" {
  value = google_cloud_run_v2_service.grafana.uri
}

output "grafana_internal_ip" {
  value = google_compute_address.grafana.address
}

output "source_cidr" {
  value = data.google_compute_subnetwork.bank.ip_cidr_range
}
