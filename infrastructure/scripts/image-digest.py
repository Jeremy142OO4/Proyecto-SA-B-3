"""Resolve a public Docker Hub image to immutable linux/amd64 content."""
import hashlib
import json
import re
import sys
import time
import urllib.parse
import urllib.request

ACCEPT = ', '.join(['application/vnd.oci.image.index.v1+json', 'application/vnd.docker.distribution.manifest.list.v2+json',
                    'application/vnd.oci.image.manifest.v1+json', 'application/vnd.docker.distribution.manifest.v2+json'])


def resolve(image, open_url=urllib.request.urlopen):
    match = re.fullmatch(r'docker.io/([a-z0-9_.-]+/[a-z0-9_.-]+):([A-Za-z0-9_.-]+)', image)
    if not match:
        raise ValueError('Expected a tagged public docker.io/owner/image:tag reference')
    repository, tag = match.groups()
    auth = 'https://auth.docker.io/token?' + urllib.parse.urlencode({'service': 'registry.docker.io', 'scope': 'repository:' + repository + ':pull'})
    with open_url(auth, timeout=20) as response:
        token = json.load(response)['token']

    def manifest(reference):
        request = urllib.request.Request('https://registry-1.docker.io/v2/' + repository + '/manifests/' + reference,
                                         headers={'Authorization': 'Bearer ' + token, 'Accept': ACCEPT})
        with open_url(request, timeout=20) as response:
            body = response.read()
            digest = 'sha256:' + hashlib.sha256(body).hexdigest()
            if response.headers.get('Docker-Content-Digest', digest) != digest:
                raise ValueError('Registry manifest digest mismatch')
            return json.loads(body), digest

    data, digest = manifest(tag)
    if 'manifests' in data:
        candidates = [m for m in data['manifests'] if m.get('platform', {}).get('os') == 'linux' and m.get('platform', {}).get('architecture') == 'amd64']
        if len(candidates) != 1:
            raise ValueError('Expected exactly one linux/amd64 image')
        requested = candidates[0]['digest']
        _, digest = manifest(requested)
        if digest != requested:
            raise ValueError('Platform manifest digest mismatch')
    return 'docker.io/' + repository + '@' + digest


if __name__ == '__main__':
    for attempt in range(3):
        try:
            print(resolve(sys.argv[1]))
            break
        except (OSError, ValueError, KeyError):
            if attempt == 2:
                raise SystemExit('Cannot verify the published Docker Hub image; deployment stopped before changing Cloud Run')
            time.sleep(2 ** attempt)
