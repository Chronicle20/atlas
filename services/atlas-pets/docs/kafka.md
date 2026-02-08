# Kafka Integration

## Topics Consumed

| Environment Variable | Direction | Description |
|---------------------|-----------|-------------|
| EVENT_TOPIC_CHARACTER_STATUS | Event | Character status events |
| EVENT_TOPIC_ASSET_STATUS | Event | Unified asset status events |
| COMMAND_TOPIC_PET | Command | Pet commands |
| COMMAND_TOPIC_PET_MOVEMENT | Command | Pet movement commands |

## Topics Produced

| Environment Variable | Direction | Description |
|---------------------|-----------|-------------|
| EVENT_TOPIC_PET_STATUS | Event | Pet status events |

## Message Types

### Character Status Event (Consumed)

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "characterId": 12345,
  "type": "LOGIN|LOGOUT|DELETED|MAP_CHANGED|CHANNEL_CHANGED",
  "body": {}
}
```

#### Event Types

| Type | Body Fields | Description |
|------|-------------|-------------|
| DELETED | none | Character was deleted; all pets for the character are deleted |
| LOGIN | channelId, mapId, instance | Character logged in; registered in character registry, pet positions cleared |
| LOGOUT | channelId, mapId, instance | Character logged out; removed from character registry, pet positions cleared |
| MAP_CHANGED | channelId, oldMapId, oldInstance, targetMapId, targetInstance, targetPortalId | Character changed maps; registry updated, pet positions cleared |
| CHANNEL_CHANGED | channelId, oldChannelId, mapId, instance | Character changed channels; registry updated, pet positions cleared |

### Asset Status Event (Consumed)

```json
{
  "characterId": 12345,
  "compartmentId": "uuid",
  "assetId": 1,
  "templateId": 5000017,
  "slot": 15,
  "type": "DELETED",
  "body": {}
}
```

The service only processes DELETED events for assets where:
- The item's inventory type is Cash (derived from templateId)
- The item's classification is Pet (derived from templateId)

When matched, the service calls `DeleteOnRemove` using `characterId`, `templateId`, and `slot` from the event.

### Pet Command (Consumed)

```json
{
  "transactionId": "uuid",
  "actorId": 12345,
  "petId": 1,
  "type": "SPAWN|DESPAWN|ATTEMPT_COMMAND|AWARD_CLOSENESS|AWARD_FULLNESS|AWARD_LEVEL|EXCLUDE",
  "body": {}
}
```

#### Command Types

| Type | Body Fields | Description |
|------|-------------|-------------|
| SPAWN | lead (bool) | Spawn a pet; if lead=true, takes slot 0 and shifts others |
| DESPAWN | none | Despawn a pet with reason "NORMAL" |
| ATTEMPT_COMMAND | commandId (byte), byName (bool) | Execute a pet command/trick |
| AWARD_CLOSENESS | amount (uint16) | Award closeness to a pet; transactionId from command envelope is forwarded |
| AWARD_FULLNESS | amount (byte) | Award fullness to a pet |
| AWARD_LEVEL | amount (byte) | Award levels to a pet |
| EXCLUDE | items ([]uint32) | Set excluded items for auto-loot |

### Pet Movement Command (Consumed)

```json
{
  "worldId": 0,
  "channelId": 1,
  "mapId": 100000000,
  "instance": "uuid",
  "objectId": 1,
  "observerId": 12345,
  "x": 100,
  "y": 200,
  "stance": 2
}
```

### Pet Status Event (Produced)

```json
{
  "petId": 1,
  "ownerId": 12345,
  "type": "CREATED|DELETED|SPAWNED|DESPAWNED|COMMAND_RESPONSE|CLOSENESS_CHANGED|FULLNESS_CHANGED|LEVEL_CHANGED|SLOT_CHANGED|EXCLUDE_CHANGED",
  "body": {}
}
```

#### Event Types

| Type | Body Fields | Description |
|------|-------------|-------------|
| CREATED | none | Pet was created |
| DELETED | none | Pet was deleted |
| SPAWNED | templateId, name, slot, level, closeness, fullness, x, y, stance, fh | Pet was spawned to a slot |
| DESPAWNED | templateId, name, slot, level, closeness, fullness, oldSlot, reason | Pet was despawned |
| COMMAND_RESPONSE | slot, closeness, fullness, commandId, success | Response to a command attempt |
| CLOSENESS_CHANGED | slot, closeness, amount, transactionId | Closeness was modified |
| FULLNESS_CHANGED | slot, fullness, amount | Fullness was modified |
| LEVEL_CHANGED | slot, level, amount | Level was modified |
| SLOT_CHANGED | oldSlot, newSlot | Slot was modified (due to spawn/despawn shifting) |
| EXCLUDE_CHANGED | items | Excluded items were replaced |

#### Despawn Reasons

| Reason | Description |
|--------|-------------|
| NORMAL | Normal despawn by user command |
| HUNGER | Despawned due to low fullness (<= 5) |
| EXPIRED | Despawned due to expiration |

## Transaction Semantics

- Pet commands include a `transactionId` field for correlation
- `AWARD_CLOSENESS` commands forward the `transactionId` to the `CLOSENESS_CHANGED` event
- All state-mutating operations are wrapped in database transactions; Kafka messages are buffered and emitted only after the transaction commits successfully

## Required Headers

- Span header (tracing)
- Tenant header (multi-tenancy)
