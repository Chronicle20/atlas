# atlas-maps

Mushroom game maps Service

## Overview

A service that tracks character presence in maps, manages spawning of monsters and reactors, manages map weather and mist area effects, enforces map-stay timers, tracks each character's persisted last-known map location, and records character map visit history. Maintains in-memory registries of character presence, spawn point cooldown states, active weather effects, active mists, and map-stay timers. Persists visit records and character locations to PostgreSQL.

## External Dependencies

- PostgreSQL: Persistent storage for character map visit records and character locations
- Redis: Spawn point cooldown registry storage
- Kafka: Message consumption and production for character status events, map status events, cash shop events, monster status events, map commands, map action commands, reactor commands, mist commands and events, character buff commands, and data ingestion events
- atlas-data service: REST API for map spawn point, reactor, script, and map info data
- atlas-monsters service: REST API for monster counts and creation
- atlas-reactors service: REST API for reactor queries
- atlas-character service: REST API for character position (used by the mist tick task)

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| TRACE_ENDPOINT | OTLP gRPC endpoint for distributed tracing |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| REST_PORT | Port for REST interface |
| BOOTSTRAP_SERVERS | Kafka host:port |
| DB_HOST | PostgreSQL host |
| DB_PORT | PostgreSQL port |
| DB_USER | PostgreSQL user |
| DB_PASSWORD | PostgreSQL password |
| DB_NAME | PostgreSQL database name |
| REDIS_URL | Redis host:port for spawn point registry |
| REDIS_PASSWORD | Redis password |
| EVENT_TOPIC_CHARACTER_STATUS | Topic for character status events (consumed and produced) |
| EVENT_TOPIC_MAP_STATUS | Topic for map status events (produced) |
| EVENT_TOPIC_CASH_SHOP_STATUS | Topic for cash shop status events (consumed) |
| EVENT_TOPIC_MONSTER_STATUS | Topic for monster status events (consumed) |
| EVENT_TOPIC_SESSION_STATUS | Topic for session status events (consumed) |
| EVENT_TOPIC_DATA | Topic for atlas-data ingestion events (consumed) |
| EVENT_TOPIC_MIST | Topic for mist events (produced) |
| DATA_EVENTS_CONSUMER_ENABLED | Enables/disables the EVENT_TOPIC_DATA consumer (defaults to enabled) |
| COMMAND_TOPIC_MAP | Topic for map commands (consumed) |
| COMMAND_TOPIC_CHARACTER | Topic for character commands (consumed and produced) |
| COMMAND_TOPIC_CHARACTER_CHANNEL_CHANGE_REQUEST | Topic for character channel-change request commands (consumed) |
| COMMAND_TOPIC_CHARACTER_BUFF | Topic for character buff commands (produced) |
| COMMAND_TOPIC_REACTOR | Topic for reactor commands (produced) |
| COMMAND_TOPIC_MAP_ACTIONS | Topic for map action commands (produced) |
| COMMAND_TOPIC_MIST | Topic for mist commands (consumed) |
| DATA | Root URL for atlas-data service |
| MONSTERS | Root URL for atlas-monsters service |
| REACTORS | Root URL for atlas-reactors service |
| CHARACTERS | Root URL for atlas-character service |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
