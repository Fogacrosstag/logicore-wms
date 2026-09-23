#!/usr/bin/env bash
set -euo pipefail
for topic in wms.events wms.events.DLT; do
 /opt/kafka/bin/kafka-topics.sh --bootstrap-server kafka:9092 --create --if-not-exists --topic "$topic" --partitions 3 --replication-factor 1 --config retention.ms=604800000
done
