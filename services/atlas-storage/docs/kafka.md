# Kafka

## Topics Consumed

### COMMAND_TOPIC_STORAGE

Storage operation commands.

| Command Type | Body Type | Description |
|--------------|-----------|-------------|
| `DEPOSIT` | DepositBody | Deposit an item into storage |
| `WITHDRAW` | WithdrawBody | Withdraw an item from storage |
| `UPDATE_MESOS` | UpdateMesosBody | Update stored mesos |
| `DEPOSIT_ROLLBACK` | DepositRollbackBody | Rollback a deposit operation |
| `ARRANGE` | ArrangeBody | Merge and sort storage items |
| `SHOW_STORAGE` | ShowStorageCommand | Create projection for storage UI |
| `CLOSE_STORAGE` | CloseStorageCommand | Destroy projection when closing storage |
| `EXPIRE` | ExpireBody | Expire an asset from storage |

### COMMAND_TOPIC_STORAGE_COMPARTMENT

Compartment transfer commands (saga participation).

| Command Type | Body Type | Description |
|--------------|-----------|-------------|
| `ACCEPT` | AcceptCommandBody | Accept an item into storage |
| `RELEASE` | ReleaseCommandBody | Release an item from storage |

### EVENT_TOPIC_CHARACTER_STATUS

Character status events (cleanup triggers).

| Event Type | Body Type | Description |
|------------|-----------|-------------|
| `LOGOUT` | StatusEventLogoutBody | Character logged out |
| `CHANNEL_CHANGED` | ChangeChannelEventLoginBody | Character changed channels |
| `MAP_CHANGED` | StatusEventMapChangedBody | Character changed maps |

### EVENT_TOPIC_ACCOUNT_STATUS

Account status events (cleanup triggers).

| Event Type | Body Type | Description |
|------------|-----------|-------------|
| `DELETED` | StatusEvent | Account deleted - triggers storage cleanup |

---

## Topics Produced

### EVENT_TOPIC_STORAGE_STATUS

Storage status events.

| Event Type | Body Type | Description |
|------------|-----------|-------------|
| `DEPOSITED` | DepositedEventBody | Item deposited into storage |
| `WITHDRAWN` | WithdrawnEventBody | Item withdrawn from storage |
| `MESOS_UPDATED` | MesosUpdatedEventBody | Mesos updated in storage |
| `ARRANGED` | ArrangedEventBody | Storage arranged (merged and sorted) |
| `ERROR` | ErrorEventBody | Operation error occurred |
| `PROJECTION_CREATED` | ProjectionCreatedEventBody | Storage projection created |
| `PROJECTION_DESTROYED` | ProjectionDestroyedEventBody | Storage projection destroyed |
| `EXPIRED` | ExpiredStatusEventBody | Asset expired from storage |

### EVENT_TOPIC_STORAGE_COMPARTMENT_STATUS

Compartment transfer status events.

| Event Type | Body Type | Description |
|------------|-----------|-------------|
| `ACCEPTED` | StatusEventAcceptedBody | Item accepted into storage |
| `RELEASED` | StatusEventReleasedBody | Item released from storage |
| `ERROR` | StatusEventErrorBody | Transfer error occurred |

---

## Message Types

### Command Messages

**Command[E]**
```
transactionId: UUID
worldId: world.Id
accountId: uint32
type: string
body: E
```

**AssetData** (embedded in DepositBody and AcceptCommandBody)
```
expiration: time.Time
quantity: uint32
ownerId: uint32
flag: uint16
rechargeable: uint64
strength: uint16
dexterity: uint16
intelligence: uint16
luck: uint16
hp: uint16
mp: uint16
weaponAttack: uint16
magicAttack: uint16
weaponDefense: uint16
magicDefense: uint16
accuracy: uint16
avoidability: uint16
hands: uint16
speed: uint16
jump: uint16
slots: uint16
levelType: byte
level: byte
experience: uint32
hammersApplied: uint32
cashId: int64
commodityId: uint32
purchaseBy: uint32
petId: uint32
```

**DepositBody**
```
slot: int16
templateId: uint32
(embeds AssetData)
```

**WithdrawBody**
```
assetId: asset.Id
quantity: asset.Quantity (optional)
```

**UpdateMesosBody**
```
mesos: uint32
operation: string (SET, ADD, SUBTRACT)
```

**DepositRollbackBody**
```
assetId: asset.Id
```

**ArrangeBody**
```
(empty)
```

**ShowStorageCommand**
```
transactionId: UUID
worldId: world.Id
channelId: channel.Id
characterId: uint32
npcId: uint32
accountId: uint32
type: string
```

**CloseStorageCommand**
```
characterId: uint32
type: string
```

**ExpireBody**
```
characterId: uint32
assetId: asset.Id
templateId: uint32
inventoryType: int8
slot: int16
replaceItemId: uint32
replaceMessage: string
```

**AcceptCommandBody** (compartment)
```
transactionId: UUID
templateId: uint32
(embeds AssetData)
```

**ReleaseCommandBody** (compartment)
```
transactionId: UUID
assetId: asset.Id
quantity: asset.Quantity
```

### Event Messages

**StatusEvent[E]** (storage)
```
transactionId: UUID
worldId: world.Id
accountId: uint32
type: string
body: E
```

**DepositedEventBody**
```
assetId: asset.Id
slot: int16
templateId: uint32
```

**WithdrawnEventBody**
```
assetId: asset.Id
slot: int16
templateId: uint32
quantity: asset.Quantity (optional)
```

**MesosUpdatedEventBody**
```
oldMesos: uint32
newMesos: uint32
```

**ArrangedEventBody**
```
(empty)
```

**ErrorEventBody**
```
errorCode: string
message: string (optional)
```

**ProjectionCreatedEventBody**
```
characterId: uint32
accountId: uint32
worldId: world.Id
channelId: channel.Id
npcId: uint32
```

**ProjectionDestroyedEventBody**
```
characterId: uint32
```

**ExpiredStatusEventBody**
```
isCash: bool
replaceItemId: uint32 (optional)
replaceMessage: string (optional)
```

**StatusEvent[E]** (compartment)
```
worldId: world.Id
accountId: uint32
characterId: uint32 (optional)
type: string
body: E
```

**StatusEventAcceptedBody**
```
transactionId: UUID
assetId: asset.Id
slot: int16
inventoryType: byte
```

**StatusEventReleasedBody**
```
transactionId: UUID
assetId: asset.Id
inventoryType: byte
```

**StatusEventErrorBody**
```
transactionId: UUID
errorCode: string
message: string (optional)
```

### Account Status Event Messages

**StatusEvent**
```
accountId: uint32
name: string
status: string
```

---

## Transaction Semantics

- Commands include transactionId for correlation with saga orchestration
- Status events include transactionId to enable saga coordination
- Compartment commands (ACCEPT/RELEASE) participate in two-phase transfer sagas
- Error events are emitted on operation failures with error codes
- Account deletion events trigger cascade deletion of all storage data

---

## Error Codes

| Code | Description |
|------|-------------|
| `STORAGE_FULL` | Storage capacity exceeded |
| `NOT_ENOUGH_MESOS` | Insufficient mesos for operation |
| `ONE_OF_A_KIND` | Item cannot be stored (unique) |
| `GENERIC` | General operation failure |
