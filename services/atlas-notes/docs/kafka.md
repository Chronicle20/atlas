# Kafka Integration

## Topics Consumed

### COMMAND_TOPIC_NOTE

Commands for note operations.

| Command Type | Body | Description |
|--------------|------|-------------|
| CREATE | CommandCreateBody | Creates a note for a character |
| DISCARD | CommandDiscardBody | Deletes multiple notes for a character |

### EVENT_TOPIC_CHARACTER_STATUS

Character status events.

| Event Type | Body | Description |
|------------|------|-------------|
| DELETED | StatusEventDeletedBody | Triggers deletion of all notes for the character |

## Topics Produced

### EVENT_TOPIC_NOTE_STATUS

Note status events emitted when notes change.

| Event Type | Body | Description |
|------------|------|-------------|
| CREATED | StatusEventCreatedBody | Emitted when a note is created |
| UPDATED | StatusEventUpdatedBody | Emitted when a note is updated |
| DELETED | StatusEventDeletedBody | Emitted when a note is deleted |

### COMMAND_TOPIC_SAGA

Saga commands produced when discarding notes to award fame to the sender.

## Message Types

### Command[E]

```json
{
  "worldId": 0,
  "channelId": 0,
  "characterId": 123,
  "type": "CREATE",
  "body": {}
}
```

### CommandCreateBody

```json
{
  "senderId": 456,
  "message": "Note message",
  "flag": 0
}
```

### CommandDiscardBody

```json
{
  "noteIds": [1, 2, 3]
}
```

### StatusEvent[E]

```json
{
  "characterId": 123,
  "type": "CREATED",
  "body": {}
}
```

### StatusEventCreatedBody

```json
{
  "noteId": 1,
  "senderId": 456,
  "message": "Note message",
  "flag": 0,
  "time": "2024-01-01T00:00:00Z"
}
```

### StatusEventUpdatedBody

```json
{
  "noteId": 1,
  "senderId": 456,
  "message": "Note message",
  "flag": 0,
  "time": "2024-01-01T00:00:00Z"
}
```

### StatusEventDeletedBody

```json
{
  "noteId": 1
}
```

## Transaction Semantics

- Messages are partitioned by characterId
- Status events are emitted after successful database operations
- Buffered emission ensures events are sent atomically with database changes
- Required headers: span context and tenant headers on all consumed topics
