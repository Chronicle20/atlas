# Kafka

## Topics Consumed

None.

## Topics Produced

| Topic (env var) | Direction | Trigger |
|------------------|-----------|---------|
| `EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS` | event | Service configuration created, updated, or deleted; also republished on startup backfill |
| `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS` | event | Tenant configuration created, updated, or deleted; also republished on startup backfill |

Publishing to a topic is skipped (no-op) when its env var is unset.

## Message Types

**envelope** (internal wire shape shared by both topics)
- `schema_version` (int) - envelope schema version, currently `1`
- `id` (string) - UUID of the service or tenant configuration
- `config` (object, nullable) - the REST model the record should be reconstructed from; `null` on delete (tombstone)
- `emitted_at` (string) - RFC3339 UTC timestamp

Message key: `service:<id>` for `EVENT_TOPIC_CONFIGURATION_SERVICE_STATUS`, `tenant:<id>` for `EVENT_TOPIC_CONFIGURATION_TENANT_STATUS`.

## Transaction Semantics

Messages are written to a transactional outbox table in the same database transaction as the triggering create, update, or delete, then asynchronously published to Kafka by a background drainer. A Postgres advisory lock gates which replica's drainer publishes when multiple replicas are running.

On startup, the seeder backfills the outbox from all existing service and tenant rows so a cold-start or a cluster recovering from a wiped Kafka topic has a complete snapshot to publish. Backfill is idempotent on (topic, key).
