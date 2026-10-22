"""Apply environment-specific interbank settings without printing private keys."""
import json
import os
import subprocess
from urllib.parse import urlsplit


def resources(env):
    namespace = env['KUBE_NAMESPACE']
    enabled = env.get('INTERBANK_ENABLED', 'false')
    if enabled not in ('true', 'false'):
        raise ValueError('INTERBANK_ENABLED must be true or false')
    key = env.get('INTERBANK_API_KEY', '')
    partner_key = env.get('INTERBANK_PARTNER_API_KEY', '')
    url = env.get('INTERBANK_PARTNER_BASE_URL', '').rstrip('/')
    if enabled == 'true' and len(key) < 16:
        raise ValueError('Inbound API key is required to enable interbank reception')
    if partner_key and len(partner_key) < 16:
        raise ValueError('Invalid outbound API key')
    if url:
        parsed = urlsplit(url)
        if (parsed.scheme != 'https' or not parsed.hostname or parsed.username
                or parsed.password or parsed.query or parsed.fragment
                or parsed.path != '/interbank/v1' or parsed.hostname.endswith('example.com')):
            raise ValueError('Invalid outbound HTTPS URL')
    secret = {
        'apiVersion': 'v1', 'kind': 'Secret',
        'metadata': {'name': 'bank-usac-interbank', 'namespace': namespace},
        'type': 'Opaque',
        'stringData': {'INTERBANK_API_KEY': key, 'INTERBANK_PARTNER_API_KEY': partner_key},
    }
    # Deployment overrides are revisioned: rollout undo restores the previous flag
    # and URL along with the old image. The shared base stays disabled by default.
    settings = [f'INTERBANK_ENABLED={enabled}', f'INTERBANK_PARTNER_BASE_URL={url}']
    return secret, settings


def main():
    secret, settings = resources(os.environ)
    subprocess.run(['kubectl', 'apply', '--server-side', '--field-manager=interbank-ci',
                    '-f', '-'], input=json.dumps(secret), text=True, check=True,
                   stdout=subprocess.DEVNULL)
    subprocess.run(['kubectl', '-n', os.environ['KUBE_NAMESPACE'], 'set', 'env',
                    'deployment/api-gateway', 'deployment/account-service',
                    'deployment/transaction-service', 'deployment/payment-service', *settings], check=True)
    print('Interbank reception configuration applied; no private values logged')


if __name__ == '__main__':
    main()
