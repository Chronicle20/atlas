# Kafka Integration

## Topics Consumed

| Topic | Environment Variable | Description |
|-------|---------------------|-------------|
| Session status | `EVENT_TOPIC_SESSION_STATUS` | Character login/logout events |
| Buff status | `EVENT_TOPIC_CHARACTER_BUFF_STATUS` | Buff applied/expired events |
| Asset status | `EVENT_TOPIC_ASSET_STATUS` | Equipment moved/deleted events |

## Topics Produced

None.

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
| worldId | byte | World identifier |
| channelId | byte | Channel identifier |
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
| worldId | byte | World identifier |
| channelId | byte | Channel identifier |
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
| referenceType | string | Asset reference type (`equipable` or `cash_equipable`) |

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

## Transaction Semantics

None. Events are processed independently without transactional guarantees.
