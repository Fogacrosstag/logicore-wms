# Operations and recovery

## Startup failures

- `docker compose config` validates interpolation; run `python3 scripts/init_env.py` first.
- `docker compose ps` identifies unhealthy containers.
- `docker compose logs warehouse-service inventory-service kafka` shows startup/migration failures.
- Initial Java image builds download Maven dependencies; allow several minutes and sufficient RAM.
- Changing `.env` passwords does not change passwords in existing PostgreSQL volumes. Preserve credentials or deliberately reset disposable demo volumes.

## Pending placement

Inspect `GET :8084/placements`. `NO_SUITABLE_LOCATION` means no single compatible STORAGE location can fit the full receipt. Add capacity or an appropriate storage class. Inventory rechecks the decision against its ledger; a rejected assignment returns to pending.

## Pending reservation/shipment

Inspect inventory and workflow logs, consumer lag, unpublished outbox rows and the DLT. A pending workflow is not silently treated as successful. Do not manually decrement stock to complete a stuck shipment.

```bash
docker compose exec kafka /opt/kafka/bin/kafka-consumer-groups.sh --bootstrap-server kafka:9092 --all-groups --describe
docker compose exec kafka /opt/kafka/bin/kafka-console-consumer.sh --bootstrap-server kafka:9092 --topic wms.events.DLT --from-beginning --max-messages 10 --property print.headers=true
```

Go DLT messages include consumer and error headers. Spring's recoverer adds its standard original-topic/exception headers. Preserve the original payload and eventId when replaying after fixing the cause. Never replay an entire DLT blindly: a malformed event may fail repeatedly, and the payload can reveal business data.

To replay a reviewed single JSON event saved as `event.json`:

```bash
python3 scripts/replay_event.py event.json
```

All consumers receive it; already successful consumers skip it through their inbox. Failed consumers have no committed inbox marker and can retry the business transaction. If a process crashed after publishing but before marking the outbox, delivery may be duplicated normally without operator action.

## Useful metrics

```promql
up
sum by(service, operation, outcome) (rate(wms_operations_total[5m]))
sum by(job) (rate(kafka_messages_processed_total[5m]))
histogram_quantile(0.95, sum by(job, le) (rate(http_server_requests_seconds_bucket[5m])))
```

Metrics record handler attempts/outcomes and are not an accounting ledger; retries/rolled-back transactions can affect operational counters. Audit/stock tables are the authoritative business record.

## Local data and security

This local topology has no API auth, TLS or broker ACLs. Ports bind to localhost. A real deployment needs explicit authentication, least-privilege database roles, protected observability endpoints, backup/restore drills and a secret manager. Do not publish `.env`.

## Retention

Outbox, inbox, HTTP command and audit rows are retained indefinitely for this portfolio. Operational pruning must preserve the supported idempotency/replay horizon. Kafka retains events for seven days. Jaeger uses in-memory storage in this local setup.
