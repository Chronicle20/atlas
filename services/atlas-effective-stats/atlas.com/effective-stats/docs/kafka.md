# Kafka Integration

## Topics Consumed

| Topic | Environment Variable | Description |
|-------|---------------------|-------------|
| Session status | `EVENT_TOPIC_SESSION_STATUS` | Character login/logout events |
| Buff status | `EVENT_TOPIC_CHARACTER_BUFF_STATUS` | Buff applied/expired events |
| Asset status | `EVENT_TOPIC_ASSET_STATUS` | Equipment moved/deleted events |
| Character status | `EVENT_TOPIC_CHARACTER_STATUS` | Character stat changed events |

## Topics Produced

| Topic | Environment Variable | Description |
|-------|---------------------|-------------|
| Character commands | `COMMAND_TOPIC_CHARACTER` | HP/MP clamp commands when MaxHP or MaxMP decreases |

## Message Types

### Session Status Events

#### StatusEvent

| Field | Type | Description |
|-------|------|-------------|
| sessionId | uuid.UUID | Session identifier |
| accountId | uint32 | Account identifier |
| characterId | uint32 | Character identifier |
| worldId | world.Id | World identifier |
| channelId | channel.Id | Channel identifier |
| issuer | string | Event issuer (`LOGIN` or `CHANNEL`) |
| type | string | Event type (`CREATED` or `DESTROYED`) |

##### Event Types

| Type | Description |
|------|-------------|
| `CREATED` | Session created; initializes character stats (channel issuer only) |
| `DESTROYED` | Session destroyed; removes character from registry |

---

### Buff Status Events

#### StatusEvent[AppliedStatusEventBody]

| Field | Type | Description |
|-------|------|-------------|
| worldId | world.Id | World identifier |
| channelId | channel.Id | Channel identifier |
| characterId | uint32 | Character identifier |
| type | string | Event type (`APPLIED`) |
| body | AppliedStatusEventBody | Event body |

##### AppliedStatusEventBody

| Field | Type | Description |
|-------|------|-------------|
| fromId | uint32 | Source character (for party buffs) |
| sourceId | int32 | Buff source skill ID |
| duration | int32 | Buff duration in milliseconds |
| changes | []StatChange | Stat changes from buff |
| createdAt | time.Time | Creation timestamp |
| expiresAt | time.Time | Expiration timestamp |

#### StatusEvent[ExpiredStatusEventBody]

| Field | Type | Description |
|-------|------|-------------|
| worldId | world.Id | World identifier |
| channelId | channel.Id | Channel identifier |
| characterId | uint32 | Character identifier |
| type | string | Event type (`EXPIRED`) |
| body | ExpiredStatusEventBody | Event body |

##### ExpiredStatusEventBody

| Field | Type | Description |
|-------|------|-------------|
| sourceId | int32 | Buff source skill ID |
| duration | int32 | Buff duration in milliseconds |
| changes | []StatChange | Stat changes from buff |
| createdAt | time.Time | Creation timestamp |
| expiresAt | time.Time | Expiration timestamp |

##### StatChange

| Field | Type | Description |
|-------|------|-------------|
| type | string | Stat type identifier |
| amount | int32 | Change amount |

##### Event Types

| Type | Description |
|------|-------------|
| `APPLIED` | Buff applied; adds buff bonuses |
| `EXPIRED` | Buff expired; removes buff bonuses |

---

### Asset Status Events

#### StatusEvent[MovedStatusEventBody]

| Field | Type | Description |
|-------|------|-------------|
| transactionId | uuid.UUID | Transaction identifier |
| characterId | uint32 | Character identifier |
| compartmentId | uuid.UUID | Compartment identifier |
| assetId | uint32 | Asset identifier |
| templateId | uint32 | Item template identifier |
| slot | int16 | New slot position |
| type | string | Event type (`MOVED`) |
| body | MovedStatusEventBody | Event body |

##### MovedStatusEventBody

| Field | Type | Description |
|-------|------|-------------|
| oldSlot | int16 | Previous slot position |
| createdAt | time.Time | Creation timestamp |

#### StatusEvent[DeletedStatusEventBody]

| Field | Type | Description |
|-------|------|-------------|
| transactionId | uuid.UUID | Transaction identifier |
| characterId | uint32 | Character identifier |
| compartmentId | uuid.UUID | Compartment identifier |
| assetId | uint32 | Asset identifier |
| templateId | uint32 | Item template identifier |
| slot | int16 | Slot position |
| type | string | Event type (`DELETED`) |
| body | DeletedStatusEventBody | Event body |

##### DeletedStatusEventBody

Empty.

##### Event Types

| Type | Description |
|------|-------------|
| `MOVED` | Asset moved; adds equipment bonuses on equip (positive to negative slot), removes on unequip (negative to positive slot) |
| `DELETED` | Asset deleted; removes equipment bonuses if slot was negative |

---

### Character Status Events

#### StatusEvent[StatusEventStatChangedBody]

| Field | Type | Description |
|-------|------|-------------|
| transactionId | uuid.UUID | Transaction identifier |
| worldId | world.Id | World identifier |
| characterId | uint32 | Character identifier |
| type | string | Event type (`STAT_CHANGED`) |
| body | StatusEventStatChangedBody | Event body |

##### StatusEventStatChangedBody

| Field | Type | Description |
|-------|------|-------------|
| channelId | channel.Id | Channel identifier |
| exclRequestSent | bool | Whether an exclusive request was sent |
| updates | []stat.Type | List of stat types that changed |
| values | map[string]interface{} | Map of stat names to new values |

The consumer filters for events containing relevant stats: `MAX_HP`, `MAX_MP`, `STRENGTH`, `DEXTERITY`, `INTELLIGENCE`, `LUCK`. Other stat changes are ignored.

##### Event Types

| Type | Description |
|------|-------------|
| `STAT_CHANGED` | Character base stats changed; updates base stats in registry and recomputes effective stats |

---

### Character Commands (Produced)

#### Command[ClampHPBody]

Published when a bonus removal causes MaxHP to decrease.

| Field | Type | Description |
|-------|------|-------------|
| transactionId | uuid.UUID | Transaction identifier |
| worldId | world.Id | World identifier |
| characterId | uint32 | Character identifier |
| type | string | Command type (`CLAMP_HP`) |
| body | ClampHPBody | Command body |

##### ClampHPBody

| Field | Type | Description |
|-------|------|-------------|
| channelId | channel.Id | Channel identifier |
| maxValue | uint16 | New maximum HP value |

#### Command[ClampMPBody]

Published when a bonus removal causes MaxMP to decrease.

| Field | Type | Description |
|-------|------|-------------|
| transactionId | uuid.UUID | Transaction identifier |
| worldId | world.Id | World identifier |
| characterId | uint32 | Character identifier |
| type | string | Command type (`CLAMP_MP`) |
| body | ClampMPBody | Command body |

##### ClampMPBody

| Field | Type | Description |
|-------|------|-------------|
| channelId | channel.Id | Channel identifier |
| maxValue | uint16 | New maximum MP value |

## Transaction Semantics

None. Events are processed independently without transactional guarantees.
