# atlas-maps

Mushroom game maps Service

## Overview

A service that tracks character presence in maps and manages spawning of monsters and reactors. Maintains in-memory registries of character locations and spawn point states.

## External Dependencies

- Kafka: Message consumption and production for character status events, map status events, cash shop events, and reactor commands
- atlas-data service: REST API for map spawn point and reactor data
- atlas-monsters service: REST API for monster counts and creation
- atlas-reactors service: REST API for reactor queries

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| JAEGER_HOST | Jaeger host:port for distributed tracing |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| REST_PORT | Port for REST interface |
| BOOTSTRAP_SERVERS | Kafka host:port |
| EVENT_TOPIC_CHARACTER_STATUS | Topic for character status events (consumed) |
| EVENT_TOPIC_MAP_STATUS | Topic for map status events (produced) |
| EVENT_TOPIC_CASH_SHOP_STATUS | Topic for cash shop status events (consumed) |
| COMMAND_TOPIC_REACTOR | Topic for reactor commands (produced) |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
