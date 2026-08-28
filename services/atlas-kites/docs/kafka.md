# Kite Kafka Integration

All consumers use the `Kite Service` consumer group and parse the span, tenant, and env headers.

## Topics Consumed

### COMMAND_TOPIC_KITE

Kite commands, keyed on `characterId` by the producer (atlas-channel), so one character's commands are totally ordered within a partition.

**Consumer Group:** Kite Service

**Message Types:**

#### CREATE

Places a kite for the character in the command's field.

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "characterId": 0,
  "type": "CREATE",
  "body": {
    "name": "",
    "templateId": 0,
    "message": "",
    "x": 0,
    "y": 0
  }
}
```

#### DESTROY

Removes the character's kite by kite id.

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "characterId": 0,
  "type": "DESTROY",
  "body": {
    "kiteId": 0
  }
}
```

### EVENT_TOPIC_CHARACTER_STATUS

Character status events for field-presence tracking and kite cleanup. The `LOGIN`, `LOGOUT`, `MAP_CHANGED`, and `CHANNEL_CHANGED` subset is consumed.

**Consumer Group:** Kite Service

**Message Types:**

#### LOGIN

Adds the character to the field's character-presence index.

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

Removes the character from the field's character-presence index and destroys the character's kite with reason `OWNER_LOGGED_OUT`.

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

Transitions the character between fields in the character-presence index and destroys the character's kite with reason `OWNER_LEFT`.

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

Transitions the character between fields (old channel to new channel) in the character-presence index and destroys the character's kite with reason `OWNER_LEFT`.

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

## Topics Produced

### EVENT_TOPIC_KITE_STATUS

Kite status events. Each message key is the field's map id.

**Message Types:**

#### CREATED

Emitted when a kite is placed.

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "characterId": 0,
  "type": "CREATED",
  "body": {
    "kiteId": 0,
    "name": "",
    "templateId": 0,
    "message": "",
    "x": 0,
    "y": 0
  }
}
```

#### DESTROYED

Emitted when a kite is removed. `reason` is one of `OWNER_LEFT`, `OWNER_LOGGED_OUT`.

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "characterId": 0,
  "type": "DESTROYED",
  "body": {
    "kiteId": 0,
    "reason": ""
  }
}
```

#### CREATION_FAILED

Emitted when a placement request is refused. Targets the requesting character only (`characterId` is the addressee, not a map broadcast). `reason` is one of `MAP_FULL`, `ALREADY_PLACED`, `MAP_FORBIDDEN`, `MESSAGE_TOO_LONG`.

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "characterId": 0,
  "type": "CREATION_FAILED",
  "body": {
    "reason": ""
  }
}
```

## Transaction Semantics

- Each produced status event is keyed by the field's map id.
- Consumed kite commands and character status events carry span, tenant, and env headers (`SpanHeaderParser`, `TenantHeaderParser`, `EnvHeaderParser`).
- All command and status-event handlers are registered with persistent configuration (`message.PersistentConfig`).
- `Create` and `Destroy` buffer their status events into a `message.Buffer`; `CreateAndEmit`/`DestroyAndEmit` flush the buffer to Kafka in one call to the producer.
