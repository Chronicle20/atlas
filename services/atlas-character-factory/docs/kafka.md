# Character Factory Kafka Integration

## Topics Consumed

| Topic Environment Variable     | Direction | Description                              |
|--------------------------------|-----------|------------------------------------------|
| EVENT_TOPIC_CHARACTER_STATUS   | Event     | Character status events (created)        |
| EVENT_TOPIC_SAGA_STATUS        | Event     | Saga status events (completed)           |

## Topics Produced

| Topic Environment Variable | Direction | Description                              |
|----------------------------|-----------|------------------------------------------|
| COMMAND_TOPIC_SAGA         | Command   | Saga commands to orchestrator            |
| EVENT_TOPIC_SEED_STATUS    | Event     | Seed completion status events            |

## Message Types

### Consumed Messages

#### StatusEvent[StatusEventCreatedBody] (Character Status)

| Field       | Type   |
|-------------|--------|
| CharacterId | uint32 |
| Type        | string |
| WorldId     | byte   |
| Body        | E      |

StatusEventCreatedBody:

| Field | Type   |
|-------|--------|
| Name  | string |

#### StatusEvent[StatusEventCompletedBody] (Saga Status)

| Field         | Type      |
|---------------|-----------|
| TransactionId | uuid.UUID |
| Type          | string    |
| Body          | E         |

StatusEventCompletedBody: Empty struct.

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
- Character created events trigger follow-up saga creation
- Saga completed events tracked for both character creation and follow-up sagas
- Seed completion event emitted when both sagas complete
