# Kafka Integration

## Topics Consumed

| Topic Environment Variable | Direction | Description |
|---------------------------|-----------|-------------|
| COMMAND_TOPIC_DROP | Command | Drop commands |

## Topics Produced

| Topic Environment Variable | Direction | Description |
|---------------------------|-----------|-------------|
| EVENT_TOPIC_DROP_STATUS | Event | Drop status events |

## Message Types

### Commands (Consumed)

#### SPAWN

Creates a new drop from a monster or other non-character source. Equipment stats are provided inline.

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "type": "SPAWN",
  "body": {
    "itemId": 0,
    "quantity": 0,
    "mesos": 0,
    "dropType": 0,
    "x": 0,
    "y": 0,
    "ownerId": 0,
    "ownerPartyId": 0,
    "dropperId": 0,
    "dropperX": 0,
    "dropperY": 0,
    "playerDrop": false,
    "mod": 0,
    "strength": 0,
    "dexterity": 0,
    "intelligence": 0,
    "luck": 0,
    "hp": 0,
    "mp": 0,
    "weaponAttack": 0,
    "magicAttack": 0,
    "weaponDefense": 0,
    "magicDefense": 0,
    "accuracy": 0,
    "avoidability": 0,
    "hands": 0,
    "speed": 0,
    "jump": 0,
    "slots": 0
  }
}
```

#### SPAWN_FROM_CHARACTER

Creates a new drop from a character action. Equipment stats are provided inline.

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "type": "SPAWN_FROM_CHARACTER",
  "body": {
    "itemId": 0,
    "quantity": 0,
    "mesos": 0,
    "dropType": 0,
    "x": 0,
    "y": 0,
    "ownerId": 0,
    "ownerPartyId": 0,
    "dropperId": 0,
    "dropperX": 0,
    "dropperY": 0,
    "playerDrop": false,
    "mod": 0,
    "strength": 0,
    "dexterity": 0,
    "intelligence": 0,
    "luck": 0,
    "hp": 0,
    "mp": 0,
    "weaponAttack": 0,
    "magicAttack": 0,
    "weaponDefense": 0,
    "magicDefense": 0,
    "accuracy": 0,
    "avoidability": 0,
    "hands": 0,
    "speed": 0,
    "jump": 0,
    "slots": 0
  }
}
```

#### REQUEST_RESERVATION

Requests to reserve a drop for a character.

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "type": "REQUEST_RESERVATION",
  "body": {
    "dropId": 0,
    "characterId": 0,
    "partyId": 0,
    "characterX": 0,
    "characterY": 0,
    "petSlot": -1
  }
}
```

#### CANCEL_RESERVATION

Cancels a drop reservation.

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "type": "CANCEL_RESERVATION",
  "body": {
    "dropId": 0,
    "characterId": 0
  }
}
```

#### REQUEST_PICK_UP

Requests to pick up a drop.

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "type": "REQUEST_PICK_UP",
  "body": {
    "dropId": 0,
    "characterId": 0
  }
}
```

#### CONSUME

Consumes a drop via a game mechanic (e.g., item-reactor trigger), removing it from the map.

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "type": "CONSUME",
  "body": {
    "dropId": 0
  }
}
```

### Events (Produced)

#### CREATED

Emitted when a drop is created.

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "dropId": 0,
  "type": "CREATED",
  "body": {
    "itemId": 0,
    "quantity": 0,
    "meso": 0,
    "type": 0,
    "x": 0,
    "y": 0,
    "ownerId": 0,
    "ownerPartyId": 0,
    "dropTime": "2024-01-01T00:00:00Z",
    "dropperUniqueId": 0,
    "dropperX": 0,
    "dropperY": 0,
    "playerDrop": false
  }
}
```

#### EXPIRED

Emitted when a drop expires.

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "dropId": 0,
  "type": "EXPIRED",
  "body": {}
}
```

#### PICKED_UP

Emitted when a drop is picked up. Equipment stats are provided inline via EquipmentData.

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "dropId": 0,
  "type": "PICKED_UP",
  "body": {
    "characterId": 0,
    "itemId": 0,
    "quantity": 0,
    "meso": 0,
    "petSlot": -1,
    "strength": 0,
    "dexterity": 0,
    "intelligence": 0,
    "luck": 0,
    "hp": 0,
    "mp": 0,
    "weaponAttack": 0,
    "magicAttack": 0,
    "weaponDefense": 0,
    "magicDefense": 0,
    "accuracy": 0,
    "avoidability": 0,
    "hands": 0,
    "speed": 0,
    "jump": 0,
    "slots": 0
  }
}
```

#### RESERVED

Emitted when a drop is reserved. Equipment stats are provided inline via EquipmentData.

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "dropId": 0,
  "type": "RESERVED",
  "body": {
    "characterId": 0,
    "itemId": 0,
    "quantity": 0,
    "meso": 0,
    "strength": 0,
    "dexterity": 0,
    "intelligence": 0,
    "luck": 0,
    "hp": 0,
    "mp": 0,
    "weaponAttack": 0,
    "magicAttack": 0,
    "weaponDefense": 0,
    "magicDefense": 0,
    "accuracy": 0,
    "avoidability": 0,
    "hands": 0,
    "speed": 0,
    "jump": 0,
    "slots": 0
  }
}
```

#### RESERVATION_FAILURE

Emitted when a drop reservation fails.

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "dropId": 0,
  "type": "RESERVATION_FAILURE",
  "body": {
    "characterId": 0
  }
}
```

#### CONSUMED

Emitted when a drop is consumed by a game mechanic.

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "dropId": 0,
  "type": "CONSUMED",
  "body": {}
}
```

## Transaction Semantics

- All commands and events include a `transactionId` for correlation
- All events carry the same `transactionId` from the originating command
- Messages are keyed by `dropId` for ordering guarantees within a drop's lifecycle
- Tenant and span headers are attached to all produced messages for multi-tenancy and tracing support
- Commands are consumed from a single consumer group (`drop_command`) on the `COMMAND_TOPIC_DROP` topic
- All Kafka message production uses a buffered emit pattern: messages are collected during processing and flushed atomically after the operation completes
