# Kafka

## Topics Consumed

### EVENT_TOPIC_MONSTER_STATUS

Monster status events.

**Consumer Group**: Monster Death Service

**Header Parsers**: SpanHeaderParser, TenantHeaderParser

**Handler**: Handles `KILLED` type events. On receipt, concurrently creates drops and distributes experience.

---

## Topics Produced

### COMMAND_TOPIC_DROP

Drop spawn commands. Each successful drop evaluation produces a SPAWN command.

### COMMAND_TOPIC_CHARACTER

Character commands. Experience award commands are produced for each character who contributed damage.

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
| instance | uuid | Field instance identifier |
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

Drop spawn command. Produced to COMMAND_TOPIC_DROP.

| Field | Type | Description |
|-------|------|-------------|
| worldId | byte | World identifier |
| channelId | byte | Channel identifier |
| mapId | uint32 | Map identifier |
| instance | uuid | Field instance identifier |
| type | string | Command type ("SPAWN") |
| body | spawnCommandBody | Command body |

**Message key**: Map ID

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
| strength | uint16 | Equipment STR (inline, 0 if not equipment) |
| dexterity | uint16 | Equipment DEX (inline, 0 if not equipment) |
| intelligence | uint16 | Equipment INT (inline, 0 if not equipment) |
| luck | uint16 | Equipment LUK (inline, 0 if not equipment) |
| hp | uint16 | Equipment HP (inline, 0 if not equipment) |
| mp | uint16 | Equipment MP (inline, 0 if not equipment) |
| weaponAttack | uint16 | Equipment weapon attack (inline, 0 if not equipment) |
| magicAttack | uint16 | Equipment magic attack (inline, 0 if not equipment) |
| weaponDefense | uint16 | Equipment weapon defense (inline, 0 if not equipment) |
| magicDefense | uint16 | Equipment magic defense (inline, 0 if not equipment) |
| accuracy | uint16 | Equipment accuracy (inline, 0 if not equipment) |
| avoidability | uint16 | Equipment avoidability (inline, 0 if not equipment) |
| hands | uint16 | Equipment hands (inline, 0 if not equipment) |
| speed | uint16 | Equipment speed (inline, 0 if not equipment) |
| jump | uint16 | Equipment jump (inline, 0 if not equipment) |
| slots | uint16 | Equipment upgrade slots (inline, 0 if not equipment) |

Equipment statistics fields are embedded directly in the spawn command body (via `EquipmentData`). For non-equipment drops, all equipment fields are zero-valued.

#### command[awardExperienceCommandBody]

Experience award command. Produced to COMMAND_TOPIC_CHARACTER.

| Field | Type | Description |
|-------|------|-------------|
| worldId | byte | World identifier |
| characterId | uint32 | Character identifier |
| type | string | Command type ("AWARD_EXPERIENCE") |
| body | awardExperienceCommandBody | Command body |

**Message key**: Character ID

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

Message production uses span and tenant header decoration propagated from consumed messages. Drop creation and experience distribution run concurrently via goroutines; failures in one do not affect the other.
