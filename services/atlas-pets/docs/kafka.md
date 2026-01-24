# Kafka Integration

## Topics Consumed

| Environment Variable | Direction | Description |
|---------------------|-----------|-------------|
| EVENT_TOPIC_CHARACTER_STATUS | Event | Character status events |
| EVENT_TOPIC_ASSET_STATUS | Event | Asset status events |
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
  "worldId": 0,
  "characterId": 12345,
  "type": "LOGIN|LOGOUT|DELETED|MAP_CHANGED|CHANNEL_CHANGED",
  "body": {}
}
```

#### Event Types

| Type | Body Fields | Description |
|------|-------------|-------------|
| DELETED | none | Character was deleted |
| LOGIN | channelId, mapId | Character logged in |
| LOGOUT | channelId, mapId | Character logged out |
| MAP_CHANGED | channelId, oldMapId, targetMapId, targetPortalId | Character changed maps |
| CHANNEL_CHANGED | channelId, oldChannelId, mapId | Character changed channels |

### Asset Status Event (Consumed)

```json
{
  "characterId": 12345,
  "compartmentId": "uuid",
  "assetId": 1,
  "templateId": 5000,
  "slot": 1,
  "type": "DELETED",
  "body": {}
}
```

Processed when a pet item is deleted from the cash inventory.

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
| SPAWN | lead | Spawn a pet |
| DESPAWN | none | Despawn a pet |
| ATTEMPT_COMMAND | commandId, byName | Execute a pet command |
| AWARD_CLOSENESS | amount | Award closeness to a pet |
| AWARD_FULLNESS | amount | Award fullness to a pet |
| AWARD_LEVEL | amount | Award levels to a pet |
| EXCLUDE | items | Set excluded items for auto-loot |

### Pet Movement Command (Consumed)

```json
{
  "worldId": 0,
  "channelId": 1,
  "mapId": 100000000,
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
| SPAWNED | templateId, name, slot, level, closeness, fullness, x, y, stance, fh | Pet was spawned |
| DESPAWNED | templateId, name, slot, level, closeness, fullness, oldSlot, reason | Pet was despawned |
| COMMAND_RESPONSE | slot, closeness, fullness, commandId, success | Response to a command attempt |
| CLOSENESS_CHANGED | slot, closeness, amount, transactionId | Closeness was modified |
| FULLNESS_CHANGED | slot, fullness, amount | Fullness was modified |
| LEVEL_CHANGED | slot, level, amount | Level was modified |
| SLOT_CHANGED | oldSlot, newSlot | Slot was modified |
| EXCLUDE_CHANGED | items | Excluded items were modified |

#### Despawn Reasons

| Reason | Description |
|--------|-------------|
| NORMAL | Normal despawn by user |
| HUNGER | Despawned due to low fullness |
| EXPIRED | Despawned due to expiration |

## Transaction Semantics

- Pet commands include a transactionId for correlation
- CLOSENESS_CHANGED events include transactionId when awarded via command

## Required Headers

- Span header (tracing)
- Tenant header (multi-tenancy)
