# Kafka Integration

## Topics Consumed

| Topic | Environment Variable | Description |
|-------|---------------------|-------------|
| Reactor Actions Command | `COMMAND_TOPIC_REACTOR_ACTIONS` | Reactor hit and trigger commands |

## Topics Produced

| Topic | Environment Variable | Description |
|-------|---------------------|-------------|
| Saga Command | `COMMAND_TOPIC_SAGA` | Saga orchestration commands |

## Message Types

### Consumed: Reactor Command

Command message for reactor hit or trigger events.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 100000000,
  "reactorId": 123,
  "classification": "2000",
  "reactorName": "box01",
  "reactorState": 1,
  "x": 100,
  "y": 200,
  "type": "HIT",
  "body": {}
}
```

| Field | Type | Description |
|-------|------|-------------|
| worldId | byte | World identifier |
| channelId | byte | Channel identifier |
| mapId | uint32 | Map identifier |
| reactorId | uint32 | Reactor instance identifier |
| classification | string | Reactor classification ID |
| reactorName | string | Reactor name |
| reactorState | int8 | Current reactor state |
| x | int16 | X coordinate |
| y | int16 | Y coordinate |
| type | string | Command type: `HIT` or `TRIGGER` |
| body | object | Type-specific body |

### Command Type: HIT

Body for hit commands:

```json
{
  "characterId": 12345,
  "skillId": 0,
  "isSkill": false
}
```

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character who hit the reactor |
| skillId | uint32 | Skill used (if skill attack) |
| isSkill | bool | Whether attack was a skill |

### Command Type: TRIGGER

Body for trigger commands:

```json
{
  "characterId": 12345
}
```

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character who triggered the reactor |

### Produced: Saga Command

Saga commands are produced using the `atlas-script-core/saga` package. The service produces saga commands for:

- `SpawnReactorDrops`: Spawn item drops at reactor location
- `SpawnMonster`: Spawn monsters at reactor location
- `SendMessage`: Send message to character

## Transaction Semantics

- Commands are consumed with tenant header parsing
- Saga commands are produced with transaction ID for orchestration
- Message key is the saga transaction ID
