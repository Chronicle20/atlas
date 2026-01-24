# atlas-saga-orchestrator

Coordinates distributed transactions across Atlas microservices using the saga pattern. Tracks step execution and performs compensation on failure to maintain data consistency.

## External Dependencies

- Kafka (message broker)
- Jaeger (distributed tracing)

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| BOOTSTRAP_SERVERS | Kafka bootstrap servers |
| JAEGER_HOST_PORT | Jaeger host and port |
| LOG_LEVEL | Logging level (Panic/Fatal/Error/Warn/Info/Debug/Trace) |
| REST_PORT | REST API server port |

### Kafka Topics

| Variable | Description |
|----------|-------------|
| COMMAND_TOPIC_SAGA | Saga command input |
| COMMAND_TOPIC_COMPARTMENT | Inventory compartment commands |
| COMMAND_TOPIC_CHARACTER | Character commands |
| COMMAND_TOPIC_SKILL | Skill commands |
| COMMAND_TOPIC_GUILD | Guild commands |
| COMMAND_TOPIC_INVITE | Invitation commands |
| COMMAND_TOPIC_BUDDYLIST | Buddy list commands |
| COMMAND_TOPIC_PET | Pet commands |
| COMMAND_TOPIC_MONSTER | Monster commands |
| COMMAND_TOPIC_QUEST | Quest commands |
| COMMAND_TOPIC_CONSUMABLE | Consumable commands |
| COMMAND_TOPIC_SYSTEM_MESSAGE | System message commands |
| COMMAND_TOPIC_STORAGE | Storage commands |
| COMMAND_TOPIC_STORAGE_COMPARTMENT | Storage compartment commands |
| COMMAND_TOPIC_CASHSHOP | Cash shop commands |
| COMMAND_TOPIC_CASHSHOP_COMPARTMENT | Cash shop compartment commands |
| COMMAND_TOPIC_PORTAL | Portal commands |
| COMMAND_TOPIC_BUFF | Buff commands |
| EVENT_TOPIC_SAGA_STATUS | Saga status output |
| EVENT_TOPIC_ASSET_STATUS | Asset status input |
| EVENT_TOPIC_BUDDYLIST_STATUS | Buddy list status input |
| EVENT_TOPIC_CASHSHOP_STATUS | Cash shop status input |
| EVENT_TOPIC_CASHSHOP_COMPARTMENT_STATUS | Cash shop compartment status input |
| EVENT_TOPIC_CHARACTER_STATUS | Character status input |
| EVENT_TOPIC_COMPARTMENT_STATUS | Compartment status input |
| EVENT_TOPIC_CONSUMABLE_STATUS | Consumable status input |
| EVENT_TOPIC_GUILD_STATUS | Guild status input |
| EVENT_TOPIC_PET_STATUS | Pet status input |
| EVENT_TOPIC_QUEST_STATUS | Quest status input |
| EVENT_TOPIC_SKILL_STATUS | Skill status input |
| EVENT_TOPIC_STORAGE_STATUS | Storage status input |
| EVENT_TOPIC_STORAGE_COMPARTMENT_STATUS | Storage compartment status input |

## Documentation

- [Domain Model](docs/domain.md)
- [Kafka Integration](docs/kafka.md)
- [REST API](docs/rest.md)
- [Storage](docs/storage.md)
