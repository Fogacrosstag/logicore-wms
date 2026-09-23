#!/usr/bin/env python3
"""Run the complete business workflow and assert its observable results."""
import json
from client import request,wait_for,wait_services,fixture,reserve,reservation,balances

def run_demo():
 wait_services()
 f=fixture()
 print('1. Created warehouse, five zones, rack, two locations and product')
 lot=request('inventory','/inventory/receive',{'warehouseId':f['warehouse']['id'],'productId':f['product']['id'],'quantity':100,'batchNumber':'DEMO-001'})
 placed=wait_for(lambda:request('inventory','/inventory/lots/'+lot['id']),lambda x:x['locationId'] is not None,description='automatic placement')
 assert balances(f)=={'quantity':100,'reserved':0,'available':100}
 print('2. Received and automatically placed 100 units:',placed['locationId'])
 r=reserve(f,15)
 wait_for(lambda:reservation(r['id']),lambda x:x['status']=='CONFIRMED',description='reservation')
 assert balances(f)=={'quantity':100,'reserved':15,'available':85}
 print('3. Reserved 15 units:',balances(f))
 other=next(x for x in f['locations'] if x['id']!=placed['locationId'])
 request('inventory','/inventory/move',{'stockId':lot['id'],'toLocationId':other['id'],'quantity':10})
 assert balances(f)=={'quantity':100,'reserved':15,'available':85}
 print('4. Moved 10 available units to a second location')
 shipment=request('shipment','/shipments',{'reservationId':r['id']})
 for action in ['start-picking','pack','ready','ship']:
  request('shipment',f"/shipments/{shipment['id']}/{action}",{},method='POST')
 wait_for(lambda:request('shipment','/shipments/'+shipment['id']),lambda x:x['status']=='SHIPPED',description='shipment confirmation')
 assert balances(f)=={'quantity':85,'reserved':0,'available':85}
 events=wait_for(lambda:request('audit','/audit/products/'+f['product']['id']),lambda xs:any(x['event_type']=='ShipmentCompleted' for x in xs),description='audit history')
 print('5. Shipped 15 units:',balances(f))
 print('6. Audit events:',', '.join(x['event_type'] for x in events))
 f.update({'stock':lot,'shipment':shipment,'reservation':r})
 print(json.dumps({'warehouseId':f['warehouse']['id'],'productId':f['product']['id'],'shipmentId':shipment['id']},indent=2))
 return f
if __name__=='__main__':run_demo()
