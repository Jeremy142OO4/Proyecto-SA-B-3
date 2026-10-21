# Account Service: monitoreo y administración de cuentas



## Comportamiento



Cada débito o crédito confirmado registra el movimiento y evalúa cuántas operaciones de esa cuenta existen en la ventana móvil `(ahora - M minutos, ahora]`. Solo se genera una anomalía cuando el conteo es **mayor que N**: con N=10 se alerta a partir de la operación 11. Las compensaciones, rechazos y reentregas idénticas no incrementan ese conteo.



El bloqueo de la fila de cuenta y la transacción SERIALIZABLE mantienen la consistencia entre réplicas. El movimiento, el registro en `anomalias_cuenta`, la idempotencia y los eventos del outbox se confirman o revierten juntos. Un fallo de serialización se procesa mediante los reintentos existentes del consumidor. El reloj de PostgreSQL determina la fecha del movimiento.



Se limita la notificación a una anomalía por cuenta durante M minutos con los mismos valores N/M, para evitar una alerta por cada operación de una ráfaga. Guardar nuevamente valores idénticos no evita esa supresión. La versión de configuración queda registrada como evidencia. Cambiar los límites aplica a las siguientes operaciones y considera los movimientos que todavía están dentro de la nueva ventana.



## Configuración



El ConfigMap `bank-usac-config` incorpora:



| Variable | Valor inicial | Valores admitidos |

|---|---|---|

| `ACCOUNT_ANOMALY_MAX_OPERATIONS` | `10` | Entero entre 1 y 10000 |

| `ACCOUNT_ANOMALY_WINDOW_MINUTES` | `5` | Entero entre 1 y 1440 |



Estos valores son la configuración base al iniciar el proceso; un cambio del ConfigMap mediante variables de entorno requiere reiniciar sus Pods. La pantalla **Administrar cuentas** permite guardar una configuración operativa en PostgreSQL que tiene prioridad sobre esa base. Esta configuración se comparte entre réplicas, sobrevive a reinicios y se aplica sin redesplegar. Se registra el administrador y se emite `cuenta.monitoreo.configuracion.actualizada`.



## Eventos, notificaciones y métricas



`AccountAnomalyDetected` se publica en `banco.eventos` con id de anomalía, cuenta, cliente, movimiento y operación, tipo de cuenta, conteo, umbral, ventana, versión, fecha y nivel WARNING. Conserva el correlationId y el contexto de traza de la operación mediante el outbox. No incluye correo, documento, credenciales ni descripción libre del cliente.



Notification & Audit declara el binding de ese evento, lo registra como WARNING y genera una notificación `ACCOUNT_ANOMALY_DETECTED` con severidad, cuenta y detalles estructurados. El correo utiliza el comprobante HTML existente con estilo de advertencia e incorpora conteo, umbral, ventana, fecha y correlación. El destinatario se resuelve por el id de cliente mediante el mecanismo existente. La notificación para la mensajería se conserva aunque SMTP falle: la alerta no depende de que el correo haya sido entregado.



Se emiten logs JSON de débitos, créditos y compensaciones confirmados con accountId y contexto. Una anomalía confirmada emite un log WARNING. Las métricas OTel son:



| Serie en Prometheus | Etiquetas adicionales | Significado |

|---|---|---|

| `bank_account_movements_total` | `account_type`, `direction=DEBITO/CREDITO` | Movimientos confirmados, sin duplicar reentregas |

| `bank_account_anomalies_total` | `account_type` | Anomalías confirmadas y guardadas en el outbox |



Los IDs permanecen en eventos/logs, nunca en etiquetas de métricas. Los counters son señales operativas en memoria y se reinician con el proceso; el historial SQL y el outbox son la evidencia durable. Una caída entre commit y emisión/exportación puede perder una muestra. El dashboard Bank USAC-Prueba1 incorpora la sección 02/Account_service.



## Pantallas y permisos



La ruta `/administracion/cuentas`, disponible para ADMIN, muestra cuentas paginadas, propietario, tipo, saldo y estado. Los nombres se consultan en Customer Service por el Gateway; no hay joins entre bases de microservicios. Si esa consulta falla, se conserva el id del propietario y se informa la limitación.



La cabecera del administrador incorpora el icono de mensajería junto a **Sesión protegida**. Consulta cada 15 segundos las últimas 50 notificaciones de anomalía WARNING, muestra las no leídas y permite marcar las alertas como leídas. La lectura se guarda por usuario en el navegador, no se sincroniza entre dispositivos. **Revisar cuenta** abre directamente la cuenta indicada, incluso si estaba en otra página del listado.



El bloqueo es manual, requiere confirmación y un motivo, y emite `cuenta.bloqueada` con notificación WARNING al propietario. Impide nuevos débitos/créditos y permite compensar débitos anteriores para devolver fondos. No se bloquean automáticamente cuentas por una frecuencia alta, que puede corresponder a actividad legítima. No se incorporó una acción de desbloqueo.



