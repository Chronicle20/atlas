# Monster Kafka Integration

## Topics Consumed

### EVENT_TOPIC_MAP_STATUS

Map status events for handling character enter/exit.

**Consumer Group:** Monster Registry Service

**Message Types:**

#### CHARACTER_ENTER

Triggers controller assignment to uncontrolled monsters in the field.

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
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
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "type": "CHARACTER_EXIT",
  "body": {
    "characterId": 0
  }
}
```

### COMMAND_TOPIC_MONSTER

Monster commands for damage, status effects, and skill use.

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
    "damage": 0,
    "attackType": 0
  }
}
```

`attackType`: 0=melee, 1=ranged, 2=magic, 3=energy.

#### APPLY_STATUS

Applies a status effect to a specific monster.

```json
{
  "worldId": 0,
  "channelId": 0,
  "monsterId": 0,
  "type": "APPLY_STATUS",
  "body": {
    "sourceType": "PLAYER_SKILL",
    "sourceCharacterId": 0,
    "sourceSkillId": 0,
    "sourceSkillLevel": 0,
    "statuses": {
      "STATUS_TYPE": 0
    },
    "duration": 0,
    "tickInterval": 0
  }
}
```

`duration` and `tickInterval` are in milliseconds.

#### CANCEL_STATUS

Cancels status effects from a specific monster. If `statusTypes` is empty, all status effects are cancelled.

```json
{
  "worldId": 0,
  "channelId": 0,
  "monsterId": 0,
  "type": "CANCEL_STATUS",
  "body": {
    "statusTypes": ["STATUS_TYPE"]
  }
}
```

#### USE_SKILL

Commands a monster to use a skill.

```json
{
  "worldId": 0,
  "channelId": 0,
  "monsterId": 0,
  "type": "USE_SKILL",
  "body": {
    "characterId": 0,
    "skillId": 0,
    "skillLevel": 0
  }
}
```

#### APPLY_STATUS_FIELD

Applies a status effect to all monsters in a field.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "type": "APPLY_STATUS_FIELD",
  "body": {
    "sourceType": "PLAYER_SKILL",
    "sourceCharacterId": 0,
    "sourceSkillId": 0,
    "sourceSkillLevel": 0,
    "statuses": {
      "STATUS_TYPE": 0
    },
    "duration": 0,
    "tickInterval": 0
  }
}
```

#### CANCEL_STATUS_FIELD

Cancels status effects from all monsters in a field. If `statusTypes` is empty, all status effects are cancelled.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "type": "CANCEL_STATUS_FIELD",
  "body": {
    "statusTypes": ["STATUS_TYPE"]
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
  "instance": "uuid",
  "objectId": 0,
  "observerId": 0,
  "x": 0,
  "y": 0,
  "stance": 0
}
```

## Topics Produced

### EVENT_TOPIC_MONSTER_STATUS

Monster status events emitted during lifecycle and status effect changes.

**Partitioning:** Keyed by mapId

**Message Types:**

#### CREATED

Emitted when a monster is spawned.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
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
  "instance": "uuid",
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
  "instance": "uuid",
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
  "instance": "uuid",
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
  "instance": "uuid",
  "uniqueId": 0,
  "monsterId": 0,
  "type": "DAMAGED",
  "body": {
    "x": 0,
    "y": 0,
    "actorId": 0,
    "boss": false,
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
  "instance": "uuid",
  "uniqueId": 0,
  "monsterId": 0,
  "type": "KILLED",
  "body": {
    "x": 0,
    "y": 0,
    "actorId": 0,
    "boss": false,
    "damageEntries": [
      {
        "characterId": 0,
        "damage": 0
      }
    ]
  }
}
```

#### STATUS_APPLIED

Emitted when a status effect is applied to a monster.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "uniqueId": 0,
  "monsterId": 0,
  "type": "STATUS_APPLIED",
  "body": {
    "effectId": "uuid",
    "sourceType": "PLAYER_SKILL",
    "sourceCharacterId": 0,
    "sourceSkillId": 0,
    "sourceSkillLevel": 0,
    "statuses": {
      "STATUS_TYPE": 0
    },
    "duration": 0
  }
}
```

`duration` is in milliseconds.

#### STATUS_EXPIRED

Emitted when a status effect expires naturally.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "uniqueId": 0,
  "monsterId": 0,
  "type": "STATUS_EXPIRED",
  "body": {
    "effectId": "uuid",
    "statuses": {
      "STATUS_TYPE": 0
    }
  }
}
```

#### STATUS_CANCELLED

Emitted when a status effect is cancelled explicitly (by command or on monster death).

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "uniqueId": 0,
  "monsterId": 0,
  "type": "STATUS_CANCELLED",
  "body": {
    "effectId": "uuid",
    "statuses": {
      "STATUS_TYPE": 0
    }
  }
}
```

#### DAMAGE_REFLECTED

Emitted when a monster reflects damage back to a character.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "uniqueId": 0,
  "monsterId": 0,
  "type": "DAMAGE_REFLECTED",
  "body": {
    "characterId": 0,
    "reflectDamage": 0,
    "reflectType": "WEAPON_REFLECT"
  }
}
```

`reflectType`: "WEAPON_REFLECT" or "MAGIC_REFLECT".

### COMMAND_TOPIC_CHARACTER_BUFF

Character buff commands produced when monster debuff skills target players.

**Message Types:**

#### APPLY

Applies a disease (debuff) to a character.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "characterId": 0,
  "type": "APPLY",
  "body": {
    "fromId": 0,
    "sourceId": 0,
    "level": 0,
    "duration": 0,
    "changes": [
      {
        "type": "DISEASE_NAME",
        "amount": 0
      }
    ]
  }
}
```

#### CANCEL_ALL

Cancels all buffs from a character (dispel).

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "characterId": 0,
  "type": "CANCEL_ALL",
  "body": {}
}
```

### COMMAND_TOPIC_PORTAL

Portal/warp commands produced when monster banish skills target players.

**Message Type:**

#### WARP

Warps a character to a target map.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "type": "WARP",
  "body": {
    "characterId": 0,
    "targetMapId": 0
  }
}
```

## Transaction Semantics

- All messages include span and tenant headers for tracing and multi-tenancy
- Monster status events are keyed by mapId for partition ordering within a map
- Character buff commands are keyed by characterId
- Portal commands are keyed by characterId

## Headers

**Required on all consumed messages:**
- Span headers (for distributed tracing)
- Tenant headers (for multi-tenancy)

**Added to all produced messages:**
- Span headers
- Tenant headers
