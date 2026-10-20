"""Deterministic Grafana provisioning; no credentials or invented metric sources."""
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parent
GATEWAY = 'bank_http_requests_total{service="api-gateway",environment="dev",route!~"/(health(/.*)?|healthz|salud|ready|readyz|metrics)"}'


def requests(window):
    return f'sum(increase({GATEWAY}[{window}]))'


def errors(window):
    selector = GATEWAY[:-1] + ',status_class="5xx"}'
    return f'(sum(increase({selector}[{window}])) or vector(0))'


def burn(window):
    return f'({errors(window)} / {requests(window)} / 0.005)'


def rule(uid, title, expression, severity, duration):
    return {
        'uid': uid, 'title': title, 'condition': 'A', 'for': duration,
        'noDataState': 'OK', 'execErrState': 'Error',
        'annotations': {
            'summary': title,
            'description': 'Revisar el runbook versionado antes de intervenir. No reiniciar ni compensar transferencias automáticamente.',
            'runbook_url': 'https://github.com/Jeremy142OO4/Proyecto-SA-B-3/blob/develop/Documentacion/observabilidad/alertas.md#' + uid,
        },
        'labels': {'severity': severity, 'environment': 'dev', 'team': 'bank-usac'},
        'data': [{
            'refId': 'A', 'relativeTimeRange': {'from': 600, 'to': 0},
            'datasourceUid': 'prometheus',
            'model': {'refId': 'A', 'expr': expression, 'instant': True,
                      'range': False, 'intervalMs': 15000, 'maxDataPoints': 43200},
        }],
    }


def provisioning():
    rules = [
        rule('bank-burn-fast', 'SLO 99.5%: consumo rápido del presupuesto',
             f'({burn("1h")} > 14.4) and ({burn("5m")} > 14.4) and ({requests("5m")} >= 20)', 'critical', '2m'),
        rule('bank-burn-slow', 'SLO 99.5%: consumo sostenido del presupuesto',
             f'({burn("6h")} > 6) and ({burn("30m")} > 6) and ({requests("30m")} >= 20)', 'warning', '5m'),
        rule('bank-http-latency', 'Gateway: p95 HTTP superior a 500 ms',
             '(histogram_quantile(0.95, sum by (le) (rate(bank_http_duration_seconds_bucket{service="api-gateway",environment="dev",route!~"/(health(/.*)?|healthz|salud|ready|readyz|metrics)"}[5m]))) > 0.5) and (' + requests('5m') + ' >= 20)', 'warning', '5m'),
        rule('bank-collector-down', 'Collector: Prometheus no puede recolectar métricas',
             '(up{job="bank-otel"} == 0) or absent(up{job="bank-otel"})', 'critical', '2m'),
        rule('bank-infra-telemetry', 'Infraestructura: sin métricas de despliegues',
             'absent_over_time(k8s_deployment_desired{job="bank-infra"}[5m])', 'warning', '2m'),
        rule('bank-deployment-unavailable', 'Kubernetes: despliegue sin réplicas disponibles',
             '(k8s_deployment_available{job="bank-infra"} == 0) and on(k8s_namespace_name,k8s_deployment_name) (k8s_deployment_desired{job="bank-infra"} > 0)', 'critical', '3m'),
        rule('bank-rabbitmq-backlog', 'RabbitMQ: cola con mensajes pendientes sostenidos',
             'rabbitmq_message_current{job="bank-infra",state="ready"} > 100', 'warning', '5m'),
        rule('bank-rabbitmq-telemetry', 'RabbitMQ: sin métricas de colas',
             'absent_over_time(rabbitmq_message_current{job="bank-infra"}[5m])', 'warning', '2m'),
    ]
    return {
        'apiVersion': 1,
        'groups': [{'orgId': 1, 'name': 'Bank USAC SLO', 'folder': 'Bank USAC', 'interval': '1m', 'rules': rules}],
        'contactPoints': [{'orgId': 1, 'name': 'equipo-tecnico', 'receivers': [{
            'uid': 'bank-email', 'type': 'email', 'disableResolveMessage': False,
            'settings': {'addresses': '3141193530501@ingenieria.usac.edu.gt', 'singleEmail': True},
        }]}],
        'policies': [{'orgId': 1, 'receiver': 'equipo-tecnico',
                      'group_by': ['grafana_folder', 'alertname', 'environment'],
                      'group_wait': '30s', 'group_interval': '5m', 'repeat_interval': '4h',
                      'routes': [{'receiver': 'equipo-tecnico', 'object_matchers': [['severity', '=', 'critical']], 'repeat_interval': '1h'}]}],
    }


