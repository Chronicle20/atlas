# Kafka Integration

## Topics Consumed

| Topic | Environment Variable | Direction | Description |
|-------|---------------------|-----------|-------------|
| Saga Commands | COMMAND_TOPIC_SAGA | Command | Saga creation requests |
| Asset Status | EVENT_TOPIC_ASSET_STATUS | Event | Asset service status events |
| Buddy List Status | EVENT_TOPIC_BUDDYLIST_STATUS | Event | Buddy list status events |
| Cash Shop Status | EVENT_TOPIC_CASHSHOP_STATUS | Event | Cash shop status events |
| Cash Shop Compartment Status | EVENT_TOPIC_CASHSHOP_COMPARTMENT_STATUS | Event | Cash shop compartment status events |
| Character Status | EVENT_TOPIC_CHARACTER_STATUS | Event | Character service status events |
| Compartment Status | EVENT_TOPIC_COMPARTMENT_STATUS | Event | Inventory compartment status events |
| Consumable Status | EVENT_TOPIC_CONSUMABLE_STATUS | Event | Consumable status events |
| Guild Status | EVENT_TOPIC_GUILD_STATUS | Event | Guild service status events |
| Pet Status | EVENT_TOPIC_PET_STATUS | Event | Pet service status events |
| Quest Status | EVENT_TOPIC_QUEST_STATUS | Event | Quest service status events |
| Skill Status | EVENT_TOPIC_SKILL_STATUS | Event | Skill service status events |
| Storage Status | EVENT_TOPIC_STORAGE_STATUS | Event | Storage service status events |
| Storage Compartment Status | EVENT_TOPIC_STORAGE_COMPARTMENT_STATUS | Event | Storage compartment status events |

## Topics Produced

| Topic | Environment Variable | Direction | Description |
|-------|---------------------|-----------|-------------|
| Saga Status | EVENT_TOPIC_SAGA_STATUS | Event | Saga completion and failure events |
| Compartment Commands | COMMAND_TOPIC_COMPARTMENT | Command | Inventory operations |
| Character Commands | COMMAND_TOPIC_CHARACTER | Command | Character operations |
| Skill Commands | COMMAND_TOPIC_SKILL | Command | Skill operations |
| Guild Commands | COMMAND_TOPIC_GUILD | Command | Guild operations |
| Invite Commands | COMMAND_TOPIC_INVITE | Command | Invitation operations |
| Buddy List Commands | COMMAND_TOPIC_BUDDYLIST | Command | Buddy list operations |
| Pet Commands | COMMAND_TOPIC_PET | Command | Pet operations |
| Monster Commands | COMMAND_TOPIC_MONSTER | Command | Monster spawn commands |
| Quest Commands | COMMAND_TOPIC_QUEST | Command | Quest operations |
| Consumable Commands | COMMAND_TOPIC_CONSUMABLE | Command | Consumable operations |
| System Message Commands | COMMAND_TOPIC_SYSTEM_MESSAGE | Command | System message commands |
| Storage Commands | COMMAND_TOPIC_STORAGE | Command | Storage operations |
| Storage Compartment Commands | COMMAND_TOPIC_STORAGE_COMPARTMENT | Command | Storage compartment operations |
| Cash Shop Commands | COMMAND_TOPIC_CASHSHOP | Command | Cash shop operations |
| Cash Shop Compartment Commands | COMMAND_TOPIC_CASHSHOP_COMPARTMENT | Command | Cash shop compartment operations |
| Portal Commands | COMMAND_TOPIC_PORTAL | Command | Portal blocking operations |
| Buff Commands | COMMAND_TOPIC_BUFF | Command | Buff operations |

## Message Types

### Saga Status Event

Produced when a saga completes or fails.

```
StatusEvent[E]
  transactionId: uuid.UUID
  type: string (COMPLETED, FAILED)
  body: E
```

#### Completed Body

Empty body indicating successful completion.

#### Failed Body

```
StatusEventFailedBody
  reason: string
  failedStep: string
  characterId: uint32
  sagaType: string
  errorCode: string (NOT_ENOUGH_MESOS, INVENTORY_FULL, STORAGE_FULL, UNKNOWN)
```

### Compartment Command

Produced to perform inventory operations.

```
Command[E]
  transactionId: uuid.UUID
  characterId: uint32
  inventoryType: byte
  type: string
  body: E
```

Command types: CREATE, DELETE, EQUIP, UNEQUIP, CREATE_AND_EQUIP, ACCEPT, RELEASE

### Character Command

Produced to perform character operations.

```
Command[E]
  transactionId: uuid.UUID
  worldId: world.Id
  characterId: uint32
  type: string
  body: E
```

Command types: CREATE_CHARACTER, CHANGE_MAP, CHANGE_JOB, CHANGE_HAIR, CHANGE_FACE, CHANGE_SKIN, AWARD_EXPERIENCE, DEDUCT_EXPERIENCE, AWARD_LEVEL, REQUEST_CHANGE_MESO, REQUEST_CHANGE_FAME, SET_HP

### Compartment Status Event (Consumed)

```
StatusEvent[E]
  transactionId: uuid.UUID
  characterId: uint32
  compartmentId: uuid.UUID
  type: string
  body: E
```

Status types: CREATED, DELETED, ACCEPTED, RELEASED, CREATION_FAILED, ERROR

### Character Status Event (Consumed)

```
StatusEvent[E]
  transactionId: uuid.UUID
  worldId: world.Id
  characterId: uint32
  type: string
  body: E
```

Status types: CREATED, MAP_CHANGED, JOB_CHANGED, EXPERIENCE_CHANGED, LEVEL_CHANGED, MESO_CHANGED, FAME_CHANGED, STAT_CHANGED, CREATION_FAILED, ERROR

## Transaction Semantics

- Each saga step produces a command with the saga's transactionId
- Step completion is tracked by consuming status events with matching transactionId
- Status events without matching transactionId are ignored
- Failed status events trigger step failure and compensation
- Synchronous actions (play_portal_sound, show_info, show_hint, block_portal, unblock_portal) complete immediately after command emission

## Ordering

- Commands are keyed by transactionId for partition ordering
- Steps execute sequentially within a saga
- Status events are processed in arrival order
- Compensation steps execute in reverse order of completion
