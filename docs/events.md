# Event contract v1

Topic `wms.events`; dead-letter topic `wms.events.DLT`; three partitions each. Each service has its own consumer group named after the service. Default retention is seven days. The local broker has replication factor one.

```json
{
  "eventId": "2c8faaba-bab0-4bda-a60f-850aa262054f",
  "eventType": "GoodsReceived",
  "eventVersion": 1,
  "timestamp": "2026-09-17T10:00:00Z",
  "source": "inventory-service",
  "correlationId": "9683e96b-a7c9-4f90-8dc4-594db42fbdaa",
  "traceparent": "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
  "payload": {
    "stockId": "c53fa043-7194-44d6-9849-ac488f97fe68",
    "productId": "bfde25a4-f11d-4bc6-b68f-f71d026aaee9",
    "warehouseId": "d5859798-930c-4f14-8a85-2baa81d69e71",
    "quantity": 100,
    "weight": 2.2,
    "volume": 0.012,
    "storageType": "NORMAL"
  }
}
```

`traceparent` may be empty if no parent span exists. Dates use UTC; `eventId` is stable across retries. Unknown event types are ignored by unrelated consumers; unsupported envelope versions fail and enter the recovery path.

| Event | Producer | Consumers | Payload fields |
|---|---|---|---|
| GoodsReceived | Inventory | Placement, Audit | stockId, productId, warehouseId, quantity, weight, volume, storageType |
| ProductPlacementAssigned | Placement | Inventory, Audit | stockId, locationId, warehouseId, score |
| PlacementRejected | Inventory | Placement, Audit | stockId, warehouseId, reason |
| StockAdded | Inventory | Placement, Audit | stockId, productId, locationId, quantity |
| LocationCapacityChanged | Inventory | Warehouse, Audit | locationId, warehouseId, currentWeight, currentVolume, revision |
| ReservationRequested | Reservation | Inventory, Audit | reservationId, warehouseId, items, expiresAt |
| StockReserved | Inventory | Reservation, Audit | reservationId, warehouseId, items, expiresAt |
| ReservationRejected | Inventory | Reservation, Audit | reservationId, warehouseId, reason |
| ReservationCancellationRequested | Reservation or Shipment | Inventory, Audit | reservationId, warehouseId, optional shipmentId |
| StockReservationCancelled | Inventory | Reservation, Shipment, Audit | reservationId, warehouseId |
| ReservationExpired | Inventory | Reservation, Shipment, Audit | reservationId, warehouseId |
| ReservationCancellationRejected | Inventory | Audit | reservationId, warehouseId, reason |
| ShipmentCreated | Shipment | Audit | shipmentId, reservationId, warehouseId, items |
| ShipmentStatusChanged | Shipment | Audit | shipmentId, warehouseId, status |
| ShipmentRequested | Shipment | Inventory, Audit | shipmentId, reservationId, warehouseId |
| StockShipped | Inventory | Reservation, Shipment, Audit | shipmentId, reservationId, warehouseId, items |
| ShipmentRejected | Inventory | Shipment, Audit | shipmentId, reservationId, warehouseId, reason |
| ShipmentCompleted | Shipment | Audit | shipmentId, reservationId, warehouseId, items |
| StockMoved | Inventory | Audit | stockId, destinationStockId, productId, warehouseId, quantity, fromLocationId, toLocationId |
| StockWrittenOff | Inventory | Audit | stockId, productId, warehouseId, quantity, reason |
| StockAdjusted | Inventory | Audit | stockId, productId, warehouseId, quantity, delta, reason |

Catalogue/layout events are also audited: `ProductCreated`, `ProductUpdated`, `ProductDeactivated`, `WarehouseCreated`, `WarehouseZoneCreated`, `RackCreated`, `WarehouseLocationCreated`, `WarehouseLocationStatusChanged`.

Payloads are additive within v1. A breaking change requires a new envelope version and a rollout plan. The consumer inbox is not purged automatically; retention must cover any supported replay horizon.

Occupancy events carry absolute weight/volume and a monotonic location revision. Warehouse ignores old revisions. ShipmentCompleted is a reporting event; Inventory deducts stock in response to ShipmentRequested and emits StockShipped first.
