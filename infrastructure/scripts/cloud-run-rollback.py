"""Save actual serving traffic and recover it without revalidating a failed image tag."""
import argparse
import json
from pathlib import Path
import shutil
import subprocess


def canonical(image):
    return image.replace('mirror.gcr.io/', 'docker.io/', 1) if image.startswith('mirror.gcr.io/') else image


def traffic(service):
    return sorted([{'revisionName': t['revisionName'], 'percent': t['percent']}
                   for t in service.get('status', {}).get('traffic', []) if t.get('percent', 0) > 0 and t.get('revisionName')],
                  key=lambda t: t['revisionName'])


def read(command):
    command = [shutil.which(command[0]) or command[0], *command[1:]]
    return subprocess.check_output(command, text=True, encoding='utf-8', stderr=subprocess.PIPE).strip()


def capture(service, region, path, get=read):
    command = ['gcloud', 'run', 'services', 'describe', service, '--region', region,
               '--format=json(status.traffic,status.latestReadyRevisionName)']
    try:
        data = json.loads(get(command))
    except subprocess.CalledProcessError as error:
        # Bootstrap is distinct from permission/API failures. Never swallow those errors.
        detail = str(error.stderr or '').lower()
        if not any(marker in detail for marker in ('not_found', 'not found', 'cannot find service')):
            raise
        path.write_text(json.dumps({'exists': False, 'service': service, 'region': region}), encoding='utf-8')
        return
    routes = traffic(data)
    if not routes or sum(t['percent'] for t in routes) != 100:
        raise ValueError('Cannot save a stable Cloud Run traffic allocation')
    revision = max(routes, key=lambda t: t['percent'])['revisionName']
    image = canonical(get(['gcloud', 'run', 'revisions', 'describe', revision, '--region', region, '--format=value(status.imageDigest)']))
    if '@sha256:' not in image:
        raise ValueError('Stable revision has no immutable image reference')
    path.write_text(json.dumps({'exists': True, 'service': service, 'region': region, 'traffic': routes, 'image': image}), encoding='utf-8')


def restore(state, get=read, run=subprocess.run):
    if not state.get('exists'):
        print('No previous Cloud Run service; no revision to restore')
        return
    service, region = state['service'], state['region']
    current = json.loads(get(['gcloud', 'run', 'services', 'describe', service, '--region', region,
                              '--format=json(status.traffic,spec.template.spec.containers[0].image)']))
    image = canonical(current['spec']['template']['spec']['containers'][0]['image'])
    if image == state['image'] and traffic(current) == state['traffic']:
        print('Cloud Run already matches stable image and traffic; rollback skipped')
        return
    if image != state['image']:
        # update-traffic alone also validates spec.template; repair the failed image first.
        run(['gcloud', 'run', 'services', 'update', service, '--region', region, '--image', state['image'], '--no-traffic', '--quiet'], check=True)
    routes = ','.join(t['revisionName'] + '=' + str(t['percent']) for t in state['traffic'])
    run(['gcloud', 'run', 'services', 'update-traffic', service, '--region', region, '--to-revisions', routes, '--quiet'], check=True)
    print('Cloud Run restored to the previously serving revisions')


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('action', choices=['capture', 'restore'])
    parser.add_argument('--service', required=True, choices=['bank-usac-frontend-dev', 'bank-usac-frontend-prod'])
    parser.add_argument('--region', default='us-central1')
    parser.add_argument('--state', type=Path, required=True)
    args = parser.parse_args()
    if args.action == 'capture':
        capture(args.service, args.region, args.state)
    else:
        state = json.loads(args.state.read_text(encoding='utf-8'))
        if state['service'] != args.service or state['region'] != args.region:
            raise SystemExit('Rollback state belongs to a different service or region')
        restore(state)
