# Promoción de Fase 3 a producción

La release `v3.0.0` conserva Docker Hub como origen de las imágenes. El frontend
se copia con verificación de digest a Artifact Registry para Cloud Run.

1. Validar y publicar `release/v3.0.0`, que incluye todos los servicios y el mock.
2. Etiquetar exactamente ese commit como `v3.0.0` y esperar su publicación.
3. Promover el mismo commit a `main`. El pipeline rechaza un HEAD sin tag exacto,
   para no desplegar silenciosamente una versión anterior.
4. Antes de detener dev, comprobar el Role lector `bank-observability` en
   `bank-usac`. Su bootstrap requiere administrador; CI no recibe permisos IAM
   adicionales. El Role permite únicamente get/list/watch de recursos de
   observabilidad, nunca Secrets ni administración del cluster.
5. Detener todos los consumidores de dev, incluido el laboratorio, antes de
   iniciar producción, ya que las bases externas se conservan.
6. Desplegar producción, actualizar la misma instalación persistente de Grafana,
   Prometheus, Loki y Tempo, y mover sus selectores a `prod` / `bank-usac`.
7. Desplegar el laboratorio aislado en producción, conservando las credenciales
   de su esquema `bank_scenarios`; no reinicia saldos ni usa el esquema bancario.
8. Validar HTTPS, disponibilidad y Swagger antes de retirar el frontend dev.

Grafana mantiene URL HTTPS, usuarios, UIDs de dashboards, PVC e históricos.
El dashboard de escenarios conserva el entorno `interbank-cloud-tests`, separado
del tráfico bancario `prod`. Los paneles de negocio requieren tráfico real nuevo;
no se inventan resultados ni se envían pagos a BANK14 para llenarlos.

En la primera promoción fallida, el rollback detiene producción antes de restaurar
dev y su Service, evitando dos conjuntos de consumidores sobre las mismas bases.
No se hacen ajustes retroactivos de pagos ni se borran las bases de datos.
