# ADR-005: Language boundaries

Status: Accepted

## Decision and consequences

Java/Spring is used for JPA-backed catalogue and workflow APIs, validation and generated OpenAPI. Go is used for transaction-heavy stock logic and durable event workers. Each application has a distinct executable. Shared infrastructure libraries reduce duplication but are released from the same monorepo; a future independent-release system would version them separately.
