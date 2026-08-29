# atlas-maker

Maker skill crafting service for the Atlas platform.

The service resolves Maker skill crafting eligibility and draws crafting
rewards. It owns no mutable game state and stores no craft-audit rows — the
saga record kept by `atlas-saga-orchestrator` is the durable history of a
craft. `atlas-maker` reads seeded, tenant-owned reference tables (reagent
requirements and crystal-band reward weights) and cross-service state
(character level/mesos, Maker skill level, inventory compartment snapshots,
quest state, recipe and equip data) to answer "can this character craft this
recipe" and "what did the craft award."

## REST Endpoints

_None yet — Task 24 adds the crafting endpoint(s)._

## Kafka Commands / Events

_None yet — this service does not currently produce or consume Kafka
messages._

## Upstream Dependencies

| Upstream | Token | Used for |
|---|---|---|
| `atlas-data` | `DATA` | recipes (`data/item-makes`), equip `reqLevel` (`data/equipment/{id}`) |
| `atlas-character` | `CHARACTERS` | level, mesos |
| `atlas-skills` | `SKILLS` | Maker skill level |
| `atlas-inventory` | `INVENTORY` | compartment snapshots, accommodation check |
| `atlas-quests` | `QUESTS` | `reqQuest` state |

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| TRACE_ENDPOINT | OpenTelemetry Collector gRPC endpoint |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| REST_PORT | HTTP server port |
| DB_HOST | PostgreSQL host |
| DB_PORT | PostgreSQL port |
| DB_USER | PostgreSQL user |
| DB_PASSWORD | PostgreSQL password |
| DB_NAME | PostgreSQL database name |
| DATA | Base URL for `atlas-data` |
| CHARACTERS | Base URL for `atlas-character` |
| SKILLS | Base URL for `atlas-skills` |
| INVENTORY | Base URL for `atlas-inventory` |
| QUESTS | Base URL for `atlas-quests` |
