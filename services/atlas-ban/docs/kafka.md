# Kafka Integration

## Topics Consumed

| Topic Environment Variable | Consumer Group | Description |
|---------------------------|----------------|-------------|
| COMMAND_TOPIC_BAN | Ban Service | Ban commands (create, delete) |
| EVENT_TOPIC_ACCOUNT_SESSION_STATUS | Ban Service | Account session status events |
| COMMAND_TOPIC_REPORT | Ban Service | Report commands (create) |

## Topics Produced

| Topic Environment Variable | Description |
|---------------------------|-------------|
| EVENT_TOPIC_BAN_STATUS | Ban status events |
| EVENT_TOPIC_REPORT_STATUS | Report status events |

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
| ExpiresAt | time.Time |
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
| EXPIRED | Ban expired |

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
| Until | time.Time |
| IPAddress | string |
| HWID | string |

### Report Commands

#### Command[E]

Consumed from COMMAND_TOPIC_REPORT. Generic envelope with typed body.

| Field | Type |
|-------|------|
| Type | string |
| Body | varies |

##### Report Command Types

| Type | Body Type | Description |
|------|-----------|-------------|
| CREATE | CreateCommandBody | Report creation |

##### CreateCommandBody

Accused identity is mechanism-dependent: claim and v95 sue supply
`AccusedName`; legacy sue (v83/v84/v87) supplies `AccusedId`. The consumer
resolves the missing half via atlas-character and rejects unresolvable
targets.

| Field | Type |
|-------|------|
| Kind | string (`sue`\|`claim`) |
| WorldId | world.Id |
| ChannelId | channel.Id |
| ReporterId | uint32 |
| AccusedId | uint32 |
| AccusedName | string |
| ReasonType | byte |
| Description | string |
| ChatClaim | bool |
| ChatLog | string |

### Report Events

#### StatusEvent

Produced to EVENT_TOPIC_REPORT_STATUS.

| Field | Type |
|-------|------|
| ReportId | uuid.UUID (uuid.Nil on ERROR) |
| Kind | string (`sue`\|`claim`) |
| WorldId | world.Id |
| ReporterId | uint32 |
| Status | string (`CREATED`\|`ERROR`) |
| ErrorCode | string (`NOT_FOUND`\|`INTERNAL`\|`QUOTA_EXCEEDED`, empty on CREATED) |
| HasRemaining | bool (set on `claim` CREATED only; zero-valued otherwise) |
| Remaining | int32 (set on `claim` CREATED only; zero-valued otherwise) |

## Transaction Semantics

- Commands are processed with persistent configuration
- Events are buffered and emitted after successful command processing
- Headers required: span (tracing), tenant
