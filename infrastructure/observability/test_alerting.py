"""Regression checks for alert contracts and secret handling."""
import base64
import json
import os
import unittest
from unittest.mock import patch
import alerting
import smtp_secret


class AlertingTests(unittest.TestCase):
    def test_dashboard_query_budget(self):
        panels = alerting.dashboard()['panels']
        cards = [panel for panel in panels if panel['type'] == 'stat']
        self.assertEqual(len(cards), 6)
        for panel in panels:
            if 'targets' not in panel:
                continue
            self.assertEqual(panel['maxDataPoints'], 240)
            for target in panel['targets']:
                self.assertEqual(target['instant'], panel['type'] == 'stat')
                self.assertEqual(target['range'], panel['type'] != 'stat')

    def test_slo_and_guardrails(self):
        rules = alerting.provisioning()['groups'][0]['rules']
        self.assertGreaterEqual(len(rules), 4)
        self.assertEqual(len({r['uid'] for r in rules}), len(rules))
        for rule in rules:
            self.assertTrue(rule['annotations']['runbook_url'].endswith(rule['uid']))
            self.assertEqual(rule['execErrState'], 'Error')
        fast, slow = rules[:2]
        for rule, windows, threshold in [(fast, ['1h', '5m'], '14.4'), (slow, ['6h', '30m'], '6')]:
            expression = rule['data'][0]['model']['expr']
            self.assertIn('/ 0.005', expression)
            self.assertIn('>= 20', expression)
            self.assertIn('> ' + threshold, expression)
            self.assertTrue(all('[' + window + ']' in expression for window in windows))
            self.assertNotIn('status_class="4xx"', expression)
            self.assertIn('route!~', expression)

    def test_notifications(self):
        config = alerting.provisioning()
        receiver = config['contactPoints'][0]['receivers'][0]
        self.assertEqual(receiver['settings']['addresses'], '3141193530501@ingenieria.usac.edu.gt')
        self.assertFalse(receiver['disableResolveMessage'])
        self.assertEqual(config['policies'][0]['group_wait'], '30s')

    def test_secret_stays_in_stdin(self):
        data = {key: base64.b64encode(value.encode()).decode() for key, value in {
            'SMTP_USERNAME': 'test', 'SMTP_APP_PASSWORD': 'test-only-secret', 'SMTP_FROM': 'Bank USAC <test@example.invalid>',
            'SMTP_HOST': 'smtp.example.invalid', 'SMTP_PORT': '587',
            'RABBITMQ_USUARIO': 'observer', 'RABBITMQ_CLAVE': 'test-broker-secret'}.items()}
        with patch.dict(os.environ, {'KUBE_NAMESPACE': 'bank-usac-dev', 'OBSERVABILITY_NAMESPACE': 'bank-usac-observability'}), \
             patch('subprocess.check_output', return_value=json.dumps({'data': data}).encode()), \
             patch('subprocess.run') as run:
            smtp_secret.main()
        self.assertEqual(run.call_count, 2)
        for call in run.call_args_list:
            self.assertIn('--server-side', call.args[0])
            self.assertNotIn('test-only-secret', str(call.args))
            self.assertEqual(json.loads(call.kwargs['input'])['metadata']['namespace'], 'bank-usac-observability')
        copied = json.loads(run.call_args_list[0].kwargs['input'])['data']
        self.assertEqual(base64.b64decode(copied['FROM_ADDRESS']).decode(), 'test@example.invalid')

    def test_production_secret_copy_forbidden(self):
        with patch.dict(os.environ, {'KUBE_NAMESPACE': 'bank-usac'}), patch('subprocess.check_output') as read:
            with self.assertRaises(SystemExit):
                smtp_secret.main()
            read.assert_not_called()


if __name__ == '__main__':
    unittest.main()
