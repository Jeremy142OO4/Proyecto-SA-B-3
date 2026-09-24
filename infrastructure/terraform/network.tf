resource "google_compute_network" "bank_usac" {
  name                    = "bank-usac-vpc"
  auto_create_subnetworks = false

  depends_on = [google_project_service.compute]
}

resource "google_compute_subnetwork" "database" {
  name          = "bank-usac-database-subnet"
  ip_cidr_range = "10.20.0.0/24"
  region        = var.region
  network       = google_compute_network.bank_usac.id

  private_ip_google_access = true
}

# IP publica regional reservada para el LoadBalancer del frontend de GKE.
# Se mantiene aunque se recreen los pods o los nodos del cluster.
resource "google_compute_address" "frontend" {
  name         = "bank-usac-frontend-public-ip"
  region       = var.region
  address_type = "EXTERNAL"
  network_tier = "PREMIUM"

  depends_on = [google_project_service.compute]
}

resource "google_compute_firewall" "frontend_nodeport" {
  name    = "bank-usac-frontend-nodeport"
  network = google_compute_network.bank_usac.name

  direction     = "INGRESS"
  source_ranges = ["0.0.0.0/0"]
  target_tags   = ["bank-usac-gke"]

  allow {
    protocol = "tcp"
    ports    = ["30080", "30081"]
  }
}
