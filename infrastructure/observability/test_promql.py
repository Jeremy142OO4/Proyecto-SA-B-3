"""Execute the actual alert expressions against synthetic series with promtool."""
import json
from pathlib import Path
import subprocess
import tempfile
import alerting


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
