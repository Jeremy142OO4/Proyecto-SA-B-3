resource "google_container_cluster" "bank_usac" {
  name     = var.gke_cluster_name
  location = var.zone

  network    = google_compute_network.bank_usac.id
  subnetwork = google_compute_subnetwork.database.id

  remove_default_node_pool = true
  initial_node_count       = 1
  deletion_protection      = false
  networking_mode          = "VPC_NATIVE"

  ip_allocation_policy {}

  release_channel {
    channel = "REGULAR"
  }

  depends_on = [google_project_service.container]
}

resource "google_container_node_pool" "bank_usac" {
  name       = "${var.gke_cluster_name}-nodes"
  location   = var.zone
  cluster    = google_container_cluster.bank_usac.name
  node_count = var.gke_node_count

  node_config {
    machine_type = var.gke_machine_type
    disk_type    = "pd-balanced"
    disk_size_gb = var.gke_disk_size_gb
    image_type   = "COS_CONTAINERD"

    oauth_scopes = [
      "https://www.googleapis.com/auth/cloud-platform",
    ]

    labels = {
      environment = var.environment
      managed_by  = "terraform"
      project     = "bank-usac"
    }

    tags = ["bank-usac-gke"]
  }

  management {
    auto_repair  = true
    auto_upgrade = true
  }

  depends_on = [google_project_service.container]
}
