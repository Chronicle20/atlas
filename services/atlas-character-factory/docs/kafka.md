# Character Factory Kafka Integration

## Topics Consumed

| Topic Environment Variable              | Direction | Description                                         |
|------------------------------------------|-----------|------------------------------------------------------|
| EVENT_TOPIC_SAGA_STATUS                 | Event     | Saga status events (completed and failed)            |
| EVENT_TOPIC_CONFIGURATION_TENANT_STATUS | Event     | Tenant configuration projection (config-status outbox envelopes) |

## Topics Produced

| Topic Environment Variable | Direction | Description                   |
|----------------------------|-----------|-------------------------------|
| COMMAND_TOPIC_SAGA         | Command   | Saga commands to orchestrator |
| EVENT_TOPIC_SEED_STATUS    | Event     | Seed completion status events |

## Message Types

### Consumed Messages

#### StatusEvent[StatusEventCompletedBody] (Saga Status)

| Field         | Type      |
|---------------|-----------|
| TransactionId | uuid.UUID |
| Type          | string    |
| Body          | E         |

StatusEventCompletedBody:

| Field    | Type           |
|----------|----------------|
| SagaType | string         |
| Results  | map[string]any |

Only events with `Type` = `"COMPLETED"` and `SagaType` = `"character_creation"` are processed. The handler extracts `accountId` and `characterId` from `Results` and emits a seed completion event.

#### StatusEvent[StatusEventFailedBody] (Saga Status)

| Field         | Type      |
|---------------|-----------|
| TransactionId | uuid.UUID |
| Type          | string    |
| Body          | E         |

StatusEventFailedBody:

| Field       | Type   |
|-------------|--------|
| Reason      | string |
| FailedStep  | string |
| CharacterId | uint32 |
| AccountId   | uint32 |
| SagaType    | string |
| ErrorCode   | string |

Only events with `Type` = `"FAILED"` and `SagaType` = `"character_creation"` are processed. The handler emits a seed FAILED event keyed by `AccountId`; events with `AccountId` = 0 are dropped.

#### TenantEnvelope (Tenant Configuration Projection)

| Field         | Type           |
|---------------|----------------|
| SchemaVersion | int            |
| Id            | string         |
| Config        | json.RawMessage |
| EmittedAt     | string         |

A nil message value is a log-compaction tombstone (key `tenant:<uuid>`) and removes the tenant from the in-memory snapshot instead of being decoded as an envelope. Envelopes with `SchemaVersion` greater than the supported version (1) are skipped.

### Produced Messages

#### Saga (Command)

| Field         | Type        |
|---------------|-------------|
| TransactionId | uuid.UUID   |
| SagaType      | Type        |
| InitiatedBy   | string      |
| Steps         | []Step[any] |

#### StatusEvent[CreatedStatusEventBody] (Seed Status)

| Field     | Type   |
|-----------|--------|
| AccountId | uint32 |
| Type      | string |
| Body      | E      |

CreatedStatusEventBody:

| Field       | Type   |
|-------------|--------|
| CharacterId | uint32 |

#### StatusEvent[FailedStatusEventBody] (Seed Status)

| Field     | Type   |
|-----------|--------|
| AccountId | uint32 |
| Type      | string |
| Body      | E      |

FailedStatusEventBody:

| Field  | Type   |
|--------|--------|
| Reason | string |

## Transaction Semantics

- Saga commands are keyed by transaction ID for ordering
- Seed status events are keyed by account ID
- Saga COMPLETED events for `CharacterCreation` type trigger a seed CREATED event emission
- Saga FAILED events for `CharacterCreation` type trigger a seed FAILED event emission
- Required headers: tenant headers and span headers (set via producer decorators)
- Consumer group for `EVENT_TOPIC_SAGA_STATUS`: `"Character Factory Service"`
- Consumer group for the tenant configuration projection: a per-process id (`"Character Factory Service - projection - <uuid>"`) so the full compacted log replays from FirstOffset on every container start
- Consumer header parsers: SpanHeaderParser, TenantHeaderParser
