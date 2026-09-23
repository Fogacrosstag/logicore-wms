# ADR-004: Inventory is the stock authority

Status: Accepted

## Decision and consequences

Inventory owns physical stock, allocations and the capacity ledger. Reservation and Shipment expose asynchronous workflow projections. PostgreSQL warehouse-scoped advisory transaction locks serialize conflicting stock mutations. Redis is an optimization, not a stock correctness dependency. Coarse locking favours explainability over maximum throughput.
