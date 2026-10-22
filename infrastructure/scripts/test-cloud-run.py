"""Cloud Run deployment regressions; no real deployments, credentials or traffic changes."""
import hashlib
import importlib.util
import io
import json
from pathlib import Path
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parent


def module(name):
    spec = importlib.util.spec_from_file_location(name, ROOT / (name + '.py'))
    result = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(result)
    return result


rollback = module('cloud-run-rollback')
images = module('image-digest')
IMAGE = 'docker.io/example/frontend@sha256:' + 'a' * 64
ROUTES = [{'revisionName': 'frontend-stable', 'percent': 100}]


class RollbackTests(unittest.TestCase):
    def test_failed_tag_is_repaired_even_when_old_revision_still_serves(self):
        current = {'spec': {'template': {'spec': {'containers': [{'image': 'docker.io/example/frontend:missing'}]}}}, 'status': {'traffic': ROUTES}}
        calls = []
        rollback.restore({'exists': True, 'service': 'bank-usac-frontend-dev', 'region': 'us-central1', 'image': IMAGE, 'traffic': ROUTES},
                         get=lambda _: json.dumps(current), run=lambda c, **kw: calls.append(c))
        self.assertEqual(len(calls), 2)
        self.assertIn('--image', calls[0])
        self.assertIn(IMAGE, calls[0])
        self.assertIn('--no-traffic', calls[0])
        self.assertIn('update-traffic', calls[1])
        self.assertIn('frontend-stable=100', calls[1])

    def test_unchanged_frontend_needs_no_update(self):
        current = {'spec': {'template': {'spec': {'containers': [{'image': IMAGE.replace('docker.io/', 'mirror.gcr.io/')}]}}}, 'status': {'traffic': ROUTES}}
        def forbidden(*args, **kwargs):
            self.fail('Unchanged service must not be updated')
        rollback.restore({'exists': True, 'service': 'bank-usac-frontend-dev', 'region': 'us-central1', 'image': IMAGE, 'traffic': ROUTES},
                         get=lambda _: json.dumps(current), run=forbidden)

    def test_failed_repair_stops_before_traffic_update(self):
        current = {'spec': {'template': {'spec': {'containers': [{'image': 'bad:tag'}]}}}, 'status': {'traffic': ROUTES}}
        calls = []
        def fail(command, **kwargs):
            calls.append(command)
            raise subprocess.CalledProcessError(1, command)
        with self.assertRaises(subprocess.CalledProcessError):
            rollback.restore({'exists': True, 'service': 'bank-usac-frontend-dev', 'region': 'us-central1', 'image': IMAGE, 'traffic': ROUTES}, get=lambda _: json.dumps(current), run=fail)
        self.assertEqual(len(calls), 1)

    def test_capture_uses_serving_revision_not_latest_ready(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'state.json'
            queries = []
            def get(command):
                queries.append(command)
                if 'services' in command:
                    return json.dumps({'status': {'latestReadyRevisionName': 'unused-new-revision', 'traffic': ROUTES}})
                return IMAGE.replace('docker.io/', 'mirror.gcr.io/')
            rollback.capture('bank-usac-frontend-dev', 'us-central1', path, get=get)
            self.assertIn('frontend-stable', queries[1])
            self.assertEqual(json.loads(path.read_text())['image'], IMAGE)

    def test_capture_does_not_hide_permission_failures(self):
        def denied(command):
            raise subprocess.CalledProcessError(1, command, stderr='PERMISSION_DENIED')
        with self.assertRaises(subprocess.CalledProcessError):
            rollback.capture('bank-usac-frontend-dev', 'us-central1', Path('must-not-exist.json'), get=denied)

    def test_first_deployment_has_no_prior_revision_to_restore(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'state.json'
            def absent(command):
                raise subprocess.CalledProcessError(1, command, stderr='Cannot find service [bank-usac-frontend-dev]')
            rollback.capture('bank-usac-frontend-dev', 'us-central1', path, get=absent)
            self.assertFalse(json.loads(path.read_text())['exists'])


class ImageTests(unittest.TestCase):
    def response(self, data):
        body = json.dumps(data).encode()
        response = io.BytesIO(body)
        response.headers = {'Docker-Content-Digest': 'sha256:' + hashlib.sha256(body).hexdigest()}
        return response

    def test_tag_is_replaced_by_verified_manifest_digest(self):
        body = {'schemaVersion': 2, 'config': {'digest': 'sha256:config'}, 'layers': []}
        expected = 'sha256:' + hashlib.sha256(json.dumps(body).encode()).hexdigest()
        calls = []
        def open_url(request, **kwargs):
            calls.append(request)
            return self.response({'token': 'public-pull-token'}) if isinstance(request, str) else self.response(body)
        self.assertEqual(images.resolve('docker.io/example/frontend:new', open_url), 'docker.io/example/frontend@' + expected)
        self.assertIn('registry-1.docker.io', calls[1].full_url)

    def test_platform_index_selects_linux_amd64(self):
        image = {'schemaVersion': 2, 'layers': []}
        digest = 'sha256:' + hashlib.sha256(json.dumps(image).encode()).hexdigest()
        index = {'manifests': [{'digest': digest, 'platform': {'os': 'linux', 'architecture': 'amd64'}},
                               {'digest': 'sha256:arm', 'platform': {'os': 'linux', 'architecture': 'arm64'}}]}
        def open_url(request, **kwargs):
            if isinstance(request, str):
                return self.response({'token': 'public-pull-token'})
            return self.response(image if request.full_url.endswith(digest) else index)
        self.assertTrue(images.resolve('docker.io/example/frontend:new', open_url).endswith('@' + digest))

    def test_digest_mismatch_blocks_deployment(self):
        def open_url(request, **kwargs):
            response = self.response({'token': 'public-pull-token'})
            if not isinstance(request, str):
                response.headers['Docker-Content-Digest'] = 'sha256:wrong'
            return response
        with self.assertRaisesRegex(ValueError, 'mismatch'):
            images.resolve('docker.io/example/frontend:new', open_url)


if __name__ == '__main__':
    unittest.main()
