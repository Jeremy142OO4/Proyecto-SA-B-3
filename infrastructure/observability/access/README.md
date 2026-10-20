# Acceso HTTPS de Grafana

Cloud Run funciona como proxy HTTPS hacia un balanceador **interno** de GKE. No expone Prometheus, Loki, Tempo, el Collector ni otra IP pública. Grafana conserva autenticación, registro deshabilitado y cookies Secure. El proxy no recibe credenciales bancarias ni permisos de acceso a bases de datos.

## Recursos y costos

Terraform administra únicamente la IP interna, la identidad sin roles de aplicación y Cloud Run con su permiso público de transporte. La autenticación real sigue siendo la de Grafana, no acceso anónimo. Cloud Run: 0 instancias mínimas, máximo 2, 1 CPU/128 MiB por instancia, Direct VPC egress sin connector dedicado. Helm administra el Service interno y su NetworkPolicy. Hay cargos del balanceador interno y consumo de Cloud Run; no se crea otro nodo ni disco.

## Provisionamiento

Este módulo tiene estado separado de las bases de datos y del clúster. En la laptop el estado se conserva en `/home/jeremy/.local/state/bank-usac-grafana/terraform.tfstate`, fuera del repositorio. Respaldarlo antes de cambiar de computadora. No volver a aplicar desde un estado vacío ni usar destroy sobre el módulo principal del banco.

1. Autenticarse en GCP e inicializar este módulo con `terraform init -backend-config=path=/home/jeremy/.local/state/bank-usac-grafana/terraform.tfstate`. Es obligatorio apuntar al estado existente, no al directorio temporal usado para las validaciones.
2. Ejecutar plan; verificar que solo se crean/modifican los recursos de Grafana. Aplicar el plan revisado. El backend configurado conserva el estado tanto para plan como para apply.
3. Leer los outputs `grafana_url`, `grafana_internal_ip` y `source_cidr`.
4. Guardar un ConfigMap `grafana-access` en `bank-usac-observability` con las claves `publicURL`, `internalIP` y `sourceCIDR`. No contiene contraseñas.
5. Ejecutar `infrastructure/observability/deploy.sh`. El CI/CD vuelve a leer ese ConfigMap para conservar la configuración en futuros despliegues.
6. Verificar HTTPS: dashboard sin sesión redirige a login; `/api/search` sin sesión responde 401; iniciar sesión y probar paneles/Explore. `/api/health` es público y no sustituye la prueba autenticada.

El enlace depende de que el clúster y Grafana sigan encendidos; no depende del túnel SSH ni de mantener una laptop encendida. No cambia al recrear el pod.

## Equipo y permisos

No compartir la cuenta administradora ni publicar su contraseña. Crear usuarios individuales desde Grafana cuando se conozcan los integrantes y asignar el mínimo rol que permita sus consultas. No habilitar acceso anónimo ni registro público para resolver permisos de Explore.

En la versión fijada de Grafana 13.2.4 se usa `viewers_can_edit=true` para habilitar Explore a usuarios Viewer y permitir ediciones temporales sin guardarlas. Esta opción está deprecada por Grafana; al actualizar de versión se debe revisar su reemplazo y volver a probar permisos. No equivale a asignar Editor ni a habilitar escritura de alertas. Las cuentas de Paula y Damaris se crean fuera del código, con contraseñas distintas y sin permisos de administrador.

## Apagado posterior a la calificación

Para detener estos cargos, deshabilitar primero el acceso en Helm y retirar `grafana-access`, comprobar que GKE eliminó el balanceador y después revisar/aplicar destroy **solo de este módulo y con su estado existente**. Apagar Cloud Run por sí solo no elimina el costo del balanceador. No ejecutar esta limpieza hasta que el equipo lo solicite.
