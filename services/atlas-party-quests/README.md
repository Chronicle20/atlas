# atlas-party-quests

Manages party quest definitions and runtime instances. Definitions describe the structure of a party quest (stages, conditions, rewards, registration rules). Instances track the live state of an active party quest run, including character participation, stage progression, timers, and stage-specific state such as item counts and monster kills.

The service orchestrates party quest lifecycle through Kafka commands and emits status events as instances transition through registration, active play, stage clearing, completion, and failure.

## External Dependencies

- **PostgreSQL** — Persistent storage for party quest definitions
- **Kafka** — Command ingestion and status event emission
- **atlas-parties** — REST client for resolving party membership
- **atlas-guilds** — REST client for resolving guild membership (affinity)
- **atlas-tenants** — REST client for loading tenant configuration at startup

## Runtime Configuration

| Variable | Purpose |
|---|---|
| `BOOTSTRAP_SERVERS` | Kafka broker address |
| `DB_USER`, `DB_PASSWORD`, `DB_HOST`, `DB_PORT`, `DB_NAME` | PostgreSQL connection |
| `REST_PORT` | HTTP server listen port |
| `COMMAND_TOPIC_PARTY_QUEST` | Kafka topic for inbound PQ commands |
| `EVENT_TOPIC_PARTY_QUEST_STATUS` | Kafka topic for outbound PQ status events |
| `COMMAND_TOPIC_CHARACTER` | Kafka topic for outbound character commands |
| `PARTY_QUEST_DEFINITIONS_PATH` | Filesystem path for JSON definition files (default: `/party-quests`) |

## Documentation

- [Domain](docs/domain.md)
- [Kafka](docs/kafka.md)
- [REST](docs/rest.md)
- [Storage](docs/storage.md)
