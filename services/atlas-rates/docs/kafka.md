# Kafka

## Topics Consumed

| Topic | Environment Variable | Description |
|-------|---------------------|-------------|
| Character Buff Status | `EVENT_TOPIC_CHARACTER_BUFF_STATUS` | Buff applied/expired events |
| World Rate | `EVENT_TOPIC_WORLD_RATE` | World rate change events |
| Asset Status | `EVENT_TOPIC_ASSET_STATUS` | Inventory asset lifecycle events |
| Character Status | `EVENT_TOPIC_CHARACTER_STATUS` | Character status events (map changes) |

## Topics Produced

None.

## Message Types

### Buff Status Events

**StatusEvent[AppliedStatusEventBody]** (`kafka/message/buff/kafka.go`)

| Field | Type | Description |
|-------|------|-------------|
| worldId | world.Id | World identifier |
| channelId | channel.Id | Channel identifier |
| characterId | uint32 | Character identifier |
| type | string | `APPLIED` |
| body.fromId | uint32 | Source of buff application |
| body.sourceId | int32 | Buff source identifier |
| body.duration | int32 | Buff duration |
| body.changes | []StatChange | Stat changes from buff |
| body.createdAt | time.Time | When buff was created |
| body.expiresAt | time.Time | When buff expires |

**StatusEvent[ExpiredStatusEventBody]** (`kafka/message/buff/kafka.go`)

| Field | Type | Description |
|-------|------|-------------|
| worldId | world.Id | World identifier |
| channelId | channel.Id | Channel identifier |
| characterId | uint32 | Character identifier |
| type | string | `EXPIRED` |
| body.sourceId | int32 | Buff source identifier |
| body.duration | int32 | Buff duration |
| body.changes | []StatChange | Stat changes from buff |
| body.createdAt | time.Time | When buff was created |
| body.expiresAt | time.Time | When buff expired |

**StatChange** (`kafka/message/buff/kafka.go`)

| Field | Type | Description |
|-------|------|-------------|
| type | string | Stat type (e.g., `HOLY_SYMBOL`, `MESO_UP`) |
| amount | int32 | Stat amount |

**Buff-to-Rate Mappings** (`kafka/message/buff/kafka.go`)

| Stat Type | Rate Type | Conversion |
|-----------|-----------|------------|
| HOLY_SYMBOL | exp | Additive: `1.0 + (amount / 100.0)` |
| MESO_UP | meso | Direct: `amount / 100.0` |

### World Rate Events

**WorldRateEvent** (`kafka/message/rate/kafka.go`)

| Field | Type | Description |
|-------|------|-------------|
| type | string | `RATE_CHANGED` |
| worldId | world.Id | World identifier |
| rateType | RateType | `exp`, `meso`, `item_drop`, or `quest_exp` |
| multiplier | float64 | New rate multiplier |

### Asset Status Events

**StatusEvent[CreatedStatusEventBody]** (`kafka/message/asset/kafka.go`)

| Field | Type | Description |
|-------|------|-------------|
| transactionId | uuid.UUID | Transaction identifier |
| characterId | uint32 | Character identifier |
| compartmentId | uuid.UUID | Compartment identifier |
| assetId | uint32 | Asset identifier |
| templateId | uint32 | Item template identifier |
| slot | int16 | Inventory slot |
| type | string | `CREATED` |
| body.expiration | time.Time | Asset expiration |
| body.createdAt | time.Time | Asset creation time |
| body.quantity | uint32 | Asset quantity |

**StatusEvent[AcceptedStatusEventBody]** (`kafka/message/asset/kafka.go`)

| Field | Type | Description |
|-------|------|-------------|
| transactionId | uuid.UUID | Transaction identifier |
| characterId | uint32 | Character identifier |
| compartmentId | uuid.UUID | Compartment identifier |
| assetId | uint32 | Asset identifier |
| templateId | uint32 | Item template identifier |
| slot | int16 | Inventory slot |
| type | string | `ACCEPTED` |
| body.expiration | time.Time | Asset expiration |
| body.createdAt | time.Time | Asset creation time |
| body.quantity | uint32 | Asset quantity |

**StatusEvent[DeletedStatusEventBody]** (`kafka/message/asset/kafka.go`)

| Field | Type | Description |
|-------|------|-------------|
| transactionId | uuid.UUID | Transaction identifier |
| characterId | uint32 | Character identifier |
| compartmentId | uuid.UUID | Compartment identifier |
| assetId | uint32 | Asset identifier |
| templateId | uint32 | Item template identifier |
| slot | int16 | Inventory slot |
| type | string | `DELETED` |

**StatusEvent[ReleasedStatusEventBody]** (`kafka/message/asset/kafka.go`)

| Field | Type | Description |
|-------|------|-------------|
| transactionId | uuid.UUID | Transaction identifier |
| characterId | uint32 | Character identifier |
| compartmentId | uuid.UUID | Compartment identifier |
| assetId | uint32 | Asset identifier |
| templateId | uint32 | Item template identifier |
| slot | int16 | Inventory slot |
| type | string | `RELEASED` |

**StatusEvent[MovedStatusEventBody]** (`kafka/message/asset/kafka.go`)

| Field | Type | Description |
|-------|------|-------------|
| transactionId | uuid.UUID | Transaction identifier |
| characterId | uint32 | Character identifier |
| compartmentId | uuid.UUID | Compartment identifier |
| assetId | uint32 | Asset identifier |
| templateId | uint32 | Item template identifier |
| slot | int16 | New inventory slot |
| type | string | `MOVED` |
| body.oldSlot | int16 | Previous inventory slot |
| body.createdAt | time.Time | Asset creation time |

### Character Status Events

**StatusEvent[StatusEventMapChangedBody]** (`kafka/message/character/kafka.go`)

| Field | Type | Description |
|-------|------|-------------|
| transactionId | uuid.UUID | Transaction identifier |
| worldId | world.Id | World identifier |
| characterId | uint32 | Character identifier |
| type | string | `MAP_CHANGED` |
| body.channelId | channel.Id | Channel identifier |
| body.oldMapId | map.Id | Previous map identifier |
| body.oldInstance | uuid.UUID | Previous map instance |
| body.targetMapId | map.Id | Target map identifier |
| body.targetInstance | uuid.UUID | Target map instance |
| body.targetPortalId | uint32 | Target portal identifier |

## Transaction Semantics

None. This service does not produce Kafka messages.
