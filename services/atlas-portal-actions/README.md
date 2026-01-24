# atlas-portal-actions

Portal script execution service that processes JSON-based rules to control portal entry behaviors.

The `atlas-portal-actions` service manages portal scripts that determine whether characters can use portals and what operations should be executed when they do. Scripts consist of rules with conditions that are evaluated in order, with the first matching rule determining the outcome.

## External Dependencies

- PostgreSQL (GORM)
- Kafka
- OpenTelemetry (tracing)
- Character validation service (HTTP)

## Configuration

| Environment Variable | Description |
|---------------------|-------------|
| REST_PORT | HTTP server port |
| PORTAL_SCRIPTS_DIR | Directory containing portal script JSON files (default: /scripts/portals) |
| COMMAND_TOPIC_PORTAL_ACTIONS | Kafka topic for portal commands |
| EVENT_TOPIC_CHARACTER_STATUS | Kafka topic for character status events |
| COMMAND_TOPIC_CHARACTER | Kafka topic for character commands |
| COMMAND_TOPIC_SAGA | Kafka topic for saga commands |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
