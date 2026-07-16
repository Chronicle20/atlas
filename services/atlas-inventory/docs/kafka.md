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
| ACCEPT | Accepts an asset into compartment |
| RELEASE | Releases an asset from compartment |
| EXPIRE | Expires an asset with optional replacement |
| MODIFY_EQUIPMENT | Updates equipment stats on an asset |
| SET_OWNER | Stamps the owner field onto an asset in a given slot |
| APPLY_LOCK | Applies a permanent or timed lock (FlagLock + expiration) to an asset in a given slot; rejects a non-locked asset that already has a non-zero expiration |
| CHANGE_TEMPLATE | Swaps a pet asset's templateId in place, resolved by petId within the Cash compartment |

### EVENT_TOPIC_DROP_STATUS

Drop reservation events.

| Type | Handler |
|------|---------|
| RESERVED | Attempts equipment or item pickup based on item type |

---

## Topics Produced

### EVENT_TOPIC_ASSET_STATUS

Asset state change events.

| Type | Description |
|------|-------------|
| CREATED | Asset created in compartment |
| UPDATED | Asset equipment stats updated |
| DELETED | Asset deleted from compartment |
| MOVED | Asset moved to different slot |
| QUANTITY_CHANGED | Asset quantity updated |
| ACCEPTED | Asset accepted into compartment from external source |
| RELEASED | Asset released from compartment to external destination |
| EXPIRED | Asset expired from compartment |

Asset UPDATED is also emitted when a pet asset's templateId is changed via CHANGE_TEMPLATE.

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
| ACCEPTED | Asset accepted into compartment |
| RELEASED | Asset released from compartment |
| ERROR | Operation failed (ACCEPT_COMMAND_FAILED, RELEASE_COMMAND_FAILED) |
| CREATION_FAILED | Asset creation failed (CREATE_ASSET_TEMPLATE_NOT_FOUND, CREATE_ASSET_INVENTORY_FULL, CREATE_ASSET_UNKNOWN_ERROR) |

### EVENT_TOPIC_INVENTORY_STATUS

Inventory lifecycle events.

| Type | Description |
|------|-------------|
| CREATED | Inventory created for character |
| CREATION_FAILED | Inventory creation failed |
| DELETED | Inventory deleted for character |

### COMMAND_TOPIC_ITEM_CONSUMED_ON_PICKUP

| Type | Description |
|------|-------------|
| ITEM_CONSUMED_ON_PICKUP | Item flagged consumeOnPickup was consumed directly from a drop pickup without entering the inventory |

### COMMAND_TOPIC_DROP

Drop operation commands.

| Type | Description |
|------|-------------|
| SPAWN_FROM_CHARACTER | Creates a drop on the map (equipment or item) |
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

Body types:
- `CreatedStatusEventBody` - embeds `AssetData` (all asset fields)
- `UpdatedStatusEventBody` - embeds `AssetData` (all asset fields)
- `DeletedStatusEventBody` - empty
- `MovedStatusEventBody` - oldSlot (int16), createdAt (time)
- `QuantityChangedEventBody` - quantity (uint32)
- `AcceptedStatusEventBody` - embeds `AssetData` (all asset fields)
- `ReleasedStatusEventBody` - embeds `AssetData` (all asset fields)
- `ExpiredStatusEventBody` - isCash (bool), replaceItemId (uint32), replaceMessage (string)

`AssetData` contains: expiration, createdAt, quantity, ownerId, owner, flag, rechargeable, strength, dexterity, intelligence, luck, hp, mp, weaponAttack, magicAttack, weaponDefense, magicDefense, accuracy, avoidability, hands, speed, jump, slots, levelType, level, experience, hammersApplied, equippedSince, cashId, commodityId, purchaseBy, petId.

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