Todos los endpoints administrativos exigen ADMIN en el Gateway. El cajero y el cliente no pueden cambiar límites ni bloquear cuentas:



| Método y ruta bajo `/api` | Operación |

|---|---|

| `GET /administracion/cuentas?limite=25&desplazamiento=0&cuenta=UUID` | Listado; el filtro de cuenta es opcional |

| `GET /administracion/cuentas/monitoreo` | Configuración efectiva |

| `PUT /administracion/cuentas/monitoreo` | Guardar `maxOperaciones` y `ventanaMinutos` |

| `PATCH /administracion/cuentas/:idCuenta/bloqueo` | Bloquear con `motivo` |

| `GET /auditoria/notificaciones?tipo=ACCOUNT_ANOMALY_DETECTED&nivel=WARNING&limite=50` | Mensajería de anomalías |



El Gateway continúa comunicándose por RabbitMQ. Las respuestas administrativas se filtran por tipo para que un evento de negocio con la misma correlación no satisfaga por error la espera HTTP.



## Migraciones y validación



Aplicar todas las migraciones pendientes antes del despliegue, incluidas `account-service/000006_monitoreo_cuentas` y `service-notification-audit/000003_nivel_notificaciones`. La primera crea la configuración e historial; la segunda agrega nivel, cuenta y detalles de notificación. No se modificaron las bases existentes del servidor.



Se validaron:



- Pruebas de Account Service con PostgreSQL 17: N frente a N+1, ventana móvil, supresión, cambios de configuración, reentregas, rollback, concurrencia entre instancias y compensación de una cuenta bloqueada.

- Pruebas de Notification & Audit: clasificación WARNING, comprobante HTML, idempotencia, filtros y persistencia de detalles, incluso con fallo SMTP.

- Pruebas del Gateway: ADMIN permitido, CLIENTE/TELLER rechazados y respuesta administrativa distinguida de eventos con la misma correlación.

- `go test -race ./...` y `go vet ./...` de los tres módulos en el servidor con PostgreSQL/RabbitMQ temporales; comprobaciones locales adicionales después de los ajustes finales.

- Recorrido real con cuatro servicios: tres depósitos con N=2/M=5 produjeron una única alerta WARNING consultable desde el Gateway.

- Revisión del frontend con ese backend: icono y alerta, marcar como leída, abrir la cuenta correcta, cambiar N/M y confirmar el bloqueo. TypeScript, compilación Vite y lint de los archivos modificados pasaron.



Los datos de esa revisión fueron ficticios y se ejecutaron en contenedores aislados. No se enviaron correos reales; se verificó su estructura con un remitente de pruebas.





### Bloqueo desde mensajería y acceso del propietario



Cada alerta permite confirmar el bloqueo con el motivo «Se han detectado múltiples transacciones en poco tiempo.». El evento `cuenta.bloqueada` envía al propietario un comprobante con ese motivo. La cuenta aparece como BLOQUEADA en su listado y detalle, actualizados cada 15 segundos y al recuperar el foco. El historial sigue disponible; depósitos, pagos y transferencias desde cuentas bloqueadas se rechazan tanto en Gateway como en Account Service. Las compensaciones de operaciones anteriores siguen permitidas para devolver fondos.



Las migraciones son archivos SQL versionados, independientes por servicio: cuentas `000006` crea la configuración y el registro de anomalías; notificaciones `000003` agrega nivel, cuenta y detalles a las notificaciones. Deben ejecutarse sobre la base correspondiente mediante el mecanismo de migraciones del despliegue, después de las versiones anteriores y antes de iniciar el nuevo backend. No trasladan dinero ni crean cuentas de clientes. Las pruebas aisladas no aplican estos cambios a las bases del entorno desplegado.



### Aplicación de migraciones en el servidor de pruebas (18 de octubre de 2026)

En `201.216.128.230` se respaldaron las bases existentes antes de modificar sus esquemas. Los respaldos quedaron en `/home/servidor/bank-usac-test/cuentas-before-monitoring-20261018.dump` y `/home/servidor/bank-usac-test/audit-before-monitoring-20261018.dump`, con permisos 600 y formato PostgreSQL custom, cuya tabla de contenido se verificó.

- `bank-usac-postgres-cuentas`: versión 4 → 6, incluyendo contexto de traza (5) y monitoreo (6). La cuenta preexistente se conservó.
- `bank_postgres_audit`: versión 1 → 3, incluyendo severidad de auditoría (2) y detalles de notificaciones (3).

Cada actualización se ejecutó dentro de una transacción con comprobación de versión inicial, bloqueo de la tabla de versiones y límites de espera. Ambas finalizaron con `dirty=false`; se verificaron las tablas y columnas nuevas.

