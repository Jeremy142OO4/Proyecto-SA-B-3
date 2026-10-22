"""Namespace-only Metrics API bridge. No dependencies, Secrets or business payloads."""
import json
import os
import ssl
import threading
import time
import urllib.request
from decimal import Decimal
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

SA = Path('/var/run/secrets/kubernetes.io/serviceaccount')


def quantity(value):
    text = str(value)
    for suffix, factor in [('Ki', 1024), ('Mi', 1024**2), ('Gi', 1024**3),
                           ('Ti', 1024**4), ('n', 1e-9), ('u', 1e-6), ('m', 1e-3),
                           ('k', 1000), ('M', 1000**2), ('G', 1000**3)]:
        if text.endswith(suffix):
            return float(Decimal(text[:-len(suffix)]) * Decimal(str(factor)))
    return float(Decimal(text))


def sample(name, labels, value):
    attributes = ','.join(k + '=' + json.dumps(str(v), ensure_ascii=True) for k, v in labels.items())
    return f'{name}{{{attributes}}} {float(value)}\n'


def resource_samples(data, namespace, environment, now):
    lines = []
    for pod in data['items']:
        # Metrics Server can retain a last sample: never export it as current forever.
        timestamp = pod.get('timestamp')
        if timestamp:
            from datetime import datetime
            if now - datetime.fromisoformat(timestamp.replace('Z', '+00:00')).timestamp() > 120:
                continue
        for container in pod.get('containers', []):
            labels = dict(namespace=namespace, environment=environment,
                          pod=pod['metadata']['name'], container=container['name'])
            for key, metric in [('cpu', 'bank_pod_cpu_usage_cores'), ('memory', 'bank_pod_memory_usage_bytes')]:
                if key in container.get('usage', {}):
                    lines.append(sample(metric, labels, quantity(container['usage'][key])))
    return ''.join(lines)


def revision_samples(data, namespace, environment):
    return ''.join(sample('bank_deployment_revision', dict(namespace=namespace, environment=environment,
                  deployment=d['metadata']['name']), revision)
                  for d in data['items']
                  if (revision := d['metadata'].get('annotations', {}).get('deployment.kubernetes.io/revision', '')).isdigit())


def read_api(path):
    # Re-read projected token so rotation does not break a long-running exporter.
    request = urllib.request.Request('https://kubernetes.default.svc' + path,
                                    headers={'Authorization': 'Bearer ' + (SA / 'token').read_text().strip()})
    with urllib.request.urlopen(request, context=ssl.create_default_context(cafile=str(SA / 'ca.crt')), timeout=8) as response:
        return json.load(response)


def collect(namespace, environment, read=read_api, now=None):
    now = time.time() if now is None else now
    result = []
    sources = [('resources', '/apis/metrics.k8s.io/v1beta1/namespaces/' + namespace + '/pods',
                lambda data: resource_samples(data, namespace, environment, now)),
               ('deployments', '/apis/apps/v1/namespaces/' + namespace + '/deployments',
                lambda data: revision_samples(data, namespace, environment))]
    for source, path, render in sources:
        labels = dict(namespace=namespace, environment=environment, source=source)
        try:
            result.extend([render(read(path)), sample('bank_system_source_up', labels, 1)])
        except Exception:
            # Drop stale series on failure; do not report zero CPU or leak API/token details.
            result.append(sample('bank_system_source_up', labels, 0))
    return ''.join(result).encode()


def main():
    namespace = os.environ['APPLICATION_NAMESPACE']
    environment = os.environ['ENVIRONMENT']
    state = {'body': b''}

    def poll():
        while True:
            state['body'] = collect(namespace, environment)
            time.sleep(30)

    class Handler(BaseHTTPRequestHandler):
        def do_GET(self):
            if self.path not in ('/metrics', '/health'):
                self.send_error(404)
                return
            self.send_response(200)
            self.send_header('Content-Type', 'text/plain; version=0.0.4; charset=utf-8')
            self.end_headers()
            self.wfile.write(state['body'] if self.path == '/metrics' else b'ok\n')

        def log_message(self, *args):
            pass

    threading.Thread(target=poll, daemon=True).start()
    ThreadingHTTPServer(('0.0.0.0', 9466), Handler).serve_forever()


if __name__ == '__main__':
    main()
