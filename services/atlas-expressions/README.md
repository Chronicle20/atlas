# atlas-expressions

A service that manages character expressions in the game. It tracks active expressions in memory and automatically reverts them after expiration.

## External Dependencies

- Kafka

## Runtime Configuration

### Required
- BOOTSTRAP_SERVERS - Kafka [host]:[port]
- EVENT_TOPIC_EXPRESSION - Kafka topic for expression events
- COMMAND_TOPIC_EXPRESSION - Kafka topic for expression commands
- EVENT_TOPIC_MAP_STATUS - Kafka topic for map status events

### Optional
- JAEGER_HOST - Jaeger [host]:[port] for distributed tracing
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
