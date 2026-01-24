# Kafka

## Topics Consumed

| Environment Variable | Topic Purpose |
|---------------------|---------------|
| `COMMAND_TOPIC_SKILL` | Skill commands (create, update, set cooldown) |
| `COMMAND_TOPIC_SKILL_MACRO` | Macro update commands |
| `EVENT_TOPIC_CHARACTER_STATUS` | Character status events (logout, deleted) |

## Topics Produced

| Environment Variable | Topic Purpose |
|---------------------|---------------|
| `EVENT_TOPIC_SKILL_STATUS` | Skill status events (created, updated, cooldown applied, cooldown expired) |
| `STATUS_EVENT_TOPIC_SKILL_MACRO` | Macro status events (updated) |

## Message Types

### Skill Commands

#### Command Structure

```json
{
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

### Skill Status Events

#### StatusEvent Structure

```json
{
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

### Macro Commands

#### Command Structure

```json
{
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

Clears all cooldowns for the character.

#### DELETED

No body fields.

Deletes all skills, macros, and cooldowns for the character.

## Transaction Semantics

- Skill create and update operations use database transactions.
- Status events are emitted after successful transaction completion.
- Messages are partitioned by character ID.
