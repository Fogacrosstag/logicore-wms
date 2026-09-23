# Interview and live-demo guide

## 30-second introduction

LogiCore WMS is a seven-service warehouse backend using Java and Go. The interesting part is inventory correctness across asynchronous workflows: two requests cannot reserve the same final unit, a retried message cannot repeat a stock mutation, and a shipment is confirmed only after Inventory has committed the deduction.

## Five-minute walkthrough

1. **Problem and boundaries (45 sec).** Show the service table and explain Inventory ownership.
2. **Working scenario (90 sec).** Run `python3 scripts/demo.py`; show 100 → 15 reserved → 85 remaining. Open the audit endpoint.
3. **Correctness (60 sec).** Show the warehouse transaction lock, allocation loop and database constraints. Explain that Redis is not the overselling guard.
4. **Failure handling (60 sec).** Show outbox/inbox SQL. Describe a crash after Kafka acknowledgement but before marking the row published; the same ID is delivered again and deduplicated.
5. **Evidence and trade-offs (45 sec).** Open CI results, integration tests and ADR-006. Explain the administrative blocking race and coarse-lock throughput cost candidly.

## Questions worth preparing

- Why seven services? To demonstrate explicit domain ownership and polyglot integration; a small real team might start with fewer services.
- Why not a Redis lock for stock? Its lease can expire during an operation. PostgreSQL commits the guarded state and constraints together.
- Is this exactly once? Delivery is at least once. Local effects are idempotent within each consumer's transaction.
- What if the broker is down? APIs can commit state plus outbox while Kafka delivery waits; asynchronous workflows stay pending.
- What if a reservation expires while shipping? Inventory serializes both operations and checks expiry under the same warehouse lock.
- Can a stale placement decision overfill a location? It is rechecked against the authoritative capacity ledger before commit.
- What remains to productionize? Authentication, fencing for administrative changes, retention, higher-availability infrastructure, operational recovery automation and performance measurements.

## Demonstration recording

Record the terminal demo, one audit query, the architecture diagram, the concurrency test and a Grafana/Jaeger view. Do not describe skipped or unexecuted tests as passing. Screenshots should come from your own successful run.
