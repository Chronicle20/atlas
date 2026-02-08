# Kafka Integration

## Topics Consumed

| Topic | Environment Variable | Description |
|-------|---------------------|-------------|
| Account Status | EVENT_TOPIC_ACCOUNT_STATUS | Account login/logout status events |
| Account Session Status | EVENT_TOPIC_ACCOUNT_SESSION_STATUS | Account session lifecycle events |
| Seed Status | EVENT_TOPIC_SEED_STATUS | Character creation completion events |

## Topics Produced

| Topic | Environment Variable | Description |
|-------|---------------------|-------------|
| Account Session Command | COMMAND_TOPIC_ACCOUNT_SESSION | Account session commands |
| Session Status | EVENT_TOPIC_SESSION_STATUS | Session lifecycle events |

## Message Types

### Commands Produced

#### Account Session Commands

Topic: COMMAND_TOPIC_ACCOUNT_SESSION

**Command Structure**

```go
type Command[E any] struct {
    SessionId uuid.UUID `json:"sessionId"`
    AccountId uint32    `json:"accountId"`
    Issuer    string    `json:"author"`
    Type      string    `json:"type"`
    Body      E         `json:"body"`
}
```

**Create Command**

Type: `CREATE`

Issuer: `LOGIN`

```go
type CreateCommandBody struct {
    AccountName string `json:"accountName"`
    Password    string `json:"password"`
    IPAddress   string `json:"ipAddress"`
    HWID        string `json:"hwid"`
}
```

**Progress State Command**

Type: `PROGRESS_STATE`

Issuer: `LOGIN`

```go
type ProgressStateCommandBody struct {
    State  uint8       `json:"state"`
    Params interface{} `json:"params"`
}
```

**Logout Command**

Type: `LOGOUT`

Issuer: `LOGIN`

```go
type LogoutCommandBody struct {
}
```

### Events Produced

#### Session Status Events

Topic: EVENT_TOPIC_SESSION_STATUS

```go
type StatusEvent struct {
    SessionId   uuid.UUID  `json:"sessionId"`
    AccountId   uint32     `json:"accountId"`
    CharacterId uint32     `json:"characterId"`
    WorldId     world.Id   `json:"worldId"`
    ChannelId   channel.Id `json:"channelId"`
    Issuer      string     `json:"issuer"`
    Type        string     `json:"type"`
}
```

**Event Types**

| Type | Description |
|------|-------------|
| CREATED | Session created |
| DESTROYED | Session destroyed |

Issuer: `LOGIN`

### Events Consumed

#### Account Status Events

Topic: EVENT_TOPIC_ACCOUNT_STATUS

```go
type StatusEvent struct {
    AccountId uint32 `json:"account_id"`
    Name      string `json:"name"`
    Status    string `json:"status"`
}
```

**Status Values**

| Status | Description |
|--------|-------------|
| LOGGED_IN | Account logged in |
| LOGGED_OUT | Account logged out |

#### Account Session Status Events

Topic: EVENT_TOPIC_ACCOUNT_SESSION_STATUS

```go
type StatusEvent[E any] struct {
    SessionId uuid.UUID `json:"sessionId"`
    AccountId uint32    `json:"accountId"`
    Type      string    `json:"type"`
    Body      E         `json:"body"`
}
```

**Event Types**

| Type | Body Type | Description |
|------|-----------|-------------|
| CREATED | CreatedStatusEventBody | Session created successfully |
| STATE_CHANGED | StateChangedEventBody | Session state changed |
| REQUEST_LICENSE_AGREEMENT | any | License agreement required |
| ERROR | ErrorStatusEventBody | Session error |

**CreatedStatusEventBody**

```go
type CreatedStatusEventBody struct {
}
```

**StateChangedEventBody**

```go
type StateChangedEventBody[E any] struct {
    State  uint8 `json:"state"`
    Params E     `json:"params"`
}
```

**ErrorStatusEventBody**

```go
type ErrorStatusEventBody struct {
    Code   string `json:"code"`
    Reason byte   `json:"reason"`
    Until  uint64 `json:"until"`
}
```

**Error Codes**

| Code | Description |
|------|-------------|
| SYSTEM_ERROR | System error |
| NOT_REGISTERED | Account not registered |
| DELETED_OR_BLOCKED | Account deleted or blocked |
| ALREADY_LOGGED_IN | Account already logged in |
| INCORRECT_PASSWORD | Incorrect password |
| TOO_MANY_ATTEMPTS | Too many login attempts |

#### Seed Status Events

Topic: EVENT_TOPIC_SEED_STATUS

```go
type StatusEvent[E any] struct {
    AccountId uint32 `json:"accountId"`
    Type      string `json:"type"`
    Body      E      `json:"body"`
}
```

**Event Types**

| Type | Body Type | Description |
|------|-----------|-------------|
| CREATED | CreatedStatusEventBody | Character created |

**CreatedStatusEventBody**

```go
type CreatedStatusEventBody struct {
    CharacterId uint32 `json:"characterId"`
}
```

## Transaction Semantics

- All messages include tenant header via TenantHeaderDecorator.
- All messages include span header via SpanHeaderDecorator for distributed tracing.
- Message keys are based on account ID for partition ordering.
- Consumer group ID follows pattern: `Login Service - {service-id}`.
