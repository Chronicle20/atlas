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
| COMMAND_TOPIC_COMPARTMENT | Command | Compartment commands (item template change) |

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
  "type": "SPAWN|DESPAWN|ATTEMPT_COMMAND|AWARD_CLOSENESS|AWARD_FULLNESS|AWARD_LEVEL|EXCLUDE|EVOLVE|SET_SKILL|REVIVE|RENAME",
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
| EVOLVE | none | Evolve a pet to a new template; transactionId from command envelope is forwarded |
| SET_SKILL | skill (string), enabled (bool) | Set or clear a pet skill flag bit identified by a semantic skill key; unknown keys are dropped |
| REVIVE | sourceTemplateId (uint32) | Restore a dried-up pet's lifespan from the consumed source item's life value; transactionId from command envelope is forwarded |
| RENAME | name (string) | Rename a pet; transactionId from command envelope is forwarded |

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
  "type": "CREATED|DELETED|SPAWNED|DESPAWNED|COMMAND_RESPONSE|CLOSENESS_CHANGED|FULLNESS_CHANGED|LEVEL_CHANGED|SLOT_CHANGED|EXCLUDE_CHANGED|EVOLVED|FLAG_CHANGED|REVIVED|REVIVE_FAILED|NAME_CHANGED",
  "body": {}
}
```

#### Event Types

| Type | Body Fields | Description |
|------|-------------|-------------|
| CREATED | none | Pet was created |
| DELETED | none | Pet was deleted |
| SPAWNED | templateId, name, slot, level, closeness, fullness, x, y, stance, fh, cashId | Pet was spawned to a slot |
| DESPAWNED | templateId, name, slot, level, closeness, fullness, oldSlot, reason | Pet was despawned |
| COMMAND_RESPONSE | slot, closeness, fullness, commandId, success | Response to a command attempt |
| CLOSENESS_CHANGED | slot, closeness, amount, transactionId | Closeness was modified |
| FULLNESS_CHANGED | slot, fullness, amount | Fullness was modified |
| LEVEL_CHANGED | slot, level, amount | Level was modified |
| SLOT_CHANGED | oldSlot, newSlot | Slot was modified (due to spawn/despawn shifting) |
| EXCLUDE_CHANGED | items | Excluded items were replaced |
| EVOLVED | slot, oldTemplateId, newTemplateId, transactionId | Pet template was changed via evolution |
| FLAG_CHANGED | slot, flag | A pet skill flag bit was set or cleared |
| REVIVED | slot, expiration, transactionId | Pet's expiration was reset following a successful revive |
| REVIVE_FAILED | reason, transactionId | Revive was rejected (pet not found, not owned, not dried up, or source item data unavailable) |
| NAME_CHANGED | slot, name, previousName, transactionId | Pet was renamed |

### Change Template Command (Produced)

```json
{
  "transactionId": "uuid",
  "characterId": 12345,
  "inventoryType": 6,
  "type": "CHANGE_TEMPLATE",
  "body": {
    "petId": 1,
    "newTemplateId": 5000018
  }
}
```

Produced to the compartment command topic when a pet's template changes (evolution or egg hatching), so the corresponding cash inventory asset's template is updated to match.

### Reset Pet Expiration Command (Produced)

```json
{
  "transactionId": "uuid",
  "characterId": 12345,
  "inventoryType": 6,
  "type": "RESET_PET_EXPIRATION",
  "body": {
    "petId": 1,
    "expiration": "2026-11-26T00:00:00Z",
    "sourceTemplateId": 2022002
  }
}
```

Produced to the compartment command topic when a pet is revived, so the corresponding cash inventory asset's expiration is reset to match. `expiration` is an absolute instant, resolved by (`characterId`, `petId`).

#### Despawn Reasons

| Reason | Description |
|--------|-------------|
| NORMAL | Normal despawn by user command |
| HUNGER | Despawned due to low fullness (<= 5) |
| EXPIRED | Despawned due to expiration |

## Transaction Semantics

- Pet commands include a `transactionId` field for correlation
- `AWARD_CLOSENESS` commands forward the `transactionId` to the `CLOSENESS_CHANGED` event
- `EVOLVE` commands forward the `transactionId` to the `EVOLVED` event and to the produced `CHANGE_TEMPLATE` command
- `REVIVE` commands forward the `transactionId` to the `REVIVED`/`REVIVE_FAILED` event and to the produced `RESET_PET_EXPIRATION` command
- `RENAME` commands forward the `transactionId` to the `NAME_CHANGED` event
- All state-mutating operations are wrapped in database transactions; outgoing messages are written to a transactional outbox (github.com/Chronicle20/atlas/libs/atlas-outbox) within the same database transaction as the state change, and are published to Kafka asynchronously by a background drainer once the transaction commits
- A redelivered `REVIVE` command for a `transactionId` already recorded as the pet's last revive re-emits `REVIVED` and re-issues the `RESET_PET_EXPIRATION` cascade without writing to the pet row again
- `RENAME` re-emits `NAME_CHANGED` on every delivery, including redeliveries

## Required Headers

- Span header (tracing)
- Tenant header (multi-tenancy)
