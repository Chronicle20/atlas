# Kafka Integration

## Topics Consumed

| Topic Environment Variable     | Consumer Group | Description                |
|-------------------------------|----------------|----------------------------|
| EVENT_TOPIC_CHARACTER_STATUS  | Key Service    | Character status events    |

## Topics Produced

None.

## Message Types

### StatusEvent

Character status event consumed from the character status topic.

| Field         | Type      | Description                     |
|---------------|-----------|---------------------------------|
| transactionId | uuid.UUID | Transaction identifier          |
| characterId   | uint32    | Character identifier            |
| type          | string    | Event type (CREATED, DELETED)   |
| worldId       | world.Id  | World identifier                |
| body          | object    | Event-specific payload          |

### CreatedStatusBody

Body for CREATED events.

| Field | Type   | Description    |
|-------|--------|----------------|
| name  | string | Character name |

### DeletedStatusEventBody

Body for DELETED events. Empty structure.

## Transaction Semantics

- Messages are processed with tenant context parsed from headers.
- Span headers are parsed for tracing propagation.
- CREATED events trigger creation of default key bindings.
- DELETED events trigger deletion of all key bindings for the character.
