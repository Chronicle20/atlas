# atlas-saga-orchestrator

Coordinates distributed transactions across Atlas microservices using the saga pattern. Tracks step execution and performs compensation on failure to maintain data consistency. Sagas are stored in-memory and are not persisted across service restarts.

The orchestrator receives saga commands via Kafka or REST, then executes each step sequentially by producing commands to downstream services and consuming their status events. High-level transfer actions (storage and cash shop transfers) are expanded at runtime into concrete accept/release step pairs by fetching asset data from the relevant inventory services.

## External Dependencies

- Kafka (message broker)
- Jaeger (distributed tracing)

## External Service Dependencies

This service makes REST calls to:

- **Inventory Service** (`INVENTORY`): Retrieve character inventory compartments and assets for step expansion
- **Storage Service** (`STORAGE`): Retrieve storage projections for step expansion
- **Rate Service** (`RATES`): Get character rate multipliers for reactor drop calculations
- **Quest Service** (`QUESTS`): Get started quests for quest-aware drop filtering
- **Drop Information Service** (`DROP_INFORMATION`): Get reactor drop tables
- **Data Service** (`DATA`): Calculate drop positions, foothold lookups, portal data, NPC data
- **Cash Shop Service** (`CASHSHOP`): Retrieve cash shop compartments for step expansion
- **Gachapon Service** (`GACHAPON`): Select random gachapon rewards and retrieve gachapon metadata
- **Transport Service** (`TRANSPORTS`): Start instance-based transports
- **Saved Location Service** (`SAVED_LOCATIONS`): Save and retrieve character locations
- **Monster Service** (`MONSTERS`): Spawn monsters via REST

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| BOOTSTRAP_SERVERS | Kafka bootstrap servers |
| JAEGER_HOST_PORT | Jaeger host and port |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| REST_PORT | REST API server port |
| RATES | Base URL for rate service |
| QUESTS | Base URL for quest service |
| DROP_INFORMATION | Base URL for drop information service |
| DATA | Base URL for data service |
| INVENTORY | Base URL for inventory service |
| STORAGE | Base URL for storage service |
| CASHSHOP | Base URL for cash shop service |
| GACHAPON | Base URL for gachapon service |
| TRANSPORTS | Base URL for transport service |
| SAVED_LOCATIONS | Base URL for saved location service |

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
| EVENT_TOPIC_STORAGE_STATUS | Storage status input |
| EVENT_TOPIC_STORAGE_COMPARTMENT_STATUS | Storage compartment status input |
| EVENT_TOPIC_GACHAPON_REWARD_WON | Gachapon reward win events output |

## Documentation

- [Domain Model](docs/domain.md)
- [Kafka Integration](docs/kafka.md)
- [REST API](docs/rest.md)
- [Storage](docs/storage.md)
