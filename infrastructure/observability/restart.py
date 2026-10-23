"""Restart dev exporters without surge pods triggering the node autoscaler."""
import json
import os
import subprocess

namespace = os.environ.get('KUBE_NAMESPACE', 'bank-usac-dev')
if namespace not in ('bank-usac-dev', 'bank-usac'):
    raise SystemExit('Unsupported application namespace')
for name in ['api-gateway','customer-service','account-service','transaction-service','payment-service','notification-audit-service']:
    command=['kubectl','-n',namespace]
    deployment=json.loads(subprocess.check_output(command+['get','deployment',name,'-o','json']))
    previous=deployment['spec']['strategy']
    def strategy(value):
        subprocess.run(command+['patch','deployment',name,'--type=merge','-p',json.dumps({'spec':{'strategy':value}})],check=True)
    try:
        strategy({'type':'RollingUpdate','rollingUpdate':{'maxSurge':0,'maxUnavailable':1}})
        subprocess.run(command+['rollout','restart','deployment/'+name],check=True)
        subprocess.run(command+['rollout','status','deployment/'+name,'--timeout=180s'],check=True)
    finally:
        strategy(previous)
