# atlas-buffs

Buff management service for the Atlas platform.

The service manages temporary stat modifications (buffs) for game characters. It maintains an in-memory registry of active buffs per character, tracks buff durations, and handles automatic expiration. Buffs are applied and cancelled via Kafka commands, with status events emitted for buff lifecycle changes.

## External Dependencies

- Kafka: Message-based command and event processing
- Jaeger: Distributed tracing

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| JAEGER_HOST | Jaeger host:port for distributed tracing |
| JAEGER_HOST_PORT | Alternative Jaeger endpoint specification |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| BOOTSTRAP_SERVERS | Kafka bootstrap servers |
| COMMAND_TOPIC_CHARACTER_BUFF | Topic for buff commands |
| EVENT_TOPIC_CHARACTER_BUFF_STATUS | Topic for buff status events |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
