"""Copy a verified public image to private Artifact Registry before Cloud Run deployment."""
import importlib.util
import json
import re
import subprocess
import sys
from pathlib import Path


def stage(source, project, tag, run=subprocess.run):
    if not re.fullmatch(r'[a-z][a-z0-9-]{4,61}[a-z0-9]', project):
        raise ValueError('Invalid project ID')
    if not re.fullmatch(r'[A-Za-z0-9_.-]{1,128}', tag):
        raise ValueError('Invalid image tag')
    match = re.fullmatch(r'docker.io/[a-z0-9_.-]+/[a-z0-9_.-]+@(sha256:[a-f0-9]{64})', source)
    if not match:
        raise ValueError('Verified Docker Hub digest required')
    digest = match[1]
    repository = f'us-central1-docker.pkg.dev/{project}/bank-usac-cloud-run/frontend'
    target = repository + ':' + tag
    def execute(args, capture=False):
        return run(args, check=True, text=True, stdout=subprocess.PIPE if capture else sys.stderr)
    # The repository is bootstrapped separately. CI cannot create repositories or grant IAM.
    execute(['gcloud', 'artifacts', 'repositories', 'describe', 'bank-usac-cloud-run', '--location=us-central1', '--project='+project])
    execute(['gcloud', 'auth', 'configure-docker', 'us-central1-docker.pkg.dev', '--quiet'])
    execute(['docker', 'pull', '--platform=linux/amd64', source])
    execute(['docker', 'tag', source, target])
    execute(['docker', 'push', target])
    result = execute(['docker', 'image', 'inspect', '--format={{json .RepoDigests}}', target], capture=True)
    if repository+'@'+digest not in json.loads(result.stdout):
        raise ValueError('Artifact Registry digest does not match verified source; deployment stopped')
    return repository+'@'+digest


if __name__ == '__main__':
    print(stage(*sys.argv[1:]))
