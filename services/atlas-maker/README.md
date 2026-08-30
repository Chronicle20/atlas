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

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/characters/{characterId}/maker/recipes` | Recipes the character currently qualifies for. Paginated (`page[number]`, `page[size]`). |
| `GET` | `/characters/{characterId}/maker/recipes/{itemId}` | One recipe with the character's eligibility verdict and its material/meso cost. |
| `POST` | `/characters/{characterId}/maker/crafts` | Validate a craft request and, on success, emit the craft saga. Returns the saga's transaction id. |

Recipe data is read-only: `POST`/`PUT`/`PATCH`/`DELETE` on either recipe route return 405.

Every error is a JSON:API error carrying a stable `code`:

| Condition | Status | Code |
|---|---|---|
| Recipe id not found | 404 | `recipe_not_found` |
| Character below `reqLevel` | 422 | `level_too_low` |
| Maker skill absent or below `reqSkillLevel` | 422 | `skill_level_too_low` |
| Missing a `recipe` material | 422 | `insufficient_materials` |
| Missing `reqItem` / `reqEquip` | 422 | `missing_prerequisite_item` |
| Missing a required quest state | 422 | `missing_prerequisite_quest` |
| Insufficient mesos | 422 | `insufficient_mesos` |
| No free inventory slot for an award | 422 | `inventory_full` |
| Equip not in the named slot (disassemble) | 422 | `equip_not_found` |
| Leftover has no crystal mapping | 422 | `no_crystal_mapping` |
| A craft is already in flight for the character | 409 | `craft_in_progress` |

## Kafka Commands / Events

| Direction | Topic (env var) | Purpose |
|---|---|---|
| Produces | `COMMAND_TOPIC_SAGA` | The craft saga (`InventoryTransaction`) an accepted `POST .../crafts` emits for `atlas-saga-orchestrator` to execute. |
| Consumes | `EVENT_TOPIC_SAGA_STATUS` | The craft saga's terminal event (`COMPLETED` or `FAILED`). Releases the character's in-flight craft guard (`craft_in_progress`), the only way an accepted craft's guard entry is ever released short of a pod restart. |

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
| COMMAND_TOPIC_SAGA | Kafka topic name the craft saga command is produced to |
| EVENT_TOPIC_SAGA_STATUS | Kafka topic name the saga terminal event is consumed from |
| BOOTSTRAP_SERVERS | Kafka broker bootstrap servers |
