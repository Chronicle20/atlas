# atlas-messengers

A RESTful resource which provides messenger (party chat) services. Messengers are ephemeral group chat rooms that allow up to 3 characters to communicate in real-time.

This service uses Redis for state storage. Messenger and character registries are backed by Redis tenant-scoped registries. A Redis-based distributed lock coordinates messenger creation.

## External Dependencies

- Kafka - Message broker for commands and events
- Redis - State storage for messenger and character registries, ID generation, and distributed locking
- OpenTelemetry Collector - Distributed tracing via OTLP/gRPC
- atlas-character - Foreign service for character information lookup

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| TRACE_ENDPOINT | OpenTelemetry Collector OTLP/gRPC endpoint |
| LOG_LEVEL | Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace |
| REST_PORT | HTTP server port |
| BOOTSTRAP_SERVERS | Kafka [host]:[port] |
| COMMAND_TOPIC_MESSENGER | Kafka topic for messenger commands |
| EVENT_TOPIC_MESSENGER_STATUS | Kafka topic for messenger status events |
| EVENT_TOPIC_MESSENGER_MEMBER_STATUS | Kafka topic for member status events |
| EVENT_TOPIC_CHARACTER_STATUS | Kafka topic for character status events to consume |
| COMMAND_TOPIC_INVITE | Kafka topic for invite commands |
| EVENT_TOPIC_INVITE_STATUS | Kafka topic for invite status events to consume |
| CHARACTERS | Base URL for atlas-character service |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
