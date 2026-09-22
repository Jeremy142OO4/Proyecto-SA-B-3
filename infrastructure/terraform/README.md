# Terraform - bases de datos de Bank USAC

Este modulo crea una instancia independiente de Cloud SQL para cada microservicio:

- Customer Service: `customer_db`.
- Account Service: `cuentas_db`.
- Transaction Service: `transacciones_db`.
- Payment Service: `pagos_db`.
- Notification & Audit Service: `auditoria_db`.

La VPC y la subred se conservan para que el cluster Kubernetes pueda compartir la
red de GCP. Las bases ya no se ejecutan en una VM ni dentro de Kubernetes.

## Requisitos

- Google Cloud CLI autenticado.
- Terraform >= 1.6.
- Permisos para Compute Engine, Cloud SQL, redes, IAM y Service Usage.

## Uso

Desde la raiz del repositorio:

```bash
gcloud auth application-default login
gcloud config set project bank-usac
cd infrastructure/terraform
terraform init
terraform fmt -recursive
terraform validate
terraform plan
terraform apply
```

Las contrasenas de Cloud SQL se generan automaticamente con Terraform. No se
guardan en archivos del repositorio; Terraform las conserva en el estado y las
expone unicamente mediante el output sensible `database_urls`.

Antes de aplicar en un entorno real, restringe `cloud_sql_authorized_networks`
al rango de red desde el que se conectara Kubernetes. El valor `0.0.0.0/0` del
ejemplo es solo para una primera prueba y deja las instancias accesibles desde
cualquier IP.

Para consultar las URLs sensibles:

```bash
terraform output -json database_urls
```

No guardar ese resultado en Git. Debe utilizarse para crear el Secret de
Kubernetes con las variables `URL_BASE_DATOS_*`.

## Destruir recursos

`terraform destroy` elimina las cinco instancias de Cloud SQL y sus bases. No se
debe ejecutar hasta tener respaldos y confirmacion del equipo.
