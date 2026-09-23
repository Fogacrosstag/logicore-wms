#!/usr/bin/env python3
"""Populate three warehouses and five products. Safe to rerun using stable keys."""
from client import request,wait_services
wait_services()
for code,name in [('LON','London Distribution Centre'),('MAN','Manchester Fulfilment Centre'),('BHM','Birmingham Warehouse')]:
 w=request('warehouse','/warehouses',{'code':code,'name':name,'address':name.split()[0]+', UK'},key='seed-warehouse-'+code)
 for zone_type in ['RECEIVING','STORAGE','PICKING','PACKING','SHIPPING']:
  z=request('warehouse',f"/warehouses/{w['id']}/zones",{'code':zone_type,'name':zone_type.title(),'type':zone_type},key=f'seed-zone-{code}-{zone_type}')
  if zone_type!='STORAGE':continue
  r=request('warehouse',f"/zones/{z['id']}/racks",{'code':'R-01'},key='seed-rack-'+code)
  for n in range(1,5):
   request('warehouse',f"/racks/{r['id']}/locations",{'code':f'R-01-A-{n:02d}','maxWeight':1000,'maxVolume':30,'distance':n*5,'storageType':'NORMAL'},key=f'seed-location-{code}-{n}')
for sku,name,weight in [('LAPTOP-001','Laptop',2.2),('MONITOR-001','Monitor',5),('KEYBOARD-001','Keyboard',0.5),('CHAIR-001','Office Chair',12),('PRINTER-001','Printer',8)]:
 request('product','/products',{'sku':sku,'name':name,'description':'Seed product','category':'Office','barcode':sku,'weight':weight,'length':0.5,'width':0.4,'height':0.3,'storageType':'NORMAL'},key='seed-product-'+sku)
print('Created 3 warehouses, 15 zones, 3 racks, 12 locations and 5 products.')
