"""Deploy the isolated scenario laboratory in GKE; databases stay outside GKE.

Uses existing dev images and external PostgreSQL servers, but a separate DB role,
schema, RabbitMQ and PVC. No partner credentials are copied into the lab.
"""
import base64
import hashlib
import json
import os
from pathlib import Path
import secrets
import subprocess
from urllib.parse import parse_qsl, urlencode, urlsplit, urlunsplit

NS = 'bank-usac-dev'
NAME = 'interbank-scenarios'
SCHEMA = 'bank_scenarios'
ROOT = Path(__file__).resolve().parents[2]


def kubectl(*args, data=None):
    result = subprocess.run(['kubectl', '-n', NS, *args], input=data,
                            text=True, capture_output=True)
    if result.returncode:
        # Never include stdin (contains credentials) in exceptions.
        raise RuntimeError('kubectl failed: ' + ' '.join(args[:3]) + '\n' + result.stderr)
    return result.stdout


def apply(value):
    kubectl('apply', '-f', '-', data=json.dumps(value))


def resource(kind, name, **fields):
    return dict(apiVersion='v1', kind=kind,
                metadata=dict(name=name, namespace=NS), **fields)


def isolated_url(raw, password):
    u = urlsplit(raw)
    if u.scheme not in ('postgres', 'postgresql') or not u.hostname or not u.path.strip('/'):
        raise ValueError('An external PostgreSQL URI is required')
    host = f'[{u.hostname}]' if ':' in u.hostname else u.hostname
    if u.port:
        host += f':{u.port}'
    query = dict(parse_qsl(u.query))
    # No inherited role/options that could select the bank's public schema.
    for key in ('options', 'search_path', 'user', 'password'):
        query.pop(key, None)
    query['search_path'] = SCHEMA
    return urlunsplit((u.scheme, f'{SCHEMA}:{password}@{host}', u.path, urlencode(query), ''))


def migration_sql(module):
    parts = ["SELECT pg_advisory_xact_lock(hashtext('bank-scenarios-bootstrap'));",
             "SELECT format('CREATE ROLE bank_scenarios LOGIN PASSWORD %L', :'password') WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='bank_scenarios') \\gexec",
             'CREATE SCHEMA IF NOT EXISTS bank_scenarios AUTHORIZATION bank_scenarios;',
             'SET LOCAL ROLE bank_scenarios;',
             'SET LOCAL search_path TO bank_scenarios;',
             'CREATE TABLE IF NOT EXISTS lab_migrations(name text PRIMARY KEY, checksum text NOT NULL);']
    for path in sorted((ROOT / 'services' / module / 'database/migrations').glob('*.up.sql')):
        sql = path.read_text(encoding='utf-8-sig')
        # pgcrypto is already provisioned by the bank; gen_random_uuid is built in.
        sql = '\n'.join(line for line in sql.splitlines() if not line.strip().startswith('CREATE EXTENSION'))
        digest = hashlib.sha256(sql.encode()).hexdigest()
        parts += [f"SELECT NOT EXISTS(SELECT 1 FROM lab_migrations WHERE name='{path.name}') AS pending \\gset",
                  '\\if :pending', sql,
                  f"INSERT INTO lab_migrations VALUES('{path.name}','{digest}');", '\\else',
                  f"DO $$ BEGIN IF NOT EXISTS(SELECT 1 FROM lab_migrations WHERE name='{path.name}' AND checksum='{digest}') THEN RAISE EXCEPTION 'Applied lab migration changed'; END IF; END $$;",
                  '\\endif']
    if module == 'account-service':
        parts.append("INSERT INTO cuentas(id_cuenta,id_cliente,numero_cuenta,tipo_cuenta,saldo_centavos,moneda,estado) VALUES('00000000-0000-4000-8000-000000000005','00000000-0000-4000-8000-000000000003','900000000003','CORRIENTE',10000000,'GTQ','ACTIVA') ON CONFLICT DO NOTHING;")
    return '\n'.join(parts)


