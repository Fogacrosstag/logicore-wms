#!/usr/bin/env python3
"""Black-box tests against the seven running services, including actual Kafka."""
import sys,time,uuid,json,subprocess
from pathlib import Path
from concurrent.futures import ThreadPoolExecutor
sys.path.insert(0,str(Path(__file__).resolve().parents[1]/'scripts'))
from client import request,wait_for,fixture,reserve,reservation,balances,APIError
from demo import run_demo

def rejects(fn,status=409):
 try:fn()
 except APIError as e:
  assert e.status==status,(e.status,e.body)
  return
 raise AssertionError('Request unexpectedly succeeded')
f=run_demo()
with ThreadPoolExecutor(max_workers=2) as pool:
 results=list(pool.map(lambda _:reserve(f,60),range(2)))
 settled=[wait_for(lambda id=r['id']:reservation(id),lambda x:x['status'] in ('CONFIRMED','REJECTED')) for r in results]
assert sorted(x['status'] for x in settled)==['CONFIRMED','REJECTED']
assert balances(f)=={'quantity':85,'reserved':60,'available':25}
winner=next(x for x in settled if x['status']=='CONFIRMED')
request('reservation','/reservations/'+winner['id'],method='DELETE')
wait_for(lambda:reservation(winner['id']),lambda x:x['status']=='CANCELLED')
assert balances(f)['reserved']==0
print('PASS: concurrent reservation and cancellation')
short=reserve(f,1,3)
wait_for(lambda:reservation(short['id']),lambda x:x['status'] in ('CONFIRMED','EXPIRED','REJECTED'))
wait_for(lambda:reservation(short['id']),lambda x:x['status']=='EXPIRED',timeout=30)
assert balances(f)['reserved']==0
print('PASS: reservation expiry releases stock')
rejects(lambda:request('shipment',f"/shipments/{f['shipment']['id']}/ship",{}))
assert balances(f)['quantity']==85
key=str(uuid.uuid4());payload={'warehouseId':f['warehouse']['id'],'productId':f['product']['id'],'quantity':2,'batchNumber':'IDEMPOTENCY'}
a=request('inventory','/inventory/receive',payload,key=key);b=request('inventory','/inventory/receive',payload,key=key)
assert a['id']==b['id'];rejects(lambda:request('inventory','/inventory/receive',{**payload,'quantity':3},key=key))
rejects(lambda:request('inventory','/inventory/receive',{**payload,'quantity':-1}),400)
wait_for(lambda:request('inventory','/inventory/lots/'+a['id']),lambda x:x['locationId'] is not None)
assert balances(f)['quantity']==87
print('PASS: HTTP idempotency, payload mismatch, negative quantity, double shipment')
for code,capacity,status in [('FULL',0.1,'AVAILABLE'),('BLOCK',1000,'BLOCKED')]:
 loc=request('warehouse',f"/racks/{f['rack']['id']}/locations",{'code':code,'maxWeight':capacity,'maxVolume':10,'distance':0,'storageType':'NORMAL'})
 if status=='BLOCKED':request('warehouse','/locations/'+loc['id']+'/status',{'status':status},method='PATCH')
 rejects(lambda:request('inventory','/inventory/move',{'stockId':a['id'],'toLocationId':loc['id'],'quantity':1}))
assert balances(f)['quantity']==87
print('PASS: blocked/full destination leaves stock unchanged')
# Replay the original GoodsReceived envelope using its eventId.
rows=wait_for(lambda:request('audit','/audit/products/'+f['product']['id']),lambda xs:any(x['event_type']=='GoodsReceived' and x['payload']['stockId']==a['id'] for x in xs))
e=next(x for x in rows if x['event_type']=='GoodsReceived' and x['payload']['stockId']==a['id'])
event={'eventId':e['event_id'],'eventType':e['event_type'],'eventVersion':1,'timestamp':e['timestamp'],'source':e['source'],'correlationId':e['correlation_id'],'traceparent':'','payload':e['payload']}
subprocess.run(['docker','compose','exec','-T','kafka','/opt/kafka/bin/kafka-console-producer.sh','--bootstrap-server','kafka:9092','--topic','wms.events','--property','parse.key=true','--property','key.separator=\t'],input=f['warehouse']['id']+'\t'+json.dumps(event)+'\n',text=True,check=True,timeout=30)
time.sleep(3)
assert balances(f)['quantity']==87
rows=request('audit','/audit/products/'+f['product']['id'])
assert sum(x['event_id']==event['eventId'] for x in rows)==1
print('PASS: duplicate Kafka event is idempotent')
print('ALL ACCEPTANCE CHECKS PASSED')