def dashboard():
    panels = []
    specs = [
        ('Disponibilidad HTTP observada (30 días); objetivo 99.5%', f'100 * (1 - {errors("30d")} / {requests("30d")})', 'percent'),
        ('Presupuesto restante (solicitudes; puede ser negativo)', f'0.005 * {requests("30d")} - {errors("30d")}', 'short'),
        ('Solicitudes elegibles observadas (30 días)', requests('30d'), 'short'),
        ('Respuestas 5xx observadas (30 días)', f'{errors("30d")} and ({requests("30d")} > 0)', 'short'),
        ('Burn rate 1h (1 = consumo sostenible)', burn('1h'), 'short'),
        ('Historia TSDB disponible (no acredita cobertura completa del Gateway)', 'time() - prometheus_tsdb_lowest_timestamp_seconds', 's'),
        ('HTTP p95 por servicio', 'histogram_quantile(0.95, sum by (service, le) (rate(bank_http_duration_seconds_bucket{environment="dev"}[5m])))', 's'),
        ('Gateway: p95 observado en 30 días (objetivo 500 ms)', 'histogram_quantile(0.95, sum by(le) (increase(bank_http_duration_seconds_bucket{service="api-gateway",environment="dev"}[30d])))', 's'),
        ('Procesamiento RabbitMQ p95 por servicio', 'histogram_quantile(0.95, sum by (service, le) (rate(bank_message_processing_seconds_bucket{environment="dev"}[5m])))', 's'),
        ('Clientes KYC por estado', 'bank_kyc_customers', 'short'),
        ('KYC: decisiones por resultado', 'sum by (outcome) (increase(bank_kyc_decisions_total[$__range]))', 'short'),
        ('KYC: creación hasta verificación p95', 'histogram_quantile(0.95, sum by(le) (rate(bank_kyc_duration_seconds_bucket[1h])))', 's'),
        ('Cuentas: operaciones y anomalías detectadas', 'sum by(__name__,account_type,direction) (increase({__name__=~"bank_account_(movements|anomalies)_total"}[$__range]))', 'short'),
        ('Transferencias internas: resultados terminales', 'sum by(outcome) (increase(bank_transfer_results_total[$__range]))', 'short'),
        ('Saga interna: duración p95 hasta resultado terminal', 'histogram_quantile(0.95, sum by(le,outcome) (rate(bank_transfer_flow_duration_seconds_bucket[1h])))', 's'),
        ('Pagos: resultados del proveedor configurable', 'sum by(outcome) (increase(bank_payment_external_results_total[$__range]))', 'short'),
        ('Transiciones interbancarias (no confundir con éxito definitivo)', 'sum by (direction, status) (increase(bank_interbank_transitions_total[$__range]))', 'short'),
        ('Despliegues: réplicas disponibles', 'k8s_deployment_available{job="bank-infra"}', 'short'),
        ('Disponibilidad Ready observada por despliegue (30 días; objetivo 98.5%)', '100 * avg_over_time((k8s_deployment_available{job="bank-infra"} > bool 0)[30d:1m])', 'percent'),
        ('HPA: réplicas actuales y deseadas', '{__name__=~"k8s_hpa_(current|desired)_replicas",job="bank-infra"}', 'short'),
        ('RabbitMQ: mensajes por cola y estado', 'rabbitmq_message_current{job="bank-infra"}', 'short'),
        ('RabbitMQ: consumidores', 'rabbitmq_consumer_count{job="bank-infra"}', 'short'),
    ]
    panels.append({'id': 100, 'type': 'text', 'title': 'BANK03 / Centro de operaciones',
                   'gridPos': {'x': 0, 'y': 0, 'w': 24, 'h': 3},
                   'options': {'mode': 'markdown', 'content': '### Salud del banco · Desarrollo\nSLO **99.5%** · Métricas reales · Actualización cada **30 s**\n\nLa ventana SLO es de 30 días, pero la historia disponible puede ser menor. **Sin datos ≠ cero ni servicio saludable.**'}})
    short_titles = ['Disponibilidad HTTP · 30 días', 'Presupuesto disponible', 'Solicitudes observadas',
                    'Errores del servidor · 5xx', 'Consumo de presupuesto · 1 h', 'Historia disponible']
    legends = {6: '{{service}}', 7: 'Gateway', 8: '{{service}}', 9: '{{status}}',
               10: '{{outcome}}', 11: 'Verificación KYC', 12: '{{account_type}} · {{direction}}',
               13: '{{outcome}}', 14: '{{outcome}}', 15: '{{outcome}}', 16: '{{direction}} · {{status}}',
               17: '{{k8s_deployment_name}}', 18: '{{k8s_deployment_name}}',
               19: '{{k8s_hpa_name}} · {{__name__}}', 20: '{{rabbitmq_queue_name}} · {{state}}',
               21: '{{rabbitmq_queue_name}}'}
    sections = {6: '01 / Rendimiento y experiencia', 9: '02 / Clientes y verificación KYC',
                12: '03 / Operaciones, Saga e integración', 17: '04 / Kubernetes y escalado',
                20: '05 / RabbitMQ y consumidores'}
    y = 3
    for index, (title, expression, unit) in enumerate(specs):
        card = index < 6
        if index in sections:
            panels.append({'id': 100 + index, 'type': 'row', 'title': sections[index],
                           'collapsed': False, 'panels': [], 'gridPos': {'x': 0, 'y': y, 'w': 24, 'h': 1}})
            y += 1
        position = index if card else index - max(start for start in sections if start <= index)
        width, height, columns = (4, 5, 6) if card else (12, 8, 2)
        defaults = {'unit': unit, 'noValue': 'Sin datos', 'color': {'mode': 'palette-classic'}}
        options = {'legend': {'displayMode': 'table', 'placement': 'bottom', 'calcs': ['lastNotNull']},
                   'tooltip': {'mode': 'multi', 'sort': 'desc'}}
        if card:
            defaults.update({'decimals': 3 if index == 0 else (2 if index == 4 else 0),
                             'color': {'mode': 'thresholds'}, 'thresholds': {'mode': 'absolute', 'steps': [{'color': 'blue', 'value': None}]}})
            thresholds = {0: [('red', None), ('green', 99.5)], 1: [('red', None), ('green', 0)],
                          3: [('green', None), ('orange', 1)], 4: [('green', None), ('orange', 1), ('red', 14.4)]}
            if index in thresholds:
                defaults['thresholds']['steps'] = [{'color': color, 'value': value} for color, value in thresholds[index]]
            options = {'reduceOptions': {'values': False, 'calcs': ['lastNotNull'], 'fields': ''},
                       'orientation': 'horizontal', 'textMode': 'value', 'colorMode': 'value',
                       'graphMode': 'none', 'justifyMode': 'center'}
        else:
            defaults['custom'] = {'drawStyle': 'line', 'lineInterpolation': 'smooth', 'lineWidth': 2,
                                  'fillOpacity': 12, 'showPoints': 'never', 'spanNulls': False,
                                  'axisBorderShow': False, 'axisCenteredZero': False}
        panels.append({'id': index + 1, 'title': short_titles[index] if card else title,
                       'description': title + '. Los valores se calculan sobre las muestras disponibles; no se rellenan ausencias con cero.',
                       'type': 'stat' if card else 'timeseries',
                       'datasource': {'type': 'prometheus', 'uid': 'prometheus'},
                       'gridPos': {'x': (position % columns) * width, 'y': y, 'w': width, 'h': height},
                       'maxDataPoints': 240, 'interval': '1m',
                       'targets': [{'refId': 'A', 'expr': expression, 'legendFormat': legends.get(index, 'Observado'),
                                    'instant': card, 'range': not card}],
                       'fieldConfig': {'defaults': defaults, 'overrides': []}, 'options': options})
        if position % columns == columns - 1 or index == len(specs) - 1 or index + 1 in sections:
            y += height
    return {'uid': 'bank-slo', 'title': 'Bank USAC — SLO 99.5% y servicios', 'schemaVersion': 39,
            'version': 2, 'refresh': '30s', 'timezone': 'browser', 'time': {'from': 'now-6h', 'to': 'now'},
            'tags': ['BANK03', 'SLO', 'operaciones', 'dev'],
            'links': [{'title': 'Consultar métricas, logs y trazas', 'url': '/explore', 'type': 'link'},
                      {'title': 'Reglas de alerta', 'url': '/alerting/list', 'type': 'link'}],
            'description': 'Datos parciales no certifican 30 días. Sin tráfico: sin datos, nunca 100%. Retención limitada también por 2 GB. Alertas de ausencia no prueban una caída del servicio.',
            'panels': panels}


if __name__ == '__main__':
    import sys
    outputs = {'alerting/bank.json': provisioning(), 'dashboards/slo.json': dashboard()}
    for name, value in outputs.items():
        content = json.dumps(value, ensure_ascii=False, indent=2) + '\n'
        path = ROOT / 'chart' / name
        if '--check' in sys.argv:
            assert path.read_text(encoding='utf-8') == content, f'Regenerate {name}'
        else:
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(content, encoding='utf-8')
