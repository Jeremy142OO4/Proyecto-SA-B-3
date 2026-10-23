"""Create NEW load-only fixtures through the public API; never seed real accounts.

Tokens are supplied by the operator in the environment, not saved in source.
The output contains a customer token/password: keep it in ignored tmp/ only.
Every write is sent once. An ambiguous response stops setup, without retrying.
"""
import argparse
import json
import os
from pathlib import Path
import secrets
import time
import urllib.error
import urllib.request


class API:
    def __init__(self, base):
        if not base.startswith('https://') or '/' in base[8:]:
            raise ValueError('A bare HTTPS origin is required')
        self.base = base
        class NoRedirect(urllib.request.HTTPRedirectHandler):
            def redirect_request(self, req, fp, code, msg, headers, newurl):
                return None
        self.opener = urllib.request.build_opener(NoRedirect)

    def call(self, path, token, method='GET', body=None):
        req = urllib.request.Request(self.base + path, method=method,
            headers={'Authorization': 'Bearer ' + token, 'Content-Type': 'application/json'},
            data=json.dumps(body).encode() if body is not None else None)
        # Never retry writes or include request tokens/bodies in exceptions.
        try:
            with self.opener.open(req, timeout=25) as r:
                return json.load(r)
        except urllib.error.HTTPError as exc:
            raise RuntimeError(f'{method} {path}: HTTP {exc.code}; inspect before retrying') from None


def rows(value, key):
    return value if isinstance(value, list) else value[key]


def wait_for(fn, predicate, seconds=90):
    deadline = time.monotonic() + seconds
    while time.monotonic() < deadline:
        value = fn()
        if predicate(value):
            return value
        time.sleep(1)
    raise RuntimeError('Setup did not settle; do not repeat writes blindly')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--base-url', required=True)
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--pairs', type=int, default=8, choices=range(1, 21))
    parser.add_argument('--confirm', required=True, choices=['create-new-test-fixtures'])
    args = parser.parse_args()
    if 'tmp' not in args.output.resolve().parts or args.output.exists():
        parser.error('Use a NEW private output under tmp/; never overwrite a fixture')
    teller, admin = os.environ['TELLER_TOKEN'], os.environ['ADMIN_TOKEN']
    api = API(args.base_url)
    run = 'carga-' + time.strftime('%Y%m%d-%H%M%S', time.gmtime()) + '-' + secrets.token_hex(3)
    password = secrets.token_urlsafe(24)
    email = run + '@load.bank-usac.invalid'
    document = str(8000000000000 + secrets.randbelow(999999999999))
    state = {'purpose': 'load-test', 'runId': run, 'baseUrl': args.base_url,
             'email': email, 'password': password, 'document': document, 'pairs': [],
             'fundingPerAccountCentavos': 100000, 'ready': False}
    args.output.parent.mkdir(parents=True, exist_ok=True)

    def save():
        # This is private runtime data, not a checked-in artifact.
        args.output.write_text(json.dumps(state, indent=2), encoding='utf-8')

    save()  # Preserve identity before the first write for ambiguous-response recovery.
    api.call('/api/clientes/registro', teller, 'POST', {
        'nombres': 'Carga', 'apellidos': run, 'documento': document,
        'correo': email, 'fechaNacimiento': '1990-01-01',
        'direccion': 'Fixture exclusivo de prueba de carga', 'password': password,
    })
    customers = rows(api.call('/api/administracion/clientes?limite=100', admin), 'clientes')
    customer = next((c for c in customers if c.get('email', c.get('correo')) == email), None)
    if customer is None:
        raise RuntimeError('New customer not in first page; locate by private fixture email before continuing')
    identifier = customer.get('customerId', customer.get('clienteId'))
    state['customerId'] = identifier
    save()
    for body in ({'estado': 'ACTIVO'}, {'estadoKyc': 'VERIFIED'}):
        api.call('/api/administracion/clientes/' + identifier + '/estado', admin, 'PATCH', body)
    login = api.call('/api/clientes/login', '', 'POST', {'username': email, 'password': password})
    state['token'] = login['token']
    save()
    token = state['token']

    def accounts():
        return rows(api.call('/api/cuentas?limite=100', token), 'cuentas')

    for index in range(args.pairs * 2):
        before = {a['idCuenta'] for a in accounts()}
        operation = api.call('/api/cuentas', teller, 'POST', {
            'tipoCuenta': 'CORRIENTE', 'documento': document,
            'saldoMinimoQuetzales': '0.00', 'comisionTransaccionQuetzales': '0.00',
        })
        state['lastSetupOperation'] = operation
        save()
        current = wait_for(accounts, lambda xs: len({a['idCuenta'] for a in xs} - before) == 1)
        account = next(a for a in current if a['idCuenta'] not in before)
        if index % 2 == 0:
            state['pairs'].append({'source': account['idCuenta']})
        else:
            state['pairs'][-1]['destination'] = account['idCuenta']
        save()
        api.call('/api/cuentas/' + account['idCuenta'] + '/deposito', token, 'POST',
                 {'montoCentavos': state['fundingPerAccountCentavos']})
        wait_for(lambda: api.call('/api/cuentas/' + account['idCuenta'], token),
                 lambda a: a['saldoCentavos'] == state['fundingPerAccountCentavos'] and
                 a['saldoReservadoCentavos'] == 0)
        print(f'Fixture funded: {index + 1}/{args.pairs * 2}', flush=True)
    state['initialTotalCentavos'] = sum(a['saldoCentavos'] for a in accounts())
    state['ready'] = True
    state.pop('lastSetupOperation', None)
    save()
    print(f'Ready: {run}; {args.pairs} pairs. Private credentials saved, not printed.', flush=True)


if __name__ == '__main__':
    main()
