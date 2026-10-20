"""Render the chart and check safety properties and executable component configs."""
import json
import re
from pathlib import Path
import subprocess
import tempfile
import sys
import yaml

directory=Path(__file__).resolve().parent
subprocess.run([sys.executable,str(directory/'alerting.py'),'--check'],check=True)
subprocess.run([sys.executable,str(directory/'test_alerting.py')],check=True)
subprocess.run([sys.executable,str(directory/'test_promql.py')],check=True)
subprocess.run([sys.executable,str(directory/'test_grafana.py')],check=True)
subprocess.run([sys.executable,str(directory/'test_scripts.py')],check=True)
subprocess.run(['helm','lint',str(directory/'chart')],check=True)
rendered=subprocess.check_output(['helm','template','bank-observability',str(directory/'chart'),'--namespace','bank-usac-observability'],text=True)
objects=list(yaml.safe_load_all(rendered))
assert len([o for o in objects if o['kind']=='PersistentVolumeClaim'])==4
assert not any(o['kind']=='Secret' for o in objects),'Credentials must not be rendered into Helm values'
for obj in objects:
    if obj['kind']=='Service':assert obj['spec']['type']=='ClusterIP','Public observability endpoint forbidden'
    if obj['kind'] in ['StatefulSet','Deployment']:
        assert obj['spec']['replicas']==1
        spec = obj['spec']['template']['spec']
        assert spec.get('automountServiceAccountToken', True) == (obj['metadata']['name'] == 'otel-collector')
        if obj['metadata']['name'] == 'otel-collector':
            assert spec['serviceAccountName'] == 'bank-observability'
    if obj['kind'] == 'Role':
        assert obj['metadata']['namespace'] == 'bank-usac-dev'
        assert all(set(rule['verbs']) <= {'get', 'list', 'watch'} and 'secrets' not in rule['resources'] for rule in obj['rules'])
    assert obj['kind'] not in ['ClusterRole', 'ClusterRoleBinding'], 'Monitoring must stay namespace scoped'
configmaps={o['metadata']['name']:o['data'] for o in objects if o['kind']=='ConfigMap'}
json.loads(configmaps['grafana-dashboards']['overview.json'])
datasources=yaml.safe_load(configmaps['grafana-config']['config.yaml'])['datasources']
matcher=datasources[1]['jsonData']['derivedFields'][0]['matcherRegex']
assert re.search(matcher,json.dumps({'traceId':'a'*32})),'Log-to-trace link does not match JSON'
for data in configmaps.values():
    if 'config.yaml' in data:yaml.safe_load(data['config.yaml'])
configs=[
 ('otel-collector-config','otel/opentelemetry-collector-contrib:0.163.1',None,['validate','--config=/etc/bank.yaml']),
 ('otel-logs-config','otel/opentelemetry-collector-contrib:0.163.1',None,['validate','--config=/etc/bank.yaml']),
 ('prometheus-config','prom/prometheus:v3.15.0','/bin/promtool',['check','config','/etc/bank.yaml']),
 ('loki-config','grafana/loki:3.7.8',None,['-config.file=/etc/bank.yaml','-verify-config=true'])
]
with tempfile.TemporaryDirectory(prefix='bank-observability-validation-') as temp:
    for name,image,entrypoint,args in configs:
        path=Path(temp)/(name+'.yaml');path.write_text(configmaps[name]['config.yaml'],encoding='utf-8')
        command=['docker','run','--rm','-e','RABBITMQ_USER=validation','-e','RABBITMQ_PASSWORD=validation','-v',f'{path}:/etc/bank.yaml:ro,Z']
        if entrypoint:command+=['--entrypoint',entrypoint]
        subprocess.run(command+[image]+args,check=True)
    subprocess.run([sys.executable,str(directory/'test_agent_ingestion.py'),'--config',str(Path(temp)/'otel-logs-config.yaml'),'--image','otel/opentelemetry-collector-contrib:0.163.1'],check=True)
print('Private services, external Secret, persistence, dashboard and component configurations validated')
