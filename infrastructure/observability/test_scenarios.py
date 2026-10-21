"""The scenario dashboard is additive and never uses unscoped bank telemetry."""
import json
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parent / 'chart'


class ScenarioDashboardTests(unittest.TestCase):
    def setUp(self):
        self.dashboard = json.loads((ROOT / 'dashboards/scenarios.json').read_text(encoding='utf-8'))

    def test_separate_uid_and_fixed_environment(self):
        previous = [json.loads((ROOT / 'dashboards' / file).read_text(encoding='utf-8'))['uid']
                    for file in ('overview.json', 'slo.json', 'prueba1.json')]
        self.assertNotIn(self.dashboard['uid'], previous)
        ids = [p['id'] for p in self.dashboard['panels']]
        self.assertEqual(len(ids), len(set(ids)))
        for panel in self.dashboard['panels']:
            for target in panel.get('targets', []):
                query = target.get('expr') or target['query']
                self.assertIn('interbank-cloud-tests', query)
                self.assertNotIn('bank_interbank_transitions_total', query)
                self.assertNotIn(':raw}', query)
                if panel['datasource']['uid'] in ('loki', 'tempo'):
                    self.assertIn('${depositId:regex}', query)

    def test_provisioned_without_replacing_previous_mounts(self):
        config = (ROOT / 'templates/configmaps.yaml').read_text()
        workloads = (ROOT / 'templates/workloads.yaml').read_text()
        for file in ('overview.json', 'slo.json', 'prueba1.json', 'scenarios.json'):
            self.assertIn('dashboards/' + file, config)
            self.assertIn('subPath: ' + file, workloads)
        self.assertIn('checksum/scenarios:', workloads)

    def test_known_limitations_are_visible(self):
        text = json.dumps(self.dashboard, ensure_ascii=False)
        self.assertIn('no son métricas contables ni depósitos únicos', text)
        self.assertIn('no se estiman con logs', text)
        self.assertIn('3 minutos', text)


if __name__ == '__main__':
    unittest.main()
