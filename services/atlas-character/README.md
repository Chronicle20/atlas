# atlas-character

Character management service for the Atlas game platform. This service handles character lifecycle, stats, appearance, progression, and position tracking. It provides both REST APIs for synchronous operations and Kafka messaging for asynchronous event-driven workflows.

The service coordinates with external services for skill management, drop handling, and session tracking through Kafka messaging.

## External Dependencies

- PostgreSQL database
- Kafka message broker
- Jaeger tracing (optional)
- atlas-skill service (via Kafka)
- atlas-drop service (via Kafka)
- atlas-session service (via Kafka events)

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| JAEGER_HOST | Jaeger host:port |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| DB_USER | Postgres user name |
| DB_PASSWORD | Postgres user password |
| DB_HOST | Postgres database host |
| DB_PORT | Postgres database port |
| DB_NAME | Postgres database name |
| BASE_SERVICE_URL | Base service URL (scheme://host:port/api/) |
| BOOTSTRAP_SERVERS | Kafka host:port |
| REST_PORT | REST server port |
| COMMAND_TOPIC_CHARACTER | Character commands topic |
| COMMAND_TOPIC_CHARACTER_MOVEMENT | Character movement commands topic |
| COMMAND_TOPIC_DROP | Drop commands topic |
| COMMAND_TOPIC_SKILL | Skill commands topic |
| EVENT_TOPIC_CHARACTER_STATUS | Character status events topic |
| EVENT_TOPIC_SESSION_STATUS | Session status events topic |
| EVENT_TOPIC_DROP_STATUS | Drop status events topic |
| EVENT_TOPIC_ACCOUNT_STATUS | Account status events topic (consumed) |

## Documentation

- [Domain](docs/domain.md) - Domain models and processors
- [Kafka](docs/kafka.md) - Kafka integration
- [REST](docs/rest.md) - REST API endpoints
- [Storage](docs/storage.md) - Database schema
