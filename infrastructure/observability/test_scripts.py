"""No-cluster regression tests for the dev-only restart helper."""
import json
from pathlib import Path
import runpy
import subprocess
import unittest
from unittest.mock import patch


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
