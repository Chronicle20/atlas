# atlas-equipables

A RESTful microservice that manages equipment instances for characters. This service creates, retrieves, updates, and deletes equipable items with their associated statistics and properties.

## External Dependencies

- PostgreSQL database for persistent storage
- Kafka for messaging (command consumption and event production)
- atlas-data service for equipment template information
- Jaeger for distributed tracing

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
| REST_PORT | HTTP server port |
| COMMAND_TOPIC_EQUIPABLE | Kafka topic for equipable commands |
| EVENT_TOPIC_EQUIPABLE_STATUS | Kafka topic for equipable status events |

## Documentation

- [Domain](docs/domain.md)
- [REST API](docs/rest.md)
- [Kafka Integration](docs/kafka.md)
- [Storage](docs/storage.md)
