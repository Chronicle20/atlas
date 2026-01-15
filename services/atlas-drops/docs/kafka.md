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

Creates a new drop from a monster or other source.

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
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
    "mod": 0
  }
}
```

#### SPAWN_FROM_CHARACTER

Creates a new drop from a character.

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "type": "SPAWN_FROM_CHARACTER",
  "body": {
    "itemId": 0,
    "equipmentId": 0,
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
    "mod": 0
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
  "type": "REQUEST_RESERVATION",
  "body": {
    "dropId": 0,
    "characterId": 0,
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
  "type": "REQUEST_PICK_UP",
  "body": {
    "dropId": 0,
    "characterId": 0
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
  "dropId": 0,
  "type": "EXPIRED",
  "body": {}
}
```

#### PICKED_UP

Emitted when a drop is picked up.

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "dropId": 0,
  "type": "PICKED_UP",
  "body": {
    "characterId": 0,
    "itemId": 0,
    "equipmentId": 0,
    "quantity": 0,
    "meso": 0,
    "petSlot": -1
  }
}
```

#### RESERVED

Emitted when a drop is reserved.

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "dropId": 0,
  "type": "RESERVED",
  "body": {
    "characterId": 0,
    "itemId": 0,
    "equipmentId": 0,
    "quantity": 0,
    "meso": 0
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
  "dropId": 0,
  "type": "RESERVATION_FAILURE",
  "body": {
    "characterId": 0
  }
}
```

## Transaction Semantics

- All commands include a `transactionId` for correlation
- All events include the same `transactionId` from the originating command
- Messages are keyed by `dropId` for ordering guarantees within a drop's lifecycle
- Tenant headers are required for multi-tenancy support
