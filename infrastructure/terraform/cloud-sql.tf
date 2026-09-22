locals {
  cloud_sql_databases = {
    customer = {
      instance_name = "bank-usac-customer-db"
      database_name = "customer_db"
      user_name     = "customer_user"
    }
    account = {
      instance_name = "bank-usac-account-db"
      database_name = "cuentas_db"
      user_name     = "cuentas_usuario"
    }
    transaction = {
      instance_name = "bank-usac-transaction-db"
      database_name = "transacciones_db"
      user_name     = "transacciones_usuario"
    }
    payment = {
      instance_name = "bank-usac-payment-db"
      database_name = "pagos_db"
      user_name     = "pagos_usuario"
    }
    notification_audit = {
      instance_name = "bank-usac-notification-audit-db"
      database_name = "auditoria_db"
      user_name     = "audit_user"
    }
  }
}

resource "random_password" "database" {
  for_each = local.cloud_sql_databases

  length           = 32
  special          = false
  override_special = ""
}

resource "google_sql_database_instance" "database" {
  for_each = local.cloud_sql_databases

  name             = each.value.instance_name
  database_version = "POSTGRES_16"
  region           = var.region

  settings {
    tier              = var.cloud_sql_tier
    edition           = "ENTERPRISE"
    availability_type = "ZONAL"
    disk_type         = "PD_SSD"
    disk_size         = var.cloud_sql_storage_gb
    disk_autoresize   = true

    backup_configuration {
      enabled                        = true
      point_in_time_recovery_enabled = true
    }

    ip_configuration {
      ipv4_enabled = true

      dynamic "authorized_networks" {
        for_each = var.cloud_sql_authorized_networks

        content {
          name  = "cidr-${authorized_networks.key}"
          value = authorized_networks.value
        }
      }
    }
  }

  deletion_protection = false

  depends_on = [google_project_service.sqladmin]
}

resource "google_sql_database" "database" {
  for_each = local.cloud_sql_databases

  name     = each.value.database_name
  instance = google_sql_database_instance.database[each.key].name
}

resource "google_sql_user" "database" {
  for_each = local.cloud_sql_databases

  name     = each.value.user_name
  instance = google_sql_database_instance.database[each.key].name
  password = random_password.database[each.key].result
}
