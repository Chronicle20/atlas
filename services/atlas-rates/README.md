# atlas-rates

Aggregates rate multipliers for characters from multiple sources (world settings, buffs, items) and computes final rate values for experience, meso, item drop, and quest experience. Other services query this service when applying rate multipliers to gameplay calculations.

## External Dependencies

- **Redis**: Stores character rate models, tracked items, and initialization state
- **Kafka**: Consumes events from `EVENT_TOPIC_CHARACTER_BUFF_STATUS`, `EVENT_TOPIC_WORLD_RATE`, `EVENT_TOPIC_ASSET_STATUS`, `EVENT_TOPIC_CHARACTER_STATUS`
- **atlas-inventory**: Queries equipped items and cash assets during lazy initialization
- **atlas-data**: Queries equipment bonusExp properties and cash item rate properties
- **atlas-buffs**: Queries active buffs during lazy initialization
- **atlas-character**: Queries session history for time-based bonus calculations

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| `REST_PORT` | Port for REST API |
| `BOOTSTRAP_SERVERS` | Kafka broker addresses |
| `BASE_SERVICE_URL` | Base URL for outbound REST requests |
| `REDIS_URL` | Redis connection address |
| `REDIS_PASSWORD` | Redis password |
| `TRACE_ENDPOINT` | OpenTelemetry tracing endpoint |
| `LOG_LEVEL` | Logging level |
| `EVENT_TOPIC_CHARACTER_BUFF_STATUS` | Kafka topic for buff events |
| `EVENT_TOPIC_WORLD_RATE` | Kafka topic for world rate events |
| `EVENT_TOPIC_ASSET_STATUS` | Kafka topic for inventory asset events |
| `EVENT_TOPIC_CHARACTER_STATUS` | Kafka topic for character status events |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
