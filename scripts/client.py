"""Dependency-free API client used by the demo and acceptance suite."""
import json, time, uuid
from urllib.request import Request, urlopen
from urllib.error import HTTPError, URLError
PORTS={'warehouse':8081,'product':8082,'inventory':8083,'placement':8084,'reservation':8085,'shipment':8086,'audit':8087}
class APIError(Exception):
 def __init__(self,status,body):
  self.status,self.body=status,body
  super().__init__(f'{status}: {body}')
def request(service,path,body=None,method=None,key=None):
 method=method or ('POST' if body is not None else 'GET')
 headers={'Content-Type':'application/json','X-Correlation-ID':str(uuid.uuid4())}
 if method not in ('GET','HEAD'):headers['Idempotency-Key']=key or str(uuid.uuid4())
 req=Request(f'http://127.0.0.1:{PORTS[service]}'+path,data=json.dumps(body).encode() if body is not None else None,headers=headers,method=method)
 try:
  with urlopen(req,timeout=15) as response:
   result=json.load(response)
   return result.get('data',result)
 except HTTPError as e:
  raise APIError(e.code,e.read().decode()) from e

def wait_for(fn,predicate,timeout=90,description='condition'):
 deadline=time.monotonic()+timeout
 last=None
 while time.monotonic()<deadline:
  last=fn()
  if predicate(last):return last
  time.sleep(0.5)
 raise AssertionError(f'Timed out waiting for {description}: {last}')
def wait_services(timeout=180):
 deadline=time.monotonic()+timeout
 for name in PORTS:
  while True:
   try:
    value=request(name,'/health')
    if value.get('status')=='UP':break
   except (APIError,URLError,TimeoutError):pass
   if time.monotonic()>deadline:raise TimeoutError(f'{name} did not become healthy')
   time.sleep(1)
def fixture(label='Demo'):
 suffix=uuid.uuid4().hex[:8]
 warehouse=request('warehouse','/warehouses',{'code':f'LON-{suffix}','name':f'London Distribution Centre / {label}','address':'London, UK'})
 zones={}
 for name in ['RECEIVING','STORAGE','PICKING','PACKING','SHIPPING']:
  zones[name]=request('warehouse',f"/warehouses/{warehouse['id']}/zones",{'code':name,'name':name.title(),'type':name})
 rack=request('warehouse',f"/zones/{zones['STORAGE']['id']}/racks",{'code':'R-01'})
 locations=[]
 for i in range(2):
  locations.append(request('warehouse',f"/racks/{rack['id']}/locations",{'code':f'R-01-A-0{i+1}','maxWeight':1000,'maxVolume':20,'distance':i*10,'storageType':'NORMAL'}))
 product=request('product','/products',{'sku':f'LAPTOP-{suffix}','name':'Business Laptop','description':'Demonstration product','category':'Electronics','barcode':suffix,'weight':2.2,'length':0.4,'width':0.3,'height':0.1,'storageType':'NORMAL'})
 return {'warehouse':warehouse,'product':product,'rack':rack,'locations':locations}
def balances(f):
 rows=request('inventory',f"/inventory/products/{f['product']['id']}")
 return {'quantity':sum(x['quantity'] for x in rows),'reserved':sum(x['reservedQuantity'] for x in rows),'available':sum(x['availableQuantity'] for x in rows)}
def reserve(f,quantity,ttl=1800):
 return request('reservation','/reservations',{'warehouseId':f['warehouse']['id'],'items':[{'productId':f['product']['id'],'quantity':quantity}],'ttlSeconds':ttl})
def reservation(id):return request('reservation','/reservations/'+id)
