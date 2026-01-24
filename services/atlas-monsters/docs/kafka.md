# Monster Kafka Integration

## Topics Consumed

### EVENT_TOPIC_MAP_STATUS

Map status events for handling character enter/exit.

**Consumer Group:** Monster Registry Service

**Message Types:**

#### CHARACTER_ENTER

Triggers controller assignment to uncontrolled monsters in the map.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "type": "CHARACTER_ENTER",
  "body": {
    "characterId": 0
  }
}
```

#### CHARACTER_EXIT

Triggers control stop for monsters controlled by the exiting character and reassignment to remaining characters.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "type": "CHARACTER_EXIT",
  "body": {
    "characterId": 0
  }
}
```

### COMMAND_TOPIC_MONSTER

Monster damage commands.

**Consumer Group:** Monster Registry Service

**Message Types:**

#### DAMAGE

Applies damage to a monster from a character.

```json
{
  "worldId": 0,
  "channelId": 0,
  "monsterId": 0,
  "type": "DAMAGE",
  "body": {
    "characterId": 0,
    "damage": 0
  }
}
```

### COMMAND_TOPIC_MONSTER_MOVEMENT

Monster movement commands.

**Consumer Group:** Monster Registry Service

**Message Type:**

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "objectId": 0,
  "observerId": 0,
  "x": 0,
  "y": 0,
  "stance": 0
}
```

## Topics Produced

### EVENT_TOPIC_MONSTER_STATUS

Monster status events emitted during lifecycle changes.

**Partitioning:** Keyed by mapId

**Message Types:**

#### CREATED

Emitted when a monster is spawned.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "uniqueId": 0,
  "monsterId": 0,
  "type": "CREATED",
  "body": {
    "actorId": 0
  }
}
```

#### DESTROYED

Emitted when a monster is manually destroyed.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "uniqueId": 0,
  "monsterId": 0,
  "type": "DESTROYED",
  "body": {
    "actorId": 0
  }
}
```

#### START_CONTROL

Emitted when a character begins controlling a monster.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "uniqueId": 0,
  "monsterId": 0,
  "type": "START_CONTROL",
  "body": {
    "actorId": 0,
    "x": 0,
    "y": 0,
    "stance": 0,
    "fh": 0,
    "team": 0
  }
}
```

#### STOP_CONTROL

Emitted when a character stops controlling a monster.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "uniqueId": 0,
  "monsterId": 0,
  "type": "STOP_CONTROL",
  "body": {
    "actorId": 0
  }
}
```

#### DAMAGED

Emitted when a monster takes damage but survives.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "uniqueId": 0,
  "monsterId": 0,
  "type": "DAMAGED",
  "body": {
    "x": 0,
    "y": 0,
    "actorId": 0,
    "damageEntries": [
      {
        "characterId": 0,
        "damage": 0
      }
    ]
  }
}
```

#### KILLED

Emitted when a monster is killed.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "uniqueId": 0,
  "monsterId": 0,
  "type": "KILLED",
  "body": {
    "x": 0,
    "y": 0,
    "actorId": 0,
    "damageEntries": [
      {
        "characterId": 0,
        "damage": 0
      }
    ]
  }
}
```

## Transaction Semantics

- All messages include span and tenant headers for tracing and multi-tenancy
- Monster status events are keyed by mapId for partition ordering within a map

## Headers

**Required on all consumed messages:**
- Span headers (for distributed tracing)
- Tenant headers (for multi-tenancy)

**Added to all produced messages:**
- Span headers
- Tenant headers
