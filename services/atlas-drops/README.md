# atlas-drops

A microservice that manages item and meso drops within game maps. Drops are ephemeral entities that exist in-memory and are tracked per tenant, world, channel, map, and instance. Equipment stats are carried inline on the drop model rather than referencing an external equipment service.

## External Dependencies

- Kafka: Message broker for command consumption and event emission
- Jaeger: Distributed tracing
- atlas-configurations: External service for runtime configuration (REST)

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| BOOTSTRAP_SERVERS | Kafka bootstrap server address |
| JAEGER_HOST | Jaeger tracing endpoint `[host]:[port]` |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| REST_PORT | HTTP server port |
| SERVICE_ID | UUID identifying this service instance |
| CONFIGURATIONS | Base URL for atlas-configurations service |
| COMMAND_TOPIC_DROP | Kafka topic for drop commands |
| EVENT_TOPIC_DROP_STATUS | Kafka topic for drop status events |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
