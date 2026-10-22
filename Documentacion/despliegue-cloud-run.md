# Frontend en Cloud Run: imágenes y rollback

El 21 de octubre de 2026, ejecución 130, el build y publicación del frontend fueron
correctos, pero Cloud Run no pudo resolver la etiqueta del commit `ee61f42` en
`mirror.gcr.io`. La misma etiqueta estaba activa en Docker Hub. La consulta directa
al mirror devolvió 404 por etiqueta y 200 por el digest linux/amd64 publicado.

El rollback Kubernetes no era el fallo original. Después del fallo de importación,
Cloud Run conservó una plantilla con la imagen fallida y el tráfico siguió en la
revisión estable `00018-sz4`. `update-traffic` también falló al validar esa plantilla.
Por ello un mensaje de tráfico 100% estable no demuestra que la configuración del
servicio haya quedado reparada.

## Corrección versionada

- `image-digest.py` consulta el registro oficial de Docker Hub con acceso público de
  lectura, verifica el hash del manifiesto y selecciona linux/amd64 si existe un índice
  multiplataforma. Dev y prod despliegan `docker.io/...@sha256:...` en lugar de resolver
  una etiqueta nueva mediante el mirror de Cloud Run. No crea repositorios ni cambia IAM.
- Si la imagen no se puede verificar, el paso falla antes de modificar Cloud Run.
  Una indisponibilidad del propio registro sigue siendo posible; no se oculta con una
  imagen anterior presentada como nueva.
- `cloud-run-rollback.py capture` guarda el reparto de tráfico **realmente servido**
  y la imagen inmutable de la revisión principal. No usa una revisión latestReady que
  podría no estar atendiendo tráfico. El archivo temporal no contiene variables de
  entorno, credenciales ni configuración de bases del banco.
- En rollback, si la plantilla todavía apunta a la imagen fallida, se repara su imagen
  con el digest estable, creando una revisión sin tráfico. Después se restablece el
  reparto entre las revisiones originales, que conservan su configuración completa.
  No se intenta reemplazar o editar revisiones inmutables.
- Solo se restaura Cloud Run si su paso de despliegue llegó a ejecutarse. Kubernetes
  solo recibe rollout undo si su paso de aplicación llegó a ejecutarse. Una falla
  anterior no provoca rollbacks innecesarios del resto de los servicios.
- Un primer despliegue sin servicio previo no tiene revisión a la que regresar;
  el helper no elimina el servicio automáticamente. Los errores de permisos al guardar
  estado detienen el pipeline y no se confunden con un servicio inexistente.

Las URLs con tags de revisión no se reconfiguran con esta corrección; se restaura el
tráfico de la URL principal. No se modifican saldos, transacciones, esquemas SQL ni datos
de clientes para verificar el pipeline.

## Verificación y operación

```sh
python3 infrastructure/scripts/test-cloud-run.py
python3 infrastructure/scripts/image-digest.py docker.io/jeremy142004/bank-usac-frontend:COMMIT
```

Las pruebas simulan el registro y gcloud, incluidos tag inválido, digest no coincidente,
selección amd64, reparación antes de cambiar tráfico, revisión estable distinta de
latestReady y permisos insuficientes. Se ejecutan en la validación existente del CI.

Publicar estos cambios y lanzar una ejecución nueva del pipeline en develop. Reejecutar
un job de un commit anterior conserva el workflow anterior y no incorpora la corrección.
Tras desplegar, verificar Ready=True, imagen por digest, revisión servida y respuesta HTTP
del frontend. No se requiere migración SQL por esta corrección.

Referencia: [despliegue por digest y caché de imágenes públicas en Cloud Run](https://docs.cloud.google.com/run/docs/deploying).

## Publicación en Artifact Registry

El pipeline verifica el digest linux/amd64 en Docker Hub y copia ese contenido con Docker al repositorio privado `us-central1-docker.pkg.dev/PROYECTO/bank-usac-cloud-run/frontend`. Antes de modificar Cloud Run comprueba que el RepoDigest publicado coincide con el original. Cloud Run recibe la referencia de Artifact Registry por digest; no depende del mirror público de Docker Hub. Se aplica a dev y prod.

El repositorio `bank-usac-cloud-run` se aprovisiona una vez en us-central1. CI necesita `roles/artifactregistry.writer` limitado a ese repositorio; el workflow no crea repositorios ni modifica IAM. Su asignación requiere autorización administrativa explícita. Las copias conservan etiquetas por commit; su almacenamiento queda sujeto a la política de retención configurada en Artifact Registry.

La prueba `test-cloud-run.py` comprueba también que la copia mantiene el digest y que una discrepancia impide continuar con el despliegue. Publicar un commit nuevo activa el workflow corregido; reintentar un commit anterior usa su workflow antiguo.
