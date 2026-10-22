# Postmortem Blameless — Contrato interbancario incompleto o discrepante con BANK14

| Campo | Valor |
|---|---|
| **ID del incidente** | PM-F3-04 |
| **Fecha** | 2026-10-20 |
| **Hora de inicio** | 23:12:00 UTC (primera entrada `ERROR` registrada) |
| **Severidad** | SEV-3 (fallo de integración entre bancos, sin pérdida de datos) |
| **Servicios involucrados** | Transaction Service, Payment Service, API Gateway |
| **Componente fallido** | Contrato de integración BANK03 ↔ BANK14  |
| **Banco pareja** | BANK14 |
| **Autores** | Equipo BANK03 |
| **Estado** |Mitigado  |

---

## 1. Resumen ejecutivo

Durante las pruebas de integración interbancaria con BANK14 se registraron fallos
recurrentes en `transaction-service` al procesar solicitudes cuyo contrato de datos no
coincidía con lo que el banco pareja esperaba recibir o devolver.

A diferencia de PM-F3-01, donde la causa fue un identificador de cuenta con formato
inválido enviado por nuestro lado, este incidente se refiere a **discrepancias en la
definición del contrato**: campos faltantes, nombres distintos, tipos incompatibles o
endpoints no acordados entre ambos equipos.

**Resultado observado:** falla en la conciliación por no seguir el contrato en la linea 571 del contrato

---

## 2. Alcance y diferencia con los otros postmortems

| Postmortem | Naturaleza del fallo | Quién origina el fallo |
|---|---|---|
| PM-F3-01 | Cuenta destino inexistente por UUID malformado | BANK03 (dato de entrada inválido) |
| PM-F3-03 | Caída de pod o del broker | Infraestructura propia |
| **PM-F3-04** | **Contrato de integración incompleto o discrepante** | **Definición conjunta entre ambos bancos** |

Este escenario es el único de los tres que no se resuelve con cambios de código
aislados: requiere una actualización del acuerdo técnico y coordinación con el equipo
de BANK14.

---

## 3. Evidencia disponible

### 3.1 Registro de errores en el visor de logs

Captura del panel «Logs de los microservicios — abrir TraceID para ver traza»:

| Campo | Valor observado |
|---|---|
| Nivel | `ERROR` |
| `k8s_container_name` | `transaction-service` |
| `k8s_pod_name` | `transaction-service-...` (truncado en la captura) |
| `k8s_pod_uid` | `ed9f5e4f-95b1-409e-...` (truncado) |
| `log_file_path` | `/var/log/pods/ban...` (truncado) |
| Rango de marcas de tiempo | `2026-10-20 23:12:00.598` a `2026-10-20 23:14:03.153` |
| Cantidad de entradas `ERROR` visibles | 10 |

**Limitación de la evidencia:** en la captura la columna del mensaje queda fuera del
área visible, por lo que **el texto del error no está documentado**. Sin ese texto no
es posible afirmar cuál campo o endpoint específico del contrato falló.



## 4. Hipótesis a validar

| # | Hipótesis | Cómo se verifica |  
|---|---|---|
| H1 | BANK03 envía un campo que BANK14 no reconoce | Comparar payload enviado con el esquema acordado  |
| H2 | BANK14 espera un campo obligatorio que BANK03 no envía | Revisar la respuesta de error de BANK14 |
| H3 | El nombre o la ruta del endpoint difiere del acordado | Contrastar con el documento de acuerdo técnico |
| H4 | Hay incompatibilidad de tipos (numérico vs cadena, fecha con/sin zona horaria) | Inspeccionar el payload y el error de validación  |
| H5 | El formato del identificador de cuenta no está normalizado entre bancos | Verificar PM-F3-01 y el acuerdo técnico  |
| H6 | La versión del contrato usada por cada equipo es distinta | Solicitar a BANK14 la versión que tienen desplegada |

---

## 5. Procedimiento para reproducir y documentar el fallo

> Ejecutar en el namespace de desarrollo. Sustituir los valores entre corchetes.

**Paso 1 — Capturar el payload exacto que envía BANK03:**

```bash
kubectl --context [CONTEXTO] -n [NAMESPACE] logs deploy/transaction-service --tail=200   | grep -i "interbank" | grep -i "payload\|request"
```

**Paso 2 — Enviar una solicitud de prueba y guardar petición y respuesta completas:**