Body types:
- `EquipCommandBody` - source (int16), destination (int16)
- `UnequipCommandBody` - source (int16), destination (int16)
- `MoveCommandBody` - source (int16), destination (int16)
- `DropCommandBody` - worldId, channelId, mapId, instance (UUID), source (int16), quantity (int16), x (int16), y (int16)
- `RequestReserveCommandBody` - transactionId (UUID), items ([]ItemBody{source, itemId, quantity})
- `ConsumeCommandBody` - transactionId (UUID), slot (int16)
- `DestroyCommandBody` - slot (int16), quantity (uint32), removeAll (bool)
- `CancelReservationCommandBody` - transactionId (UUID), slot (int16)
- `IncreaseCapacityCommandBody` - amount (uint32)
- `CreateAssetCommandBody` - templateId (uint32), quantity (uint32), expiration (time), ownerId (uint32), flag (uint16), rechargeable (uint64), useAverageStats (bool, optional)
- `RechargeCommandBody` - slot (int16), quantity (uint32)
- `MergeCommandBody` - empty
- `SortCommandBody` - empty
- `AcceptCommandBody` - transactionId (UUID), templateId (uint32), embeds AssetData
- `ReleaseCommandBody` - transactionId (UUID), assetId (uint32), quantity (uint32)
- `ExpireCommandBody` - assetId (uint32), templateId (uint32), slot (int16), replaceItemId (uint32), replaceMessage (string)
- `ModifyEquipmentCommandBody` - assetId (uint32), all equipment stat fields, flag (uint16), expiration (time)
- `ChangeTemplateCommandBody` - petId (uint32), newTemplateId (uint32)

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

Body types:
- `CreatedStatusEventBody` - type (byte), capacity (uint32)
- `DeletedStatusEventBody` - empty
- `CapacityChangedEventBody` - type (byte), capacity (uint32)
- `ReservedEventBody` - transactionId (UUID), itemId (uint32), slot (int16), quantity (uint32)
- `ReservationCancelledEventBody` - itemId (uint32), slot (int16), quantity (uint32)
- `MergeCompleteEventBody` - type (byte)
- `SortCompleteEventBody` - type (byte)
- `AcceptedEventBody` - transactionId (UUID)
- `ReleasedEventBody` - transactionId (UUID)
- `ErrorEventBody` - errorCode (string), transactionId (UUID)
- `CreationFailedStatusEventBody` - errorCode (string), message (string)

### Inventory Status Event

```
StatusEvent[Body] {
  transactionId: UUID
  characterId: uint32
  type: string
  body: Body
}
```

Body types:
- `CreatedStatusEventBody` - empty
- `CreationFailedStatusEventBody` - reason (string)
- `DeletedStatusEventBody` - empty

### Character Status Event

```
StatusEvent[Body] {
  transactionId: UUID
  worldId: byte
  characterId: uint32
  type: string
  body: Body
}
```

Body types:
- `CreatedStatusEventBody` - name (string)
- `DeletedStatusEventBody` - empty

### Pickup Command

```
Command {
  tenantId: UUID
  characterId: uint32
  itemId: uint32
  transactionId: UUID
  type: string
}
```

Type: `ITEM_CONSUMED_ON_PICKUP`. Unlike other command/event messages in this service, this struct is flat and does not wrap a separate body type.

### Drop Command

```
Command[Body] {
  worldId: byte
  channelId: byte
  mapId: uint32
  instance: UUID
  type: string
  body: Body
}
```

Body types:
- `SpawnFromCharacterCommandBody` - itemId (uint32), quantity (uint32), mesos (uint32), dropType (byte), x (int16), y (int16), ownerId (uint32), dropperId (uint32), dropperX (int16), dropperY (int16), playerDrop (bool), embeds EquipmentData
- `CancelReservationCommandBody` - dropId (uint32), characterId (uint32)
- `RequestPickUpCommandBody` - dropId (uint32), characterId (uint32)

### Drop Status Event

```
StatusEvent[Body] {
  worldId: byte
  channelId: byte
  mapId: uint32
  instance: UUID
  dropId: uint32
  type: string
  body: Body
}
```

Body types:
- `ReservedStatusEventBody` - characterId (uint32), itemId (uint32), quantity (uint32), meso (uint32), embeds EquipmentData

---

## Transaction Semantics

- Commands include `transactionId` for correlation
- Events include `transactionId` matching originating command
- Reservations have 30-second timeout
- All database mutations within a single command are wrapped in a transaction
- Kafka messages produced during a transaction are buffered and, via the transactional outbox provider, persisted as outbox rows in the same database transaction; a background drainer publishes them to Kafka after the transaction commits
- Compensating messages emitted after a transaction has rolled back (for example drop cancel-reservation on a failed pickup) are emitted directly to the producer on a separate, throwaway buffer, outside the outbox
