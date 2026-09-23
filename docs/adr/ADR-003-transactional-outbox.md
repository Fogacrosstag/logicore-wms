# ADR-003: Transactional outbox and inbox

Status: Accepted

## Decision and consequences

Every state-changing use case writes its outbox in the same local database transaction. Publishers retry until Kafka acknowledges. Consumers commit inbox markers with their domain effects. This closes the write/publish gap while accepting duplicate delivery; it does not create a distributed exactly-once transaction.
