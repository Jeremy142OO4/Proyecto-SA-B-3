"""Execute the real route-selection portion of the workflow without network or jq."""
import os
from pathlib import Path
import shlex
import shutil
import subprocess
import unittest
import yaml

ROOT = Path(__file__).resolve().parents[2]


class FilterTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.workflow = yaml.safe_load((ROOT/'.github/workflows/ci-cd.yml').read_text(encoding='utf-8'))
        steps = cls.workflow['jobs']['detectar-cambios']['steps']
        cls.script = next(s['run'] for s in steps if s.get('id') == 'filtros').split('matriz_go=', 1)[0]
        cls.bash = 'C:/Program Files/Git/bin/bash.exe' if os.name == 'nt' else shutil.which('bash')
        if not cls.bash or not Path(cls.bash).exists():
            raise unittest.SkipTest('Bash is required to execute workflow filters')

    def selected(self, paths, ref='refs/heads/develop'):
        names = ['interbancaria', 'observabilidad', 'gateway', 'account', 'payment', 'transaction', 'customer', 'notification', 'frontend']
        fake_git = "git(){ printf '%s\\n' " + ' '.join(shlex.quote(p) for p in paths) + '; }\n'
        inspect = '\n'+''.join('printf "'+name+'=%s\\n" "$'+name+'"\n' for name in names)
        env = dict(os.environ, EVENTO='push', REF=ref, SHA_ANTERIOR='base', GITHUB_SHA='head')
        result = subprocess.run([self.bash, '-s'], input=fake_git+self.script+inspect,
                                text=True, capture_output=True, env=env, check=True, timeout=10)
        return {line.split('=')[0] for line in result.stdout.splitlines() if line.endswith('=true')}

    def test_contract_only_does_not_build_apps(self):
        self.assertEqual(self.selected(['contracts/interbank/openapi.yaml']), {'interbancaria'})
        self.assertEqual(self.selected(['tools/interbank_mock/app.py']), {'interbancaria'})

    def test_documentation_only(self):
        self.assertEqual(self.selected(['Documentacion/interbancaria/acuerdo-bank03-bank14.md']), set())

    def test_service_filter_unchanged(self):
        self.assertEqual(self.selected(['services/account-service/main.go']), {'account'})
        self.assertEqual(self.selected(['tools/interbank_mock/app.py', 'gateway/api-gateway/main.go']), {'interbancaria', 'gateway'})

    def test_global_and_release_validate_everything(self):
        self.assertEqual(len(self.selected(['.github/workflows/ci-cd.yml'])), 9)
        self.assertEqual(len(self.selected(['README.md'], 'refs/heads/release/v3')), 9)

    def test_failed_contract_prevents_publication_gate(self):
        gate = self.workflow['jobs']['validacion-sistema']
        self.assertIn('validar-interbancaria', gate['needs'])
        self.assertIn("needs.validar-interbancaria.result == 'success'", gate['if'])
        self.assertIn("needs.validar-interbancaria.result == 'skipped'", gate['if'])


if __name__ == '__main__':
    unittest.main()