```bash
curl -sS -i -X POST "https://[HOST_API_GATEWAY]/api/interbancarias/"   -H "Content-Type: application/json"   -H "Authorization: Bearer [TOKEN]"   -d '{
    "originAccount": "107102621189",
    "destinationAccount": "[CUENTA_VALIDA_EN_BANK14]",
    "amount": 100.00,
    "currency": "GTQ"
  }' | tee evidencia-pm04-respuesta.txt
```

**Paso 3 — Obtener el `TraceID` y revisar la traza:**

```logql
{k8s_container_name="transaction-service"} |= "ERROR" | json
```

Filtrar por la ventana `2026-10-20 23:12:00` a `23:14:10` y copiar el `traceId` de una
de las entradas para abrirla en Tempo.

**Paso 4 — Contrastar contra el acuerdo técnico:**

Comparar campo por campo el payload de BANK03 con la sección de contrato de
`interbancaria/acuerdo-tecnico-bank03-bank14.md` y anotar cada diferencia en la tabla
de la sección 6.

---



## 6. Cronología de los hechos (UTC)

| Hora | Evento | Fuente de evidencia |
|---|---|---|
| 23:12:00.598 | Primera entrada `ERROR` en `transaction-service` | Visor de logs |
| 23:12:01.651 | Segunda entrada `ERROR` | Visor de logs |
| 23:12:30.971 | Tercera entrada `ERROR` | Visor de logs |
| 23:12:32.024 | Cuarta entrada `ERROR` | Visor de logs |
| 23:13:01.340 | Quinta entrada `ERROR` | Visor de logs |
| 23:13:02.394 | Sexta entrada `ERROR` | Visor de logs |
| 23:13:13.778 | Séptima entrada `ERROR` | Visor de logs |
| 23:13:31.725 | Octava entrada `ERROR` | Visor de logs |
| 23:14:02.100 | Novena entrada `ERROR` | Visor de logs |
| 23:14:03.153 | Décima entrada `ERROR` | Visor de logs |
|  | Identificación de la causa | Análisis del mensaje de error |
|  | Comunicación con el equipo de BANK14 | Canal de coordinación |
| | Corrección aplicada o acuerdo actualizado | Commit / documento |


---

## 7. Detección y diagnóstico

- **Método de detección:** revisión manual del panel de logs. No hubo alerta automática.

- **Consulta en Loki para aislar el error:**

```logql
{k8s_container_name="transaction-service"} |= "ERROR"
  | json
  | line_format "{{.timestamp}} {{.message}}"
```

- **Consulta en Loki acotada a la ventana del incidente:**

```logql
{k8s_container_name="transaction-service"} |= "ERROR"
  | json
  | __error__ = ""
```

Aplicar el rango de tiempo `2026-10-20 23:12:00` a `2026-10-20 23:14:10` en el selector
del panel.

- **Verificación en base de datos:**

```sql
SELECT id, status, reason_code, created_at, updated_at
FROM interbank_transfers
WHERE created_at BETWEEN '2026-10-20 23:00:00' AND '2026-10-20 23:30:00'
ORDER BY created_at DESC;
```

---


**Factores contribuyentes:**

- El acuerdo técnico no define un esquema formal (JSON Schema u OpenAPI) validable por ambas partes.
- No existe un entorno de pruebas conjunto previo a la integración real.
- Los errores de validación de la contraparte no se traducen a un mensaje accionable en nuestros logs.

---

## 8. Acciones correctivas

| # | Acción | Tipo | Responsable | 
|---|---|---|---|
| 1 | Formalizar el contrato con JSON Schema u OpenAPI compartido | Prevención | Ambos equipos | 
| 2 | Versionar el contrato y registrar la versión desplegada por cada banco | Prevención | Ambos equipos | 
| 3 | Implementar validación de esquema en la frontera del API Gateway | Prevención | Backend | 
| 4 | Registrar el cuerpo completo de la respuesta de BANK14 ante error | Detección | Backend |
| 5 | Alerta sobre tasa de errores de validación por `reasonCode` | Detección | SRE | 
| 6 | Establecer un entorno de pruebas conjunto (sandbox) con BANK14 | Mitigación | Ambos equipos |
| 7 | Documentar cada discrepancia resuelta en el acuerdo técnico | Prevención | Ambos equipos |

---

### Evidencia
![contratos](../imgF3/contraatos.jpeg)
