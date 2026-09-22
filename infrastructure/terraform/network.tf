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
