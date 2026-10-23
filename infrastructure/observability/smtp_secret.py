"""Copy existing SMTP credentials without persisting or printing secret values."""
import base64
from email.utils import parseaddr
import json
import os
import subprocess


def main():
    source = os.environ.get('KUBE_NAMESPACE', 'bank-usac-dev')
    target = os.environ.get('OBSERVABILITY_NAMESPACE', 'bank-usac-observability')
    if source not in ('bank-usac-dev', 'bank-usac') or target != 'bank-usac-observability':
        raise SystemExit('SMTP copy is restricted to the bank application namespaces')
    raw = subprocess.check_output(['kubectl', '-n', source, 'get', 'secret', 'bank-usac-secrets', '-o', 'json'])
    data = json.loads(raw)['data']
    keys = {'USER': 'SMTP_USERNAME', 'PASSWORD': 'SMTP_APP_PASSWORD', 'FROM_ADDRESS': 'SMTP_FROM'}
    if any(not data.get(key) or not base64.b64decode(data[key]).strip() for key in [*keys.values(), 'SMTP_HOST', 'SMTP_PORT']):
        raise SystemExit('Existing application SMTP configuration is incomplete; refusing to deploy mail alerts')
    host = base64.b64decode(data['SMTP_HOST']).decode().strip()
    port = base64.b64decode(data['SMTP_PORT']).decode().strip()
    if port != '587':
        raise SystemExit('This SMTP profile requires STARTTLS on port 587')
    secret = {'apiVersion': 'v1', 'kind': 'Secret', 'type': 'Opaque',
              'metadata': {'name': 'observability-smtp', 'namespace': target},
              'data': {name: data[key] for name, key in keys.items()}}
    secret['data']['HOST'] = base64.b64encode(f'{host}:{port}'.encode()).decode()
    # The app accepts "Bank USAC <sender@example>"; Grafana requires a bare address.
    _, sender = parseaddr(base64.b64decode(data['SMTP_FROM']).decode().strip())
    if '@' not in sender or any(character in sender for character in '\r\n'):
        raise SystemExit('SMTP_FROM does not contain a valid sender address')
    secret['data']['FROM_ADDRESS'] = base64.b64encode(sender.encode()).decode()
    # Server-side apply avoids a last-applied annotation containing credentials.
    subprocess.run(['kubectl', 'apply', '--server-side', '--field-manager=bank-observability', '-f', '-'],
                   input=json.dumps(secret), text=True, check=True, stdout=subprocess.DEVNULL)
    broker = {'username': 'RABBITMQ_USUARIO', 'password': 'RABBITMQ_CLAVE'}
    if any(not data.get(key) for key in broker.values()):
        raise SystemExit('Existing RabbitMQ credentials are missing')
    secret['metadata']['name'] = 'observability-rabbitmq'
    secret['data'] = {name: data[key] for name, key in broker.items()}
    subprocess.run(['kubectl', 'apply', '--server-side', '--field-manager=bank-observability', '-f', '-'],
                   input=json.dumps(secret), text=True, check=True, stdout=subprocess.DEVNULL)
    print('Observability SMTP Secret synchronized (values hidden)')


if __name__ == '__main__':
    main()
