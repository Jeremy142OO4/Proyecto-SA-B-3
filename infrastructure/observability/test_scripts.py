"""No-cluster regression tests for the dev-only restart helper."""
import json
from pathlib import Path
import runpy
import subprocess
import unittest
from unittest.mock import patch
from rbac_preflight import check_role


class RBACTests(unittest.TestCase):
    def test_matching_role_does_not_require_ci_patch_permission(self):
        role = {'rules': [{'apiGroups': ['metrics.k8s.io'], 'resources': ['pods'], 'verbs': ['get', 'list']}]}
        self.assertIn('no RBAC update', check_role(role, role, False))

    def test_different_role_requires_bootstrap_for_limited_ci(self):
        with self.assertRaisesRegex(RuntimeError, 'administrador'):
            check_role({'rules': []}, {'rules': [{'resources': ['pods']}]}, False)

    def test_administrator_can_reconcile_role(self):
        self.assertIn('can patch', check_role({'rules': []}, {'rules': [{}]}, True))

    def test_rule_order_is_reconciled_before_helm(self):
        with self.assertRaises(RuntimeError):
            check_role({'rules': [{'resources': ['pods']}, {'resources': ['deployments']}]},
                       {'rules': [{'resources': ['deployments']}, {'resources': ['pods']}]}, False)


class RestartTests(unittest.TestCase):
    def exercise(self, fail):
        calls=[]
        previous={'type':'RollingUpdate','rollingUpdate':{'maxSurge':1,'maxUnavailable':0}}
        def run(command,check=True):
            calls.append(command)
            if fail and 'status' in command:
                raise subprocess.CalledProcessError(1,command)
        with patch('subprocess.check_output',return_value=json.dumps({'spec':{'strategy':previous}}).encode()),patch('subprocess.run',side_effect=run):
            if fail:
                with self.assertRaises(subprocess.CalledProcessError):
                    runpy.run_path(str(Path(__file__).with_name('restart.py')))
            else:
                runpy.run_path(str(Path(__file__).with_name('restart.py')))
        self.assertEqual(len(calls),4 if fail else 24)
        self.assertEqual(json.loads(calls[0][-1])['spec']['strategy']['rollingUpdate']['maxSurge'],0)
        self.assertEqual(json.loads(calls[-1][-1])['spec']['strategy'],previous)
        self.assertTrue(all(command[2]=='bank-usac-dev' for command in calls))

    def test_success_restores_strategy(self):
        self.exercise(False)

    def test_failure_restores_strategy_and_stops(self):
        self.exercise(True)


if __name__=='__main__':
    unittest.main()
