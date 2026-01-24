# Kafka

## Topics Consumed

### EVENT_TOPIC_MONSTER_STATUS

Monster status events.

**Consumer Group**: Monster Death Service

**Header Parsers**: SpanHeaderParser, TenantHeaderParser

---

## Topics Produced

### COMMAND_TOPIC_DROP

Drop spawn commands.

### COMMAND_TOPIC_CHARACTER

Character commands including experience awards.

---

## Message Types

### Consumed

#### statusEvent[statusEventKilledBody]

Monster killed event.

| Field | Type | Description |
|-------|------|-------------|
| worldId | byte | World identifier |
| channelId | byte | Channel identifier |
| mapId | uint32 | Map identifier |
| uniqueId | uint32 | Monster unique instance identifier |
| monsterId | uint32 | Monster type identifier |
| type | string | Event type ("KILLED") |
| body | statusEventKilledBody | Event body |

#### statusEventKilledBody

| Field | Type | Description |
|-------|------|-------------|
| x | int16 | Monster X position |
| y | int16 | Monster Y position |
| actorId | uint32 | Character who killed the monster |
| damageEntries | []damageEntry | Damage dealt by characters |

#### damageEntry

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character identifier |
| damage | uint32 | Damage dealt |

### Produced

#### command[spawnCommandBody]

Drop spawn command.

| Field | Type | Description |
|-------|------|-------------|
| worldId | byte | World identifier |
| channelId | byte | Channel identifier |
| mapId | uint32 | Map identifier |
| type | string | Command type ("SPAWN") |
| body | spawnCommandBody | Command body |

#### spawnCommandBody

| Field | Type | Description |
|-------|------|-------------|
| itemId | uint32 | Item identifier (0 for meso) |
| quantity | uint32 | Item quantity |
| mesos | uint32 | Meso amount (0 for items) |
| dropType | byte | Drop type |
| x | int16 | Drop X position |
| y | int16 | Drop Y position |
| ownerId | uint32 | Owner character identifier |
| ownerPartyId | uint32 | Owner party identifier |
| dropperId | uint32 | Monster unique identifier |
| dropperX | int16 | Monster X position |
| dropperY | int16 | Monster Y position |
| playerDrop | bool | Whether dropped by player |
| mod | byte | Drop modifier |

#### command[awardExperienceCommandBody]

Experience award command.

| Field | Type | Description |
|-------|------|-------------|
| worldId | byte | World identifier |
| characterId | uint32 | Character identifier |
| type | string | Command type ("AWARD_EXPERIENCE") |
| body | awardExperienceCommandBody | Command body |

#### awardExperienceCommandBody

| Field | Type | Description |
|-------|------|-------------|
| channelId | byte | Channel identifier |
| distributions | []experienceDistributions | Experience distributions |

#### experienceDistributions

| Field | Type | Description |
|-------|------|-------------|
| experienceType | string | Distribution type |
| amount | uint32 | Experience amount |
| attr1 | uint32 | Additional attribute |

**Experience Distribution Types**:
- WHITE
- YELLOW
- CHAT
- MONSTER_BOOK
- MONSTER_EVENT
- PLAY_TIME
- WEDDING
- SPIRIT_WEEK
- PARTY
- ITEM
- INTERNET_CAFE
- RAINBOW_WEEK
- PARTY_RING
- CAKE_PIE

---

## Transaction Semantics

Message production uses tenant header decoration propagated from consumed messages.
