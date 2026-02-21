# Atlas Family Service

Manages hierarchical character relationships (senior-junior) and reputation tracking for family members.

## External Dependencies

- PostgreSQL: Primary database (SQLite supported for testing)
- Kafka: Message broker for command consumption and event production
- OpenTelemetry: Distributed tracing (OTLP/gRPC export)

## Runtime Configuration

### Database

| Variable | Description |
|----------|-------------|
| DB_HOST | Database host |
| DB_PORT | Database port |
| DB_USER | Database username |
| DB_PASSWORD | Database password |
| DB_NAME | Database name |
| DB_SCHEMA | Database schema |

### Kafka

| Variable | Description |
|----------|-------------|
| BOOTSTRAP_SERVERS | Kafka broker address |
| COMMAND_TOPIC_FAMILY | Family command topic |
| EVENT_TOPIC_FAMILY_STATUS | Family status event topic |
| EVENT_TOPIC_FAMILY_REPUTATION | Family reputation event topic |
| EVENT_TOPIC_FAMILY_ERRORS | Family error event topic |
| EVENT_TOPIC_CHARACTER_STATUS | Character status event topic (consumed) |

### Scheduler

| Variable | Description |
|----------|-------------|
| REPUTATION_RESET_HOUR | Hour for daily reset (0-23) |
| REPUTATION_RESET_MINUTE | Minute for daily reset (0-59) |
| REPUTATION_RESET_TIMEZONE | Timezone for reset |

### Logging and Tracing

| Variable | Description |
|----------|-------------|
| LOG_LEVEL | Logging level |
| TRACE_ENDPOINT | OpenTelemetry collector endpoint |
| REST_PORT | REST server port |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
