# Punto 4: plataforma de observabilidad

El perfil de desarrollo instala Collector, Prometheus, Loki, Tempo y Grafana mediante el chart de `infrastructure/observability/chart`. Un agente por nodo recoge exclusivamente los logs de los seis componentes backend. Los servicios envían métricas y trazas al Collector; las consultas del negocio siguen usando sus canales habituales.

## Acceso privado

No se crea IP pública ni Ingress para estas herramientas. Con las credenciales vigentes del cluster:

```sh
kubectl -n bank-usac-observability port-forward service/grafana 3000:3000
```

Abrir http://localhost:3000. Usuario `admin`; obtener la contraseña sin guardarla en el repositorio:

```sh
kubectl -n bank-usac-observability get secret observability-grafana -o jsonpath='{.data.password}' | base64 --decode
```

Grafana provisiona el tablero **Bank USAC — Observabilidad dev** y las fuentes Prometheus, Loki y Tempo. Los logs JSON permiten abrir su TraceID en Tempo. El valor 99.5 del tablero es el objetivo de disponibilidad, no una certificación de cumplimiento.

## Despliegue y CI/CD

```sh
KUBE_NAMESPACE=bank-usac-dev bash infrastructure/observability/deploy.sh
```

El script comprueba recursos disponibles, conserva la contraseña existente y usa Helm con espera y rollback automático si falla la instalación. Al cambiar el endpoint reinicia secuencialmente los seis componentes. Durante este reinicio inicial de dev se permite indisponibilidad de una réplica para evitar crear pods adicionales; después se restaura la estrategia anterior.

Los cambios exclusivamente en `infrastructure/observability/` validan y despliegan el chart en develop sin reconstruir imágenes del negocio. Cambios globales siguen activando todas las validaciones. Este perfil no despliega observabilidad en prod y no cambia sus réplicas. El workflow está integrado, pero su ejecución remota requiere hacer push.

## Recursos y datos

- Una réplica por herramienta; no es un perfil de alta disponibilidad.
- Con dos nodos: solicitudes totales aproximadas de 60m CPU y 608Mi RAM. La comprobación de capacidad no garantiza la colocación exacta ni evita por sí sola que el autoescalador actúe.
- Cuatro volúmenes persistentes: 3Gi para Prometheus, Loki y Tempo; 1Gi para Grafana. Generan costo de almacenamiento.
- Retención de métricas, logs y trazas: 48 horas; Prometheus limita también a 2GB.
- Los PVC se conservan al desinstalar Helm. Para dejar de pagar deben eliminarse explícitamente cuando ya no se necesiten; esto destruye los datos.
- Los identificadores de correlación y traza no se usan como etiquetas indexadas. No registrar contraseñas, tokens ni datos bancarios sensibles.
- Hay políticas de red declaradas, pero su aplicación depende de que el cluster tenga habilitado un motor que las haga cumplir.

## Verificación

Se comprobaron en dev solicitudes de lectura de perfil, cuentas, transferencias y pagos, series HTTP reales en Prometheus, logs con el mismo TraceID en Loki y la traza distribuida correspondiente en Tempo. También se verificó el estado de los pods y que producción permaneciera en cero réplicas.

Las alertas, las ventanas completas de SLO/error budget y las métricas específicas de KYC/anomalías no forman parte de este punto.

Durante la reducción del nodo temporal creado por el autoescalador se reinició RabbitMQ y se detectaron consumidores que no recuperaban la conexión. La corrección posterior reconstruye las sesiones de Cuentas, Transferencias, Pagos y Auditoría sin reiniciar los procesos. Ver [recuperación de mensajería](../sre/recuperacion-mensajeria.md). El overlay de dev ahora evita pods adicionales durante los rollouts; acepta indisponibilidad breve de cada servicio para priorizar el presupuesto de dos nodos. Prod conserva su estrategia anterior.
