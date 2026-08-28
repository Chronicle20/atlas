# atlas-monsters

Manages monster instances in game maps, including spawning, movement, damage, control assignment, skill execution, status effects, and destruction.

## Overview

This service maintains a Redis-backed registry of active monster instances across all tenants, worlds, channels, and maps. It handles monster lifecycle events, assigns character controllers to monsters (including puppet-vicinity bias, damage-leader takeover, and GM-hide exclusion), tracks damage dealt by characters (with idle aggro decay and a separate client-driven auto-aggro lease), manages monster status effects (buffs, debuffs, reflects, DoT), executes monster skills (stat buffs, heals, debuffs, dispel, banish, summons, area-effect mist), predicts and broadcasts a monster's next skill via a sweep-driven picker, applies HP/MP recovery, manages friendly monster drop timers, resolves self-destructing monsters (threshold, timer, and contact triggers) and bridle (catch-item) capture attempts, tracks GM-hidden characters for controller-election purposes, and emits status events for downstream consumers.

## External Dependencies

- Redis: All state storage (monster instances, skill/attack cooldowns, ID allocation, drop timers, puppet tracking, self-destruct timers, hidden-character tracking)
- Kafka: Consumes map status events, monster commands, character buff-status events, and monster-data cache-invalidation events; produces monster status events, character buff commands, portal/warp commands, mist commands, system-message commands, and drop spawn commands
- atlas-data: REST API for retrieving monster information (HP, MP, boss, resistances, skills, revives, banish, animation times, attack metadata, HP/MP recovery, self-destruction), mob skill definitions, and catch-item (bridle) definitions
- atlas-drops: REST API for retrieving monster drop tables
- atlas-maps: REST API for retrieving character IDs in maps and character locations
- atlas-buffs: REST API for retrieving a character's active buffs (hidden-character reconciliation)
- atlas-character: REST API for retrieving character positions (mob-skill AoE target selection)
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
| BUFFS | atlas-buffs REST API base URL |
| CHARACTERS | atlas-character REST API base URL |
| EVENT_TOPIC_MAP_STATUS | Kafka topic for map status events (consumed) |
| EVENT_TOPIC_CHARACTER_BUFF_STATUS | Kafka topic for character buff-status events (consumed) |
| EVENT_TOPIC_MONSTER_STATUS | Kafka topic for monster status events (produced) |
| EVENT_TOPIC_MONSTER_CATCH | Kafka topic for bridle (catch-item) capture outcomes (produced) |
| EVENT_TOPIC_DATA | Kafka topic for atlas-data cache-invalidation events (consumed) |
| COMMAND_TOPIC_MONSTER | Kafka topic for monster commands (consumed) |
| COMMAND_TOPIC_MONSTER_MOVEMENT | Kafka topic for monster movement commands (consumed) |
| COMMAND_TOPIC_CHARACTER_BUFF | Kafka topic for character buff commands (produced) |
| COMMAND_TOPIC_PORTAL | Kafka topic for portal/warp commands (produced) |
| COMMAND_TOPIC_SYSTEM_MESSAGE | Kafka topic for system-message commands (produced) |
| COMMAND_TOPIC_DROP | Kafka topic for drop spawn commands (produced) |
| COMMAND_TOPIC_MIST | Kafka topic for mist (area-effect) commands (produced) |
| DATA_EVENTS_CONSUMER_ENABLED | Enables/disables the EVENT_TOPIC_DATA consumer (default true) |
| MONSTER_DATA_CACHE_ENABLED | Enables/disables the in-Redis atlas-data monster information cache (default true) |
| MONSTER_DATA_CACHE_TTL | TTL for cached monster information (default 5m, range 1s-24h) |
| MONSTER_DATA_CACHE_NEGATIVE_TTL | TTL for cached lookup misses (default 30s, range 0s-5m) |
| MONSTER_LEADER_ELECTION_ENABLED | Enables leader election gating for sweep tasks (default true) |
| MONSTER_LEADER_TTL | Leader-election lock TTL (default 30s, range 5s-5m) |
| MONSTER_LEADER_REFRESH | Leader-election lock refresh interval (default TTL/3, min 1s) |
| MONSTER_LEADER_BACKOFF | Leader-election retry backoff (default 5s, range 1s-1m) |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
