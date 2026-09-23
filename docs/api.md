# API cookbook

The automated runnable equivalent is `python3 scripts/demo.py`. In the manual examples, copy actual IDs from previous responses into the variables; never send the literal placeholders. Use a fresh idempotency key for each distinct operation.

## Create layout

```bash
curl -sS http://localhost:8081/warehouses -H 'Content-Type: application/json' -H 'Idempotency-Key: manual-warehouse-1' -d '{"code":"LONDON-01","name":"London Distribution Centre","address":"London, UK"}'
WAREHOUSE_ID='<returned warehouse id>'
curl -sS "http://localhost:8081/warehouses/$WAREHOUSE_ID/zones" -H 'Content-Type: application/json' -H 'Idempotency-Key: manual-zone-1' -d '{"code":"STORAGE","name":"Storage","type":"STORAGE"}'
ZONE_ID='<returned zone id>'
curl -sS "http://localhost:8081/zones/$ZONE_ID/racks" -H 'Content-Type: application/json' -H 'Idempotency-Key: manual-rack-1' -d '{"code":"R-01"}'
RACK_ID='<returned rack id>'
curl -sS "http://localhost:8081/racks/$RACK_ID/locations" -H 'Content-Type: application/json' -H 'Idempotency-Key: manual-location-1' -d '{"code":"R-01-A-04","maxWeight":1000,"maxVolume":20,"distance":5,"storageType":"NORMAL"}'
```

## Create and receive a product

```bash
curl -sS http://localhost:8082/products -H 'Content-Type: application/json' -H 'Idempotency-Key: manual-product-1' -d '{"sku":"LAPTOP-001","name":"Business Laptop","description":"Demo laptop","category":"Electronics","barcode":"001","weight":2.2,"length":0.4,"width":0.3,"height":0.1,"storageType":"NORMAL"}'
PRODUCT_ID='<returned product id>'
curl -sS http://localhost:8083/inventory/receive -H 'Content-Type: application/json' -H 'Idempotency-Key: manual-receive-1' -d "{\"warehouseId\":\"$WAREHOUSE_ID\",\"productId\":\"$PRODUCT_ID\",\"quantity\":100,\"batchNumber\":\"BATCH-001\"}"
STOCK_ID='<returned stock id>'
curl -sS "http://localhost:8083/inventory/lots/$STOCK_ID"
curl -sS "http://localhost:8084/placements/$STOCK_ID"
```

Wait until the lot's `locationId` is non-null. Physical quantity includes unplaced lots, but only placed, unexpired stock is eligible for reservation.

## Reserve and ship

```bash
curl -sS http://localhost:8085/reservations -H 'Content-Type: application/json' -H 'Idempotency-Key: manual-reserve-1' -d "{\"warehouseId\":\"$WAREHOUSE_ID\",\"items\":[{\"productId\":\"$PRODUCT_ID\",\"quantity\":15}]}"
RESERVATION_ID='<returned reservation id>'
curl -sS "http://localhost:8085/reservations/$RESERVATION_ID"
# Wait until status is CONFIRMED.
curl -sS http://localhost:8086/shipments -H 'Content-Type: application/json' -H 'Idempotency-Key: manual-shipment-1' -d "{\"reservationId\":\"$RESERVATION_ID\"}"
SHIPMENT_ID='<returned shipment id>'
for action in start-picking pack ready ship; do
 curl -sS -X POST "http://localhost:8086/shipments/$SHIPMENT_ID/$action" -H "Idempotency-Key: manual-$SHIPMENT_ID-$action"
done
curl -sS "http://localhost:8086/shipments/$SHIPMENT_ID"
curl -sS "http://localhost:8083/inventory/products/$PRODUCT_ID"
curl -sS "http://localhost:8087/audit/products/$PRODUCT_ID"
```

Wait for `SHIPPED`; the stock total is then 85. A transport-level success on a workflow command is not yet business completion.

## Other stock operations

```json
POST /inventory/move
{"stockId":"UUID","toLocationId":"UUID","quantity":10}

POST /inventory/write-off
{"stockId":"UUID","quantity":2,"reason":"Damaged packaging"}

POST /inventory/adjust
{"stockId":"UUID","quantity":83,"reason":"Verified cycle count"}
```

`adjust.quantity` is the **absolute counted quantity**, not a delta. All three requests require `Idempotency-Key`. Movement targets a stock lot so batch/expiration identity is unambiguous.

## Errors

```json
{"code":"INSUFFICIENT_STOCK","message":"Not enough available stock","details":{},"timestamp":"2026-09-17T10:00:00Z","traceId":"..."}
```

Business conflicts use 409, input errors 400, absent resources 404 and dependency failures 503 where classified. Reservation rejections arrive asynchronously as lifecycle state plus reason.

## List limits

Catalogue/layout/inventory/placement lists return at most 500 rows. Audit supports `limit` (1–500), `offset` and `eventType`. Large-scale cursor pagination is a documented future improvement. The demo stays far below those limits.
