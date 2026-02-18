# Portal Actions Kafka Integration

## Topics Consumed

| Environment Variable | Direction | Description |
|---------------------|-----------|-------------|
| COMMAND_TOPIC_PORTAL_ACTIONS | command | Portal entry commands |
| EVENT_TOPIC_SAGA_STATUS | event | Saga completion and failure events |

## Topics Produced

| Environment Variable | Direction | Description |
|---------------------|-----------|-------------|
| EVENT_TOPIC_CHARACTER_STATUS | event | Character status events (enable actions) |
| COMMAND_TOPIC_SAGA | command | Saga commands for operations |

## Message Types

### Portal Entry Command (Consumed)

Topic: `COMMAND_TOPIC_PORTAL_ACTIONS`

```json
{
  "worldId": 0,
  "channelId": 1,
  "mapId": 100000000,
  "instance": "00000000-0000-0000-0000-000000000000",
  "portalId": 0,
  "type": "ENTER",
  "body": {
    "characterId": 12345,
    "portalName": "east00"
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| worldId | byte | World identifier |
| channelId | byte | Channel identifier |
| mapId | uint32 | Map identifier |
| instance | uuid | Map instance identifier |
| portalId | uint32 | Numeric portal identifier |
| type | string | Command type (ENTER) |
| body.characterId | uint32 | Character identifier |
| body.portalName | string | Portal name for script lookup |

### Saga Status Event - Completed (Consumed)

Topic: `EVENT_TOPIC_SAGA_STATUS`

```json
{
  "transactionId": "uuid",
  "type": "COMPLETED",
  "body": {}
}
```

| Field | Type | Description |
|-------|------|-------------|
| transactionId | uuid | Saga transaction identifier |
| type | string | Status event type (COMPLETED) |

### Saga Status Event - Failed (Consumed)

Topic: `EVENT_TOPIC_SAGA_STATUS`

```json
{
  "transactionId": "uuid",
  "type": "FAILED",
  "body": {
    "errorCode": "TRANSPORT_CAPACITY_FULL",
    "reason": "string",
    "failedStep": "string"
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| transactionId | uuid | Saga transaction identifier |
| type | string | Status event type (FAILED) |
| body.errorCode | string | Error code |
| body.reason | string | Failure reason |
| body.failedStep | string | Step that failed |

### Stat Changed Event (Produced)

Topic: `EVENT_TOPIC_CHARACTER_STATUS`

```json
{
  "characterId": 12345,
  "type": "STAT_CHANGED",
  "worldId": 0,
  "body": {
    "channelId": 1,
    "exclRequestSent": true
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character identifier |
| type | string | Event type (STAT_CHANGED) |
| worldId | byte | World identifier |
| body.channelId | byte | Channel identifier |
| body.exclRequestSent | bool | Enable character actions |

### Saga Command (Produced)

Topic: `COMMAND_TOPIC_SAGA`

Saga messages are produced for operations including warp, play_portal_sound, drop_message, show_hint, block_portal, create_skill, update_skill, start_instance_transport, apply_consumable_effect, cancel_consumable_effect, save_location, and warp_to_saved_location.

## Transaction Semantics

- Portal entry commands are processed with tenant context from Kafka headers
- Character actions are enabled after processing completes (success or failure)
- Operations are executed via saga messages for coordination
- For `start_instance_transport` operations, a pending action is registered in the action registry; saga completion removes the entry, and saga failure sends a failure message to the character before cleanup
