"""Generate the system dashboard from real instruments and the approved SLOs."""
import json
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent
PROM = {'type': 'prometheus', 'uid': 'prometheus'}
EXCLUDE = '/(health(/.*)?|healthz|salud|ready|readyz|metrics)'
HTTP = 'bank_http_requests_total{service="api-gateway",environment="dev",route!~"' + EXCLUDE + '"}'
LATENCY = 'bank_http_duration_seconds_bucket{service="api-gateway",environment="dev",route!~"' + EXCLUDE + '"}'
INFRA = 'job="bank-infra",k8s_namespace_name="bank-usac-dev"'


def total(window):
    return f'sum(increase({HTTP}[{window}]))'


def bad(window):
    return f'(sum(increase({HTTP[:-1]},status_class="5xx"}}[{window}])) or (0 * {total(window)}))'


def ratio(numerator, denominator):
    return f'(({numerator}) / ({denominator})) and (({denominator}) > 0)'


def dashboard():
    panels = []
    y = 0
    panel_id = 0

    def next_id():
        nonlocal panel_id
        panel_id += 1
        return panel_id

    def row(title):
        nonlocal y
        panels.append(dict(id=next_id(), type='row', title=title, collapsed=False, panels=[],
                           gridPos=dict(x=0, y=y, w=24, h=1)))
        y += 1

    def panel(title, targets, unit='short', kind='timeseries', width=12, x=0, description='', height=8, datasource=PROM):
        nonlocal y
        defaults = dict(unit=unit, noValue='Sin datos', color=dict(mode='palette-classic'))
        if kind == 'timeseries':
            defaults['custom'] = dict(drawStyle='line', lineInterpolation='linear', lineWidth=2,
                                      fillOpacity=8, showPoints='auto', spanNulls=False)
        item = dict(id=next_id(), title=title, description=description, type=kind,
                    gridPos=dict(x=x, y=y, w=width, h=height), datasource=datasource,
                    targets=[dict(refId=chr(65+i), expr=expr, legendFormat=legend,
                                  instant=kind in ('stat', 'table'), range=kind not in ('stat', 'table'))
                             for i, (expr, legend) in enumerate(targets)],
                    fieldConfig=dict(defaults=defaults, overrides=[]), maxDataPoints=300,
                    options=dict(legend=dict(displayMode='table', placement='bottom', calcs=['lastNotNull']),
                                 tooltip=dict(mode='multi', sort='desc')))
        if kind == 'stat':
            item['options'] = dict(reduceOptions=dict(values=False, calcs=['lastNotNull'], fields=''),
                                   graphMode='none', colorMode='value', textMode='value_and_name')
        panels.append(item)
        if x + width == 24:
            y += height
        return item

    row('01 / Estado del sistema')
    panel('Disponibilidad Gateway · 30 días · objetivo 99.5%',
          [(f'100 * ({ratio(f"{total('30d')} - {bad('30d')}", total("30d"))})', 'No 5xx')],
          'percent', 'stat', 12, 0, 'Obligatorio: disponibilidad. Respuestas no 5xx; incluye 4xx. Excluye salud/readiness/métricas. Sin tráfico: sin datos. Ventana móvil con la historia disponible. Puede permanecer estable durante operaciones exitosas.', 5)
    panel('Error budget restante · SLO 99.5%',
          [(f'100 * (1 - ({ratio(bad("30d"), "0.005 * " + total("30d"))}))', 'Presupuesto restante')],
          'percent', 'stat', 12, 12, 'Obligatorio: error budget. 100% = presupuesto intacto; 0% = agotado; puede ser negativo. Fallos permitidos = 0.005 × solicitudes observadas. No debe variar artificialmente si no hay errores.', 5)
    next_id()  # Retired scrape-status card; preserve existing panel links/IDs.
    panel('Réplicas disponibles por despliegue', [(f'k8s_deployment_available{{{INFRA}}}', '{{k8s_deployment_name}}')])
    panel('Solicitudes elegibles y errores · 30 días', [(total('30d'), 'Solicitudes observadas'), (bad('30d'), 'Respuestas 5xx')],
          kind='stat', x=12, description='Prometheus estima incrementos entre scrapes; no es un registro contable exacto. No certifica cobertura completa de 30 días.')

    row('02 / SLI · latencia, errores y throughput')
    panel('Latencia de transferencias · p50, p95 y p99',
          [(f'histogram_quantile({q},sum by(le)(rate(bank_transfer_flow_duration_seconds_bucket{{service="transaction-service",environment="dev"}}[$__rate_interval])))', label)
           for q, label in [(.5, 'p50'), (.95, 'p95'), (.99, 'p99')]], 's', width=24,
          description='Histograma desde creación hasta resultado terminal de la transferencia, incluidas esperas y compensaciones. No es promedio. Sin resultados recientes no hay percentil.')
    panel('Errores HTTP del Gateway · 5xx', [(f'100 * ({ratio(bad("5m"), total("5m"))})', 'Errores técnicos HTTP (%)')], 'percent')
    transfer = 'bank_transfer_results_total{service="transaction-service",environment="dev"}'
    den = f'sum(increase({transfer}[$__range]))'
    panel('Transferencias terminadas · por resultado en el intervalo',
          [(f'(sum(increase({transfer[:-1]},outcome="{outcome}"}}[$__range])) or (0 * {den}))', label)
           for outcome, label in [('success', 'Completadas'), ('failure', 'Fallidas / compensadas')]], 'short', kind='stat', x=12,
          description='Resultados terminales acumulados en el intervalo seleccionado; cambia al completar transferencias, incluso si todas tienen éxito. Prometheus estima incrementos, no es el historial contable exacto. Un rechazo/compensación no equivale a fallo técnico; la tasa HTTP 5xx se muestra aparte. Sin la serie base no se inventa cero.')
    panel('Throughput · eventos por segundo por servicio',
          [('sum by(service,action)(rate(bank_messages_total{environment="dev"}[$__rate_interval]))', '{{service}} · {{action}}')], 'ops',
          description='Intentos de publicación/procesamiento. Se separan por acción: no sumar productor y consumidor como eventos únicos.')
    panel('Throughput · eventos por segundo por routing key',
          [('sum by(service,event_type,action)(rate(bank_messages_total{environment="dev"}[$__rate_interval]))', '{{service}} · {{event_type}} · {{action}}')], 'ops', x=12,
          description='event_type es la routing key del catálogo permitido. Claves no reconocidas se agrupan en other para limitar cardinalidad.')
    row('03 / RabbitMQ · actividad y consumer lag')
    broker_throughput = panel('Throughput RabbitMQ · por cola',
          [(f'sum by(rabbitmq_queue_name)(rate(rabbitmq_message_{metric}_total{{job="bank-infra"}}[$__rate_interval]))', '{{rabbitmq_queue_name}} · ' + label)
           for metric, label in [('published', 'publicados'), ('delivered', 'entregados'), ('acknowledged', 'confirmados')]], 'ops', width=24,
          description='Contadores del management API del broker; los reintentos son entregas adicionales. Colas RPC temporales excluidas del almacenamiento.')
    broker_throughput['options']['legend']['calcs'] = ['lastNotNull', 'max']

    queue = 'rabbitmq_message_current{job="bank-infra",state="ready"}'
    consumers = 'rabbitmq_consumer_count{job="bank-infra"}'
    for _ in range(3):
        next_id()  # Retired ready/unacked/consumer-count panels; preserve IDs.
    queue_panels = [panel('Pendientes por consumidor activo · pico del promedio',
          [(f'max_over_time(({queue} / on(rabbitmq_queue_name,rabbitmq_vhost_name,rabbitmq_node_name) ({consumers} > 0))[$__interval:15s])', '{{rabbitmq_queue_name}}')], width=24,
          description='Obligatorio: consumer lag del broker. Calcula ready / consumidores activos y conserva su máximo observado por intervalo. Es un promedio por cola, no mensajes asignados individualmente. Un cero es normal si el flujo procesa los mensajes entre recolecciones; no se crea acumulación artificial. No divide entre cero: una cola sin consumidores no tiene un promedio definido y queda fuera de esta razón. No certifica salud de todas las colas ni detecta picos no recolectados.')]
    for queue_panel in queue_panels:
        queue_panel['interval'] = '30s'
        queue_panel['fieldConfig']['defaults'].update(min=0, decimals=0)
        queue_panel['fieldConfig']['defaults']['custom']['lineInterpolation'] = 'stepAfter'
        queue_panel['options']['legend']['calcs'] = ['lastNotNull', 'max']
    next_id()  # Retired blocked-queue card. Broker alerts/collection remain enabled.

    row('04 / SLO y error budget · ventana móvil de 30 días')
    panel('SLO-01 · errores permitidos y presupuesto restante',
          [(f'0.005 * {total("30d")}', 'Fallos permitidos'), (f'0.005 * {total("30d")} - {bad("30d")}', 'Fallos restantes')], width=24)
    ready = f'(k8s_deployment_available{{{INFRA}}} > bool 0)'
    availability = f'avg_over_time({ready}[30d:1m])'
    panel('SLO-02 · disponibilidad Ready · objetivo 98.5%', [(f'100 * {availability}', '{{k8s_deployment_name}}')], 'percent',
          description='Proporción de minutos observados con al menos una réplica disponible; proxy de disponibilidad del servicio. Intervalos sin telemetría no se convierten en éxitos.')
    panel('SLO-02 · presupuesto restante por despliegue', [(f'100 * (1 - (1 - {availability}) / 0.015)', '{{k8s_deployment_name}}')], 'percent', x=12)
    buckets = f'sum by(le)(increase({LATENCY}[30d]))'
    count = f'sum(increase({LATENCY.replace("_bucket", "_count")}[30d]))'
    fast = f'sum(increase({LATENCY[:-1]},le="0.5"}}[30d]))'
    panel('SLO-03 · p95 HTTP · objetivo ≤ 500 ms', [(f'histogram_quantile(0.95,{buckets})', 'p95 HTTP')], 's')
    panel('SLO-03 · presupuesto de solicitudes lentas',
          [(f'100 * (1 - ({ratio(f"{count} - {fast}", f"0.05 * {count}")}))', 'Presupuesto restante')], 'percent', x=12,
          description='p95 ≤ 500 ms equivale a permitir hasta 5% de respuestas por encima de 500 ms. Presupuesto negativo = incumplimiento observado.')
    panel('Burn rate · SLO-01 · objetivo sostenible = 1',
          [(f'({ratio(bad(w), total(w))}) / 0.005', w) for w in ['5m', '1h', '6h']], width=24)
    panel('Cobertura de medición · muestras HTTP y minutos Ready',
          [(count, 'Respuestas del histograma HTTP'), (f'count_over_time({ready}[30d:1m])', '{{k8s_deployment_name}} · minutos observados')], width=24,
          description='Una ventana parcial no demuestra cumplimiento mensual. Retención máxima 30 días o 2 GB; revisar huecos y despliegues antes de certificar.')
    panels.append(dict(id=next_id(), type='text', title='Alcance de los SLO medidos', gridPos=dict(x=0,y=y,w=24,h=5),
                       options=dict(mode='markdown', content='SLO-01, SLO-02 y SLO-03 muestran medición y presupuesto con las fuentes actuales. **Los restantes SLO no se declaran cumplidos:** requieren denominadores de operaciones iniciadas, distinción de fallo técnico, deduplicación, entrega/DLQ, trazas completas y notificaciones enviadas. El p95 del consumidor no sustituye la duración completa del negocio. Consulte la matriz de cobertura versionada en `Documentacion/observabilidad/metricas-sistema.md`.')))
    y += 5

    panels.append(dict(id=next_id(), type='alertlist', title='Alertas del sistema',
                       gridPos=dict(x=0,y=y,w=24,h=6), options=dict(viewMode='list', groupMode='default',
                       maxItems=12, sortOrder=1, dashboardAlerts=False, alertName='',
                       alertInstanceLabelFilter='{team="bank-usac",environment="dev"}',
                       stateFilter=dict(firing=True,pending=True,error=True,noData=True,normal=True))))
    y += 6

    row('05 / Kubernetes · consumo real y HPA')
    selector = 'namespace="bank-usac-dev",environment="dev"'
    panel('CPU utilizada por pod', [(f'sum by(pod)(bank_pod_cpu_usage_cores{{{selector}}})', '{{pod}}')], 'cores', description='Consumo real informado por Metrics Server; no son requests ni límites.')
    panel('Memoria utilizada por pod', [(f'sum by(pod)(bank_pod_memory_usage_bytes{{{selector}}})', '{{pod}}')], 'bytes', x=12,
          description='Working set de Metrics Server. Muestras con antigüedad superior a 120 s se omiten.')
    panel('HPA · réplicas actuales y deseadas', [(f'{{__name__=~"k8s_hpa_(current|desired)_replicas",{INFRA}}}', '{{k8s_hpa_name}} · {{__name__}}')],
          description='Obligatorio: evidenciar autoescalado bajo carga. Es normal conservar una réplica si no se supera el objetivo de CPU; no se exige que cambie en cada operación ni se modifica el HPA para mover la gráfica.')
    panel('Número de pods observados por fase', [(f'count_values("fase",k8s_pod_phase{{{INFRA}}})', 'Fase {{fase}}')], x=12,
          description='Fases del receiver: 1 Pending, 2 Running, 3 Succeeded, 4 Failed, 5 Unknown. No equivale a Ready.')
    next_id()  # Retired source-health graph; source-health alerts are retained.

    row('06 / Errores en tiempo real y trazas distribuidas')
    logs = panel('Errores y advertencias del sistema', [], kind='logs', width=24, height=10, datasource={'type':'loki','uid':'loki'},
                 description='JSON original preservado. Expandir un log para consultar correlationId y abrir TraceID mediante el enlace de la fuente Loki.')
    logs['targets'] = [dict(refId='A', queryType='range', expr='{deployment_environment_name="dev",service_name=~"api-gateway|customer-service|account-service|transaction-service|payment-service|notification-audit-service"} | json | level=~"ERROR|WARN|WARNING"', maxLines=300)]
    logs['options'] = dict(showTime=True, showLabels=False, wrapLogMessage=False, sortOrder='Descending', enableLogDetails=True, dedupStrategy='none')
    traces = panel('Trazas del sistema · solicitudes de negocio', [], kind='table', width=24, height=10, datasource={'type':'tempo','uid':'tempo'})
    traces['targets'] = [dict(refId='A', queryType='traceql', query='{ resource.service.name = "api-gateway" && resource.deployment.environment.name = "dev" && kind = server && span.http.route =~ "/api/.*" }', tableType='traces', limit=30, spss=5)]

    # Show the actual compliance boundary; never replace a missing observation by a target.
    for item in panels:
        title = item['title']
        boundary = None
        if title.startswith(('Disponibilidad Gateway', 'SLO-02 · disponibilidad')):
            boundary = [('red', None), ('green', 99.5 if title.startswith('Disponibilidad') else 98.5)]
        elif 'presupuesto' in title.lower() and item.get('fieldConfig', {}).get('defaults', {}).get('unit') == 'percent':
            boundary = [('red', None), ('green', 0.000001)]
        elif title.startswith('SLO-03 · p95'):
            boundary = [('green', None), ('red', 0.500001)]
        if boundary:
            defaults = item['fieldConfig']['defaults']
            defaults['thresholds'] = dict(mode='absolute', steps=[dict(color=color, value=value) for color, value in boundary])
            if item['type'] == 'stat':
                defaults['color'] = dict(mode='thresholds')
            else:
                defaults['custom']['thresholdsStyle'] = dict(mode='line')

    return dict(uid='bank-system-metrics', title='Métricas sistema', schemaVersion=39, version=4, editable=False,
                timezone='browser', refresh='30s', time=dict(**{'from':'now-30m','to':'now'}), tags=['bank-usac','SLI','SLO','dev'],
                description='SLI base y SLO medibles con datos reales de dev. Sin muestras no se presume éxito. Despliegues nuevos se detectan por cambios de revisión Kubernetes.',
                links=[dict(title='Servicios y búsqueda de logs',type='link',url='/d/bank-usac-prueba1'),
                       dict(title='Alertas y runbooks',type='link',url='/alerting/list')],
                annotations=dict(list=[dict(name='Despliegues Kubernetes', datasource=PROM, enable=True, hide=False,
                     iconColor='rgba(109,179,185,1)', expr='changes(bank_deployment_revision{namespace="bank-usac-dev",environment="dev"}[1m]) > 0',
                     step='30s', titleFormat='Despliegue {{deployment}}', textFormat='Cambio de revisión observado en {{deployment}}', tagKeys='deployment', useValueForTime=False)]),
                panels=panels)


if __name__ == '__main__':
    path = ROOT / 'chart/dashboards/sistema.json'
    content = json.dumps(dashboard(), ensure_ascii=False, indent=2) + '\n'
    if '--check' in sys.argv:
        assert path.read_text(encoding='utf-8') == content, 'Regenerate sistema.json'
    else:
        path.write_text(content, encoding='utf-8')
