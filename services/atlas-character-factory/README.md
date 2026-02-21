# atlas-character-factory

Character creation service using saga-based orchestration. Validates character creation requests against tenant-configured templates and emits a unified saga to the Atlas Saga Orchestrator containing character creation, item awards, equipment, and skill creation steps.

## External Dependencies

- Kafka (message broker for saga commands and events)
- OpenTelemetry Collector (distributed tracing)
- Atlas Configuration Service (tenant configuration)
- Atlas Saga Orchestrator (saga execution)

## Runtime Configuration

| Variable                | Description                                             |
|-------------------------|---------------------------------------------------------|
| TRACE_ENDPOINT          | OpenTelemetry collector endpoint                        |
| LOG_LEVEL               | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| BOOTSTRAP_SERVERS       | Kafka host:port                                         |
| SERVICE_ID              | Service identifier (UUID)                               |
| REST_PORT               | REST API port                                           |
| CONFIGURATIONS          | Base URL for configuration service                      |
| COMMAND_TOPIC_SAGA      | Topic for saga commands                                 |
| EVENT_TOPIC_SAGA_STATUS | Topic for saga status events                            |
| EVENT_TOPIC_SEED_STATUS | Topic for seed status events                            |

## Documentation

- [Domain](docs/domain.md)
- [Kafka Integration](docs/kafka.md)
- [REST API](docs/rest.md)
- [Storage](docs/storage.md)
