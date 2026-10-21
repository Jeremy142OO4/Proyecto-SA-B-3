from pathlib import Path
import unittest
import yaml


class SandboxIsolationTests(unittest.TestCase):
    def test_compose_has_no_cloud_dependencies_or_public_data_ports(self):
        root = Path(__file__).resolve().parents[2]
        config = yaml.safe_load((root/'infrastructure/interbank-sandbox/compose.yaml').read_text())
        self.assertTrue(config['networks']['sandbox']['internal'])
        self.assertEqual(config['name'], 'bank-usac-interbank-sandbox')
        services = config['services']
        self.assertEqual(services['mock']['network_mode'], 'service:gateway')
        self.assertEqual(services['transactions']['network_mode'], 'service:gateway')
        for name, service in services.items():
            if name != 'frontend':
                self.assertNotIn('ports', service)
            self.assertNotIn('external', service)
            for value in service.get('environment', {}).values():
                self.assertNotIn('pitagoricos.tech', str(value))
                self.assertNotIn('run.app', str(value))
        self.assertEqual(services['frontend']['ports'], ['127.0.0.1:8180:80'])
        gateway = services['gateway']['environment']
        self.assertEqual(gateway['ENTORNO'], 'interbank-sandbox')
        self.assertEqual(gateway['INTERBANK_PARTNER_BASE_URL'], gateway['INTERBANK_MOCK_CONTROL_URL']+'/interbank/v1')
        self.assertNotEqual(gateway['MOCK_CONTROL_KEY'], gateway['INTERBANK_PARTNER_API_KEY'])
