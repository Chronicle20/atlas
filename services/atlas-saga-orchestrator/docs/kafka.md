# Kafka Integration

## Topics Consumed

| Topic | Environment Variable | Direction | Description |
|-------|---------------------|-----------|-------------|
| Saga Commands | COMMAND_TOPIC_SAGA | Command | Saga creation requests |
| Asset Status | EVENT_TOPIC_ASSET_STATUS | Event | Asset service status events (CREATED, DELETED, MOVED, QUANTITY_CHANGED) |
| Buddy List Status | EVENT_TOPIC_BUDDY_LIST_STATUS | Event | Buddy list status events |
| Wallet Status | EVENT_TOPIC_WALLET_STATUS | Event | Wallet status events (UPDATED) |
| Cash Shop Compartment Status | EVENT_TOPIC_CASH_COMPARTMENT_STATUS | Event | Cash shop compartment status events (ACCEPTED, RELEASED, ERROR) |
| Character Status | EVENT_TOPIC_CHARACTER_STATUS | Event | Character service status events |
| Compartment Status | EVENT_TOPIC_COMPARTMENT_STATUS | Event | Inventory compartment status events (CREATED, DELETED, ACCEPTED, RELEASED, CREATION_FAILED, ERROR) |
| Consumable Status | EVENT_TOPIC_CONSUMABLE_STATUS | Event | Consumable status events |
| Guild Status | EVENT_TOPIC_GUILD_STATUS | Event | Guild service status events |
| Invite Status | EVENT_TOPIC_INVITE_STATUS | Event | Invite status events (CREATED, ACCEPTED, REJECTED) |
| Pet Status | EVENT_TOPIC_PET_STATUS | Event | Pet service status events |
| Quest Status | EVENT_TOPIC_QUEST_STATUS | Event | Quest service status events (STARTED, COMPLETED) |
| Skill Status | EVENT_TOPIC_SKILL_STATUS | Event | Skill service status events (CREATED, UPDATED) |
| Storage Status | EVENT_TOPIC_STORAGE_STATUS | Event | Storage service status events (DEPOSITED, WITHDRAWN, MESOS_UPDATED, ERROR) |
| Storage Compartment Status | EVENT_TOPIC_STORAGE_COMPARTMENT_STATUS | Event | Storage compartment status events (ACCEPTED, RELEASED, ERROR) |

## Topics Produced

| Topic | Environment Variable | Direction | Description |
|-------|---------------------|-----------|-------------|
| Saga Status | EVENT_TOPIC_SAGA_STATUS | Event | Saga completion and failure events |
| Compartment Commands | COMMAND_TOPIC_COMPARTMENT | Command | Inventory operations (CREATE_ASSET, DESTROY, EQUIP, UNEQUIP, ACCEPT, RELEASE) |
| Character Commands | COMMAND_TOPIC_CHARACTER | Command | Character operations |
| Skill Commands | COMMAND_TOPIC_SKILL | Command | Skill operations |
| Guild Commands | COMMAND_TOPIC_GUILD | Command | Guild operations |
| Invite Commands | COMMAND_TOPIC_INVITE | Command | Invitation operations |
| Buddy List Commands | COMMAND_TOPIC_BUDDY_LIST | Command | Buddy list operations |
| Pet Commands | COMMAND_TOPIC_PET | Command | Pet operations |
| Quest Commands | COMMAND_TOPIC_QUEST | Command | Quest operations |
| Consumable Commands | COMMAND_TOPIC_CONSUMABLE | Command | Consumable operations |
| System Message Commands | COMMAND_TOPIC_SYSTEM_MESSAGE | Command | System message commands |
| Storage Commands | COMMAND_TOPIC_STORAGE | Command | Storage operations (DEPOSIT, WITHDRAW, UPDATE_MESOS, DEPOSIT_ROLLBACK, SHOW_STORAGE) |
| Storage Compartment Commands | COMMAND_TOPIC_STORAGE_COMPARTMENT | Command | Storage compartment operations (ACCEPT, RELEASE) |
| Wallet Commands | COMMAND_TOPIC_WALLET | Command | Cash shop wallet operations (ADJUST_CURRENCY) |
| Cash Shop Compartment Commands | COMMAND_TOPIC_CASH_COMPARTMENT | Command | Cash shop compartment operations (ACCEPT, RELEASE) |
| Portal Commands | COMMAND_TOPIC_PORTAL | Command | Portal blocking operations (BLOCK, UNBLOCK) |
| Buff Commands | COMMAND_TOPIC_CHARACTER_BUFF | Command | Buff operations (CANCEL_ALL) |
| Gachapon Reward Won | EVENT_TOPIC_GACHAPON_REWARD_WON | Event | Gachapon reward win announcements |

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

### Asset Status Event (Consumed)

```
StatusEvent[E]
  transactionId: uuid.UUID
  characterId: uint32
  compartmentId: uuid.UUID
  assetId: uint32
  templateId: uint32
  slot: int16
  type: string
  body: E
```

