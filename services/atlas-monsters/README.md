# atlas-monsters

Manages monster instances in game maps, including spawning, movement, damage, control assignment, skill execution, status effects, and destruction.

## Overview

This service maintains a Redis-backed registry of active monster instances across all tenants, worlds, channels, and maps. It handles monster lifecycle events, assigns character controllers to monsters, tracks damage dealt by characters, manages monster status effects (buffs, debuffs, DoT), executes monster skills (stat buffs, heals, debuffs, summons), manages friendly monster drop timers, and emits status events for downstream consumers.

## External Dependencies

- Redis: All state storage (monster instances, skill cooldowns, ID allocation, drop timers)
- Kafka: Consumes map status events and monster commands; produces monster status events, character buff commands, portal commands, and drop spawn commands
- atlas-data: REST API for retrieving monster information (HP, MP, boss, resistances, skills, revives, banish, animation times) and mob skill definitions
- atlas-drops: REST API for retrieving monster drop tables
- atlas-maps: REST API for retrieving character IDs in maps
- OpenTelemetry: Distributed tracing via OTLP/gRPC

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| TRACE_ENDPOINT | OpenTelemetry collector endpoint |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| BOOTSTRAP_SERVERS | Kafka host:port |
| REST_PORT | HTTP server port |
| DATA | atlas-data REST API base URL |
| DROPS_INFORMATION | atlas-drops REST API base URL |
| MAPS | atlas-maps REST API base URL |
| EVENT_TOPIC_MAP_STATUS | Kafka topic for map status events (consumed) |
| EVENT_TOPIC_MONSTER_STATUS | Kafka topic for monster status events (produced) |
| COMMAND_TOPIC_MONSTER | Kafka topic for monster commands (consumed) |
| COMMAND_TOPIC_MONSTER_MOVEMENT | Kafka topic for monster movement commands (consumed) |
| COMMAND_TOPIC_CHARACTER_BUFF | Kafka topic for character buff commands (produced) |
| COMMAND_TOPIC_PORTAL | Kafka topic for portal/warp commands (produced) |
| COMMAND_TOPIC_DROP | Kafka topic for drop spawn commands (produced) |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
