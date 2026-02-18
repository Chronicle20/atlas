# Kafka

## Topics Consumed

| Topic | Environment Variable | Description |
|-------|---------------------|-------------|
| Character Chat Command | `COMMAND_TOPIC_CHARACTER_CHAT` | Receives chat commands from characters |

## Topics Produced

| Topic | Environment Variable | Description |
|-------|---------------------|-------------|
| Character Chat Event | `EVENT_TOPIC_CHARACTER_CHAT` | Emits chat events for relay |
| Saga Command | `COMMAND_TOPIC_SAGA` | Emits saga commands for orchestration |
| Character Buff | `COMMAND_TOPIC_CHARACTER_BUFF` | Emits buff application commands |
| Monster Command | `COMMAND_TOPIC_MONSTER` | Emits monster status effect commands |
| Party Quest Command | `COMMAND_TOPIC_PARTY_QUEST` | Emits party quest commands |
| Map Command | `COMMAND_TOPIC_MAP` | Emits map commands |

## Message Types

### Consumed Messages

#### chatCommand

Generic chat command structure consumed from `COMMAND_TOPIC_CHARACTER_CHAT`.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 100000000,
  "instance": "00000000-0000-0000-0000-000000000000",
  "actorId": 12345,
  "message": "Hello world",
  "type": "GENERAL",
  "body": {}
}
```

| Field | Type | Description |
|-------|------|-------------|
| worldId | byte | World identifier |
| channelId | byte | Channel identifier |
| mapId | uint32 | Map identifier |
| instance | uuid.UUID | Map instance identifier |
| actorId | uint32 | Character or pet ID |
| message | string | Chat message content |
| type | string | Chat type |
| body | object | Type-specific body |

#### Chat Types and Bodies

| Type | Body Type | Body Fields |
|------|-----------|-------------|
| GENERAL | generalChatBody | balloonOnly (bool) |
| BUDDY | multiChatBody | recipients ([]uint32) |
| PARTY | multiChatBody | recipients ([]uint32) |
| GUILD | multiChatBody | recipients ([]uint32) |
| ALLIANCE | multiChatBody | recipients ([]uint32) |
| WHISPER | whisperChatBody | recipientName (string) |
| MESSENGER | messengerChatBody | recipients ([]uint32) |
| PET | petChatBody | ownerId (uint32), petSlot (int8), type (byte), action (byte), balloon (bool) |

### Produced Messages

#### ChatEvent

Chat event structure produced to `EVENT_TOPIC_CHARACTER_CHAT`.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 100000000,
  "instance": "00000000-0000-0000-0000-000000000000",
  "actorId": 12345,
  "message": "Hello world",
  "type": "GENERAL",
  "body": {}
}
```

| Field | Type | Description |
|-------|------|-------------|
| worldId | byte | World identifier |
| channelId | byte | Channel identifier |
| mapId | uint32 | Map identifier |
| instance | uuid.UUID | Map instance identifier |
| actorId | uint32 | Character or pet ID |
| message | string | Chat message content |
| type | string | Chat type |
| body | object | Type-specific body |

#### Chat Event Types and Bodies

| Type | Body Type | Body Fields |
|------|-----------|-------------|
| GENERAL | GeneralChatBody | BalloonOnly (bool) |
| BUDDY | MultiChatBody | Recipients ([]uint32) |
| PARTY | MultiChatBody | Recipients ([]uint32) |
| GUILD | MultiChatBody | Recipients ([]uint32) |
| ALLIANCE | MultiChatBody | Recipients ([]uint32) |
| WHISPER | WhisperChatBody | Recipient (uint32) |
| MESSENGER | MessengerChatBody | Recipients ([]uint32) |
| PET | PetChatBody | OwnerId (uint32), PetSlot (int8), Type (byte), Action (byte), Balloon (bool) |
| PINK_TEXT | PinkTextChatBody | Recipients ([]uint32) |

#### Saga

Saga command structure produced to `COMMAND_TOPIC_SAGA`.

```json
{
  "transactionId": "uuid",
  "sagaType": "quest_reward",
  "initiatedBy": "COMMAND",
  "steps": []
}
```

| Field | Type | Description |
|-------|------|-------------|
| transactionId | uuid.UUID | Unique transaction identifier |
| sagaType | string | Type of saga |
| initiatedBy | string | Initiator of the saga |
| steps | []Step | Steps in the saga |

