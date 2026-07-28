# Kafka — atlas-map-actions

## Topics Consumed

| Environment Variable | Direction | Consumer Group |
|---------------------|-----------|----------------|
| `COMMAND_TOPIC_MAP_ACTIONS` | Command | `Map Actions Service` |
| `EVENT_TOPIC_SAGA_STATUS` | Event | `Map Actions Service` |

## Topics Produced

| Environment Variable | Direction |
|---------------------|-----------|
| `COMMAND_TOPIC_SAGA` | Command |
| `EVENT_TOPIC_CHARACTER_STATUS` | Event |

## Message Types

### COMMAND_TOPIC_MAP_ACTIONS

**commandEvent[enterBody]** (`script/consumer.go`)

Consumed when type is `"ENTER"`.

```
{
  "worldId":   world.Id,
  "channelId": channel.Id,
  "mapId":     _map.Id,
  "instance":  uuid.UUID,
  "type":      "ENTER",
  "body": {
    "characterId": uint32,
    "scriptName":  string,
    "scriptType":  string
  }
}
```

Headers: `SpanHeaderParser`, `TenantHeaderParser`.

### EVENT_TOPIC_SAGA_STATUS

**StatusEvent[StatusEventCompletedBody]** (`kafka/message/saga/kafka.go`)

Consumed when type is `"COMPLETED"`. Logged at debug level.

```
{
  "transactionId": uuid.UUID,
  "type":          "COMPLETED",
  "body":          {}
}
```

**StatusEvent[StatusEventFailedBody]** (`kafka/message/saga/kafka.go`)

Consumed when type is `"FAILED"`. Logged at warn level.

```
{
  "transactionId": uuid.UUID,
  "type":          "FAILED",
  "body": {
    "errorCode":  string,
    "reason":     string,
    "failedStep": string
  }
}
```

Headers: `SpanHeaderParser`, `TenantHeaderParser`.

### COMMAND_TOPIC_SAGA

**scriptsaga.Saga** (`saga/producer.go`)

Produced when operations are executed. Key is the saga transaction ID.

### EVENT_TOPIC_CHARACTER_STATUS

**statusEvent[statChangedBody]** (`character/producer.go`)

Produced after script processing completes to re-enable character actions.

```
{
  "characterId": uint32,
  "type":        "STAT_CHANGED",
  "worldId":     world.Id,
  "body": {
    "channelId":       channel.Id,
    "exclRequestSent": true
  }
}
```

Key: character ID.

## Transaction Semantics

Each consumed `ENTER` command results in:
1. Script lookup and rule evaluation
2. Zero or one saga command produced (for the first matched rule's operations)
3. One `STAT_CHANGED` event produced (always, regardless of match result or error)
