# atlas-messengers

A RESTful resource which provides messenger (party chat) services. Messengers are ephemeral group chat rooms that allow up to 3 characters to communicate in real-time.

This service uses in-memory storage instead of database persistence. Messenger state is intentionally ephemeral and does not persist across service restarts.

## External Dependencies

- Kafka - Message broker for commands and events
- Jaeger - Distributed tracing
- atlas-character - Foreign service for character information lookup

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| JAEGER_HOST | Jaeger [host]:[port] |
| LOG_LEVEL | Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace |
| REST_PORT | HTTP server port |
| BOOTSTRAP_SERVERS | Kafka [host]:[port] |
| COMMAND_TOPIC_MESSENGER | Kafka topic for messenger commands |
| EVENT_TOPIC_MESSENGER_STATUS | Kafka topic for messenger status events |
| EVENT_TOPIC_MESSENGER_MEMBER_STATUS | Kafka topic for member status events |
| EVENT_TOPIC_CHARACTER_STATUS | Kafka topic for character status events to consume |
| COMMAND_TOPIC_INVITE | Kafka topic for invite commands |
| EVENT_TOPIC_INVITE_STATUS | Kafka topic for invite status events to consume |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
