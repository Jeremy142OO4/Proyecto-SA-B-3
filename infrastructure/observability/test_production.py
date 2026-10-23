"""Promotion regressions without cloud credentials or banking operations."""
import json
import os
from pathlib import Path
import unittest
from unittest.mock import patch
import yaml
import smtp_secret

ROOT = Path(__file__).resolve().parents[2]


class ProductionTests(unittest.TestCase):
    def test_release_is_exact_and_read_permission_checked_before_dev_shutdown(self):
        workflow = yaml.safe_load((ROOT / '.github/workflows/ci-cd.yml').read_text(encoding='utf-8'))
        steps = workflow['jobs']['desplegar-produccion']['steps']
        scripts = '\n'.join(s.get('run', '') for s in steps)
        self.assertIn('--exact-match HEAD', scripts)
        names = [s.get('name','') for s in steps]
        self.assertLess(names.index('Comprobar permiso lector de observabilidad antes de detener dev'),
                        names.index('Evitar competencia de Outbox con desarrollo'))
        self.assertLess(names.index('Verificar HTTPS y disponibilidad de produccion'),
                        names.index('Desactivar frontend dev solo tras validar produccion'))
        shutdown = next(s['run'] for s in steps if s.get('id') == 'detener-dev')
        self.assertIn('deployment/interbank-scenarios', shutdown)
        self.assertNotIn('|| true', shutdown)
        self.assertIn('KUBE_NAMESPACE=bank-usac-dev bash infrastructure/observability/deploy.sh', scripts)
        self.assertIn('"$FRONTEND_URL/interbank/v1/health"', scripts)
        for name in ('desplegar-desarrollo','desplegar-observabilidad-dev'):
            self.assertIn("vars.DEPLOY_DEV != 'false'", workflow['jobs'][name]['if'])

    def test_prod_exports_telemetry_and_keeps_activation_origin(self):
        directory = ROOT / 'infrastructure/kubernetes/overlays/prod'
        config = yaml.safe_load((directory / 'patch-configuracion-prod.yaml').read_text())['data']
        self.assertEqual(config['ENTORNO'], 'produccion')
        self.assertIn('bank-usac-observability', config['OTEL_EXPORTER_OTLP_ENDPOINT'])
        self.assertEqual(config['CORS_ORIGINS'], '__FRONTEND_URL__')

    def test_prod_smtp_reads_only_prod_application_secret(self):
        with patch.dict(os.environ, {'KUBE_NAMESPACE':'bank-usac'}), \
             patch('subprocess.check_output', return_value=json.dumps({'data':{}}).encode()) as read:
            with self.assertRaises(SystemExit):
                smtp_secret.main()
            self.assertEqual(read.call_args.args[0][2], 'bank-usac')


if __name__ == '__main__':
    unittest.main()
