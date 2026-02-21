# atlas-portal-actions

Portal script execution service that processes JSON-based rules to control portal entry behaviors.

The `atlas-portal-actions` service manages portal scripts that determine whether characters can use portals and what operations should be executed when they do. Scripts consist of rules with conditions that are evaluated in order, with the first matching rule determining the outcome.

## External Dependencies

- PostgreSQL (GORM)
- Redis (pending action registry)
- Kafka
- OpenTelemetry (tracing)
- Character validation service (HTTP)

## Configuration

| Environment Variable | Description |
|---------------------|-------------|
| REST_PORT | HTTP server port |
| PORTAL_SCRIPTS_DIR | Directory containing portal script JSON files (default: /scripts/portals) |
| COMMAND_TOPIC_PORTAL_ACTIONS | Kafka topic for portal entry commands |
| EVENT_TOPIC_CHARACTER_STATUS | Kafka topic for character status events |
| COMMAND_TOPIC_SAGA | Kafka topic for saga commands |
| EVENT_TOPIC_SAGA_STATUS | Kafka topic for saga status events |
| QUERY_AGGREGATOR | Base URL for character validation service |
| BOOTSTRAP_SERVERS | Kafka broker addresses |
| DB_USER | PostgreSQL user |
| DB_PASSWORD | PostgreSQL password |
| DB_HOST | PostgreSQL host |
| DB_PORT | PostgreSQL port |
| DB_NAME | PostgreSQL database name |
| TRACE_ENDPOINT | OpenTelemetry collector endpoint |
| LOG_LEVEL | Log level (default: info) |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
