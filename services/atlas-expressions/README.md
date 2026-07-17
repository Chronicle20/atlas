# atlas-expressions

A service that manages character facial expressions. It tracks active expressions in Redis with automatic TTL-based expiration and reverts them to the default expression after a fixed duration.

## External Dependencies

- Kafka
- Redis

## Runtime Configuration

### Required
- BOOTSTRAP_SERVERS - Kafka [host]:[port]
- REST_PORT - HTTP server port
- COMMAND_TOPIC_EXPRESSION - Kafka topic for expression commands
- EVENT_TOPIC_EXPRESSION - Kafka topic for expression events
- EVENT_TOPIC_MAP_STATUS - Kafka topic for map status events

### Optional
- REDIS_URL - Redis connection address (default `localhost:6379`)
- REDIS_PASSWORD - Redis authentication password
- TRACE_ENDPOINT - OpenTelemetry collector endpoint for distributed tracing
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
