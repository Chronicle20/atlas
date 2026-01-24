# Fame Kafka Integration

## Topics Consumed

| Topic | Environment Variable | Consumer Group |
|-------|---------------------|----------------|
| Fame command topic | COMMAND_TOPIC_FAME | Fame Service |

## Topics Produced

| Topic | Environment Variable |
|-------|---------------------|
| Fame status event topic | EVENT_TOPIC_FAME_STATUS |
| Character command topic | COMMAND_TOPIC_CHARACTER |

## Message Types

### Command: Request Change

Direction: Consumed

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

### Event: Fame Status Error

Direction: Produced

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

Error values:
- `NOT_TODAY`
- `NOT_THIS_MONTH`
- `INVALID_NAME`
- `NOT_MINIMUM_LEVEL`
- `UNEXPECTED`

### Command: Request Change Fame (Character)

Direction: Produced

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

## Transaction Semantics

- Commands are processed with tenant header parsing
- Messages are keyed by characterId for partition ordering
