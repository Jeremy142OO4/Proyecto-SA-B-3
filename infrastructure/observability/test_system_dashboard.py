"""Contract checks for new telemetry: units, failure visibility and SLO guardrails."""
import importlib.util
import json
from pathlib import Path
import unittest
import system_dashboard

ROOT = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location('system_exporter', ROOT / 'chart/config/system_exporter.py')
exporter = importlib.util.module_from_spec(spec)
spec.loader.exec_module(exporter)


class SystemTests(unittest.TestCase):
    def test_metrics_api_quantities_and_staleness(self):
        data = {'items': [{'metadata': {'name': 'account-1'}, 'timestamp': '2026-10-21T12:00:00Z',
                          'containers': [{'name': 'account', 'usage': {'cpu': '250000000n', 'memory': '64Mi'}}]}]}
        from datetime import datetime
        now = datetime.fromisoformat('2026-10-21T12:00:30+00:00').timestamp()
        body = exporter.resource_samples(data, 'bank-usac-dev', 'dev', now)
        self.assertIn('bank_pod_cpu_usage_cores', body)
        self.assertIn(' 0.25\n', body)
        self.assertIn(' 67108864.0\n', body)
        self.assertEqual(exporter.resource_samples(data, 'bank-usac-dev', 'dev', now + 121), '')

    def test_partial_failure_drops_usage_and_keeps_revision(self):
        def read(path):
            if 'metrics.k8s.io' in path:
                raise PermissionError('DO-NOT-EXPOSE-TOKEN')
            return {'items': [{'metadata': {'name': 'api-gateway', 'annotations': {'deployment.kubernetes.io/revision': '42'}}}]}
        body = exporter.collect('bank-usac-dev', 'dev', read=read).decode()
        self.assertIn('source="resources"} 0.0', body)
        self.assertIn('source="deployments"} 1.0', body)
        self.assertIn('deployment="api-gateway"} 42.0', body)
        self.assertNotIn('bank_pod_cpu', body)
        self.assertNotIn('DO-NOT-EXPOSE', body)

    def test_approved_objectives_and_absence(self):
        dashboard = system_dashboard.dashboard()
        self.assertEqual(dashboard['title'], 'Métricas sistema')
        ids = [p['id'] for p in dashboard['panels']]
        self.assertEqual(len(ids), len(set(ids)))
        text = json.dumps(dashboard, ensure_ascii=False)
        self.assertNotIn('avg(bank_transfer_flow', text)
        self.assertIn('histogram_quantile(0.99', text)
        self.assertIn('0.005', text)
        self.assertIn('0.015', text)
        self.assertIn('0.05', text)
        self.assertIn('denominadores', text)
        self.assertNotIn('or vector(100)', text)
        self.assertIn('bank_deployment_revision', text)

    def test_provisioning_and_namespace_only_permissions(self):
        config = (ROOT/'chart/templates/configmaps.yaml').read_text()
        workloads = (ROOT/'chart/templates/workloads.yaml').read_text()
        rbac = (ROOT/'chart/templates/rbac.yaml').read_text()
        self.assertIn('dashboards/sistema.json', config)
        self.assertIn('checksum/sistema:', workloads)
        self.assertIn('subPath: sistema.json', workloads)
        self.assertIn('metrics.k8s.io', rbac)
        self.assertNotIn('ClusterRole', rbac)
        self.assertNotIn('secrets', rbac.lower())

    def test_evaluation_cleanup_preserves_mandatory_observability(self):
        dashboard = system_dashboard.dashboard()
        panels = {p['title']: p for p in dashboard['panels']}
        for retired in ('Recolección de métricas', 'Mensajes pendientes por cola · pico de ready',
                        'Mensajes sin confirmar · pico de unacked', 'Consumidores activos por cola · máximo observado',
                        'Colas con pendientes y sin consumidores', 'Fuentes de uso de recursos y revisiones'):
            self.assertNotIn(retired, panels)
        for mandatory in ('CPU utilizada por pod', 'Memoria utilizada por pod',
                          'HPA · réplicas actuales y deseadas', 'Número de pods observados por fase',
                          'Pendientes por consumidor activo · pico del promedio',
                          'Latencia de transferencias · p50, p95 y p99', 'Errores HTTP del Gateway · 5xx',
                          'Disponibilidad Gateway · 30 días · objetivo 99.5%',
                          'Error budget restante · SLO 99.5%', 'SLO-03 · p95 HTTP · objetivo ≤ 500 ms',
                          'Errores y advertencias del sistema', 'Trazas del sistema · solicitudes de negocio',
                          'Throughput · eventos por segundo por servicio',
                          'Throughput · eventos por segundo por routing key', 'Throughput RabbitMQ · por cola'):
            self.assertIn(mandatory, panels)
        self.assertTrue(dashboard['annotations']['list'][0]['enable'])
        self.assertEqual(dashboard['time']['from'], 'now-30m')
        # Existing panel links must still identify the same observation.
        for title, identifier in [('CPU utilizada por pod', 31), ('Memoria utilizada por pod', 32),
                                  ('HPA · réplicas actuales y deseadas', 33),
                                  ('Pendientes por consumidor activo · pico del promedio', 18),
                                  ('SLO-03 · p95 HTTP · objetivo ≤ 500 ms', 24)]:
            self.assertEqual(panels[title]['id'], identifier)
        for i, p in enumerate(dashboard['panels']):
            if p['type'] == 'row':
                continue
            a = p['gridPos']
            for other in dashboard['panels'][i+1:]:
                if other['type'] == 'row':
                    continue
                b = other['gridPos']
                self.assertFalse(a['x'] < b['x']+b['w'] and b['x'] < a['x']+a['w'] and
                                 a['y'] < b['y']+b['h'] and b['y'] < a['y']+a['h'],
                                 f'Overlapping panels: {p["title"]} / {other["title"]}')

    def test_normal_flow_uses_real_counts_not_constant_success_percentage(self):
        panel = next(p for p in system_dashboard.dashboard()['panels'] if p['id'] == 10)
        self.assertEqual(panel['type'], 'stat')
        self.assertEqual(panel['fieldConfig']['defaults']['unit'], 'short')
        self.assertTrue(all('increase(' in t['expr'] and '[$__range]' in t['expr'] for t in panel['targets']))
        self.assertNotIn('or vector(0)', json.dumps(panel))


if __name__ == '__main__':
    unittest.main()
