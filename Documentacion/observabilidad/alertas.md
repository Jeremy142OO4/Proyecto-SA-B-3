# Alertas y operación — entorno dev

Responsable: equipo técnico. Destino de correo: `3141193530501@ingenieria.usac.edu.gt`.
Grafana → Alerting → Alert rules muestra estados Pending/Firing/Normal/Error. No son notificaciones del panel de clientes ni del administrador bancario.
El canal de correo usa el SMTP existente mediante un Secret separado, STARTTLS obligatorio y validación de certificados. Nunca versionar credenciales.
Agrupación por alerta y ambiente: espera inicial 30 s, agrupación 5 min, recordatorio 4 h (críticas 1 h); se envía recuperación.

## SLO y límites de interpretación

Disponibilidad del Gateway: 99.5% en ventana móvil de 30 días, respuestas no 5xx / respuestas de negocio. Los 4xx cuentan como respuestas válidas; se excluyen health (incluidos sus subpaths), healthz, salud, ready, readyz y metrics.
Presupuesto restante = `0.005 * solicitudes - respuestas_5xx`. Puede ser negativo. Sin tráfico no hay datos, no se asume 100%.
Burn rate = fracción de 5xx / 0.005. Se requieren al menos 20 solicitudes en la ventana corta para evitar alarmas por una única prueba.
Retención: máximo 30 días **o 2 GB**, lo que se alcance primero, conservando el disco existente de 3 GiB. El panel muestra historia TSDB disponible; esto no acredita cobertura continua del Gateway. No certificar un mes completo hasta reunir evidencia suficiente.
El p95 HTTP de 500 ms mide aceptación/consulta, no finalización de transferencias interbancarias. Los plazos interbancarios no se reemplazan por el SLO interno de 10 segundos.
Los receivers existentes del Collector recogen despliegues/HPA del namespace dev y métricas reales del management API de RabbitMQ cada 30 s. No se crean nodos, discos, balanceadores ni servicios públicos adicionales. El acceso Kubernetes es únicamente get/list/watch sin permisos de Secrets ni escritura.

## Runbooks

<a id="bank-burn-fast"></a>
### Consumo rápido del presupuesto

- Condición: burn >14.4 en 1 h **y** 5 min, durante 2 min; mínimo 20 solicitudes/5 min.
- Revisar panel SLO y errores por ruta/servicio; correlacionar logs Loki con trazas Tempo y la hora del despliegue.
- Comprobar DB, broker y dependencias. Si coincide con un despliegue defectuoso, seguir el procedimiento existente de rollback; no ejecutarlo a ciegas.
- Recuperación: ambas ventanas dejan de satisfacer la condición. Conservar evidencia e impacto del presupuesto.

<a id="bank-burn-slow"></a>
### Consumo sostenido del presupuesto

- Condición: burn >6 en 6 h y 30 min, durante 5 min; mínimo 20 solicitudes/30 min.
- Buscar errores persistentes o una ruta degradada; comparar volumen, versiones y dependencias.
- Corregir causa y verificar caída del burn en ambas ventanas; documentar seguimiento aunque se recupere solo.

<a id="bank-http-latency"></a>
### Latencia HTTP

- Condición: p95 del Gateway >500 ms durante 5 min, mínimo 20 solicitudes/5 min.
- Abrir trazas lentas; separar espera RabbitMQ, DB y procesamiento. Consultar réplicas disponibles y HPA.
- No aumentar nodos/costos automáticamente. Corregir cuello de botella o revertir cambio autorizado; verificar recuperación del p95.

<a id="bank-collector-down"></a>
### Recolección caída

- Condición: target bank-otel caído o ausente durante 2 min.
- Revisar pod, eventos, memoria y logs del Collector, su Service y NetworkPolicy; comprobar también Prometheus.
- Recuperar el componente afectado sin reiniciar servicios financieros innecesariamente. Confirmar up=1 y muestras recientes.
- Si Grafana o Prometheus están caídos, estas alertas no pueden garantizar entrega: se requiere comprobación externa para cubrir esa limitación.

<a id="bank-infra-telemetry"></a>
### Métricas Kubernetes ausentes

- Condición: sin muestras de réplicas deseadas por 5 min, sostenido 2 min.
- Revisar receiver k8s_cluster, ServiceAccount y Role/RoleBinding del namespace dev; no ampliar permisos a administrador.
- Verificar errores 403 y Collector. Recuperación: réplicas y HPA reaparecen con etiquetas de despliegue.

<a id="bank-deployment-unavailable"></a>
### Despliegue sin disponibilidad

- Condición: disponibles=0 y deseadas>0 durante 3 min; evita alertar despliegues apagados intencionalmente.
- Revisar pods Pending/CrashLoopBackOff, probes, recursos y eventos. Relacionar con despliegue o pérdida de broker/DB.
- Aplicar corrección o rollback autorizado; comprobar réplicas Ready y operación de lectura. No generar transferencias como prueba automática.

<a id="bank-rabbitmq-backlog"></a>
### Acumulación RabbitMQ

- Condición: una cola tiene >100 mensajes ready por 5 min. Se excluyen colas RPC temporales amq.gen-*.
- Revisar consumidores, tasa de llegada y procesamiento, desconexiones y errores. No purgar colas ni reencolar pagos indiscriminadamente.
- Recuperar consumidor afectado; confirmar descenso del backlog y ausencia de duplicación. Este umbral mide volumen, no antigüedad ni Sagа detenida.

<a id="bank-rabbitmq-telemetry"></a>
### RabbitMQ sin observación

- Condición: sin métricas de colas durante 5 min, sostenido 2 min.
- Revisar management API, conectividad privada y Secret del receiver; nunca imprimir contraseñas. Si el broker está caído, revisar PVC y pod.
- Confirmar retorno de métricas y consumidores. No confundir esta ausencia con una cola vacía (valor 0).

## Verificación

`python infrastructure/observability/alerting.py --check` valida que los artefactos generados coincidan con su fuente. `validate.py` valida Helm, seguridad, componentes y regresiones antes del despliegue.
Cambios en reglas/dashboard se realizan en `alerting.py`, regenerando sus JSON; las modificaciones desde la UI no son la fuente de verdad.
Tras desplegar: comprobar estado y consultas de cada regla, paneles, datos de RabbitMQ/HPA y enviar una prueba del contacto al correo técnico. Confirmar recepción con el equipo; aceptación SMTP por sí sola no prueba que el mensaje llegó a la bandeja.
No inyectar fallos en el banco ni hacer transferencias reales para probar notificaciones. Las pruebas de fallos y sus postmortems pertenecen a la etapa de evidencia.

## Comprobación inicial, 20 de octubre de 2026

Las ocho reglas se evaluaron en Grafana 13.2.4 con `health=ok`, sin errores de consulta. Prometheus recibió métricas de ocho despliegues, cinco HPA y colas reales de RabbitMQ. El contacto técnico respondió `status=success` a la prueba de correo desde Grafana; falta la confirmación humana de recepción en bandeja.
La validación incluye arranque real de la imagen de Grafana, provisión de reglas/contacto y expresiones ejecutadas con promtool en 16 escenarios sintéticos, sin modificar saldos ni enviar transferencias. Las pruebas del CI usan SMTP deshabilitado y no envían correos reales.
Las métricas de dominio basadas en eventos pueden mostrar «sin datos» si no hubo actividad reciente. Esto no demuestra una falla ni certifica que todos los casos funcionales hayan sido probados.
