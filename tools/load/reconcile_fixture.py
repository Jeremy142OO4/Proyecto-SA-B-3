"""Read-only settlement check for dedicated load accounts. Never resubmit a transfer."""
import argparse
from collections import Counter
from datetime import datetime, timezone
import json
from pathlib import Path
from prepare_fixture import API, rows


def reconcile(fixture):
    if fixture.get('purpose') != 'load-test' or not fixture.get('ready'):
        raise ValueError('Ready, dedicated load fixture required')
    api = API(fixture['baseUrl'])
    token = fixture['token']
    profile = api.call('/api/clientes/perfil', token)
    if profile['correo'] != fixture['email'] or not profile['correo'].endswith('.invalid'):
        raise ValueError('Customer is not the dedicated fixture owner')
    accounts = rows(api.call('/api/cuentas?limite=100', token), 'cuentas')
    expected = {p[k] for p in fixture['pairs'] for k in ('source', 'destination')}
    if {a['idCuenta'] for a in accounts} != expected:
        raise ValueError('Account set differs from private fixture; inspect before continuing')
    records = []
    for offset in range(0, 10000, 100):
        batch = rows(api.call(f'/api/transferencias?limite=100&desplazamiento={offset}', token), 'transferencias')
        records.extend(batch)
        if len(batch) < 100:
            break
    else:
        raise ValueError('Unexpected history size; incomplete reconciliation')
    states = Counter(t['estado'] for t in records)
    ids = {t['idTransferencia'] for t in records}
    owned = all(t['descripcion'] == fixture['runId'] and t['idCuentaOrigen'] in expected and
                t['idCuentaDestino'] in expected and t['montoCentavos'] == 1 for t in records)
    total = sum(a['saldoCentavos'] for a in accounts)
    held = sum(a['saldoReservadoCentavos'] for a in accounts)
    return {'timestamp': datetime.now(timezone.utc).isoformat(), 'runId': fixture['runId'],
            'accountCount': len(accounts), 'transfers': len(records), 'states': dict(states),
            'initialTotalCentavos': fixture['initialTotalCentavos'], 'totalCentavos': total,
            'reservedCentavos': held,
            'consistent': owned and len(ids) == len(records) and
            all(a['comisionTransaccionCentavos'] == 0 and a['saldoCentavos'] >= 0 for a in accounts) and
            total == fixture['initialTotalCentavos'] and held == 0 and
            all(s == 'COMPLETADA' for s in states)}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--fixture', type=Path, required=True)
    parser.add_argument('--output', type=Path)
    args = parser.parse_args()
    result = reconcile(json.loads(args.fixture.read_text(encoding='utf-8')))
    safe = json.dumps(result, indent=2)
    if args.output:
        with args.output.open('x', encoding='utf-8') as out:
            out.write(safe + '\n')
    print(safe)
    raise SystemExit(0 if result['consistent'] else 1)


if __name__ == '__main__':
    main()