#### BuffCommand

Buff application command produced to `COMMAND_TOPIC_CHARACTER_BUFF`.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 100000000,
  "instance": "00000000-0000-0000-0000-000000000000",
  "characterId": 12345,
  "type": "APPLY",
  "body": {
    "fromId": 0,
    "sourceId": 0,
    "level": 1,
    "duration": 60000,
    "changes": [{"type": "PAD", "amount": 20}]
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| worldId | byte | World identifier |
| channelId | byte | Channel identifier |
| mapId | uint32 | Map identifier |
| instance | uuid.UUID | Map instance identifier |
| characterId | uint32 | Target character ID |
| type | string | Command type (APPLY) |
| body | ApplyCommandBody | Buff application details |

#### ApplyCommandBody

| Field | Type | Description |
|-------|------|-------------|
| fromId | uint32 | Source character ID |
| sourceId | int32 | Source skill ID |
| level | byte | Skill level |
| duration | int32 | Buff duration in milliseconds |
| changes | []StatChange | Stat changes to apply |

#### StatChange

| Field | Type | Description |
|-------|------|-------------|
| type | string | Stat type identifier |
| amount | int32 | Stat change amount |

#### MonsterFieldCommand

Monster field command structure produced to `COMMAND_TOPIC_MONSTER`.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 100000000,
  "instance": "00000000-0000-0000-0000-000000000000",
  "type": "APPLY_STATUS_FIELD",
  "body": {}
}
```

| Field | Type | Description |
|-------|------|-------------|
| worldId | byte | World identifier |
| channelId | byte | Channel identifier |
| mapId | uint32 | Map identifier |
| instance | uuid.UUID | Map instance identifier |
| type | string | Command type |
| body | object | Type-specific body |

#### Monster Command Types

| Type | Body Type | Body Fields |
|------|-----------|-------------|
| APPLY_STATUS_FIELD | ApplyStatusFieldBody | sourceType (string), sourceCharacterId (uint32), sourceSkillId (uint32), sourceSkillLevel (uint32), statuses (map[string]int32), duration (uint32), tickInterval (uint32) |
| CANCEL_STATUS_FIELD | CancelStatusFieldBody | statusTypes ([]string) |
| USE_SKILL_FIELD | UseSkillFieldBody | skillId (uint16), skillLevel (uint16) |
| DESTROY_FIELD | DestroyFieldBody | (empty) |

#### PartyQuestCommand

Party quest command structure produced to `COMMAND_TOPIC_PARTY_QUEST`.

```json
{
  "worldId": 0,
  "characterId": 12345,
  "type": "REGISTER",
  "body": {}
}
```

| Field | Type | Description |
|-------|------|-------------|
| worldId | byte | World identifier |
| characterId | uint32 | Character ID |
| type | string | Command type |
| body | object | Type-specific body |

#### Party Quest Command Types

| Type | Body Type | Body Fields |
|------|-----------|-------------|
| REGISTER | RegisterCommandBody | questId (string), partyId (uint32, optional), channelId (byte), mapId (uint32) |
| STAGE_ADVANCE | StageAdvanceCommandBody | instanceId (uuid.UUID) |

#### MapCommand

Map command structure produced to `COMMAND_TOPIC_MAP`.

```json
{
  "transactionId": "uuid",
  "worldId": 0,
  "channelId": 0,
  "mapId": 100000000,
  "instance": "00000000-0000-0000-0000-000000000000",
  "type": "WEATHER_START",
  "body": {}
}
```

| Field | Type | Description |
|-------|------|-------------|
| transactionId | uuid.UUID | Unique transaction identifier |
| worldId | byte | World identifier |
| channelId | byte | Channel identifier |
| mapId | uint32 | Map identifier |
| instance | uuid.UUID | Map instance identifier |
| type | string | Command type |
| body | object | Type-specific body |

#### Map Command Types

| Type | Body Type | Body Fields |
|------|-----------|-------------|
| WEATHER_START | WeatherStartCommandBody | itemId (uint32), message (string), durationMs (uint32) |

## Transaction Semantics

- Chat events are partitioned by actor ID for ordering
- Saga commands are partitioned by transaction ID
- Buff commands are partitioned by character ID
- Monster commands are partitioned by map ID
- Party quest commands are partitioned by character ID
- Map commands are partitioned by map ID
- Headers include span context for distributed tracing
- Headers include tenant context for multi-tenancy
