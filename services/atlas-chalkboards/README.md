# atlas-chalkboards

Chalkboard management service for the Atlas platform.

The service manages chalkboard messages for game characters. It tracks which characters have active chalkboard messages, handles setting and clearing messages via Kafka commands, and automatically clears chalkboards on character logout, map change, or channel change. The service maintains Redis-backed registries for both chalkboard messages and character locations.

## External Dependencies

- Redis: Chalkboard message and character location storage
- Kafka: Message-based command and event processing
- OpenTelemetry (OTLP/gRPC): Distributed tracing

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| TRACE_ENDPOINT | OTLP gRPC endpoint for distributed tracing |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| REST_PORT | Port for REST server |
| BOOTSTRAP_SERVERS | Kafka bootstrap servers |
| COMMAND_TOPIC_CHALKBOARD | Topic for chalkboard commands |
| EVENT_TOPIC_CHALKBOARD_STATUS | Topic for chalkboard status events |
| EVENT_TOPIC_CHARACTER_STATUS | Topic for character status events (consumed) |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
