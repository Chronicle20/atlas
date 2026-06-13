# Atlas Mounts Service

A microservice for managing per-character mount progression. This service maintains a single mount progression record per character (level, exp, tiredness) and advances tiredness over time for active (tamed) mounts. It exposes the current mount progression over a read-only REST endpoint.

The service consumes character buff status events, taming-mob food events, and character status events via Kafka, and produces mount status events. Mount progression is persisted in PostgreSQL. A Redis-backed registry tracks which characters currently have an active tamed mount so a periodic ticker can advance their tiredness.

## External Dependencies

- PostgreSQL database
- Kafka cluster
- Redis
- OpenTelemetry (tracing via OTLP)

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| TRACE_ENDPOINT | OTLP trace exporter endpoint |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| REST_PORT | Port for the REST API |
| DB_HOST | PostgreSQL database host |
| DB_PORT | PostgreSQL database port |
| DB_USER | PostgreSQL database username |
| DB_PASSWORD | PostgreSQL database password |
| DB_NAME | PostgreSQL database name |
| BOOTSTRAP_SERVERS | Kafka broker address |
| EVENT_TOPIC_CHARACTER_BUFF_STATUS | Character buff status events topic (consumed) |
| EVENT_TOPIC_TAMING_MOB_FOOD | Taming-mob food events topic (consumed) |
| EVENT_TOPIC_CHARACTER_STATUS | Character status events topic (consumed) |
| EVENT_TOPIC_MOUNT_STATUS | Mount status events topic (produced) |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
