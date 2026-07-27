# atlas-character-factory

Character creation service using saga-based orchestration. Supports two creation paths: validating a request against tenant-configured character-creation templates, and creating a character from a tenant-configured preset. Both paths emit a unified saga to the Atlas Saga Orchestrator containing character creation, item awards, equipment, and skill creation steps.

## External Dependencies

- Kafka (message broker for saga commands/events, seed status events, and the tenant configuration projection)
- OpenTelemetry Collector (distributed tracing)
- Atlas Character service (character name-validity checks)
- Atlas Data service (item and skill existence/attribute lookups)
- Atlas Saga Orchestrator (saga execution)

## Runtime Configuration

| Variable                | Description                                             |
|-------------------------|---------------------------------------------------------|
| TRACE_ENDPOINT          | OpenTelemetry collector endpoint                        |
| LOG_LEVEL               | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| BOOTSTRAP_SERVERS       | Kafka host:port                                         |
| SERVICE_ID              | Service identifier (UUID)                               |
| REST_PORT               | REST API port                                           |
| EVENT_TOPIC_CONFIGURATION_TENANT_STATUS | Topic for the tenant configuration projection          |
| PROJECTION_CATCHUP_TIMEOUT_S | Optional. Seconds to wait for projection catch-up (default 300)    |
| COMMAND_TOPIC_SAGA      | Topic for saga commands                                 |
| EVENT_TOPIC_SAGA_STATUS | Topic for saga status events                            |
| EVENT_TOPIC_SEED_STATUS | Topic for seed status events                            |
| CHARACTERS_SERVICE_URL  | Base URL for the Atlas Character service (falls back to BASE_SERVICE_URL) |
| DATA_SERVICE_URL        | Base URL for the Atlas Data service (falls back to BASE_SERVICE_URL) |
| BASE_SERVICE_URL        | Fallback base URL for service-to-service REST calls     |

## Documentation

- [Domain](docs/domain.md)
- [Kafka Integration](docs/kafka.md)
- [REST API](docs/rest.md)
- [Storage](docs/storage.md)