Los fuentes de Account Service, Notification & Audit, Gateway y frontend quedaron preparados en `/home/servidor/bank-usac-test/releases/account-monitoring-20261018`. Esto no implica que ya estén atendiendo tráfico. El Minikube del servidor no contenía la aplicación bancaria y el contexto GKE configurado no respondía durante la revisión; debe definirse el destino del despliegue antes de instalar servicios nuevos o reemplazar el entorno remoto.


### Despliegue por CI/CD hacia Google Cloud

El destino es el entorno administrado por GitHub Actions, no una instalación nueva en Minikube. `infrastructure/scripts/prepare-telemetry-schema.sh` prepara, mediante Jobs de Kubernetes y las URLs de los Secrets del entorno, los esquemas de trazas y funcionalidades: Customer Service 000007, Account Service 000006 y Notification & Audit 000003, incluyendo sus requisitos de telemetría y severidad. Los cambios son aditivos y reejecutables. Cada Job usa una transacción y un bloqueo consultivo para evitar cambios concurrentes en una misma base. Si falla, el pipeline se detiene antes de desplegar el nuevo frontend o las imágenes del backend. Los registros de versión `schema_migrations` gestionados por otros mecanismos no se sobrescriben: este paso sigue el mecanismo de DDL aditivo ya utilizado por el proyecto.

La validación de sistema ejecuta `python3 infrastructure/scripts/test-feature-schema.py` contra un contenedor PostgreSQL desechable, sin puertos públicos ni acceso a las bases reales. Comprueba los cinco servicios, la reejecución de los esquemas, la conservación de un saldo y la reversión de una migración fallida. La prueba se ejecutó satisfactoriamente en el servidor el 18 de octubre de 2026. También se validó el YAML del workflow y la posición del paso de migraciones antes de Cloud Run en desarrollo y producción.

El workflow conserva los disparadores existentes: un push a `develop` permite desplegar desarrollo cuando pasan las validaciones; el flujo de producción conserva sus propias condiciones. Cambiar archivos locales o ejecutar pruebas en el servidor no inicia GitHub Actions ni demuestra un despliegue en Google Cloud. El Minikube arrancado para diagnóstico se volvió a detener y se restauró el contexto GKE original.

## Grafana: 02/Account_service

El dashboard `Bank USAC-Prueba1` incorpora:

- Columnas: estimación de movimientos confirmados por AHORRO/CORRIENTE y DEBITO/CREDITO durante el rango seleccionado, con `sum by (account_type, direction) (increase(bank_account_movements_total{...}[$__range]))`. Crédito es entrada; débito es salida. `increase` puede devolver fracciones y requiere al menos dos muestras; no es un conteo contable exacto. MONETARIA legado no se mezcla con CORRIENTE.
- Gráfico de puntos: pegar un UUID en **UUID de cuenta (historial)**. Dos series cuentan entradas y salidas por minuto, desde logs Loki de `account movement committed`. El eje Y es número de operaciones, no dinero ni saldo. La variable vacía no selecciona cuentas.
- Historial observable: logs originales de esa cuenta, hasta 100 líneas del rango seleccionado. Incluye UUID, tipo y operación; expandir los detalles para correlación/traza cuando estén presentes.
- Alertas: logs WARNING de `AccountAnomalyDetected` del entorno. La notificación de negocio continúa en auditoría y mensajería administrativa.

Los UUID se extraen del JSON al consultar Loki; no se añaden etiquetas de cuenta a Prometheus ni al índice de Loki. Los logs se conservan 48 h según la configuración actual. El gráfico de puntos no reconstruye movimientos anteriores a la recolección ni sustituye el historial SQL. Los conteos Prometheus incluyen todas las cuentas de los dos tipos, independientemente del UUID seleccionado.

### Endpoint existente de movimientos

`GET /api/cuentas/{idCuenta}/movimientos?limite=50&desplazamiento=0`, autenticado con JWT del rol CLIENTE y validación de propiedad de la cuenta. Devuelve movimientos con `idMovimiento`, `tipoMovimiento`, `montoCentavos`, `saldoNuevoCentavos`, `fechaCreacion` y descripción. No es un endpoint administrativo y no se habilita acceso a cuentas ajenas. El UUID de la cuenta está en la ruta. No se debe guardar un JWT de cliente en el dashboard.

Grafana utiliza los datasources ya aprovisionados Prometheus y Loki; no requiere consultar ese endpoint ni instalar un plugin HTTP. Para probar manualmente, generar un depósito y una transferencia de una cuenta, esperar exportación/recolección, seleccionar su UUID y comprobar entradas/salidas en los puntos y logs. Validar el historial contable mediante el endpoint del propietario. Configuración de N/M y bloqueo siguen en administración; el dashboard no modifica esos valores.

Cambios de publicación: JSON del dashboard y despliegue Helm existente, sin migraciones ni modificaciones de backend para estos paneles. Falta verificar visualmente las consultas contra los datasources del entorno desplegado.
