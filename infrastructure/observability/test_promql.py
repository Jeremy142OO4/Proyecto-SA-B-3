"""Execute the actual alert expressions against synthetic series with promtool."""
import json
from pathlib import Path
import subprocess
import tempfile
import alerting
import system_dashboard


def rabbitmq_dashboard_cases():
    panel = next(p for p in system_dashboard.dashboard()['panels']
                 if p['title'] == 'Pendientes por consumidor activo · pico del promedio')
    expression = 'sum(' + panel['targets'][0]['expr'].replace('$__interval', '5m') + ')'
    attributes = 'job="bank-infra",rabbitmq_queue_name="cuentas.comandos",rabbitmq_vhost_name="/",rabbitmq_node_name="rabbit@test"'
    cases = []
    for name, ready, consumers, expected in [
        ('empty queue with consumer', 0, 1, 0),
        ('pending with one consumer', 5, 1, 5),
        ('average across two consumers', 5, 2, 2.5),
        ('pending without consumers is undefined, not zero', 5, 0, None),
        ('empty DLQ without consumers is undefined', 0, 0, None),
        ('missing broker telemetry', None, None, None),
        ('missing consumer telemetry', 5, None, None),
    ]:
        inputs = []
        if ready is not None:
            inputs.append({'series': 'rabbitmq_message_current{' + attributes + ',state="ready"}', 'values': str(ready) + '+0x10'})
        if consumers is not None:
            inputs.append({'series': 'rabbitmq_consumer_count{' + attributes + '}', 'values': str(consumers) + '+0x10'})
        cases.append({'name': 'RabbitMQ dashboard: ' + name, 'interval': '1m', 'input_series': inputs,
                      'promql_expr_test': [{'expr': expression, 'eval_time': '5m',
                                           'exp_samples': [] if expected is None else [{'labels': '{}', 'value': expected}]}]})
    # A graph evaluated every five minutes must preserve a collected one-minute spike.
    inputs = [
        {'series': 'rabbitmq_message_current{' + attributes + ',state="ready"}', 'values': '0 0 12 0 0 0'},
        {'series': 'rabbitmq_consumer_count{' + attributes + '}', 'values': '1+0x10'},
    ]
    cases.append({'name': 'RabbitMQ consumer lag retains brief peak', 'interval': '1m', 'input_series': inputs,
                  'promql_expr_test': [{'expr': expression, 'eval_time': '5m', 'exp_samples': [{'labels': '{}', 'value': 12}]}]})
    # A normal successful transfer must be visible even if the success rate stays 100%.
    results = next(p for p in system_dashboard.dashboard()['panels'] if p['id'] == 10)
    success = results['targets'][0]['expr'].replace('$__range', '5m')
    failure = results['targets'][1]['expr'].replace('$__range', '5m')
    counter = 'bank_transfer_results_total{service="transaction-service",environment="dev",outcome="success"}'
    for name, series, expected in [('three completed transfers', [{'series': counter, 'values': '0 0 1 2 3 3'}], 3),
                                    ('missing counters', [], None)]:
        cases.append({'name': name, 'interval': '1m', 'input_series': series,
                      'promql_expr_test': [
                          {'expr': success, 'eval_time': '5m', 'exp_samples': [] if expected is None else [{'labels':'{}','value':expected}]},
                          {'expr': failure, 'eval_time': '5m', 'exp_samples': [] if expected is None else [{'labels':'{}','value':0}]},
                      ]})
    return cases


