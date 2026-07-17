# Summon Kafka Integration

## Topics Consumed

### COMMAND_TOPIC_SUMMON

Summon commands relayed from atlas-channel.

**Consumer Group:** Summon Registry Service

**Envelope:**

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "summonId": 0,
  "type": "SPAWN",
  "body": {}
}
```

**Message Types:**

#### SPAWN

```json
{
  "type": "SPAWN",
  "body": {
    "ownerCharacterId": 0,
    "skillId": 0,
    "skillLevel": 0,
    "x": 0,
    "y": 0,
    "auraLevel": 0,
    "hexLevel": 0
  }
}
```

`auraLevel`/`hexLevel` carry the caster's trained AURA_OF_THE_BEHOLDER
(1320008) and HEX_OF_THE_BEHOLDER (1320009) levels for a Beholder summon; 0
for all other summons.

#### MOVE

```json
{
  "type": "MOVE",
  "body": {
    "summonId": 0,
    "senderCharacterId": 0,
    "x": 0,
    "y": 0,
    "stance": 0,
    "rawMovement": "base64"
  }
}
```

#### ATTACK

```json
{
  "type": "ATTACK",
  "body": {
    "summonId": 0,
    "senderCharacterId": 0,
    "direction": 0,
    "targets": [
      { "monsterId": 0, "damage": 0 }
    ]
  }
}
```

`damage` is the raw client-reported value; atlas-summons clamps it before use.

#### DAMAGE

```json
{
  "type": "DAMAGE",
  "body": {
    "summonId": 0,
    "senderCharacterId": 0,
    "damage": 0,
    "monsterIdFrom": 0
  }
}
```

### EVENT_TOPIC_CHARACTER_STATUS

Character status events consumed to despawn a character's summons on logout,
channel change, or map change.

**Consumer Group:** Summon Registry Service

**Envelope:**

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "characterId": 0,
  "type": "LOGOUT",
  "body": {}
}
```

**Message Types:**

#### LOGOUT

```json
{
  "type": "LOGOUT",
  "body": {
    "channelId": 0,
    "mapId": 0,
    "instance": "uuid"
  }
}
```

#### CHANNEL_CHANGED

```json
{
  "type": "CHANNEL_CHANGED",
  "body": {
    "channelId": 0,
    "oldChannelId": 0,
    "mapId": 0,
    "instance": "uuid"
  }
}
```

#### MAP_CHANGED

```json
{
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

Only `characterId` from the envelope is used; other message types on this
topic are decoded into the same envelope shape but produce no summon action.

## Topics Produced

### EVENT_TOPIC_SUMMON_STATUS

Summon lifecycle and status events.

**Partitioning:** Keyed by mapId

**Envelope:**

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "summonId": 0,
  "ownerCharacterId": 0,
  "skillId": 0,
  "type": "CREATED",
  "body": {}
}
```

**Message Types:**

#### CREATED

```json
{
  "type": "CREATED",
  "body": {
    "skillLevel": 0,
    "movementType": 0,
    "x": 0,
    "y": 0,
    "stance": 0,
    "puppet": false,
    "animated": true
  }
}
```

#### MOVED

```json
{
  "type": "MOVED",
  "body": {
    "x": 0,
    "y": 0,
    "stance": 0,
    "rawMovement": "base64"
  }
}
```

#### ATTACKED

```json
{
  "type": "ATTACKED",
  "body": {
    "direction": 0,
    "targets": [
      { "monsterId": 0, "damage": 0 }
    ]
  }
}
```

`damage` is the server-clamped value, not the raw client report.

#### DAMAGED

```json
{
  "type": "DAMAGED",
  "body": {
    "damage": 0,
    "monsterIdFrom": 0
  }
}
```

#### DESTROYED

```json
{
  "type": "DESTROYED",
  "body": {
    "animated": true
  }
}
```

#### SKILL

Emitted by the Beholder aura sweep so atlas-channel rebroadcasts the
server-driven heal/buff pulse visual map-wide.

```json
{
  "type": "SKILL",
  "body": {
    "newStance": 6
  }
}
```

`newStance` is 5 for the heal pulse and 6-8 for the buff pulse; the sweep
always emits 6 (see docs/domain.md, Beholder Aura Sweep).

### COMMAND_TOPIC_MONSTER

Commands produced to atlas-monsters to credit owner damage, apply monster
status effects, and register/clear puppet controller bias.

**Message Types:**

#### ADD_PUPPET

Flat envelope (no `body`). Partitioning: keyed by ownerCharacterId.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "type": "ADD_PUPPET",
  "ownerCharacterId": 0,
  "x": 0,
  "y": 0
}
```

#### REMOVE_PUPPET

Flat envelope (no `body`). Partitioning: keyed by ownerCharacterId.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "type": "REMOVE_PUPPET",
  "ownerCharacterId": 0
}
```

#### DAMAGE

Partitioning: keyed by monsterId.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "monsterId": 0,
  "type": "DAMAGE",
  "body": {
    "characterId": 0,
    "damages": [0],
    "attackType": 0
  }
}
```

`characterId` is the summon's owner. `attackType` is always 0.

#### APPLY_STATUS

Partitioning: keyed by monsterId.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 0,
  "instance": "uuid",
  "monsterId": 0,
  "type": "APPLY_STATUS",
  "body": {
    "sourceType": "PLAYER_SKILL",
    "sourceCharacterId": 0,
    "sourceSkillId": 0,
    "sourceSkillLevel": 0,
    "statuses": { "STATUS_TYPE": 0 },
    "duration": 0,
    "tickInterval": 0
  }
}
```

`duration`/`tickInterval` are in milliseconds; `tickInterval` is always 0.

### COMMAND_TOPIC_CHARACTER_BUFF

Commands produced to atlas-buffs by the Beholder aura sweep to apply the Hex
buff to the summon's owner.

**Message Type:**

#### APPLY

Partitioning: keyed by characterId.

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
      { "type": "STAT_TYPE", "amount": 0 }
    ],
    "accumulate": true
  }
}
```

`accumulate` asks atlas-buffs to store each change as its own
independently-timed buff under the same sourceId rather than replacing the
whole sourceId buff; the Beholder sweep always sets it true.

### COMMAND_TOPIC_CHARACTER

Commands produced to atlas-character by the Beholder aura sweep to heal the
summon's owner.

**Message Type:**

#### CHANGE_HP

Partitioning: keyed by characterId.

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "characterId": 0,
  "type": "CHANGE_HP",
  "body": {
    "channelId": 0,
    "amount": 0
  }
}
```

## Transaction Semantics

- All consumed messages require span and tenant headers
- Summon status events are keyed by mapId for partition ordering within a map
- atlas-monsters ADD_PUPPET/REMOVE_PUPPET commands are keyed by ownerCharacterId; DAMAGE/APPLY_STATUS commands are keyed by monsterId
- atlas-buffs APPLY commands are keyed by characterId
- atlas-character CHANGE_HP commands are keyed by characterId and carry a freshly generated transactionId

## Headers

**Required on all consumed messages:**
- Span headers (for distributed tracing)
- Tenant headers (for multi-tenancy)

**Added to all produced messages:**
- Span headers
- Tenant headers
