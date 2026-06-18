# Door Kafka Integration

All consumers use the `Door Registry Service` consumer group and parse the span and tenant headers.

## Topics Consumed

### COMMAND_TOPIC_DOOR

Door commands.

**Consumer Group:** Door Registry Service

**Message Types:**

#### SPAWN

Spawns a Mystic Door for the owner in the command's field.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "ownerCharacterId": 0,
  "type": "SPAWN",
  "body": {
    "skillId": 0,
    "skillLevel": 0,
    "x": 0,
    "y": 0
  }
}
```

#### REMOVE

Removes the owner's door. An empty `reason` defaults to `RECAST`.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "ownerCharacterId": 0,
  "type": "REMOVE",
  "body": {
    "reason": ""
  }
}
```

### EVENT_TOPIC_CHARACTER_STATUS

Character status events for door cleanup. Only the `LOGOUT`, `CHANNEL_CHANGED`, and `MAP_CHANGED` subset is consumed.

**Consumer Group:** Door Registry Service

**Message Types:**

#### LOGOUT

Removes the owner's doors with reason `LOGOUT`.

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "characterId": 0,
  "type": "LOGOUT",
  "body": {
    "channelId": 0,
    "mapId": 0,
    "instance": "uuid"
  }
}
```

#### CHANNEL_CHANGED

Removes the owner's doors with reason `CHANNEL_CHANGED`.

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "characterId": 0,
  "type": "CHANNEL_CHANGED",
  "body": {
    "channelId": 0,
    "oldChannelId": 0,
    "mapId": 0,
    "instance": "uuid"
  }
}
```

#### MAP_CHANGED

Removes the owner's door only when the target field is neither the door's source field nor its town map (reason `LEFT_FIELD`).

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "characterId": 0,
  "type": "MAP_CHANGED",
  "body": {
    "channelId": 0,
    "oldMapId": 0,
    "oldInstance": "uuid",
    "targetMapId": 0,
    "targetInstance": "uuid",
    "targetPortalId": 0
  }
}
```

### EVENT_TOPIC_PARTY_STATUS

Party status events that trigger town-portal reslotting of affected members' doors. The `JOINED`, `LEFT`, `EXPEL`, `DISBAND`, and `CHANGE_LEADER` subset is consumed.

**Consumer Group:** Door Registry Service

**Message Types:**

#### JOINED

Reslots all current party members' doors to their party slot.

```json
{
  "actorId": 0,
  "worldId": 0,
  "partyId": 0,
  "type": "JOINED",
  "body": {}
}
```

#### LEFT

Reslots remaining members' doors and reslots the actor (leaver) to solo.

```json
{
  "actorId": 0,
  "worldId": 0,
  "partyId": 0,
  "type": "LEFT",
  "body": {}
}
```

#### EXPEL

Reslots remaining members' doors and reslots the expelled character to solo.

```json
{
  "actorId": 0,
  "worldId": 0,
  "partyId": 0,
  "type": "EXPEL",
  "body": {
    "characterId": 0
  }
}
```

#### DISBAND

Reslots all listed members' doors to solo (party gone).

```json
{
  "actorId": 0,
  "worldId": 0,
  "partyId": 0,
  "type": "DISBAND",
  "body": {
    "members": [0]
  }
}
```

#### CHANGE_LEADER

Reslots all current party members' doors to their party slot.

```json
{
  "actorId": 0,
  "worldId": 0,
  "partyId": 0,
  "type": "CHANGE_LEADER",
  "body": {
    "characterId": 0,
    "disconnected": false
  }
}
```

## Topics Produced

### EVENT_TOPIC_DOOR_STATUS

Door status events. Each message key is the area field map id. `forCharacterId` is 0 for a broadcast to the door's eligible set (owner plus current party members) or non-zero to target a single character.

**Message Types:**

#### CREATED

Emitted when a door is spawned.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "pairId": 0,
  "ownerCharacterId": 0,
  "partyId": 0,
  "forCharacterId": 0,
  "type": "CREATED",
  "body": {
    "areaDoorId": 0,
    "townDoorId": 0,
    "townMapId": 0,
    "slot": 0,
    "townPortalId": 0,
    "areaX": 0,
    "areaY": 0,
    "townX": 0,
    "townY": 0,
    "skillId": 0,
    "skillLevel": 0,
    "expiresAt": 0
  }
}
```

#### REMOVED

Emitted when a door is removed. `reason` is one of `RECAST`, `EXPIRY`, `LOGOUT`, `CHANNEL_CHANGED`, `LEFT_FIELD`, `PARTY_LEFT`.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "pairId": 0,
  "ownerCharacterId": 0,
  "partyId": 0,
  "forCharacterId": 0,
  "type": "REMOVED",
  "body": {
    "areaDoorId": 0,
    "townDoorId": 0,
    "townMapId": 0,
    "slot": 0,
    "reason": "RECAST"
  }
}
```

#### SLOT_CHANGED

Emitted when a door's town slot changes (party reslot).

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "pairId": 0,
  "ownerCharacterId": 0,
  "partyId": 0,
  "forCharacterId": 0,
  "type": "SLOT_CHANGED",
  "body": {
    "areaDoorId": 0,
    "townDoorId": 0,
    "townMapId": 0,
    "oldSlot": 0,
    "newSlot": 0,
    "townPortalId": 0,
    "townX": 0,
    "townY": 0,
    "areaX": 0,
    "areaY": 0
  }
}
```

## Transaction Semantics

- Each produced status event is keyed by the area field map id.
- Consumed door commands and character/party status events carry span and tenant headers (`SpanHeaderParser`, `TenantHeaderParser`).
- All status-event handlers and command handlers are registered with persistent configuration.
