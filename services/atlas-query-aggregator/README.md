# atlas-query-aggregator

A RESTful service that provides composite character state validation in the Atlas ecosystem. This service queries dependent services to validate conditions against character state.

## External Dependencies

- **Character Service** (`CHARACTERS` environment variable): Character data and attributes
- **Inventory Service** (`INVENTORY` environment variable): Item quantities and equipment data
- **Quest Service** (`QUESTS` environment variable): Quest status and progress
- **Marriage Service** (`MARRIAGE` environment variable): Marriage gift data
- **Buddy Service** (`BUDDY` environment variable): Buddy list capacity
- **Pet Service** (`PETS` environment variable): Pet count data
- **Skill Service** (`SKILLS` environment variable): Skill levels
- **Buff Service** (`BUFFS` environment variable): Active buff data
- **Map Service** (`MAPS` environment variable): Map player counts
- **Transport Service** (`TRANSPORT` environment variable): Transport route availability
- **Guild Service** (`GUILDS` environment variable): Guild membership data
- **Item Service** (`ITEMS` environment variable): Item slot information
- **Jaeger** (`JAEGER_HOST` environment variable): Distributed tracing

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| `JAEGER_HOST` | Jaeger host:port for tracing |
| `LOG_LEVEL` | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| `REST_PORT` | HTTP server port |

## Documentation

- [Domain Model](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST API](docs/rest.md)
- [Storage](docs/storage.md)
