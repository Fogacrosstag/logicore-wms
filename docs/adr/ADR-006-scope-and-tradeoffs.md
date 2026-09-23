# ADR-006: Portfolio scope and deliberate trade-offs

Status: Accepted

## Decision and consequences

This project uses the Go standard-library HTTP router instead of an additional web framework. PostgreSQL, not Redis, durably owns idempotency and reservations. Product physical properties are immutable to preserve capacity accounting. Lists are capped at 500 rows except paginated Audit. Administrative location blocking has a documented check/commit race; distributed fencing is future work. One receipt must fit one location. No external authentication, payment/order integration, distributed database transaction, inventory valuation, multi-tenant authorization, load-test claim or production SLA is provided. Kafka and databases have single-node local topology. Publishers must not be scaled assuming per-aggregate ordering is already solved.
