# Kafka

## Topics Consumed

### COMMAND_TOPIC_FAME

Fame command topic for requesting fame changes.

| Message Type | Direction | Description |
|--------------|-----------|-------------|
| REQUEST_CHANGE | Command | Request to change fame between characters |

#### Command Structure

```json
{
  "transactionId": "uuid",
  "worldId": "byte",
  "characterId": "uint32",
  "type": "REQUEST_CHANGE",
  "body": {
    "channelId": "byte",
    "mapId": "uint32",
    "targetId": "uint32",
    "amount": "int8"
  }
}
```

### EVENT_TOPIC_CHARACTER_STATUS

Character status event topic for character lifecycle events.

| Message Type | Direction | Description |
|--------------|-----------|-------------|
| DELETED | Event | Character was deleted |

#### Event Structure

```json
{
  "worldId": "byte",
  "characterId": "uint32",
  "type": "DELETED",
  "body": {}
}
```

## Topics Produced

### EVENT_TOPIC_FAME_STATUS

Fame status event topic for fame operation results.

| Message Type | Direction | Description |
|--------------|-----------|-------------|
| ERROR | Event | Fame change request failed |

#### Event Structure

```json
{
  "transactionId": "uuid",
  "worldId": "world.Id",
  "characterId": "uint32",
  "type": "ERROR",
  "body": {
    "channelId": "channel.Id",
    "error": "string"
  }
}
```

#### Error Types

| Error | Description |
|-------|-------------|
| NOT_TODAY | Character already gave fame today |
| NOT_THIS_MONTH | Character already gave fame to this target this month |
| INVALID_NAME | Target character does not exist |
| NOT_MINIMUM_LEVEL | Character is below level 15 |
| UNEXPECTED | Unexpected error occurred |

### COMMAND_TOPIC_CHARACTER

Character command topic for requesting character operations.

| Message Type | Direction | Description |
|--------------|-----------|-------------|
| REQUEST_CHANGE_FAME | Command | Request to change a character's fame value |

#### Command Structure

```json
{
  "transactionId": "uuid",
  "worldId": "world.Id",
  "characterId": "uint32",
  "type": "REQUEST_CHANGE_FAME",
  "body": {
    "actorId": "uint32",
    "actorType": "CHARACTER",
    "amount": "int8"
  }
}
```

## Message Types

### Fame Messages

| Struct | Purpose |
|--------|---------|
| Command[E] | Generic fame command envelope |
| RequestChangeCommandBody | Body for REQUEST_CHANGE command |
| StatusEvent[E] | Generic fame status event envelope |
| StatusEventErrorBody | Body for ERROR status event |

### Character Messages

| Struct | Purpose |
|--------|---------|
| CommandEvent[E] | Generic character command envelope |
| RequestChangeFameBody | Body for REQUEST_CHANGE_FAME command |
| StatusEvent[E] | Generic character status event envelope |
| StatusEventDeletedBody | Body for DELETED status event |

## Transaction Semantics

- Fame change requests are processed within a database transaction
- On success, a REQUEST_CHANGE_FAME command is emitted to the character service
- On failure, an ERROR event is emitted to the fame status topic
- Message partitioning uses characterId as the key
