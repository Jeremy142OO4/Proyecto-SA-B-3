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

    def row(title):
        nonlocal y
        panels.append(dict(id=len(panels)+1, type='row', title=title, collapsed=False, panels=[],
                           gridPos=dict(x=0, y=y, w=24, h=1)))
        y += 1

    def panel(title, targets, unit='short', kind='timeseries', width=12, x=0, description='', height=8, datasource=PROM):
        nonlocal y
        defaults = dict(unit=unit, noValue='Sin datos', color=dict(mode='palette-classic'))
        if kind == 'timeseries':
            defaults['custom'] = dict(drawStyle='line', lineInterpolation='linear', lineWidth=2,
                                      fillOpacity=8, showPoints='auto', spanNulls=False)
        item = dict(id=len(panels)+1, title=title, description=description, type=kind,
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
          'percent', 'stat', 8, 0, 'Respuestas no 5xx; incluye 4xx. Excluye salud/readiness/métricas. Sin tráfico: sin datos. Ventana móvil con la historia disponible.', 5)
    panel('Error budget restante · SLO 99.5%',
          [(f'100 * (1 - ({ratio(bad("30d"), "0.005 * " + total("30d"))}))', 'Presupuesto restante')],
          'percent', 'stat', 8, 8, '100% = presupuesto intacto; 0% = agotado; puede ser negativo. Fallos permitidos = 0.005 × solicitudes observadas.', 5)
    panel('Recolección de métricas', [('up{job=~"bank-otel|bank-infra|bank-system"}', '{{job}}')],
          'short', 'stat', 8, 16, '1 = scrape disponible; no acredita disponibilidad de operaciones bancarias.', 5)
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
    den = f'sum(rate({transfer}[$__rate_interval]))'
    panel('Resultados de transferencias · éxito y fallo',
          [(f'100 * ({ratio(f"(sum(rate({transfer[:-1]},outcome=\"{outcome}\"}}[$__rate_interval])) or (0 * {den}))", den)})', label)
           for outcome, label in [('success', 'Completadas'), ('failure', 'Fallidas / compensadas')]], 'percent', x=12,
          description='Resultados de negocio terminales; un rechazo o una compensación no equivale necesariamente a error técnico ni incumplimiento del SLO.')
    panel('Throughput · eventos por segundo por servicio',
          [('sum by(service,action)(rate(bank_messages_total{environment="dev"}[$__rate_interval]))', '{{service}} · {{action}}')], 'ops',
          description='Intentos de publicación/procesamiento. Se separan por acción: no sumar productor y consumidor como eventos únicos.')
    panel('Throughput · eventos por segundo por routing key',
          [('sum by(service,event_type,action)(rate(bank_messages_total{environment="dev"}[$__rate_interval]))', '{{service}} · {{event_type}} · {{action}}')], 'ops', x=12,
          description='event_type es la routing key del catálogo permitido. Claves no reconocidas se agrupan en other para limitar cardinalidad.')
    panel('Throughput RabbitMQ · por cola',
          [(f'sum by(rabbitmq_queue_name)(rate(rabbitmq_message_{metric}_total{{job="bank-infra"}}[$__rate_interval]))', '{{rabbitmq_queue_name}} · ' + label)
           for metric, label in [('published', 'publicados'), ('delivered', 'entregados'), ('acknowledged', 'confirmados')]], 'ops', width=24,
          description='Contadores del management API del broker; los reintentos son entregas adicionales. Colas RPC temporales excluidas del almacenamiento.')

    row('03 / RabbitMQ · pendientes y consumidores')
    queue = 'rabbitmq_message_current{job="bank-infra",state="ready"}'
    consumers = 'rabbitmq_consumer_count{job="bank-infra"}'
    panel('Mensajes pendientes por cola · ready', [(queue, '{{rabbitmq_queue_name}}')], description='Backlog real de RabbitMQ, compartido por los consumidores de cada cola. No es lag por partición de Kafka.')
    panel('Mensajes entregados sin confirmar · unacked', [('rabbitmq_message_current{job="bank-infra",state="unacknowledged"}', '{{rabbitmq_queue_name}}')], x=12)
    panel('Consumidores activos por cola', [(consumers, '{{rabbitmq_queue_name}}')])
    panel('Pendientes por consumidor activo · promedio',
          [(f'{queue} / on(rabbitmq_queue_name,rabbitmq_vhost_name,rabbitmq_node_name) ({consumers} > 0)', '{{rabbitmq_queue_name}}')], x=12,
          description='ready / consumidores activos. Es un promedio por cola; RabbitMQ no asigna mensajes ready a un consumidor individual. Las colas sin consumidor se muestran en el panel siguiente.')
    blocked_queues = panel('Colas con pendientes y sin consumidores',
          [(f'(count(({queue} > 0) and on(rabbitmq_queue_name,rabbitmq_vhost_name,rabbitmq_node_name) ({consumers} == 0)) or (0 * count({queue}))) and on() (count({consumers}) > 0) and on() (up{{job="bank-infra"}} == 1)', 'Colas afectadas')],
          kind='stat', width=24, height=5,
          description='Número de colas con ready > 0 y cero consumidores. 0 = ninguna cola afectada entre las observadas. Sin datos = falta la métrica de colas/consumidores o falla la recolección; no se rellena una ausencia con cero.')
    blocked_queues['fieldConfig']['defaults'].update(
        decimals=0, min=0, color=dict(mode='thresholds'),
        thresholds=dict(mode='absolute', steps=[dict(color='green', value=None), dict(color='red', value=1)]))

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
    panels.append(dict(id=len(panels)+1, type='text', title='Alcance de los SLO medidos', gridPos=dict(x=0,y=y,w=24,h=5),
                       options=dict(mode='markdown', content='SLO-01, SLO-02 y SLO-03 muestran medición y presupuesto con las fuentes actuales. **Los restantes SLO no se declaran cumplidos:** requieren denominadores de operaciones iniciadas, distinción de fallo técnico, deduplicación, entrega/DLQ, trazas completas y notificaciones enviadas. El p95 del consumidor no sustituye la duración completa del negocio. Consulte la matriz de cobertura versionada en `Documentacion/observabilidad/metricas-sistema.md`.')))
    y += 5

    panels.append(dict(id=len(panels)+1, type='alertlist', title='Alertas del sistema',
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
    panel('HPA · réplicas actuales y deseadas', [(f'{{__name__=~"k8s_hpa_(current|desired)_replicas",{INFRA}}}', '{{k8s_hpa_name}} · {{__name__}}')])
    panel('Número de pods observados por fase', [(f'count_values("fase",k8s_pod_phase{{{INFRA}}})', 'Fase {{fase}}')], x=12,
          description='Fases del receiver: 1 Pending, 2 Running, 3 Succeeded, 4 Failed, 5 Unknown. No equivale a Ready.')
    panel('Fuentes de uso de recursos y revisiones', [(f'bank_system_source_up{{{selector}}}', '{{source}}')], width=24,
          description='1 disponible, 0 error de API/permisos. Si resources=0 revisar Metrics Server y el permiso list sobre metrics.k8s.io/pods en el namespace.')

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

    return dict(uid='bank-system-metrics', title='Métricas sistema', schemaVersion=39, version=2, editable=False,
                timezone='browser', refresh='30s', time=dict(**{'from':'now-6h','to':'now'}), tags=['bank-usac','SLI','SLO','dev'],
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
