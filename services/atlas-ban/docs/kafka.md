# Kafka Integration

## Topics Consumed

| Topic Environment Variable | Consumer Group | Description |
|---------------------------|----------------|-------------|
| COMMAND_TOPIC_BAN | Ban Service | Ban commands (create, delete) |
| EVENT_TOPIC_ACCOUNT_SESSION_STATUS | Ban Service | Account session status events |

## Topics Produced

| Topic Environment Variable | Description |
|---------------------------|-------------|
| EVENT_TOPIC_BAN_STATUS | Ban status events |

## Message Types

### Commands

#### Command[E]

Consumed from COMMAND_TOPIC_BAN. Generic envelope with typed body.

| Field | Type |
|-------|------|
| Type | string |
| Body | varies |

##### Ban Command Types

| Type | Body Type | Description |
|------|-----------|-------------|
| CREATE | CreateCommandBody | Ban creation |
| DELETE | DeleteCommandBody | Ban deletion |

##### CreateCommandBody

| Field | Type |
|-------|------|
| BanType | byte |
| Value | string |
| Reason | string |
| ReasonCode | byte |
| Permanent | bool |
| ExpiresAt | int64 |
| IssuedBy | string |

##### DeleteCommandBody

| Field | Type |
|-------|------|
| BanId | uint32 |

### Events

#### StatusEvent

Produced to EVENT_TOPIC_BAN_STATUS.

| Field | Type |
|-------|------|
| BanId | uint32 |
| Status | string |

##### Status Event Types

| Status | Description |
|--------|-------------|
| CREATED | Ban created |
| DELETED | Ban deleted |

### Consumed Events

#### SessionStatusEvent[E]

Consumed from EVENT_TOPIC_ACCOUNT_SESSION_STATUS. Generic envelope with typed body.

| Field | Type |
|-------|------|
| SessionId | uuid.UUID |
| AccountId | uint32 |
| AccountName | string |
| Type | string |
| Body | varies |

##### Consumed Session Status Event Types

| Type | Body Type | Description |
|------|-----------|-------------|
| CREATED | CreatedSessionStatusEventBody | Successful login |
| ERROR | ErrorSessionStatusEventBody | Failed login |

##### CreatedSessionStatusEventBody

| Field | Type |
|-------|------|
| IPAddress | string |
| HWID | string |

##### ErrorSessionStatusEventBody

| Field | Type |
|-------|------|
| Code | string |
| Reason | byte |
| Until | uint64 |
| IPAddress | string |
| HWID | string |

## Transaction Semantics

- Commands are processed with persistent configuration
- Events are buffered and emitted after successful command processing
- Headers required: span (tracing), tenant
