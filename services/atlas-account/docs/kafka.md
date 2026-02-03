# Kafka Integration

## Topics Consumed

| Topic Environment Variable | Consumer Group | Description |
|---------------------------|----------------|-------------|
| COMMAND_TOPIC_ACCOUNT | Account Service | Account commands (create, delete) |
| COMMAND_TOPIC_ACCOUNT_SESSION | Account Service | Session commands |

## Topics Produced

| Topic Environment Variable | Description |
|---------------------------|-------------|
| EVENT_TOPIC_ACCOUNT_STATUS | Account status events |
| EVENT_TOPIC_ACCOUNT_SESSION_STATUS | Session status events |

## Message Types

### Commands

#### Command[E]

Consumed from COMMAND_TOPIC_ACCOUNT. Generic envelope with typed body.

| Field | Type |
|-------|------|
| Type | string |
| Body | varies |

##### Account Command Types

| Type | Body Type | Description |
|------|-----------|-------------|
| CREATE | CreateCommandBody | Account creation |
| DELETE | DeleteCommandBody | Account deletion |

##### CreateCommandBody

| Field | Type |
|-------|------|
| Name | string |
| Password | string |

##### DeleteCommandBody

| Field | Type |
|-------|------|
| AccountId | uint32 |

#### SessionCommand[E]

Consumed from COMMAND_TOPIC_ACCOUNT_SESSION. Generic envelope with typed body.

| Field | Type |
|-------|------|
| SessionId | uuid.UUID |
| AccountId | uint32 |
| Issuer | string |
| Type | string |
| Body | varies |

##### Session Command Types

| Type | Body Type | Description |
|------|-----------|-------------|
| CREATE | CreateSessionCommandBody | Login attempt |
| PROGRESS_STATE | ProgressStateSessionCommandBody | State transition |
| LOGOUT | LogoutSessionCommandBody | Logout request |

##### CreateSessionCommandBody

| Field | Type |
|-------|------|
| AccountName | string |
| Password | string |
| IPAddress | string |

##### ProgressStateSessionCommandBody

| Field | Type |
|-------|------|
| State | uint8 |
| Params | interface{} |

##### LogoutSessionCommandBody

Empty body.

##### Session Command Issuers

| Value | Description |
|-------|-------------|
| INTERNAL | Internal service |
| LOGIN | Login service |
| CHANNEL | Channel service |

### Events

#### StatusEvent

Produced to EVENT_TOPIC_ACCOUNT_STATUS.

| Field | Type |
|-------|------|
| AccountId | uint32 |
| Name | string |
| Status | string |

##### Status Event Types

| Status | Description |
|--------|-------------|
| CREATED | Account created |
| LOGGED_IN | Account logged in |
| LOGGED_OUT | Account logged out |
| DELETED | Account deleted |

#### SessionStatusEvent[E]

Produced to EVENT_TOPIC_ACCOUNT_SESSION_STATUS. Generic envelope with typed body.

| Field | Type |
|-------|------|
| SessionId | uuid.UUID |
| AccountId | uint32 |
| Type | string |
| Body | varies |

##### Session Status Event Types

| Type | Body Type | Description |
|------|-----------|-------------|
| CREATED | CreatedSessionStatusEventBody | Session created |
| STATE_CHANGED | StateChangedSessionStatusEventBody | State transition completed |
| REQUEST_LICENSE_AGREEMENT | none | License agreement required |
| ERROR | ErrorSessionStatusEventBody | Error occurred |

##### CreatedSessionStatusEventBody

Empty body.

##### StateChangedSessionStatusEventBody

| Field | Type |
|-------|------|
| State | uint8 |
| Params | interface{} |

##### ErrorSessionStatusEventBody

| Field | Type |
|-------|------|
| Code | string |
| Reason | byte |
| Until | uint64 |

## Transaction Semantics

- Commands are processed with persistent configuration
- Events are buffered and emitted after successful command processing
- Headers required: span (tracing), tenant
