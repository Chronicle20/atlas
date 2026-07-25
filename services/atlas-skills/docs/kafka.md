# Kafka

## Topics Consumed

| Environment Variable | Topic Purpose |
|---------------------|---------------|
| `COMMAND_TOPIC_SKILL` | Skill commands (create, update, set cooldown, delete, transfer SP) |
| `COMMAND_TOPIC_SKILL_MACRO` | Macro update commands |
| `EVENT_TOPIC_CHARACTER_STATUS` | Character status events (logout, deleted) |

## Topics Produced

| Environment Variable | Topic Purpose |
|---------------------|---------------|
| `EVENT_TOPIC_SKILL_STATUS` | Skill status events (created, updated, deleted, cooldown applied, cooldown expired, SP transferred, error) |
| `STATUS_EVENT_TOPIC_SKILL_MACRO` | Macro status events (updated) |

## Message Types

### Skill Commands

#### Command Structure

```json
{
  "transactionId": "uuid",
  "worldId": "byte",
  "characterId": "uint32",
  "type": "string",
  "body": {}
}
```

#### REQUEST_CREATE

| Field | Type | Description |
|-------|------|-------------|
| skillId | uint32 | Skill identifier |
| level | byte | Skill level |
| masterLevel | byte | Master level |
| expiration | time.Time | Expiration timestamp |

#### REQUEST_UPDATE

| Field | Type | Description |
|-------|------|-------------|
| skillId | uint32 | Skill identifier |
| level | byte | Skill level |
| masterLevel | byte | Master level |
| expiration | time.Time | Expiration timestamp |

#### SET_COOLDOWN

| Field | Type | Description |
|-------|------|-------------|
| skillId | uint32 | Skill identifier |
| cooldown | uint32 | Cooldown duration in seconds |

#### REQUEST_DELETE

Saga-correlated command used by the orchestrator's character-creation reverse-walk compensator. Idempotent on a missing skill row.

| Field | Type | Description |
|-------|------|-------------|
| skillId | uint32 | Skill identifier |

#### TRANSFER_SP

| Field | Type | Description |
|-------|------|-------------|
| jobId | job.Id | Character's job identifier |
| fromSkillId | uint32 | Source skill identifier |
| toSkillId | uint32 | Target skill identifier |
| itemTier | byte | Job advancement tier encoded on the consumed SP Reset item |
| targetMaxLevel | byte | Level cap for the target skill |

### Skill Status Events

#### StatusEvent Structure

```json
{
  "transactionId": "uuid",
  "worldId": "byte",
  "characterId": "uint32",
  "skillId": "uint32",
  "type": "string",
  "body": {}
}
```

#### CREATED

| Field | Type | Description |
|-------|------|-------------|
| level | byte | Skill level |
| masterLevel | byte | Master level |
| expiration | time.Time | Expiration timestamp |

#### UPDATED

| Field | Type | Description |
|-------|------|-------------|
| level | byte | Skill level |
| masterLevel | byte | Master level |
| expiration | time.Time | Expiration timestamp |

#### COOLDOWN_APPLIED

| Field | Type | Description |
|-------|------|-------------|
| cooldownExpiresAt | time.Time | Cooldown expiration timestamp |

#### COOLDOWN_EXPIRED

No body fields.

#### DELETED

Emitted once a saga-compensation REQUEST_DELETE has been processed, whether or not a matching row existed.

No body fields.

#### SP_TRANSFERRED

Saga-completion signal for a successful TRANSFER_SP. The envelope `skillId` carries the target (to) skill.

| Field | Type | Description |
|-------|------|-------------|
| fromSkillId | uint32 | Source skill identifier |
| fromLevel | byte | Source skill level after the transfer |
| toLevel | byte | Target skill level after the transfer |

#### ERROR

Emitted for a rejected TRANSFER_SP.

| Field | Type | Description |
|-------|------|-------------|
| error | string | One of `SKILL_AT_ZERO`, `SKILL_AT_CAP`, `WRONG_TIER`, `INVALID_TARGET` |
| detail | string | The offending skill id, as a string |

### Macro Commands

#### Command Structure

```json
{
  "transactionId": "uuid",
  "worldId": "byte",
  "characterId": "uint32",
  "type": "string",
  "body": {}
}
```

#### UPDATE

| Field | Type | Description |
|-------|------|-------------|
| macros | []MacroBody | Array of macro configurations |

##### MacroBody

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Macro identifier |
| name | string | Macro name |
| shout | bool | Shout flag |
| skillId1 | uint32 | First skill ID |
| skillId2 | uint32 | Second skill ID |
| skillId3 | uint32 | Third skill ID |

### Macro Status Events

#### StatusEvent Structure

```json
{
  "transactionId": "uuid",
  "worldId": "byte",
  "characterId": "uint32",
  "type": "string",
  "body": {}
}
```

#### UPDATED

| Field | Type | Description |
|-------|------|-------------|
| macros | []MacroBody | Array of macro configurations |

### Character Status Events (Consumed)

#### StatusEvent Structure

```json
{
  "transactionId": "uuid",
  "worldId": "byte",
  "characterId": "uint32",
  "type": "string",
  "body": {}
}
```

#### LOGOUT

| Field | Type | Description |
|-------|------|-------------|
| channelId | byte | Channel identifier |
| mapId | uint32 | Map identifier |
| instance | uuid | Map instance identifier |

Clears all cooldowns for the character.

#### DELETED

No body fields.

Deletes all skills, macros, and cooldowns for the character.

## Transaction Semantics

- Skill CREATE, UPDATE, and saga-compensation DELETE, and macro UPDATE operations run inside a database transaction; on success, the resulting status event is enqueued to the transactional outbox (`atlas-outbox` library) within the same transaction, then drained asynchronously to Kafka.
- SET_COOLDOWN operates on Redis (no database transaction) and emits its status event directly to Kafka via the producer, outside the outbox.
- TRANSFER_SP's skill and macro row mutations run inside a single database transaction; on success, the resulting status event (SP_TRANSFERRED or ERROR) is emitted directly to Kafka via the producer after that transaction commits, outside the outbox. A rejected transfer emits ERROR without a database transaction.
- COOLDOWN_EXPIRED (background expiration task) operates on Redis (no database transaction) and is emitted directly to Kafka via the producer, outside the outbox.
- Messages are partitioned by character ID.
