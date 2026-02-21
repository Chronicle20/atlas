# atlas-buffs

Buff management service for the Atlas platform.

The service manages temporary stat modifications (buffs) for game characters. It maintains a Redis-backed registry of active buffs per character, tracks buff durations, and handles automatic expiration. Buffs are applied and cancelled via Kafka commands, with status events emitted for buff lifecycle changes. The service also processes periodic poison damage ticks for characters with active poison debuffs.

## External Dependencies

- Redis: Buff state storage via TenantRegistry
- Kafka: Message-based command and event processing
- OpenTelemetry Collector: Distributed tracing via OTLP gRPC

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| REDIS_URL | Redis server address |
| REDIS_PASSWORD | Redis authentication password |
| TRACE_ENDPOINT | OpenTelemetry collector endpoint for distributed tracing |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| REST_PORT | HTTP server port |
| BOOTSTRAP_SERVERS | Kafka bootstrap servers |
| COMMAND_TOPIC_CHARACTER_BUFF | Topic for buff commands |
| EVENT_TOPIC_CHARACTER_BUFF_STATUS | Topic for buff status events |
| COMMAND_TOPIC_CHARACTER | Topic for character commands (poison damage) |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
