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

## Message Types

### Consumed Messages

#### chatCommand

Generic chat command structure consumed from `COMMAND_TOPIC_CHARACTER_CHAT`.

```json
{
  "worldId": 0,
  "channelId": 0,
  "mapId": 100000000,
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

## Transaction Semantics

- Messages are partitioned by actor ID for ordering
- Saga commands are partitioned by transaction ID
- Headers include span context for distributed tracing
- Headers include tenant context for multi-tenancy
