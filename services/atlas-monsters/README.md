# atlas-monsters

Manages monster instances in game maps, including spawning, movement, damage, control assignment, and destruction.

## Overview

This service maintains an in-memory registry of active monster instances across all tenants, worlds, channels, and maps. It handles monster lifecycle events, assigns character controllers to monsters, tracks damage dealt by characters, and emits status events for downstream consumers.

## External Dependencies

- Kafka: Consumes map status events and monster commands; produces monster status events
- atlas-data: REST API for retrieving monster information (HP, MP)
- atlas-maps: REST API for retrieving character IDs in maps
- Jaeger: Distributed tracing

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| JAEGER_HOST | Jaeger host:port |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| BOOTSTRAP_SERVERS | Kafka host:port |
| REST_PORT | HTTP server port |
| EVENT_TOPIC_MAP_STATUS | Kafka topic for map status events (consumed) |
| EVENT_TOPIC_MONSTER_STATUS | Kafka topic for monster status events (produced) |
| COMMAND_TOPIC_MONSTER | Kafka topic for monster damage commands (consumed) |
| COMMAND_TOPIC_MONSTER_MOVEMENT | Kafka topic for monster movement commands (consumed) |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