Status types: CREATED, DELETED, MOVED, QUANTITY_CHANGED

The asset consumer handles `CREATED` events with special logic for `CreateAndEquipAsset` steps -- it dynamically adds an `EquipAsset` step to the saga after the current step, using the slot and template from the event to determine source slot and inventory type. For CREATED and QUANTITY_CHANGED events, the step is completed with a result containing `assetId`.

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

Command types: CREATE_ASSET, DESTROY, EQUIP, UNEQUIP, ACCEPT, RELEASE

#### ACCEPT Body

```
AcceptCommandBody
  transactionId: uuid.UUID
  templateId: uint32
  <flat AssetData fields>
```

#### RELEASE Body

```
ReleaseCommandBody
  transactionId: uuid.UUID
  assetId: uint32
  quantity: uint32
```

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

Command types: CREATE_CHARACTER, CHANGE_MAP, CHANGE_JOB, CHANGE_HAIR, CHANGE_FACE, CHANGE_SKIN, AWARD_EXPERIENCE, DEDUCT_EXPERIENCE, AWARD_LEVEL, REQUEST_CHANGE_MESO, REQUEST_CHANGE_FAME, SET_HP, RESET_STATS

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

### Storage Command

Produced to perform storage operations.

```
Command[E]
  transactionId: uuid.UUID
  worldId: world.Id
  accountId: uint32
  type: string
  body: E
```

Command types: DEPOSIT, WITHDRAW, UPDATE_MESOS, DEPOSIT_ROLLBACK, SHOW_STORAGE

### Storage Status Event (Consumed)

```
StatusEvent[E]
  transactionId: uuid.UUID
  worldId: world.Id
  accountId: uint32
  type: string
  body: E
```

Status types: DEPOSITED, WITHDRAWN, MESOS_UPDATED, ERROR

### Storage Compartment Command

Produced to perform storage accept/release operations.

```
Command[E]
  worldId: world.Id
  accountId: uint32
  characterId: uint32 (optional, for client notification)
  type: string
  body: E
```

Command types: ACCEPT, RELEASE

#### ACCEPT Body

```
AcceptCommandBody
  transactionId: uuid.UUID
  templateId: uint32
  <flat AssetData fields>
```

#### RELEASE Body

```
ReleaseCommandBody
  transactionId: uuid.UUID
  assetId: asset.Id
  quantity: asset.Quantity
```

### Storage Compartment Status Event (Consumed)

```
StatusEvent[E]
  worldId: world.Id
  accountId: uint32
  characterId: uint32 (optional)
  type: string
  body: E
```

Status types: ACCEPTED, RELEASED, ERROR

### Wallet Command

Produced to adjust cash shop currency.

```
AdjustCurrencyCommand
  transactionId: uuid.UUID
  accountId: uint32
  currencyType: uint32 (1=credit, 2=points, 3=prepaid)
  amount: int32 (negative for deduction)
  type: string (ADJUST_CURRENCY)
```

### Wallet Status Event (Consumed)

```
StatusEvent[E]
  accountId: uint32
  type: string (UPDATED)
  body: E
```

#### Updated Body

```
StatusEventUpdatedBody
  credit: uint32
  points: uint32
  prepaid: uint32
  transactionId: uuid.UUID (optional, nil for non-saga updates)
```

Events without a transactionId are skipped (non-saga wallet updates).

### Cash Shop Compartment Command

Produced to perform cash shop accept/release operations.

```
Command[E]
  accountId: uint32
  characterId: uint32
  compartmentType: byte
  type: string
  body: E
```

Command types: ACCEPT, RELEASE

### Cash Shop Compartment Status Event (Consumed)

```
StatusEvent[E]
  compartmentId: uuid.UUID
  compartmentType: byte
  type: string
  body: E
```

Status types: ACCEPTED, RELEASED, ERROR

### Gachapon Reward Won Event

Produced when a gachapon reward is won (uncommon/rare tier only).

```
RewardWonEvent
  characterId: uint32
  worldId: byte
  itemId: uint32
  quantity: uint32
  tier: string
  gachaponId: string
  gachaponName: string
  assetId: uint32
```

## Transaction Semantics

- Each saga step produces a command with the saga's transactionId
- Step completion is tracked by consuming status events with matching transactionId
- Status events without matching transactionId are ignored (saga not found in cache)
- Failed status events trigger step failure and compensation
- Synchronous actions (play_portal_sound, show_info, show_info_text, update_area_info, show_hint, show_guide_hint, show_intro, block_portal, unblock_portal, emit_gachapon_win) complete immediately after command emission
- REST-based synchronous actions (start_instance_transport, save_location, warp_to_saved_location, select_gachapon_reward) complete after the REST call returns
- Asset CREATED and QUANTITY_CHANGED events carry `assetId` as step result data for downstream steps

## Ordering

- Commands are keyed by characterId for partition ordering
- Steps execute sequentially within a saga
- Status events are processed in arrival order
- Compensation steps execute in reverse order of completion
