# atlas-maps

Mushroom game maps Service

## Overview

A service that tracks character presence in maps, manages spawning of monsters and reactors, and records character map visit history. Maintains in-memory registries of character locations and spawn point cooldown states. Persists visit records to PostgreSQL.

## External Dependencies

- PostgreSQL: Persistent storage for character map visit records
- Kafka: Message consumption and production for character status events, map status events, cash shop events, and reactor commands
- atlas-data service: REST API for map spawn point and reactor data
- atlas-monsters service: REST API for monster counts and creation
- atlas-reactors service: REST API for reactor queries

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| JAEGER_HOST_PORT | Jaeger host:port for distributed tracing |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| REST_PORT | Port for REST interface |
| BOOTSTRAP_SERVERS | Kafka host:port |
| DB_HOST | PostgreSQL host |
| DB_PORT | PostgreSQL port |
| DB_USER | PostgreSQL user |
| DB_PASSWORD | PostgreSQL password |
| DB_NAME | PostgreSQL database name |
| EVENT_TOPIC_CHARACTER_STATUS | Topic for character status events (consumed) |
| EVENT_TOPIC_MAP_STATUS | Topic for map status events (produced) |
| EVENT_TOPIC_CASH_SHOP_STATUS | Topic for cash shop status events (consumed) |
| COMMAND_TOPIC_REACTOR | Topic for reactor commands (produced) |
| DATA | Root URL for atlas-data service |
| MONSTERS | Root URL for atlas-monsters service |
| REACTORS | Root URL for atlas-reactors service |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
