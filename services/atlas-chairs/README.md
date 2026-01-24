# atlas-chairs

Chair management service for the Atlas platform.

The service manages chair usage for game characters. It tracks which characters are sitting on chairs, validates chair availability and type, and handles automatic chair clearing on character logout, map change, or channel change. The service maintains in-memory registries for both chair assignments and character locations.

## External Dependencies

- Kafka: Message-based command and event processing
- Jaeger: Distributed tracing
- atlas-data: Map data for fixed chair validation

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| JAEGER_HOST_PORT | Jaeger host:port for distributed tracing |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| REST_PORT | Port for REST server |
| BOOTSTRAP_SERVERS | Kafka bootstrap servers |
| COMMAND_TOPIC_CHAIR | Topic for chair commands |
| EVENT_TOPIC_CHAIR_STATUS | Topic for chair status events |
| EVENT_TOPIC_CHARACTER_STATUS | Topic for character status events (consumed) |
| DATA | Base URL for data service |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
