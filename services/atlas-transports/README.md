# atlas-transports

A Golang service that manages transportation routes within the game server. The service supports two transport models: scheduled routes that operate on fixed time-of-day schedules with shared vessels, and instance-based routes that create on-demand transport instances per character group.

Scheduled routes transition through states (awaiting return, open entry, locked entry, in transit) based on a precomputed daily schedule and coordinate character warping between maps. Instance-based routes create ephemeral transport instances with boarding windows, capacity limits, and per-instance timers.

## External Dependencies

- Kafka: Message broker for event consumption and production
- Redis: State storage for registries (routes, instances, characters, channels)
- OpenTelemetry (OTLP): Distributed tracing
- atlas-tenants: Tenant configuration service (routes, vessels, instance routes)
- atlas-data: Portal and map data service
- atlas-maps: Character map presence service

## Runtime Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| TRACE_ENDPOINT | OpenTelemetry OTLP gRPC endpoint | - |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) | - |
| REST_PORT | REST API server port | 8080 |
| BOOTSTRAP_SERVERS | Comma-separated Kafka bootstrap servers | - |
| EVENT_TOPIC_TRANSPORT_STATUS | Kafka topic for transport status events | - |
| EVENT_TOPIC_CHARACTER_STATUS | Kafka topic for character status events | - |
| EVENT_TOPIC_CHANNEL_STATUS | Kafka topic for channel status events | - |
| EVENT_TOPIC_MAP_STATUS | Kafka topic for map status events | - |
| EVENT_TOPIC_INSTANCE_TRANSPORT | Kafka topic for instance transport events | - |
| COMMAND_TOPIC_CHARACTER | Kafka topic for character commands | - |
| COMMAND_TOPIC_INSTANCE_TRANSPORT | Kafka topic for instance transport commands | - |
| EVENT_TOPIC_CONFIGURATION_STATUS | Kafka topic for configuration status events | - |
| TENANTS | Base URL for tenants service | - |
| MAPS | Base URL for maps service | - |
| DATA | Base URL for data service | - |

## Documentation

- [Domain Model](docs/domain.md)
- [Kafka Integration](docs/kafka.md)
- [REST API](docs/rest.md)
- [Storage](docs/storage.md)
