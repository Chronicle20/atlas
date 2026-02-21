# Character Factory Kafka Integration

## Topics Consumed

| Topic Environment Variable | Direction | Description                    |
|----------------------------|-----------|--------------------------------|
| EVENT_TOPIC_SAGA_STATUS    | Event     | Saga status events (completed) |

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

## Transaction Semantics

- Saga commands are keyed by transaction ID for ordering
- Seed status events are keyed by account ID
- Saga completed events for `CharacterCreation` type trigger seed completion event emission
- Required headers: tenant headers and span headers (set via producer decorators)
- Consumer group: `"Character Factory Service"`
- Consumer header parsers: SpanHeaderParser, TenantHeaderParser
