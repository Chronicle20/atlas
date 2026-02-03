# Kafka Integration

## Topics Consumed

| Environment Variable | Direction | Description |
|---------------------|-----------|-------------|
| COMMAND_TOPIC_FAMILY | Command | Family operation commands |
| EVENT_TOPIC_CHARACTER_STATUS | Event | Character status events |

## Topics Produced

| Environment Variable | Direction | Description |
|---------------------|-----------|-------------|
| EVENT_TOPIC_FAMILY_STATUS | Event | Family relationship status events |
| EVENT_TOPIC_FAMILY_ERRORS | Event | Error events |
| EVENT_TOPIC_FAMILY_REPUTATION | Event | Reputation change events |

## Message Types

### Command Wrapper

```go
type Command[E any] struct {
    TransactionId uuid.UUID `json:"transactionId"`
    WorldId       byte      `json:"worldId"`
    CharacterId   uint32    `json:"characterId"`
    Type          string    `json:"type"`
    Body          E         `json:"body"`
}
```

### Event Wrapper

```go
type Event[E any] struct {
    WorldId     byte   `json:"worldId"`
    CharacterId uint32 `json:"characterId"`
    Type        string `json:"type"`
    Body        E      `json:"body"`
}
```

### Command Types

| Type | Body Struct | Description |
|------|-------------|-------------|
| ADD_JUNIOR | AddJuniorCommandBody | Add junior to family |
| REMOVE_MEMBER | RemoveMemberCommandBody | Remove member from family |
| BREAK_LINK | BreakLinkCommandBody | Break family link |
| AWARD_REP | AwardRepCommandBody | Award reputation |
| DEDUCT_REP | DeductRepCommandBody | Deduct reputation |

### Command Body Structures

#### AddJuniorCommandBody

| Field | Type | Description |
|-------|------|-------------|
| JuniorId | uint32 | Junior character ID |
| SeniorLevel | uint16 | Senior character level |
| JuniorLevel | uint16 | Junior character level |

#### RemoveMemberCommandBody

| Field | Type | Description |
|-------|------|-------------|
| TargetId | uint32 | Target character ID |
| Reason | string | Removal reason |

#### BreakLinkCommandBody

| Field | Type | Description |
|-------|------|-------------|
| Reason | string | Break reason |

#### AwardRepCommandBody

| Field | Type | Description |
|-------|------|-------------|
| Amount | uint32 | Reputation amount |
| Source | string | Source of reputation |
| Timestamp | time.Time | Event timestamp |

#### DeductRepCommandBody

| Field | Type | Description |
|-------|------|-------------|
| Amount | uint32 | Reputation amount |
| Reason | string | Deduction reason |

### Event Types

| Type | Body Struct | Topic |
|------|-------------|-------|
| LINK_CREATED | LinkCreatedEventBody | EVENT_TOPIC_FAMILY_STATUS |
| LINK_BROKEN | LinkBrokenEventBody | EVENT_TOPIC_FAMILY_STATUS |
| TREE_DISSOLVED | TreeDissolvedEventBody | EVENT_TOPIC_FAMILY_STATUS |
| REP_GAINED | RepGainedEventBody | EVENT_TOPIC_FAMILY_REPUTATION |
| REP_REDEEMED | RepRedeemedEventBody | EVENT_TOPIC_FAMILY_REPUTATION |
| REP_PENALIZED | RepPenalizedEventBody | EVENT_TOPIC_FAMILY_REPUTATION |
| REP_CAPPED | RepCappedEventBody | EVENT_TOPIC_FAMILY_REPUTATION |
| REP_RESET | RepResetEventBody | EVENT_TOPIC_FAMILY_REPUTATION |
| REP_ERROR | RepErrorEventBody | EVENT_TOPIC_FAMILY_ERRORS |
| LINK_ERROR | LinkErrorEventBody | EVENT_TOPIC_FAMILY_ERRORS |

### Event Body Structures

#### LinkCreatedEventBody

| Field | Type | Description |
|-------|------|-------------|
| SeniorId | uint32 | Senior character ID |
| JuniorId | uint32 | Junior character ID |
| Timestamp | time.Time | Event timestamp |

#### LinkBrokenEventBody

| Field | Type | Description |
|-------|------|-------------|
| SeniorId | uint32 | Senior character ID |
| JuniorId | uint32 | Junior character ID |
| Reason | string | Break reason |
| Timestamp | time.Time | Event timestamp |

#### TreeDissolvedEventBody

| Field | Type | Description |
|-------|------|-------------|
| SeniorId | uint32 | Senior character ID |
| AffectedIds | []uint32 | Affected character IDs |
| Reason | string | Dissolution reason |
| Timestamp | time.Time | Event timestamp |

#### RepGainedEventBody

| Field | Type | Description |
|-------|------|-------------|
| RepGained | uint32 | Reputation amount gained |
| DailyRep | uint32 | Current daily reputation |
| Source | string | Reputation source |
| Timestamp | time.Time | Event timestamp |

#### RepRedeemedEventBody

| Field | Type | Description |
|-------|------|-------------|
| RepRedeemed | uint32 | Reputation amount redeemed |
| Reason | string | Redemption reason |
| Timestamp | time.Time | Event timestamp |

#### RepPenalizedEventBody

| Field | Type | Description |
|-------|------|-------------|
| RepLost | uint32 | Reputation amount lost |
| Reason | string | Penalty reason |
| Timestamp | time.Time | Event timestamp |

#### RepCappedEventBody

| Field | Type | Description |
|-------|------|-------------|
| AttemptedAmount | uint32 | Attempted reputation amount |
| DailyRep | uint32 | Current daily reputation |
| Source | string | Reputation source |
| Timestamp | time.Time | Event timestamp |

#### RepResetEventBody

| Field | Type | Description |
|-------|------|-------------|
| PreviousDailyRep | uint32 | Previous daily reputation |
| Timestamp | time.Time | Event timestamp |

#### RepErrorEventBody

| Field | Type | Description |
|-------|------|-------------|
| ErrorCode | string | Error code |
| ErrorMessage | string | Error message |
| Amount | uint32 | Related amount |
| Timestamp | time.Time | Event timestamp |

#### LinkErrorEventBody

| Field | Type | Description |
|-------|------|-------------|
| SeniorId | uint32 | Senior character ID |
| JuniorId | uint32 | Junior character ID |
| ErrorCode | string | Error code |
| ErrorMessage | string | Error message |
| Timestamp | time.Time | Event timestamp |

## Consumed Events

### Character Status Events (EVENT_TOPIC_CHARACTER_STATUS)

#### StatusEvent Wrapper

```go
type StatusEvent[E any] struct {
    WorldId     byte   `json:"worldId"`
    CharacterId uint32 `json:"characterId"`
    Type        string `json:"type"`
    Body        E      `json:"body"`
}
```

#### Status Event Types

| Type | Body Struct | Description |
|------|-------------|-------------|
| DELETED | StatusEventDeletedBody | Character deleted event |

#### StatusEventDeletedBody

Empty body struct. Triggers family member removal.

## Transaction Semantics

- Commands require transactionId header for idempotency
- Messages partitioned by characterId for ordering
- Consumer groups: family_command, character_status
- Headers parsed: SpanHeaderParser, TenantHeaderParser
