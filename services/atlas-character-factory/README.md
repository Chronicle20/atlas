# atlas-character-factory

Character creation service using saga-based orchestration. Validates character creation requests against tenant-configured templates and delegates creation to the Atlas Saga Orchestrator. Coordinates two-phase character creation: initial character creation followed by awarding items, equipment, and skills.

## External Dependencies

- Kafka (message broker for saga commands and events)
- Jaeger (distributed tracing)
- Atlas Configuration Service (tenant configuration)
- Atlas Character Service (character creation)
- Atlas Saga Orchestrator (saga execution)

## Runtime Configuration

| Variable                       | Description                                    |
|--------------------------------|------------------------------------------------|
| JAEGER_HOST                    | Jaeger host:port                               |
| LOG_LEVEL                      | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| BOOTSTRAP_SERVERS              | Kafka host:port                                |
| SERVICE_ID                     | Service identifier (UUID)                      |
| REST_PORT                      | REST API port                                  |
| EVENT_TOPIC_CHARACTER_STATUS   | Topic for character status events              |
| EVENT_TOPIC_SAGA_STATUS        | Topic for saga status events                   |
| COMMAND_TOPIC_SAGA             | Topic for saga commands                        |
| EVENT_TOPIC_SEED_STATUS        | Topic for seed status events                   |

## Documentation

- [Domain](docs/domain.md)
- [Kafka Integration](docs/kafka.md)
- [REST API](docs/rest.md)
- [Storage](docs/storage.md)
