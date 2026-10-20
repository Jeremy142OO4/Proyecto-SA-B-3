"""Boot the pinned Grafana image and verify provisioning without sending email."""
import base64
import json
from pathlib import Path
import secrets
import subprocess
import time
import urllib.request
import uuid
import alerting


def main():
    name = 'bank-grafana-test-' + uuid.uuid4().hex[:12]
    password = secrets.token_hex(24)
    authorization = 'Basic ' + base64.b64encode(('admin:' + password).encode()).decode()
    directory = Path(__file__).resolve().parent / 'chart'
    command = ['docker', 'run', '-d', '--name', name, '-p', '127.0.0.1::3000',
               '-v', f'{directory / "alerting/bank.json"}:/etc/grafana/provisioning/alerting/bank.json:ro,Z',
               '-v', f'{directory / "config/datasources.yaml"}:/etc/grafana/provisioning/datasources/bank.yaml:ro,Z']
    environment = {'GF_SECURITY_ADMIN_PASSWORD': password, 'GF_UNIFIED_ALERTING_EXECUTE_ALERTS': 'false',
                   'GF_USERS_VIEWERS_CAN_EDIT': 'true',
                   'GF_SMTP_ENABLED': 'false', 'GF_PLUGINS_PREINSTALL_DISABLED': 'true',
                   'GF_ANALYTICS_REPORTING_ENABLED': 'false', 'GF_ANALYTICS_CHECK_FOR_UPDATES': 'false',
                   'GF_ANALYTICS_CHECK_FOR_PLUGIN_UPDATES': 'false'}
    for key, value in environment.items():
        command += ['-e', f'{key}={value}']
    try:
        subprocess.run(command + ['grafana/grafana:13.2.4'], check=True, stdout=subprocess.DEVNULL)
        binding = subprocess.check_output(['docker', 'port', name, '3000/tcp'], text=True).strip().splitlines()[0]
        base = 'http://' + binding
        def get(path, auth=authorization):
            request = urllib.request.Request(base + path, headers={'Authorization': auth})
            with urllib.request.urlopen(request, timeout=5) as response:
                return json.load(response)
        deadline = time.monotonic() + 100
        while True:
            try:
                if get('/api/health')['database'] == 'ok':
                    break
            except (OSError, ValueError):
                pass
            if time.monotonic() > deadline:
                raise RuntimeError('Grafana provisioning did not become ready')
            time.sleep(1)
        rules = get('/api/v1/provisioning/alert-rules')
        assert {r['uid'] for r in rules} == {r['uid'] for r in alerting.provisioning()['groups'][0]['rules']}
        contacts = get('/api/v1/provisioning/contact-points')
        assert any(c['uid'] == 'bank-email' and c['settings']['addresses'] == '3141193530501@ingenieria.usac.edu.gt' for c in contacts)
        viewer_password = secrets.token_hex(24)
        request = urllib.request.Request(base + '/api/admin/users',
            data=json.dumps({'login': 'query-test-viewer', 'password': viewer_password, 'OrgId': 1}).encode(),
            headers={'Authorization': authorization, 'Content-Type': 'application/json'}, method='POST')
        with urllib.request.urlopen(request, timeout=5) as response:
            assert response.status == 200
        viewer_auth = 'Basic ' + base64.b64encode(('query-test-viewer:' + viewer_password).encode()).decode()
        assert all(org['role'] == 'Viewer' for org in get('/api/user/orgs', viewer_auth))
        assert not get('/api/user', viewer_auth).get('isGrafanaAdmin')
        permissions = get('/api/access-control/user/permissions', viewer_auth)
        assert permissions.get('datasources:explore')
        assert not permissions.get('alert.rules:create')
        print('Grafana boot, alerts, contact and Viewer Explore permissions verified; no email sent')
    except Exception:
        diagnostic = subprocess.run(['docker', 'logs', name], capture_output=True, text=True)
        # Docker config is intentionally never inspected: it includes the disposable password.
        print('\n'.join(diagnostic.stdout.splitlines()[-15:]))
        raise
    finally:
        subprocess.run(['docker', 'rm', '-f', name], capture_output=True)


if __name__ == '__main__':
    main()
