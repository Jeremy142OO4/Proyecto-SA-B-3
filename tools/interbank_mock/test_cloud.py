"""Cloud deployment invariants, with no cluster calls or bank operations."""
import base64
import importlib.util
import json
import os
from pathlib import Path
import unittest
from unittest.mock import patch
from urllib.parse import parse_qs, urlsplit

ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location('cloud_lab', ROOT/'infrastructure/interbank-sandbox/deploy-cloud.py')
cloud = importlib.util.module_from_spec(spec)
spec.loader.exec_module(cloud)


class CloudIsolationTests(unittest.TestCase):
    def test_database_uri_cannot_inherit_public_search_path_or_bank_role(self):
        url = cloud.isolated_url('postgres://bank:real-password@10.2.3.4:5432/cuentas?sslmode=require&options=bad&search_path=public&user=bank&password=bad', 'fixture-password')
        parsed = urlsplit(url)
        self.assertEqual(parsed.username, 'bank_scenarios')
        self.assertEqual(parsed.password, 'fixture-password')
        self.assertEqual(parsed.path, '/cuentas')
        self.assertEqual(parse_qs(parsed.query), {'sslmode':['require'], 'search_path':['bank_scenarios']})
        self.assertNotIn('real-password', url)

    def test_migrations_use_separate_schema_and_role_without_resetting_balances(self):
        sql = cloud.migration_sql('account-service')
        self.assertIn('SET LOCAL ROLE bank_scenarios;', sql)
        self.assertLess(sql.index('GRANT bank_scenarios TO CURRENT_USER;'), sql.index('CREATE SCHEMA'))
        self.assertIn('SET LOCAL search_path TO bank_scenarios;', sql)
        self.assertIn('ON CONFLICT DO NOTHING', sql)
        self.assertIn('Applied lab migration changed', sql)
        self.assertNotIn('public.', sql)

    def test_private_cloud_resources_and_enable_only_after_ready(self):
        applied, calls = [], []
        def fake(*args, data=None):
            calls.append(args)
            if args[:3] == ('get','secret','bank-usac-secrets'):
                values = {'JWT_SECRET':'bank-jwt-secret', 'URL_BASE_DATOS_CUENTAS':'postgres://bank:bank-password@10.1.1.1/accounts', 'URL_BASE_DATOS_TRANSACCIONES':'postgres://bank:bank-password@10.1.1.1/transactions', 'URL_BASE_DATOS_PAGOS':'postgres://bank:bank-password@10.1.1.1/payments'}
                return json.dumps({'data':{k:base64.b64encode(v.encode()).decode() for k,v in values.items()}})
            if args[:2] == ('get','secret'): return ''
            if args[:2] == ('get','deployment'):
                return json.dumps({'spec':{'template':{'spec':{'containers':[{'image':'registry/'+args[2]+':immutable'}]}}}})
            if data: applied.append(json.loads(data))
            return ''
        with patch.object(cloud, 'kubectl', side_effect=fake), patch.dict(os.environ, {'MOCK_IMAGE':'registry/mock:immutable'}):
            cloud.main()
        deployment = next(r for r in applied if r['kind']=='Deployment')
        pod = deployment['spec']['template']['spec']
        self.assertEqual(len(pod['containers']), 6)
        self.assertEqual(sum(int(c['resources']['requests']['cpu'][:-1]) for c in pod['containers']),35)
        payments = next(c for c in pod['containers'] if c['name']=='payments')
        self.assertEqual(payments['image'], 'registry/payment-service:immutable')
        self.assertIn('URL_BASE_DATOS_PAGOS', json.dumps(payments['env']))
        self.assertIn('lab-pagos-', json.dumps(applied))
        self.assertFalse(pod['automountServiceAccountToken'])
        self.assertEqual(deployment['spec']['strategy']['type'], 'Recreate')
        self.assertNotIn('database', [c['name'] for c in pod['containers']])
        service = next(r for r in applied if r['kind']=='Service')
        self.assertNotIn('type', service['spec'])  # ClusterIP, no public endpoint.
        self.assertEqual(service['spec']['ports'][0]['port'],8080)
        secret = next(r for r in applied if r['kind']=='Secret')['stringData']
        self.assertNotIn('bank-password', json.dumps(secret))
        self.assertNotIn('pitagoricos', json.dumps(applied))
        self.assertIn('interbank-cloud-tests', json.dumps(pod))
        ready = calls.index(('rollout','status','deployment/interbank-scenarios','--timeout=240s'))
        enable = calls.index(('set','env','deployment/api-gateway','INTERBANK_CLOUD_TEST_RELAY=true'))
        self.assertLess(ready, enable)

    def test_readme_changes_do_not_trigger_cloud_deployment(self):
        from .test_ci_filters import FilterTests
        filters = FilterTests()
        filters.setUpClass()
        self.assertEqual(filters.selected(['infrastructure/interbank-sandbox/README.md']), set())
