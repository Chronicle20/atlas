# Portal Actions Kafka Integration

## Topics Consumed

| Environment Variable | Direction | Description |
|---------------------|-----------|-------------|
| COMMAND_TOPIC_PORTAL_ACTIONS | command | Portal entry commands |

## Topics Produced

| Environment Variable | Direction | Description |
|---------------------|-----------|-------------|
| EVENT_TOPIC_CHARACTER_STATUS | event | Character status events (enable actions) |
| COMMAND_TOPIC_CHARACTER | command | Character commands (change map) |
| COMMAND_TOPIC_SAGA | command | Saga commands for operations |

## Message Types

### Portal Entry Command (Consumed)

Topic: `COMMAND_TOPIC_PORTAL_ACTIONS`

```json
{
  "worldId": 0,
  "channelId": 1,
  "mapId": 100000000,
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
| portalId | uint32 | Numeric portal identifier |
| type | string | Command type (ENTER) |
| body.characterId | uint32 | Character identifier |
| body.portalName | string | Portal name for script lookup |

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

### Change Map Command (Produced)

Topic: `COMMAND_TOPIC_CHARACTER`

```json
{
  "worldId": 0,
  "characterId": 12345,
  "type": "CHANGE_MAP",
  "body": {
    "channelId": 1,
    "mapId": 200000000,
    "portalId": 0
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| worldId | byte | World identifier |
| characterId | uint32 | Character identifier |
| type | string | Command type (CHANGE_MAP) |
| body.channelId | byte | Channel identifier |
| body.mapId | uint32 | Target map identifier |
| body.portalId | uint32 | Target portal identifier |

### Saga Command (Produced)

Topic: `COMMAND_TOPIC_SAGA`

Saga messages are produced for operations including warp, play_portal_sound, drop_message, show_hint, block_portal, create_skill, and update_skill.

## Transaction Semantics

- Portal entry commands are processed with tenant context from Kafka headers
- Character actions are enabled after processing completes (success or failure)
- Operations are executed via saga messages for coordination
