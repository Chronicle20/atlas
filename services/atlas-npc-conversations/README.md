# atlas-npc-conversations

NPC conversation service for managing JSON-driven NPC and quest conversation state machines. Processes player interactions through a state machine engine that supports dialogue, list selection, number input, style selection, slide menus, crafting, transport, gachapon, and party quest actions. Operations execute locally or via atlas-saga-orchestrator for distributed transactions. Conditions evaluate via atlas-query-aggregator for character state validation.

## External Dependencies

- **PostgreSQL** — Persistent storage for NPC and quest conversation definitions (JSONB).
- **Kafka** — Receives conversation commands (start, continue, end) and character/saga status events. Produces NPC dialogue commands, character status events, saga commands, and guild commands.
- **atlas-saga-orchestrator** — Executes distributed operations (crafting, transport, gachapon, party quests) via saga commands.
- **atlas-query-aggregator** — Evaluates character state conditions (job, level, items, mesos, quest status, etc.) via synchronous REST calls.
- **Jaeger** — Distributed tracing.

## Runtime Configuration

| Variable                         | Description                                    |
|----------------------------------|------------------------------------------------|
| JAEGER_HOST                      | Jaeger host:port for distributed tracing       |
| LOG_LEVEL                        | Logging level                                  |
| CONFIG_FILE                      | Service configuration file location            |
| BOOTSTRAP_SERVERS                | Kafka host:port                                |
| BASE_SERVICE_URL                 | scheme://host:port/api/                        |
| REST_PORT                        | HTTP server port                               |
| COMMAND_TOPIC_NPC                | Kafka topic for NPC commands                   |
| COMMAND_TOPIC_NPC_CONVERSATION   | Kafka topic for NPC conversation commands      |
| COMMAND_TOPIC_SAGA               | Kafka topic for saga commands                  |
| COMMAND_TOPIC_GUILD              | Kafka topic for guild commands                 |
| COMMAND_TOPIC_QUEST_CONVERSATION | Kafka topic for quest conversation commands    |
| EVENT_TOPIC_CHARACTER_STATUS     | Kafka topic for character status events        |
| EVENT_TOPIC_SAGA_STATUS          | Kafka topic for saga status events             |

## Documentation

- [Domain](docs/domain.md) — Domain models, invariants, state transitions, processors
- [Kafka](docs/kafka.md) — Kafka topics consumed and produced, message types
- [REST](docs/rest.md) — HTTP endpoints for NPC and quest conversation management
- [Storage](docs/storage.md) — Database tables and schema
