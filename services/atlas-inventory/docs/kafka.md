# Kafka

## Topics Consumed

### EVENT_TOPIC_CHARACTER_STATUS

Character lifecycle events.

| Type | Handler |
|------|---------|
| CREATED | Creates inventory for character |
| DELETED | Deletes inventory for character |

### COMMAND_TOPIC_COMPARTMENT

Compartment operation commands.

| Type | Handler |
|------|---------|
| EQUIP | Equips item from source slot to destination equipment slot |
| UNEQUIP | Unequips item from equipment slot to inventory slot |
| MOVE | Moves item between slots (swaps or merges if applicable) |
| DROP | Drops item from inventory to map |
| REQUEST_RESERVE | Reserves items for a transaction |
| CONSUME | Consumes a reserved item |
| DESTROY | Destroys item or reduces quantity |
| CANCEL_RESERVATION | Cancels an item reservation |
| INCREASE_CAPACITY | Increases compartment capacity |
| CREATE_ASSET | Creates a new asset in compartment |
| RECHARGE | Recharges a rechargeable asset |
| MERGE | Merges stackable assets and compacts |
| SORT | Compacts and sorts assets by templateId |
| ACCEPT | Accepts a cash item into compartment |
| RELEASE | Releases an asset from compartment |

### EVENT_TOPIC_DROP_STATUS

Drop reservation events.

| Type | Handler |
|------|---------|
| RESERVED | Attempts equipment or item pickup |

### EVENT_TOPIC_EQUIPABLE_STATUS

Equipable update events.

| Type | Handler |
|------|---------|
| UPDATED | Relays asset update event |

---

## Topics Produced

### EVENT_TOPIC_ASSET_STATUS

Asset state change events.

| Type | Description |
|------|-------------|
| CREATED | Asset created in compartment |
| UPDATED | Asset reference data updated |
| DELETED | Asset deleted from compartment |
| MOVED | Asset moved to different slot |
| QUANTITY_CHANGED | Asset quantity updated |

### EVENT_TOPIC_COMPARTMENT_STATUS

Compartment state change events.

| Type | Description |
|------|-------------|
| CREATED | Compartment created |
| DELETED | Compartment deleted |
| CAPACITY_CHANGED | Compartment capacity updated |
| RESERVED | Items reserved for transaction |
| RESERVATION_CANCELLED | Reservation cancelled |
| MERGE_COMPLETE | Merge and compact operation completed |
| SORT_COMPLETE | Compact and sort operation completed |
| ACCEPTED | Cash item accepted into compartment |
| RELEASED | Asset released from compartment |
| ERROR | Operation failed (ACCEPT_COMMAND_FAILED, RELEASE_COMMAND_FAILED) |

### EVENT_TOPIC_INVENTORY_STATUS

Inventory lifecycle events.

| Type | Description |
|------|-------------|
| CREATED | Inventory created for character |
| DELETED | Inventory deleted for character |

### COMMAND_TOPIC_DROP

Drop operation commands.

| Type | Description |
|------|-------------|
| SPAWN_FROM_CHARACTER | Creates a drop on the map |
| CANCEL_RESERVATION | Cancels drop reservation |
| REQUEST_PICK_UP | Requests drop pickup completion |

---

## Message Types

### Asset Status Event

```
StatusEvent[Body] {
  transactionId: UUID
  characterId: uint32
  compartmentId: UUID
  assetId: uint32
  templateId: uint32
  slot: int16
  type: string
  body: Body
}
```

### Compartment Command

```
Command[Body] {
  transactionId: UUID
  characterId: uint32
  inventoryType: byte
  type: string
  body: Body
}
```

### Compartment Status Event

```
StatusEvent[Body] {
  transactionId: UUID
  characterId: uint32
  compartmentId: UUID
  type: string
  body: Body
}
```

### Inventory Status Event

```
StatusEvent[Body] {
  characterId: uint32
  type: string
  body: Body
}
```

### Drop Command

```
Command[Body] {
  worldId: byte
  channelId: byte
  mapId: uint32
  type: string
  body: Body
}
```

---

## Transaction Semantics

- Commands include `transactionId` for correlation
- Events include `transactionId` matching originating command
- Reservations have 30-second timeout
