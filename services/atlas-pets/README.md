# Atlas Pets Service

A microservice for managing in-game pets. This service handles pet creation, retrieval, spawning, despawning, attribute management, and lifecycle operations.

## External Dependencies

- PostgreSQL database
- Kafka cluster
- Jaeger (tracing)

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| JAEGER_HOST | Jaeger host and port for tracing |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| REST_PORT | Port for the REST API |
| DB_HOST | PostgreSQL database host |
| DB_PORT | PostgreSQL database port |
| DB_USER | PostgreSQL database username |
| DB_PASS | PostgreSQL database password |
| DB_NAME | PostgreSQL database name |
| KAFKA_BROKERS | Comma-separated list of Kafka brokers |
| EVENT_TOPIC_CHARACTER_STATUS | Character status events topic |
| EVENT_TOPIC_ASSET_STATUS | Asset status events topic |
| COMMAND_TOPIC_PET | Pet commands topic |
| COMMAND_TOPIC_PET_MOVEMENT | Pet movement commands topic |
| EVENT_TOPIC_PET_STATUS | Pet status events topic |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
