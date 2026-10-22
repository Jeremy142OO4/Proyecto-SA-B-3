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


if __name__ == '__main__':
    unittest.main()
