"""Read-only, timestamped HPA/pod evidence. Requires kubectl access to the cluster."""
import argparse
from datetime import datetime, timezone
import json
from pathlib import Path
import subprocess
import time


def snapshot(fetch, namespace):
    hpas = fetch(f'/apis/autoscaling/v2/namespaces/{namespace}/horizontalpodautoscalers')['items']
    deployments = fetch(f'/apis/apps/v1/namespaces/{namespace}/deployments')['items']
    pods = fetch(f'/api/v1/namespaces/{namespace}/pods')['items']
    return {
        'timestamp': datetime.now(timezone.utc).isoformat(),
        'namespace': namespace,
        'hpas': [{
            'name': h['metadata']['name'], 'target': h['spec']['scaleTargetRef']['name'],
            'min': h['spec'].get('minReplicas', 1), 'max': h['spec']['maxReplicas'],
            'targets': h['spec']['metrics'], 'status': h.get('status', {}),
        } for h in hpas],
        'deployments': [{
            'name': d['metadata']['name'], 'desired': d['spec'].get('replicas', 1),
            'ready': d.get('status', {}).get('readyReplicas', 0),
            'available': d.get('status', {}).get('availableReplicas', 0),
        } for d in deployments],
        'pods': [{
            'name': p['metadata']['name'], 'uid': p['metadata']['uid'],
            'created': p['metadata']['creationTimestamp'],
            'app': p['metadata'].get('labels', {}).get('app'),
            'phase': p['status']['phase'],
            'terminating': bool(p['metadata'].get('deletionTimestamp')),
            'conditions': p['status'].get('conditions', []),
            'containers': [{ 'name': s['name'], 'ready': s['ready'], 'restarts': s['restartCount'] }
                           for s in p['status'].get('containerStatuses', [])],
        } for p in pods],
    }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--namespace', default='bank-usac')
    parser.add_argument('--output', type=Path, required=True)
    parser.add_argument('--seconds', type=int, default=600)
    args = parser.parse_args()
    if not 1 <= args.seconds <= 1200:
        parser.error('seconds must be 1..1200')
    def fetch(path):
        return json.loads(subprocess.check_output(['kubectl', 'get', '--raw', path], timeout=25))
    deadline = time.monotonic() + args.seconds
    args.output.parent.mkdir(parents=True, exist_ok=True)
    # Exclusive output avoids silently overwriting an earlier run's evidence.
    with args.output.open('x', encoding='utf-8') as out:
        while time.monotonic() < deadline:
            out.write(json.dumps(snapshot(fetch, args.namespace)) + '\n')
            out.flush()
            time.sleep(min(10, max(0, deadline - time.monotonic())))


if __name__ == '__main__':
    main()
