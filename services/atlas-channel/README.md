# atlas-channel

Mushroom game Channel Service

## Overview

A channel server that manages active player sessions, socket connections, and real-time game events. The service handles client packet processing, coordinates with other microservices via Kafka for game state changes, and broadcasts updates to connected clients.

The channel service acts as the primary interface between game clients and the backend system. It maintains encrypted socket connections, processes incoming packets through configurable handlers, and writes responses using version-aware packet writers.

## External Dependencies

- Kafka - Message bus for event consumption and command production
- Jaeger - Distributed tracing
- External REST services:
  - ACCOUNTS - Account data
  - BUDDIES - Buddy list data
  - BUFFS - Character buff data
  - CASHSHOP - Cash shop inventory, wallet, and wishlist
  - CHAIRS - Chair state
  - CHALKBOARDS - Chalkboard state
  - CHANNELS - Channel registration
  - CHARACTERS - Character data
  - CONFIGURATIONS - Service and tenant configuration
  - DATA - Static game data (maps, NPCs, skills)
  - DROPS - Drop state
  - GUILDS - Guild data
  - GUILD_THREADS - Guild BBS
  - INVENTORY - Character inventory (compartments and unified assets)
  - KEYS - Key bindings
  - MAPS - Map character tracking
  - MESSENGERS - Messenger rooms
  - MONSTERS - Monster state
  - NOTES - Mail notes
  - NPC_SHOP - NPC shop data
  - PARTIES - Party data
  - PETS - Pet data
  - QUESTS - Quest progress data
  - REACTORS - Reactor state
  - ROUTES - Transport routes
  - SKILLS - Character skills
  - STORAGE - Storage data and projections
  - WORLDS - World data

## Runtime Configuration

| Variable | Description |
|----------|-------------|
| JAEGER_HOST | Jaeger [host]:[port] |
| LOG_LEVEL | Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace |
| BOOTSTRAP_SERVERS | Kafka [host]:[port] |
| BASE_SERVICE_URL | [scheme]://[host]:[port]/api/ |
| SERVICE_ID | Service instance UUID |
| SERVICE_TYPE | channel-service |
| EVENT_TOPIC_ACCOUNT_STATUS | Account status events |
| EVENT_TOPIC_ACCOUNT_SESSION_STATUS | Account session status events |
| EVENT_TOPIC_ASSET_STATUS | Asset status events |
| EVENT_TOPIC_CASH_COMPARTMENT_STATUS | Cash shop compartment status events |
| EVENT_TOPIC_CASH_SHOP_STATUS | Cash shop status events |
| EVENT_TOPIC_CHAIR_STATUS | Chair status events |
| EVENT_TOPIC_CHALKBOARD_STATUS | Chalkboard status events |
| EVENT_TOPIC_CHARACTER_CHAT | Character chat events |
| EVENT_TOPIC_CHARACTER_STATUS | Character status events |
| EVENT_TOPIC_COMPARTMENT_STATUS | Compartment status events |
| EVENT_TOPIC_CONSUMABLE_STATUS | Consumable status events |
| EVENT_TOPIC_DROP_STATUS | Drop status events |
| EVENT_TOPIC_EXPRESSION | Expression events |
| EVENT_TOPIC_FAME_STATUS | Fame status events |
| EVENT_TOPIC_GACHAPON_REWARD_WON | Gachapon reward won events |
| EVENT_TOPIC_INSTANCE_TRANSPORT | Instance transport events |
| EVENT_TOPIC_MAP_STATUS | Map status events |
| EVENT_TOPIC_MONSTER_STATUS | Monster status events |
| EVENT_TOPIC_NOTE_STATUS | Note status events |
| EVENT_TOPIC_PET_STATUS | Pet status events |
| EVENT_TOPIC_QUEST_STATUS | Quest status events |
| EVENT_TOPIC_REACTOR_STATUS | Reactor status events |
| EVENT_TOPIC_SAGA_STATUS | Saga status events |
| EVENT_TOPIC_SESSION_STATUS | Session status events |
| EVENT_TOPIC_STORAGE_COMPARTMENT_STATUS | Storage compartment status events |
| EVENT_TOPIC_STORAGE_STATUS | Storage status events |
| EVENT_TOPIC_TRANSPORT_STATUS | Transport status events |
| COMMAND_TOPIC_ACCOUNT_SESSION | Account session commands |
| COMMAND_TOPIC_BUDDY_LIST | Buddy list commands |
| COMMAND_TOPIC_CASH_SHOP | Cash shop commands |
| COMMAND_TOPIC_CHAIR | Chair commands |
| COMMAND_TOPIC_CHALKBOARD | Chalkboard commands |
| COMMAND_TOPIC_CHANNEL_STATUS | Channel status commands |
| COMMAND_TOPIC_CHARACTER | Character commands |
| COMMAND_TOPIC_CHARACTER_BUFF | Buff commands |
| COMMAND_TOPIC_CHARACTER_CHAT | Chat commands |
| COMMAND_TOPIC_CHARACTER_MOVEMENT | Character movement commands |
| COMMAND_TOPIC_COMPARTMENT | Compartment commands |
| COMMAND_TOPIC_CONSUMABLE | Consumable commands |
| COMMAND_TOPIC_DROP | Drop commands |
| COMMAND_TOPIC_EXPRESSION | Expression commands |
| COMMAND_TOPIC_FAME | Fame commands |
| COMMAND_TOPIC_GUILD | Guild commands |
| COMMAND_TOPIC_GUILD_THREAD | Guild thread commands |
| COMMAND_TOPIC_INVITE | Invite commands |
| COMMAND_TOPIC_MESSENGER | Messenger commands |
| COMMAND_TOPIC_MONSTER | Monster commands |
| COMMAND_TOPIC_MONSTER_MOVEMENT | Monster movement commands |
| COMMAND_TOPIC_NOTE | Note commands |
| COMMAND_TOPIC_NPC | NPC commands |
| COMMAND_TOPIC_NPC_CONVERSATION | NPC conversation commands |
| COMMAND_TOPIC_NPC_SHOP | NPC shop commands |
| COMMAND_TOPIC_PARTY | Party commands |
| COMMAND_TOPIC_PET | Pet commands |
| COMMAND_TOPIC_PET_MOVEMENT | Pet movement commands |
| COMMAND_TOPIC_PORTAL | Portal commands |
| COMMAND_TOPIC_QUEST | Quest commands |
| COMMAND_TOPIC_QUEST_CONVERSATION | Quest conversation commands |
| COMMAND_TOPIC_REACTOR | Reactor commands |
| COMMAND_TOPIC_SAGA | Saga commands |
| COMMAND_TOPIC_SKILL | Skill commands |
| COMMAND_TOPIC_SKILL_MACRO | Skill macro commands |
| COMMAND_TOPIC_STORAGE | Storage commands |
| COMMAND_TOPIC_SYSTEM_MESSAGE | System message commands |

## Documentation

- [Domain Documentation](docs/domain.md)
- [Kafka Documentation](docs/kafka.md)
- [REST Documentation](docs/rest.md)
- [Storage Documentation](docs/storage.md)
