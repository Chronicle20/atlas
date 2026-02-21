# atlas-npc-conversations

NPC conversation service for managing JSON-driven NPC and quest conversation state machines. Processes player interactions through a state machine engine that supports dialogue, list selection, number input, style selection, slide menus, crafting, transport, gachapon, and party quest actions. Operations execute locally or via atlas-saga-orchestrator for distributed transactions. Conditions evaluate via atlas-query-aggregator for character state validation.

## External Dependencies

- **PostgreSQL** — Persistent storage for NPC and quest conversation definitions (JSONB).
- **Redis** — In-memory conversation context registry. Stores active conversation state per character and saga-to-character index.
- **Kafka** — Receives conversation commands (start, continue, end) and character/saga status events. Produces NPC dialogue commands, character status events, saga commands, and guild commands.
- **atlas-saga-orchestrator** — Executes distributed operations (crafting, transport, gachapon, party quests) via saga commands.
- **atlas-query-aggregator** — Evaluates character state conditions (job, level, items, mesos, quest status, etc.) and provides character appearance data via synchronous REST calls.
- **atlas-data** — Validates cosmetic item existence (hair, face) via synchronous REST calls.
- **OpenTelemetry (OTLP/gRPC)** — Distributed tracing.

## Runtime Configuration

| Variable                         | Description                                                      |
|----------------------------------|------------------------------------------------------------------|
| TRACE_ENDPOINT                   | OpenTelemetry collector gRPC endpoint                            |
| LOG_LEVEL                        | Logging level                                                    |
| BOOTSTRAP_SERVERS                | Kafka broker addresses                                           |
| REST_PORT                        | HTTP server port                                                 |
| COMMAND_TOPIC_NPC                | Kafka topic for NPC commands                                     |
| COMMAND_TOPIC_NPC_CONVERSATION   | Kafka topic for NPC conversation commands                        |
| COMMAND_TOPIC_SAGA               | Kafka topic for saga commands                                    |
| COMMAND_TOPIC_GUILD              | Kafka topic for guild commands                                   |
| COMMAND_TOPIC_QUEST_CONVERSATION | Kafka topic for quest conversation commands                      |
| EVENT_TOPIC_CHARACTER_STATUS     | Kafka topic for character status events                          |
| EVENT_TOPIC_SAGA_STATUS          | Kafka topic for saga status events                               |
| NPC_CONVERSATIONS_PATH           | Filesystem path for NPC conversation seed files (default: /conversations/npc)       |
| QUEST_CONVERSATIONS_PATH         | Filesystem path for quest conversation seed files (default: /conversations/quests)  |

## Documentation

- [Domain](docs/domain.md) — Domain models, invariants, state transitions, processors
- [Kafka](docs/kafka.md) — Kafka topics consumed and produced, message types
- [REST](docs/rest.md) — HTTP endpoints for NPC and quest conversation management
- [Storage](docs/storage.md) — Database tables and schema
