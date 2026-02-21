# atlas-chairs

Chair management service for the Atlas platform.

The service manages chair usage for game characters. It tracks which characters are sitting on chairs, validates chair availability and type, and handles automatic chair clearing on character logout, map change, or channel change. The service maintains Redis-backed registries for both chair assignments and character locations.

## External Dependencies

- Kafka: Message-based command and event processing
- Redis: Chair assignment and character location registries
- OpenTelemetry Collector (OTLP/gRPC): Distributed tracing
- atlas-data: Map data for fixed chair validation
- atlas-query-aggregator: Item ownership validation for portable chairs

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| TRACE_ENDPOINT | OpenTelemetry Collector endpoint for distributed tracing |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| REST_PORT | Port for REST server |
| BOOTSTRAP_SERVERS | Kafka bootstrap servers |
| COMMAND_TOPIC_CHAIR | Topic for chair commands |
| EVENT_TOPIC_CHAIR_STATUS | Topic for chair status events |
| EVENT_TOPIC_CHARACTER_STATUS | Topic for character status events (consumed) |
| DATA | Base URL for data service |
| QUERY_AGGREGATOR | Base URL for query aggregator service |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
