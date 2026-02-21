# atlas-saga-orchestrator

Coordinates distributed transactions across Atlas microservices using the saga pattern. Tracks step execution and performs compensation on failure to maintain data consistency. Sagas are persisted in PostgreSQL with optimistic locking, recovered on startup, and reaped on timeout.

The orchestrator receives saga commands via Kafka or REST, then executes each step sequentially by producing commands to downstream services and consuming their status events. High-level transfer actions (storage and cash shop transfers) are expanded at runtime into concrete accept/release step pairs by fetching asset data from the relevant inventory services.

## External Dependencies

- PostgreSQL (saga persistence)
- Kafka (message broker)
- OpenTelemetry (distributed tracing)

## External Service Dependencies

This service makes REST calls to:

- **Inventory Service** (`INVENTORY`): Retrieve character inventory compartments and assets for step expansion
- **Storage Service** (`STORAGE`): Retrieve storage projections for step expansion
- **Rate Service** (`RATES`): Get character rate multipliers for reactor drop calculations
- **Quest Service** (`QUESTS`): Get started quests for quest-aware drop filtering
- **Drop Information Service** (`DROP_INFORMATION`): Get reactor drop tables
- **Data Service** (`DATA`): Calculate drop positions, foothold lookups, portal data, NPC data
- **Cash Shop Service** (`CASHSHOP`): Retrieve cash shop compartments for step expansion
- **Gachapon Service** (`GACHAPONS_URL`): Select random gachapon rewards and retrieve gachapon metadata
- **Transport Service** (`TRANSPORTS_URL`): Start instance-based transports
- **Character Service** (`CHARACTER_URL`): Save and retrieve character locations
- **Monster Service** (`MONSTERS`): Spawn monsters via REST
- **Parties Service** (`PARTIES`): Retrieve character party and members for party quest validation
- **Party Quests Service** (`PARTY_QUESTS`): Retrieve party quest definitions and start requirements
- **Query Aggregator Service** (`QUERY_AGGREGATOR`): Validate character state conditions
- **Reactors Service** (`REACTORS`): Look up reactors by name for hit commands

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| BOOTSTRAP_SERVERS | Kafka bootstrap servers |
| TRACE_ENDPOINT | OpenTelemetry collector endpoint |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| REST_PORT | REST API server port |
| SAGA_DEFAULT_TIMEOUT | Default saga timeout duration (default: 5m) |
| SAGA_RECOVERY_ENABLED | Enable saga recovery on startup (default: true) |
| SAGA_REAPER_INTERVAL | Interval between reaper sweeps (default: 30s) |
| RATES | Base URL for rate service |
| QUESTS | Base URL for quest service |
| DROP_INFORMATION | Base URL for drop information service |
| DATA | Base URL for data service |
| INVENTORY | Base URL for inventory service |
| STORAGE | Base URL for storage service |
| CASHSHOP | Base URL for cash shop service |
| GACHAPONS_URL | Base URL for gachapon service |
| TRANSPORTS_URL | Base URL for transport service |
| CHARACTER_URL | Base URL for character service (saved locations) |
| MONSTERS | Base URL for monster service |
| PARTIES | Base URL for parties service |
| PARTY_QUESTS | Base URL for party quests service |
| QUERY_AGGREGATOR | Base URL for query aggregator service |
| REACTORS | Base URL for reactors service |

### Kafka Topics

| Variable | Description |
|----------|-------------|
| COMMAND_TOPIC_SAGA | Saga command input |
| COMMAND_TOPIC_COMPARTMENT | Inventory compartment commands |
| COMMAND_TOPIC_CHARACTER | Character commands |
| COMMAND_TOPIC_SKILL | Skill commands |
| COMMAND_TOPIC_GUILD | Guild commands |
| COMMAND_TOPIC_INVITE | Invitation commands |
| COMMAND_TOPIC_BUDDY_LIST | Buddy list commands |
| COMMAND_TOPIC_PET | Pet commands |
| COMMAND_TOPIC_QUEST | Quest commands |
| COMMAND_TOPIC_CONSUMABLE | Consumable commands |
| COMMAND_TOPIC_SYSTEM_MESSAGE | System message commands |
| COMMAND_TOPIC_STORAGE | Storage commands |
| COMMAND_TOPIC_STORAGE_COMPARTMENT | Storage compartment commands |
| COMMAND_TOPIC_WALLET | Cash shop wallet commands |
| COMMAND_TOPIC_CASH_COMPARTMENT | Cash shop compartment commands |
| COMMAND_TOPIC_PORTAL | Portal commands |
| COMMAND_TOPIC_CHARACTER_BUFF | Buff commands |
| COMMAND_TOPIC_PARTY_QUEST | Party quest commands |
| COMMAND_TOPIC_REACTOR | Reactor commands |
| COMMAND_TOPIC_DROP | Drop spawn commands |
| COMMAND_TOPIC_MAP | Map commands |
| EVENT_TOPIC_SAGA_STATUS | Saga status output |
| EVENT_TOPIC_ASSET_STATUS | Asset status input |
| EVENT_TOPIC_BUDDY_LIST_STATUS | Buddy list status input |
| EVENT_TOPIC_WALLET_STATUS | Wallet status input |
| EVENT_TOPIC_CASH_COMPARTMENT_STATUS | Cash shop compartment status input |
| EVENT_TOPIC_CHARACTER_STATUS | Character status input |
| EVENT_TOPIC_COMPARTMENT_STATUS | Compartment status input |
| EVENT_TOPIC_CONSUMABLE_STATUS | Consumable status input |
| EVENT_TOPIC_GUILD_STATUS | Guild status input |
| EVENT_TOPIC_INVITE_STATUS | Invite status input |
| EVENT_TOPIC_PET_STATUS | Pet status input |
| EVENT_TOPIC_QUEST_STATUS | Quest status input |
| EVENT_TOPIC_SKILL_STATUS | Skill status input |
| EVENT_TOPIC_STORAGE_STATUS | Storage service status input |
| EVENT_TOPIC_STORAGE_COMPARTMENT_STATUS | Storage compartment status input |
| EVENT_TOPIC_GACHAPON_REWARD_WON | Gachapon reward win events output |

## Documentation

- [Domain Model](docs/domain.md)
- [Kafka Integration](docs/kafka.md)
- [REST API](docs/rest.md)
- [Storage](docs/storage.md)
