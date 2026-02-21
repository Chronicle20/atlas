# atlas-drops

A microservice that manages item and meso drops within game maps. Drops are stored in Redis and tracked per tenant, world, channel, map, and instance. Equipment stats are carried inline on the drop model rather than referencing an external equipment service. All drops are expired on service shutdown.

## External Dependencies

- Redis: Drop storage and ID generation
- Kafka: Message broker for command consumption and event emission
- OpenTelemetry Collector: Distributed tracing (via OTLP/gRPC)
- atlas-configurations: External service for runtime configuration (REST)

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| BOOTSTRAP_SERVERS | Kafka bootstrap server address |
| REDIS_URL | Redis server address |
| REDIS_PASSWORD | Redis authentication password |
| TRACE_ENDPOINT | OpenTelemetry collector gRPC endpoint |
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
