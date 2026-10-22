"""Exercise only the explicit isolated sandbox. Never reads cloud credentials."""
import argparse
import json
import time
import urllib.error
import urllib.request
from urllib.parse import urlsplit
import uuid


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--base', default='http://127.0.0.1:8180')
    parser.add_argument('--execute', action='store_true', help='Create five deposits in the sandbox fixture account')
    args = parser.parse_args()
    parsed = urlsplit(args.base)
    if parsed.hostname not in ('127.0.0.1', 'localhost') or parsed.scheme != 'http' or parsed.path:
        parser.error('Only a loopback sandbox frontend is allowed')
    token = ''

    def api(method, path, value=None, auth=True):
        headers = {'Content-Type': 'application/json', 'traceparent': '00-'+uuid.uuid4().hex+'-'+uuid.uuid4().hex[:16]+'-01'}
        if token and auth:
            headers['Authorization'] = 'Bearer '+token
        req = urllib.request.Request(args.base+'/api'+path, method=method, headers=headers,
                                     data=None if value is None else json.dumps(value).encode())
        try:
            response = urllib.request.urlopen(req, timeout=20)
        except urllib.error.HTTPError as e:
            response = e
        with response:
            raw = response.read()
            try:
                value = json.loads(raw)
            except ValueError:
                value = raw.decode(errors='replace')
            return response.code, value

    code, login = api('POST', '/clientes/login', {'username': 'sandbox-admin', 'password': 'SandboxDemo2026!'}, False)
    assert code == 200, ('Sandbox login', code, login)
    token = login['token']
    root = '/administracion/pruebas-interbancarias/'
    code, state = api('GET', root)
    assert code == 200 and state['enabled'] and state['target'] == 'MOCK', ('Not an isolated sandbox', code, state)
    assert state['callbackConfigured'] and state['accounts'] == ['TEST-BANK14-001']
    assert api('GET', root, auth=False)[0] == 401
    assert api('PUT', root, {'scenario': 'invalid'})[0] == 400
    print('Isolated mock and admin authentication verified', flush=True)
    if not args.execute:
        return
    identifiers = {}
    code, listing = api('GET', '/administracion/cuentas?cuenta=00000000-0000-4000-8000-000000000005')
    assert code == 200 and len(listing['cuentas']) == 1, (code, listing)
    before = listing['cuentas'][0]['saldoCentavos']

    def select(scenario):
        code, result = api('PUT', root, {'scenario': scenario})
        assert code == 200 and result['scenario'] == scenario, (code, result)

    def send(scenario):
        select(scenario)
        identifier = str(uuid.uuid4())
        code, result = api('POST', root+'depositos', {'id': identifier, 'amount': '100.00'})
        assert code == 202, (code, result)
        identifiers[scenario] = identifier
        print(scenario, identifier, 'created', flush=True)
        # Wait for the mock to receive this transfer before changing scenarios.
        # 12 s is above the normal bank worker/RPC interval in the sandbox.
        time.sleep(12)

    try:
        for scenario in ('no_callback', 'delayed', 'accept', 'reject', 'unavailable'):
            send(scenario)
        select('accept')  # Recovery for the 503 scenario, same transfer ID.
        pending = set(identifiers)
        deadline = time.monotonic()+300
        expected = {'no_callback': 'ACCEPTED', 'delayed': 'REVERSED', 'accept': 'ACCEPTED', 'reject': 'REVERSED', 'unavailable': 'ACCEPTED'}
        last = {}
        while pending and time.monotonic() < deadline:
            for scenario in list(pending):
                code, result = api('GET', root+'depositos/'+identifiers[scenario])
                assert code == 200, (code, result)
                status = result['status']
                if last.get(scenario) != status:
                    print(scenario, status, result.get('reasonCode', ''), flush=True)
                    last[scenario] = status
                if status in ('ACCEPTED', 'REVERSED', 'REJECTED', 'CANCELLED'):
                    assert status == expected[scenario], (scenario, result)
                    pending.remove(scenario)
            if pending:
                time.sleep(5)
        assert not pending, ('Unresolved transfers: do not resend', pending, identifiers)
        code, listing = api('GET', '/administracion/cuentas?cuenta=00000000-0000-4000-8000-000000000005')
        account = listing['cuentas'][0]
        assert code == 200 and account['saldoCentavos'] == before-30000, account
        assert account['saldoReservadoCentavos'] == 0, account
        print('PASS: all five scenarios resolved. IDs:', json.dumps(identifiers), flush=True)
    finally:
        select('accept')


if __name__ == '__main__':
    main()
