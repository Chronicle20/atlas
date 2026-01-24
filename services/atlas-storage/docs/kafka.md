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

### COMMAND_TOPIC_STORAGE_SHOW

Storage session commands.

| Command Type | Body Type | Description |
|--------------|-----------|-------------|
| `SHOW_STORAGE` | ShowStorageCommand | Create projection for storage UI |
| `CLOSE_STORAGE` | CloseStorageCommand | Destroy projection when closing storage |

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
worldId: byte
accountId: uint32
type: string
body: E
```

**DepositBody**
```
slot: int16
templateId: uint32
expiration: time.Time
referenceId: uint32
referenceType: string
referenceData: ReferenceData (optional)
```

**ReferenceData**
```
quantity: uint32
ownerId: uint32
flag: uint16
```

**WithdrawBody**
```
assetId: uint32
targetSlot: int16 (optional)
quantity: uint32 (optional)
targetStorageId: string (optional)
```

**UpdateMesosBody**
```
mesos: uint32
operation: string (SET, ADD, SUBTRACT)
```

**DepositRollbackBody**
```
assetId: uint32
```

**ArrangeBody**
```
(empty)
```

**ShowStorageCommand**
```
transactionId: UUID
worldId: byte
channelId: byte
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

**AcceptCommandBody**
```
transactionId: UUID
slot: int16
templateId: uint32
referenceId: uint32
referenceType: string
referenceData: json.RawMessage (optional)
quantity: uint32
```

**ReleaseCommandBody**
```
transactionId: UUID
assetId: uint32
quantity: uint32
```

### Event Messages

**StatusEvent[E]**
```
transactionId: UUID
worldId: byte
accountId: uint32
type: string
body: E
```

**DepositedEventBody**
```
assetId: uint32
slot: int16
templateId: uint32
referenceId: uint32
referenceType: string
expiration: time.Time
```

**WithdrawnEventBody**
```
assetId: uint32
slot: int16
templateId: uint32
quantity: uint32 (optional)
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
worldId: byte
channelId: byte
npcId: uint32
```

**ProjectionDestroyedEventBody**
```
characterId: uint32
```

**StatusEventAcceptedBody**
```
transactionId: UUID
assetId: uint32
slot: int16
inventoryType: byte
```

**StatusEventReleasedBody**
```
transactionId: UUID
assetId: uint32
inventoryType: byte
```

**StatusEventErrorBody**
```
transactionId: UUID
errorCode: string
message: string (optional)
```

---

## Transaction Semantics

- Commands include transactionId for correlation with saga orchestration
- Status events include transactionId to enable saga coordination
- Compartment commands (ACCEPT/RELEASE) participate in two-phase transfer sagas
- Error events are emitted on operation failures with error codes

---

## Error Codes

| Code | Description |
|------|-------------|
| `STORAGE_FULL` | Storage capacity exceeded |
| `NOT_ENOUGH_MESOS` | Insufficient mesos for operation |
| `ONE_OF_A_KIND` | Item cannot be stored (unique) |
| `GENERIC` | General operation failure |
