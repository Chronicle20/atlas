# Dragon Kafka Integration

All consumers use the `Dragon Registry Service` consumer group and parse the span and tenant headers.

## Topics Consumed

### COMMAND_TOPIC_DRAGON

Dragon commands.

**Consumer Group:** Dragon Registry Service

**Message Types:**

#### CREATE

Creates the dragon for the given character in the command's field, subject to the job gate.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "type": "CREATE",
  "body": {
    "characterId": 0
  }
}
```

#### DESTROY

Destroys the given character's dragon.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "type": "DESTROY",
  "body": {
    "characterId": 0
  }
}
```

#### MOVE

Updates the given character's dragon position and relays the raw movement blob.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "type": "MOVE",
  "body": {
    "characterId": 0,
    "startX": 0,
    "startY": 0,
    "stance": 0,
    "rawMovement": "base64"
  }
}
```

### EVENT_TOPIC_CHARACTER_STATUS

Character status events for dragon lifecycle. The `LOGIN`, `LOGOUT`, `MAP_CHANGED`, `CHANNEL_CHANGED`, and `JOB_CHANGED` subset is consumed. `LOGIN` and `LOGOUT` are produced by atlas-character; `MAP_CHANGED` and `CHANNEL_CHANGED` are produced by atlas-maps; `JOB_CHANGED` is produced by atlas-character.

**Consumer Group:** Dragon Registry Service

**Message Types:**

#### LOGIN

Creates the character's dragon in the field logged into.

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "characterId": 0,
  "type": "LOGIN",
  "body": {
    "channelId": 0,
    "mapId": 0,
    "instance": "uuid"
  }
}
```

#### LOGOUT

Destroys the character's dragon.

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

#### MAP_CHANGED

Destroys the character's dragon in the old field and creates it in the new field.

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

#### CHANNEL_CHANGED

Destroys the character's dragon in the old field and creates it in the new field.

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

#### JOB_CHANGED

Destroys the character's dragon if the new job id no longer resolves to a dragon-bearing Evan stage.

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "characterId": 0,
  "type": "JOB_CHANGED",
  "body": {
    "channelId": 0,
    "jobId": 0
  }
}
```

## Topics Produced

### EVENT_TOPIC_DRAGON_STATUS

Dragon status events. Each message key is the field map id.

**Message Types:**

#### CREATED

Emitted when a dragon is created (absent-to-present transition only).

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "ownerCharacterId": 0,
  "type": "CREATED",
  "body": {
    "x": 0,
    "y": 0,
    "stance": 0,
    "jobId": 0
  }
}
```

#### MOVED

Emitted when a dragon's position is updated.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "ownerCharacterId": 0,
  "type": "MOVED",
  "body": {
    "rawMovement": "base64"
  }
}
```

#### DESTROYED

Emitted when a dragon is removed.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "ownerCharacterId": 0,
  "type": "DESTROYED",
  "body": {}
}
```

## Transaction Semantics

- Each produced status event is keyed by the field map id.
- Consumed dragon commands and character status events carry span and tenant headers (`SpanHeaderParser`, `TenantHeaderParser`).
- All command handlers and status-event handlers are registered with persistent configuration.
