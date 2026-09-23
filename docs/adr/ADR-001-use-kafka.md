# ADR-001: Kafka as the event backbone

Status: Accepted

## Decision and consequences

Separate consumer groups let audit, placement and lifecycle projections consume the same committed facts independently. JSON is inspectable and adequate for this portfolio. The cost is eventual consistency, operational overhead and at-least-once delivery. Local KRaft avoids a separate ZooKeeper service. A production cluster would require replication and authentication.
