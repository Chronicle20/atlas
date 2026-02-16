# atlas-monsters

Manages monster instances in game maps, including spawning, movement, damage, control assignment, skill execution, status effects, and destruction.

## Overview

This service maintains an in-memory registry of active monster instances across all tenants, worlds, channels, and maps. It handles monster lifecycle events, assigns character controllers to monsters, tracks damage dealt by characters, manages monster status effects (buffs, debuffs, DoT), executes monster skills (stat buffs, heals, debuffs, summons), and emits status events for downstream consumers.

## External Dependencies

- Kafka: Consumes map status events and monster commands; produces monster status events, character buff commands, and portal commands
- atlas-data: REST API for retrieving monster information (HP, MP, boss, resistances, skills, revives, banish, animation times) and mob skill definitions
- atlas-maps: REST API for retrieving character IDs in maps
- OpenTelemetry: Distributed tracing via OTLP/gRPC

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| TRACE_ENDPOINT | OpenTelemetry collector endpoint |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| BOOTSTRAP_SERVERS | Kafka host:port |
| REST_PORT | HTTP server port |
| EVENT_TOPIC_MAP_STATUS | Kafka topic for map status events (consumed) |
| EVENT_TOPIC_MONSTER_STATUS | Kafka topic for monster status events (produced) |
| COMMAND_TOPIC_MONSTER | Kafka topic for monster commands (consumed) |
| COMMAND_TOPIC_MONSTER_MOVEMENT | Kafka topic for monster movement commands (consumed) |
| COMMAND_TOPIC_CHARACTER_BUFF | Kafka topic for character buff commands (produced) |
| COMMAND_TOPIC_PORTAL | Kafka topic for portal/warp commands (produced) |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