def main():
    rules = alerting.provisioning()['groups'][0]['rules']
    expressions = {rule['uid']: rule['data'][0]['model']['expr'] for rule in rules}
    labels = 'service="api-gateway",environment="dev",route="/api/cuentas"'

    def series(status, values, route=None):
        attributes = labels if route is None else labels.replace('/api/cuentas', route)
        return {'series': 'bank_http_requests_total{' + attributes + ',status_class="' + status + '"}', 'values': values}

    cases = []
    def case(name, inputs, expected):
        cases.append({'name': name, 'interval': '1m', 'input_series': inputs,
                      'promql_expr_test': [{'expr': f'count({expressions[uid]}) or vector(0)', 'eval_time': '6h',
                                            'exp_samples': [{'labels': '{}', 'value': value}]} for uid, value in expected.items()]})

    burns = ['bank-burn-fast', 'bank-burn-slow']
    case('healthy without any 5xx series', [series('2xx', '0+100x400')], {uid: 0 for uid in burns})
    case('sustained 10 percent failures', [series('2xx', '0+90x400'), series('5xx', '0+10x400')], {uid: 1 for uid in burns})
    case('4xx are valid business responses', [series('4xx', '0+100x400')], {uid: 0 for uid in burns})
    case('health errors excluded', [series('2xx', '0+100x400'), series('5xx', '0+10000x400', '/health')], {uid: 0 for uid in burns})
    case('no traffic is not an availability alarm', [], {uid: 0 for uid in burns})
    case('zero counters do not report a burn', [series('2xx', '0+0x400')], {uid: 0 for uid in burns})
    case('low traffic guarded', [series('5xx', '0+0.01x400')], {uid: 0 for uid in burns})
    case('counter reset still detects errors', [series('2xx', '0+90x180 0+90x220'), series('5xx', '0+10x180 0+10x220')], {uid: 1 for uid in burns})
    case('missing collector', [], {'bank-collector-down': 1})
    case('healthy collector', [{'series': 'up{job="bank-otel"}', 'values': '1+0x400'}], {'bank-collector-down': 0})
    case('stopped collector', [{'series': 'up{job="bank-otel"}', 'values': '0+0x400'}], {'bank-collector-down': 1})
    case('missing infrastructure observation', [], {'bank-infra-telemetry': 1, 'bank-rabbitmq-telemetry': 1})
    deploy_labels = 'job="bank-infra",k8s_namespace_name="bank-usac-dev",k8s_deployment_name="account-service"'
    case('intentionally scaled to zero', [{'series': name + '{' + deploy_labels + '}', 'values': '0+0x400'}
                                        for name in ['k8s_deployment_available', 'k8s_deployment_desired']], {'bank-deployment-unavailable': 0})
    case('unavailable deployment', [{'series': 'k8s_deployment_available{' + deploy_labels + '}', 'values': '0+0x400'},
                                    {'series': 'k8s_deployment_desired{' + deploy_labels + '}', 'values': '1+0x400'}], {'bank-deployment-unavailable': 1})
    for backlog, expected in [(0, 0), (101, 1)]:
        case('backlog ' + str(backlog), [{'series': 'rabbitmq_message_current{job="bank-infra",state="ready",rabbitmq_queue_name="cuentas"}',
                                        'values': str(backlog) + '+0x400'}], {'bank-rabbitmq-backlog': expected})

    cases.extend(rabbitmq_dashboard_cases())
    with tempfile.TemporaryDirectory(prefix='bank-promql-') as temp:
        path = Path(temp)
        path.chmod(0o755)
        # YAML accepts JSON; the exact Grafana queries, not a second implementation.
        (path / 'rules.json').write_text(json.dumps({'groups': [{'name': 'bank-test', 'rules': [
            {'alert': rule['uid'], 'expr': expressions[rule['uid']], 'for': rule['for']} for rule in rules]}]}), encoding='utf-8')
        (path / 'tests.json').write_text(json.dumps({'rule_files': ['rules.json'], 'evaluation_interval': '1m', 'tests': cases}), encoding='utf-8')
        subprocess.run(['docker', 'run', '--rm', '-v', f'{path}:/tests:ro,Z', '-w', '/tests', '--entrypoint', '/bin/promtool',
                        'prom/prometheus:v3.15.0', 'test', 'rules', '/tests/tests.json'], check=True)


if __name__ == '__main__':
    main()
