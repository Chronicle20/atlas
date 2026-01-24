# atlas-transports

A Golang service that manages transportation routes within the game server. The system simulates travel via ships or similar transports, allowing players to move between maps on a timed schedule.

The service tracks route state based on time-of-day scheduling and coordinates character warping between maps during transport operations. Routes transition through states (awaiting return, open entry, locked entry, in transit) based on a precomputed daily schedule.

## External Dependencies

- Kafka: Message broker for event consumption and production
- Jaeger: Distributed tracing
- atlas-tenants: Tenant configuration service
- atlas-data: Portal and map data service
- atlas-maps: Character map presence service

## Runtime Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| JAEGER_HOST | Jaeger host:port | - |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) | - |
| REST_PORT | REST API server port | 8080 |
| BOOTSTRAP_SERVERS | Comma-separated Kafka bootstrap servers | - |
| EVENT_TOPIC_TRANSPORT_STATUS | Kafka topic for transport status events | - |
| EVENT_TOPIC_CHARACTER_STATUS | Kafka topic for character status events | - |
| EVENT_TOPIC_CHANNEL_STATUS | Kafka topic for channel status events | - |
| COMMAND_TOPIC_CHARACTER | Kafka topic for character commands | - |
| TENANTS | Base URL for tenants service | - |
| MAPS | Base URL for maps service | - |
| DATA | Base URL for data service | - |

## Documentation

- [Domain Model](docs/domain.md)
- [Kafka Integration](docs/kafka.md)
- [REST API](docs/rest.md)
