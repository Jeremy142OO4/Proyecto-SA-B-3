output "database_instances" {
  description = "Nombres de las instancias Cloud SQL por microservicio."
  value = {
    for service, database in local.cloud_sql_databases : service => google_sql_database_instance.database[service].name
  }
}

output "database_private_or_public_ips" {
  description = "IPs publicas asignadas por Cloud SQL."
  value = {
    for service, instance in google_sql_database_instance.database : service => instance.public_ip_address
  }
}

output "database_urls" {
  description = "URLs de conexion por microservicio. Son valores sensibles."
  sensitive   = true

  value = {
    for service, database in local.cloud_sql_databases : service => format(
      "postgresql://%s:%s@%s:5432/%s?sslmode=require",
      google_sql_user.database[service].name,
      random_password.database[service].result,
      google_sql_database_instance.database[service].public_ip_address,
      google_sql_database.database[service].name,
    )
  }
}

output "database_subnetwork" {
  description = "Subred conservada para el futuro cluster Kubernetes."
  value       = google_compute_subnetwork.database.name
}

output "gke_cluster_name" {
  description = "Nombre del cluster GKE creado por Terraform."
  value       = google_container_cluster.bank_usac.name
}

output "gke_cluster_location" {
  description = "Zona del cluster GKE creado por Terraform."
  value       = google_container_cluster.bank_usac.location
}

output "gke_get_credentials_command" {
  description = "Comando para configurar kubectl contra el cluster GKE."
  value       = "gcloud container clusters get-credentials ${google_container_cluster.bank_usac.name} --zone ${google_container_cluster.bank_usac.location} --project ${var.project_id}"
}

output "frontend_public_ip" {
  description = "IP publica regional reservada para el LoadBalancer del frontend."
  value       = google_compute_address.frontend.address
}
