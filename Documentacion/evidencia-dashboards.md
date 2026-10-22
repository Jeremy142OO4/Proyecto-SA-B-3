# Evidencia de dashboards de Grafana — Bank USAC, Fase 3

## 1. Propósito y alcance

Este documento reúne capturas de Grafana como evidencia de visualización de disponibilidad, presupuesto de error, latencia, recursos de Kubernetes, réplicas y mensajería. Las lecturas se limitan a los valores y leyendas visibles; no sustituyen la revisión de las consultas de los paneles ni los resultados de una prueba de carga.

**Estado de la evidencia:** capturas revisadas visualmente; fecha, zona horaria, filtros completos y consultas pendientes de registrar. Los valores corresponden a las capturas y no al estado actual del sistema.




## 4. Evidencias e interpretación

### E-GRAF-01 — Estado del sistema

![Estado general del sistema en Grafana](/Documentacion/imgF3/estado%20del%20sistema.png)



### E-GRAF-02 — SLO-01: presupuesto de error

![Errores permitidos y restantes del SLO-01](/Documentacion/imgF3/SLO.png)



### E-GRAF-03 — SLO-02: disponibilidad Ready

![Disponibilidad Ready y presupuesto restante por despliegue](/Documentacion/imgF3/SLO1.png)
![Consumo de CPU y memoria por pod](/Documentacion/imgF3/SLO2.png)
![Consumo de CPU y memoria por pod](/Documentacion/imgF3/SLO3.png)



### E-GRAF-04 — CPU y memoria por pod

![Consumo de CPU y memoria por pod](/Documentacion/imgF3/observabilidad1.png)


### E-GRAF-05 — HPA y número de pods

![Réplicas del HPA y número de pods observados](/Documentacion/imgF3/observabilidad2.png)



### E-GRAF-06 — API Gateway: tráfico y latencia

![Tráfico HTTP y percentiles de latencia del API Gateway](/Documentacion/imgF3/escenarioInterbancario1.png)



### E-GRAF-08 — Kubernetes y escalado

![Disponibilidad de despliegues y réplicas del HPA](/Documentacion/imgF3/escenarioInterbancario2.png)


### E-GRAF-09 — RabbitMQ y consumidores

![Mensajes por cola y consumidores en RabbitMQ](/Documentacion/imgF3/escenarioInterbancario3.png)