def main():
    bank = json.loads(kubectl('get', 'secret', 'bank-usac-secrets', '-o', 'json'))['data']
    bank = {k: base64.b64decode(v).decode() for k, v in bank.items()}
    previous = kubectl('get', 'secret', NAME, '--ignore-not-found', '-o', 'json')
    old = json.loads(previous)['data'] if previous.strip() else {}
    values = {k: base64.b64decode(v).decode() for k, v in old.items()}
    for key in ('TEST_DB_PASSWORD', 'MOCK_CONTROL_KEY', 'INTERBANK_API_KEY', 'INTERBANK_PARTNER_API_KEY'):
        values.setdefault(key, secrets.token_hex(24))
    values['JWT_SECRET'] = bank['JWT_SECRET']  # Existing ADMIN login, no new public login.
    for suffix in ('CUENTAS', 'TRANSACCIONES'):
        values['URL_BASE_DATOS_' + suffix] = isolated_url(bank['URL_BASE_DATOS_' + suffix], values['TEST_DB_PASSWORD'])
    apply(resource('Secret', NAME, type='Opaque', stringData=values))

    run = os.environ.get('GITHUB_RUN_ID', 'manual-'+secrets.token_hex(4)) + '-' + os.environ.get('GITHUB_RUN_ATTEMPT', '1')
    for module, key in [('account-service', 'CUENTAS'), ('transaction-service', 'TRANSACCIONES')]:
        job = f'lab-{key.lower()}-{run}'
        apply(resource('ConfigMap', job, data={'migration.sql': migration_sql(module)}))
        manifest = resource('Job', job)
        manifest['apiVersion'] = 'batch/v1'
        manifest['spec'] = dict(backoffLimit=0, activeDeadlineSeconds=180, ttlSecondsAfterFinished=600,
            template=dict(spec=dict(restartPolicy='Never', automountServiceAccountToken=False,
                containers=[dict(name='schema', image='postgres:17-alpine',
                    command=['sh', '-ec', 'psql --no-psqlrc --set=ON_ERROR_STOP=1 --single-transaction --set=password="$TEST_DB_PASSWORD" --dbname="$DATABASE_URL" --file=/migration/migration.sql'],
                    env=[dict(name='DATABASE_URL', valueFrom=dict(secretKeyRef=dict(name='bank-usac-secrets', key='URL_BASE_DATOS_'+key))),
                         dict(name='TEST_DB_PASSWORD', valueFrom=dict(secretKeyRef=dict(name=NAME, key='TEST_DB_PASSWORD'))),
                         dict(name='PGCONNECT_TIMEOUT', value='10'), dict(name='PGOPTIONS', value='-c lock_timeout=5000 -c statement_timeout=30000')],
                    resources=dict(requests=dict(cpu='25m', memory='32Mi'), limits=dict(cpu='100m', memory='96Mi')),
                    volumeMounts=[dict(name='sql', mountPath='/migration', readOnly=True)])],
                volumes=[dict(name='sql', configMap=dict(name=job))])))
        apply(manifest)
        kubectl('wait', '--for=condition=complete', 'job/'+job, '--timeout=200s')
        kubectl('delete', 'configmap', job)

    def image(deployment):
        return json.loads(kubectl('get', 'deployment', deployment, '-o', 'json'))['spec']['template']['spec']['containers'][0]['image']

    common = dict(ENTORNO='interbank-sandbox', DEPLOYMENT_ENVIRONMENT='interbank-cloud-tests',
                  INTERBANK_TEST_MODE='true', INTERBANK_TEST_LOCATION='GCP / GKE',
                  INTERBANK_ENABLED='true', INTERBANK_BANK_CODE='BANK03', INTERBANK_PARTNER_BANK_CODE='BANK14',
                  INTERBANK_PARTNER_BASE_URL='http://127.0.0.1:8099/interbank/v1',
                  URL_RABBITMQ='amqp://guest:guest@127.0.0.1:5672/',
                  OTEL_EXPORTER_OTLP_ENDPOINT='http://otel-collector.bank-usac-observability.svc.cluster.local:4318',
                  OTEL_TRACES_SAMPLER='always_on', INTERVALO_OUTBOX_SEGUNDOS='2')

    def container(name, img, port, extra=None):
        settings = dict(common, PUERTO_HTTP=str(port), OTEL_SERVICE_NAME='lab-'+name, **(extra or {}))
        return dict(name=name, image=img, envFrom=[dict(secretRef=dict(name=NAME))],
                    env=[dict(name=k, value=v) for k,v in settings.items()],
                    resources=dict(requests=dict(cpu='25m', memory='48Mi'), limits=dict(cpu='300m', memory='192Mi')),
                    readinessProbe=dict(httpGet=dict(path='/salud', port=port), initialDelaySeconds=5, periodSeconds=5),
                    livenessProbe=dict(httpGet=dict(path='/salud', port=port), initialDelaySeconds=30, periodSeconds=10),
                    securityContext=dict(allowPrivilegeEscalation=False, capabilities=dict(drop=['ALL'])) )

    gateway = container('gateway', image('api-gateway'), 8080, dict(INTERBANK_MOCK_CONTROL_URL='http://127.0.0.1:8099', INTERBANK_TEST_GRAFANA_URL='https://bank-usac-grafana-2atv2f4ufq-uc.a.run.app/explore'))
    accounts = container('accounts', image('account-service'), 8082)
    transactions = container('transactions', image('transaction-service'), 8083)
    for c,key in [(accounts,'CUENTAS'),(transactions,'TRANSACCIONES')]:
        c['env'].append(dict(name='URL_BASE_DATOS', valueFrom=dict(secretKeyRef=dict(name=NAME,key='URL_BASE_DATOS_'+key))))
    mock = dict(name='mock', image=os.environ['MOCK_IMAGE'], envFrom=[dict(secretRef=dict(name=NAME))],
        env=[dict(name='MOCK_API_KEY', valueFrom=dict(secretKeyRef=dict(name=NAME,key='INTERBANK_PARTNER_API_KEY'))),
             dict(name='MOCK_PARTNER_API_KEY', valueFrom=dict(secretKeyRef=dict(name=NAME,key='INTERBANK_API_KEY'))),
             dict(name='MOCK_PARTNER_BASE_URL', value='http://127.0.0.1:8080/interbank/v1')],
        volumeMounts=[dict(name='state',mountPath='/app/state',subPath='mock')],
        resources=dict(requests=dict(cpu='25m',memory='32Mi'),limits=dict(cpu='200m',memory='96Mi')),
        readinessProbe=dict(exec=dict(command=['python','-c',"import socket; socket.create_connection(('127.0.0.1',8099),2).close()"]),periodSeconds=5))
    rabbit = dict(name='rabbit',image='rabbitmq:3.13-alpine',
        env=[dict(name='RABBITMQ_NODENAME',value='rabbit@interbank-scenarios'),dict(name='ERL_FLAGS',value='+S 1:1 +A 2')],
        volumeMounts=[dict(name='state',mountPath='/var/lib/rabbitmq',subPath='rabbit')],
        resources=dict(requests=dict(cpu='50m',memory='128Mi'),limits=dict(cpu='300m',memory='384Mi')))
    apply(resource('PersistentVolumeClaim', NAME, spec=dict(accessModes=['ReadWriteOnce'],resources=dict(requests=dict(storage='2Gi')))))
    manifest = resource('Deployment', NAME)
    manifest['apiVersion']='apps/v1'
    labels = dict(app=NAME)
    manifest['spec']=dict(replicas=1,strategy=dict(type='Recreate'),selector=dict(matchLabels=labels),
        template=dict(metadata=dict(labels=labels, annotations={'bank.usac/config-digest':hashlib.sha256(json.dumps(values,sort_keys=True).encode()).hexdigest()}),spec=dict(hostname=NAME,automountServiceAccountToken=False,
            initContainers=[dict(name='volume-permissions',image='busybox:1.37',command=['sh','-ec','mkdir -p /state/mock /state/rabbit; chown 10001:10001 /state/mock; chown 999:999 /state/rabbit'],volumeMounts=[dict(name='state',mountPath='/state')])],
            containers=[gateway,accounts,transactions,mock,rabbit],volumes=[dict(name='state',persistentVolumeClaim=dict(claimName=NAME))])))
    apply(manifest)
    apply(resource('Service',NAME,spec=dict(selector=labels,ports=[dict(name='http',port=8080,targetPort=8080)])))
    policy=resource('NetworkPolicy',NAME)
    policy['apiVersion']='networking.k8s.io/v1'
    policy['spec']=dict(podSelector=dict(matchLabels=labels),policyTypes=['Ingress'],ingress=[dict(
        **{'from':[dict(podSelector=dict(matchLabels=dict(app='api-gateway')))]}, ports=[dict(protocol='TCP',port=8080)])])
    apply(policy)
    kubectl('rollout','status','deployment/'+NAME,'--timeout=240s')
    # Enable only after the isolated deployment is healthy. Normal bank settings unchanged.
    kubectl('set','env','deployment/api-gateway','INTERBANK_CLOUD_TEST_RELAY=true')
    kubectl('rollout','status','deployment/api-gateway','--timeout=180s')
    print('GCP laboratory ready. BANK14 configuration and balances were not modified.')


if __name__ == '__main__':
    main()
