# atlas-party-quests

Manages party quest definitions and runtime instances. Definitions describe the structure of a party quest (stages, conditions, rewards, registration rules, bonus configuration). Instances track the live state of an active party quest run, including character participation, stage progression, timers, and stage-specific state such as item counts, monster kills, and custom data.

The service orchestrates party quest lifecycle through Kafka commands and emits status events as instances transition through registration, active play, stage clearing, bonus, completion, and failure. It also reacts to character logout and monster status events to handle automatic leave and friendly monster callbacks.

## External Dependencies

- **PostgreSQL** — Persistent storage for party quest definitions
- **Kafka** — Command ingestion and status event emission
- **atlas-parties** — REST client for resolving party membership
- **atlas-guilds** — REST client for resolving guild membership (affinity)
- **atlas-tenants** — REST client for loading tenant configuration at startup
- **atlas-monsters** — REST client for spawning and destroying monsters in fields

## Runtime Configuration

| Variable | Purpose |
|---|---|
| `BOOTSTRAP_SERVERS` | Kafka broker address |
| `DB_USER`, `DB_PASSWORD`, `DB_HOST`, `DB_PORT`, `DB_NAME` | PostgreSQL connection |
| `REST_PORT` | HTTP server listen port |
| `COMMAND_TOPIC_PARTY_QUEST` | Kafka topic for inbound PQ commands |
| `EVENT_TOPIC_PARTY_QUEST_STATUS` | Kafka topic for outbound PQ status events |
| `COMMAND_TOPIC_CHARACTER` | Kafka topic for outbound character commands |
| `COMMAND_TOPIC_REACTOR` | Kafka topic for outbound reactor commands |
| `COMMAND_TOPIC_SYSTEM_MESSAGE` | Kafka topic for outbound system message commands |
| `COMMAND_TOPIC_MAP` | Kafka topic for outbound map commands |
| `EVENT_TOPIC_CHARACTER_STATUS` | Kafka topic for inbound character status events |
| `EVENT_TOPIC_MONSTER_STATUS` | Kafka topic for inbound monster status events |
| `PARTY_QUEST_DEFINITIONS_PATH` | Filesystem path for JSON definition files (default: `/party-quests`) |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
