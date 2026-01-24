# Note Domain

## Responsibility

Manages notes sent between characters. A note represents a message from one character to another with associated metadata.

## Core Models

### Model

Immutable domain model representing a note.

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Note identifier |
| characterId | uint32 | ID of the character who owns the note |
| senderId | uint32 | ID of the character who sent the note |
| message | string | Note content |
| timestamp | time.Time | When the note was created |
| flag | byte | Note flag |

### Builder

Constructs Model instances with validation.

## Invariants

- characterId is required (must be non-zero)
- senderId is required (must be non-zero)
- message is required (must be non-empty)

## Processors

### ProcessorImpl

Coordinates note operations with database persistence and Kafka event emission.

| Method | Description |
|--------|-------------|
| Create | Creates a note and buffers a status event |
| CreateAndEmit | Creates a note and emits a CREATED status event |
| Update | Updates a note and buffers a status event |
| UpdateAndEmit | Updates a note and emits an UPDATED status event |
| Delete | Deletes a note and buffers a status event |
| DeleteAndEmit | Deletes a note and emits a DELETED status event |
| DeleteAll | Deletes all notes for a character and buffers status events |
| DeleteAllAndEmit | Deletes all notes for a character and emits DELETED status events |
| Discard | Deletes multiple notes for a character by ID and buffers status events |
| DiscardAndEmit | Deletes multiple notes for a character by ID and emits DELETED status events |
| ByIdProvider | Retrieves a note by ID |
| ByCharacterProvider | Retrieves all notes for a character |
| InTenantProvider | Retrieves all notes in a tenant |
