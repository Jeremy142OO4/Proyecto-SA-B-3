"""Check aggregate steady-state requests; not a scheduler or autoscaler guarantee."""
import json
import os
from decimal import Decimal
import subprocess

def get(resource):
    return json.loads(subprocess.check_output(['kubectl','get',resource,'--all-namespaces','-o','json']))['items']
def cpu(value):
    text=str(value)
    for suffix,factor in [('m',Decimal('0.001')),('u',Decimal('0.000001')),('n',Decimal('0.000000001'))]:
        if text.endswith(suffix):return Decimal(text[:-len(suffix)])*factor
    return Decimal(text)
def memory(value):
    text=str(value)
    for suffix,factor in [('Ki',1024),('Mi',1024**2),('Gi',1024**3),('K',1000),('M',1000**2),('G',1000**3)]:
        if text.endswith(suffix):return Decimal(text[:-len(suffix)])*factor
    return Decimal(text)
namespace=os.environ.get('OBSERVABILITY_NAMESPACE','bank-usac-observability')
nodes={n['metadata']['name']:n for n in get('nodes') if not n['spec'].get('unschedulable',False)}
free={name:[cpu(node['status']['allocatable']['cpu']),memory(node['status']['allocatable']['memory'])] for name,node in nodes.items()}
for pod in get('pods'):
    name=pod['spec'].get('nodeName')
    if name not in free or pod['metadata']['namespace']==namespace or pod['status']['phase'] in ['Succeeded','Failed']:continue
    request=[Decimal(0),Decimal(0)]
    for container in pod['spec']['containers']:
        values=container.get('resources',{}).get('requests',{})
        request[0]+=cpu(values.get('cpu','0'));request[1]+=memory(values.get('memory','0'))
    for container in pod['spec'].get('initContainers',[]):
        values=container.get('resources',{}).get('requests',{})
        request[0]=max(request[0],cpu(values.get('cpu','0')));request[1]=max(request[1],memory(values.get('memory','0')))
    free[name][0]-=request[0];free[name][1]-=request[1]
required_cpu=Decimal('0.065')+len(nodes)*Decimal('0.005')
required_memory=(640+32*len(nodes))*1024**2
available_cpu=sum(v[0] for v in free.values());available_memory=sum(v[1] for v in free.values())
assert nodes,'No schedulable nodes'
assert all(v[0]>=Decimal('0.005') and v[1]>=32*1024**2 for v in free.values()),'No room for log agent on a node; refusing to increase cluster'
assert available_cpu>=required_cpu and available_memory>=required_memory,'Insufficient capacity; no automatic node increase authorized'
print(f'Capacity OK: need {required_cpu*1000}m CPU / {required_memory//1024**2}Mi; available {available_cpu*1000}m CPU / {int(available_memory)//1024**2}Mi on {len(nodes)} existing nodes')
