# ADR-002: Database ownership

Status: Accepted

## Decision and consequences

Every service gets a private database and migrations. Sharing PostgreSQL tables across services would hide coupling and bypass event/API contracts. The additional resource cost is accepted for an explicit demonstration of ownership. Audit and placement also persist their own state.
