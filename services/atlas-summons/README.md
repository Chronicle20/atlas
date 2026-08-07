# atlas-summons

Manages summon instances cast by characters — puppets, attacker summons (hawks,
elementals, dragons, etc.), and the Dark Knight Beholder buff-aura summon. It
handles spawn/move/attack/damage/despawn lifecycle commands relayed from
atlas-channel, clamps summon attack damage to a faithful per-hit ceiling,
credits monster damage/status effects to the summon's owner, periodically
despawns expired summons, and periodically heals/buffs the Beholder's owner.

## Overview

This service maintains a Redis-backed registry of active summon instances
across all tenants, worlds, channels, and maps. Summons are despawned when
their duration expires, when their owner logs out, changes channel, or changes
map, or when a same-skill or mobility-conflicting summon is cast. A puppet
summon is registered with atlas-monsters so the monster controller picker
biases toward the puppet's owner.

## External Dependencies

- Redis: all state storage (summon instances, field/owner indexes, object-id
  allocation) and the leader-election lock for sweep tasks
- Kafka: consumes summon commands and character-status events; produces
  summon-status events, and commands to atlas-monsters, atlas-buffs, and
  atlas-character
- atlas-data: REST API for skill effect data (HP/duration/damage/proc/statup
  attributes of summon skills)
- atlas-effective-stats: REST API for a character's session-effective combat
  stats (used by the summon damage ceiling)
- atlas-inventory: REST API for a character's equipped weapon type (used by
  the summon damage ceiling)

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| TRACE_ENDPOINT | OpenTelemetry collector endpoint |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| BOOTSTRAP_SERVERS | Kafka host:port |
| REST_PORT | HTTP server port |
| DATA | atlas-data REST API base URL |
| EFFECTIVE_STATS | atlas-effective-stats REST API base URL |
| INVENTORY | atlas-inventory REST API base URL |
| COMMAND_TOPIC_SUMMON | Kafka topic for summon commands (consumed) |
| EVENT_TOPIC_CHARACTER_STATUS | Kafka topic for character status events (consumed) |
| EVENT_TOPIC_SUMMON_STATUS | Kafka topic for summon status events (produced) |
| COMMAND_TOPIC_MONSTER | Kafka topic for atlas-monsters commands (produced) |
| COMMAND_TOPIC_CHARACTER_BUFF | Kafka topic for atlas-buffs commands (produced) |
| COMMAND_TOPIC_CHARACTER | Kafka topic for atlas-character commands (produced) |
| SUMMON_LEADER_ELECTION_ENABLED | Enables leader election gating for sweep tasks (default true) |
| SUMMON_LEADER_TTL | Leader-election lock TTL (default 30s, range 5s-5m) |
| SUMMON_LEADER_REFRESH | Leader-election lock refresh interval (default TTL/3, min 1s) |
| SUMMON_LEADER_BACKOFF | Leader-election retry backoff (default 5s, range 1s-1m) |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
