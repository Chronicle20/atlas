# Kafka Integration

## Topics Consumed

| Topic | Environment Variable | Direction |
|-------|---------------------|-----------|
| Invite Commands | COMMAND_TOPIC_INVITE | Command |
| Character Status | EVENT_TOPIC_CHARACTER_STATUS | Event |

## Topics Produced

| Topic | Environment Variable | Direction |
|-------|---------------------|-----------|
| Invite Status | EVENT_TOPIC_INVITE_STATUS | Event |

## Message Types

### Command

Consumed from COMMAND_TOPIC_INVITE.

| Field | Type | Description |
|-------|------|-------------|
| transactionId | uuid.UUID | Transaction identifier |
| worldId | byte | World identifier |
| inviteType | string | Invite category |
| type | string | Command type |
| body | varies | Command-specific payload |

#### Command Types

| Type | Body Struct | Fields |
|------|-------------|--------|
| CREATE | CreateCommandBody | originatorId, targetId, referenceId |
| ACCEPT | AcceptCommandBody | targetId, referenceId |
| REJECT | RejectCommandBody | targetId, originatorId |

### StatusEvent

Produced to EVENT_TOPIC_INVITE_STATUS.

| Field | Type | Description |
|-------|------|-------------|
| transactionId | uuid.UUID | Transaction identifier |
| worldId | byte | World identifier |
| inviteType | string | Invite category |
| referenceId | uint32 | Reference entity identifier |
| type | string | Event type |
| body | varies | Event-specific payload |

#### Event Types

| Type | Body Struct | Fields |
|------|-------------|--------|
| CREATED | CreatedEventBody | originatorId, targetId |
| ACCEPTED | AcceptedEventBody | originatorId, targetId |
| REJECTED | RejectedEventBody | originatorId, targetId |

### Character StatusEvent

Consumed from EVENT_TOPIC_CHARACTER_STATUS.

| Field | Type | Description |
|-------|------|-------------|
| transactionId | uuid.UUID | Transaction identifier |
| worldId | byte | World identifier |
| characterId | uint32 | Character identifier |
| type | string | Status event type |
| body | varies | Event-specific payload |

#### Handled Event Types

| Type | Body Struct | Effect |
|------|-------------|--------|
| DELETED | StatusEventDeletedBody | Removes all invites for the character |

## Transaction Semantics

- All command messages include a transactionId for correlation.
- Status events are produced with the same transactionId as the originating command.
- Messages are keyed by referenceId for partition ordering.
- Required headers: tenant headers parsed via TenantHeaderParser, span headers parsed via SpanHeaderParser.
