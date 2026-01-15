# Kafka Integration

## Topics Consumed

| Topic Environment Variable | Consumer Group | Description |
|---------------------------|----------------|-------------|
| COMMAND_TOPIC_CHARACTER_BUFF | Buff Service | Buff commands |

## Topics Produced

| Topic Environment Variable | Description |
|---------------------------|-------------|
| EVENT_TOPIC_CHARACTER_BUFF_STATUS | Buff status events |

## Message Types

### Commands

#### Command

Consumed from COMMAND_TOPIC_CHARACTER_BUFF. Generic envelope with typed body.

| Field | Type |
|-------|------|
| WorldId | byte |
| ChannelId | byte |
| CharacterId | uint32 |
| Type | string |
| Body | varies |

##### Command Types

| Type | Body Type | Description |
|------|-----------|-------------|
| APPLY | ApplyCommandBody | Apply buff to character |
| CANCEL | CancelCommandBody | Cancel buff on character |

##### ApplyCommandBody

| Field | Type |
|-------|------|
| FromId | uint32 |
| SourceId | int32 |
| Duration | int32 |
| Changes | []StatChange |

##### CancelCommandBody

| Field | Type |
|-------|------|
| SourceId | int32 |

##### StatChange

| Field | Type |
|-------|------|
| Type | string |
| Amount | int32 |

### Events

#### StatusEvent

Produced to EVENT_TOPIC_CHARACTER_BUFF_STATUS. Generic envelope with typed body.

| Field | Type |
|-------|------|
| WorldId | byte |
| CharacterId | uint32 |
| Type | string |
| Body | varies |

##### Status Event Types

| Type | Body Type | Description |
|------|-----------|-------------|
| APPLIED | AppliedStatusEventBody | Buff applied |
| EXPIRED | ExpiredStatusEventBody | Buff expired or cancelled |

##### AppliedStatusEventBody

| Field | Type |
|-------|------|
| FromId | uint32 |
| SourceId | int32 |
| Duration | int32 |
| Changes | []StatChange |
| CreatedAt | time.Time |
| ExpiresAt | time.Time |

##### ExpiredStatusEventBody

| Field | Type |
|-------|------|
| SourceId | int32 |
| Duration | int32 |
| Changes | []StatChange |
| CreatedAt | time.Time |
| ExpiresAt | time.Time |

## Transaction Semantics

- Commands are processed with persistent configuration
- Headers required: span (tracing), tenant
