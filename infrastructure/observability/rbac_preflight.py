"""Fail before Helm if a limited CI identity cannot reconcile the reader Role."""
import json
import os
from pathlib import Path
import subprocess
import sys
import yaml


def check_role(current, desired, may_patch):
    # Preserve order too: Helm's three-way patch can otherwise replace the rules list.
    if current.get('rules', []) == desired.get('rules', []):
        return 'Collector Role matches chart; no RBAC update required'
    if may_patch:
        return 'Collector Role differs; current identity can patch it'
    raise RuntimeError(
        'El Role bank-observability difiere del chart y la identidad CI no puede modificarlo. '
        'Un administrador debe sincronizar una sola vez las reglas lectoras del Role en '
        'bank-usac-dev con chart/templates/rbac.yaml, conservando la propiedad de Helm. '
        'No conceder cluster-admin al pipeline. Ver Documentacion/observabilidad/metricas-sistema.md.'
    )


def main():
    namespace = os.environ.get('KUBE_NAMESPACE', 'bank-usac-dev')
    observability = os.environ.get('OBSERVABILITY_NAMESPACE', 'bank-usac-observability')
    if namespace not in ('bank-usac-dev', 'bank-usac') or observability != 'bank-usac-observability':
        raise SystemExit('Unsupported observability profile')
    chart = Path(__file__).resolve().parent / 'chart'
    rendered = subprocess.check_output([
        'helm', 'template', 'bank-observability', str(chart), '--namespace', observability,
        '--set', 'applicationNamespace=' + namespace, '--show-only', 'templates/rbac.yaml',
    ], text=True, encoding='utf-8')
    desired = next(o for o in yaml.safe_load_all(rendered) if o and o['kind'] == 'Role')
    output = subprocess.check_output([
        'kubectl', '-n', namespace, 'get', 'role', 'bank-observability', '--ignore-not-found', '-o', 'json',
    ], text=True, encoding='utf-8')
    if not output.strip():
        allowed = subprocess.run(['kubectl', 'auth', 'can-i', 'create', 'roles', '-n', namespace], capture_output=True)
        if allowed.returncode:
            raise RuntimeError('El Role del Collector no existe: requiere bootstrap administrativo antes del despliegue CI.')
        print('Collector Role absent; current identity can create it')
        return
    current = json.loads(output)
    if current.get('rules', []) == desired.get('rules', []):
        print(check_role(current, desired, False))
        return
    allowed = subprocess.run(['kubectl', 'auth', 'can-i', 'patch', 'role/bank-observability', '-n', namespace], capture_output=True)
    print(check_role(current, desired, allowed.returncode == 0))


if __name__ == '__main__':
    try:
        main()
    except RuntimeError as error:
        print(str(error), file=sys.stderr)
        sys.exit(1)
