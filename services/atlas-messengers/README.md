# atlas-messengers

Mushroom game messengers Service

## Overview

A RESTful resource which provides messenger (party chat) services. Messengers are ephemeral group chat rooms that allow up to 3 characters to communicate in real-time.

**Architecture Note:** This service uses in-memory storage instead of database persistence. Messenger state is intentionally ephemeral and does not persist across service restarts. See [ADR-001](../../docs/architecture/decisions/001-atlas-messengers-in-memory-storage.md) for rationale.

## Environment

- JAEGER_HOST - Jaeger [host]:[port]
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- BASE_SERVICE_URL - [scheme]://[host]:[port]/api/
- BOOTSTRAP_SERVERS - Kafka [host]:[port]
- COMMAND_TOPIC_MESSENGER - Kafka topic for messenger commands (default: MESSENGER_COMMAND)
- EVENT_TOPIC_MESSENGER_STATUS - Kafka topic for messenger status events (default: MESSENGER_STATUS)
- EVENT_TOPIC_MESSENGER_MEMBER_STATUS - Kafka topic for member status events (default: MEMBER_STATUS)
- EVENT_TOPIC_CHARACTER_STATUS - Kafka topic for character status events to consume

## API

### Header

All RESTful requests require the supplied header information to identify the server instance.

```
TENANT_ID:083839c6-c47c-42a6-9585-76492795d123
REGION:GMS
MAJOR_VERSION:83
MINOR_VERSION:1
```

## REST Endpoints

| Method | Path | Query Parameters | Description |
|--------|------|------------------|-------------|
| GET | /api/messengers | - | Get all messengers |
| GET | /api/messengers | filter[members.id]={id} | Get messengers by member ID |
| GET | /api/messengers/{messengerId} | - | Get messenger by ID |
| GET | /api/messengers/{messengerId}/members | - | Get messenger members |
| GET | /api/messengers/{messengerId}/relationships/members | - | Get messenger members (JSON:API relationship) |
| POST | /api/messengers/{messengerId}/members | - | Add member to messenger (via Kafka command) |
| GET | /api/messengers/{messengerId}/members/{memberId} | - | Get specific member |
| DELETE | /api/messengers/{messengerId}/members/{memberId} | - | Remove member from messenger (via Kafka command) |

### Example Requests

**Get all messengers:**
```bash
curl -H "TENANT_ID: 083839c6-c47c-42a6-9585-76492795d123" \
     http://localhost:8080/api/messengers
```

**Get messengers by member:**
```bash
curl -H "TENANT_ID: 083839c6-c47c-42a6-9585-76492795d123" \
     "http://localhost:8080/api/messengers?filter[members.id]=12345"
```

**Get specific messenger:**
```bash
curl -H "TENANT_ID: 083839c6-c47c-42a6-9585-76492795d123" \
     http://localhost:8080/api/messengers/1000000001
```

## Kafka Commands (Consumed)

| Topic | Command Type | Description |
|-------|--------------|-------------|
| COMMAND_TOPIC_MESSENGER | CREATE | Create a new messenger with the actor as first member |
| COMMAND_TOPIC_MESSENGER | JOIN | Join an existing messenger |
| COMMAND_TOPIC_MESSENGER | LEAVE | Leave a messenger |
| COMMAND_TOPIC_MESSENGER | REQUEST_INVITE | Request to invite another character to messenger |

### Command Message Format

```json
{
  "transactionId": "uuid",
  "actorId": 12345,
  "type": "CREATE|JOIN|LEAVE|REQUEST_INVITE",
  "body": {}
}
```

**JOIN body:**
```json
{
  "messengerId": 1000000001
}
```

**LEAVE body:**
```json
{
  "messengerId": 1000000001
}
```

**REQUEST_INVITE body:**
```json
{
  "characterId": 67890
}
```

## Kafka Events (Produced)

### Messenger Status Events

| Topic | Event Type | Description |
|-------|------------|-------------|
| EVENT_TOPIC_MESSENGER_STATUS | CREATED | Messenger was successfully created |
| EVENT_TOPIC_MESSENGER_STATUS | JOINED | Member successfully joined messenger |
| EVENT_TOPIC_MESSENGER_STATUS | LEFT | Member successfully left messenger |
| EVENT_TOPIC_MESSENGER_STATUS | ERROR | An error occurred during operation |

### Member Status Events

| Topic | Event Type | Description |
|-------|------------|-------------|
| EVENT_TOPIC_MESSENGER_MEMBER_STATUS | LOGIN | Member logged into the game |
| EVENT_TOPIC_MESSENGER_MEMBER_STATUS | LOGOUT | Member logged out of the game |

### Status Event Message Format

```json
{
  "transactionId": "uuid",
  "actorId": 12345,
  "worldId": 0,
  "messengerId": 1000000001,
  "type": "CREATED|JOINED|LEFT|ERROR",
  "body": {}
}
```

**JOINED/LEFT body:**
```json
{
  "slot": 0
}
```

**ERROR body:**
```json
{
  "type": "ERROR_CODE",
  "characterName": "CharacterName"
}
```

## Kafka Events (Consumed)

| Topic | Event Type | Description |
|-------|------------|-------------|
| EVENT_TOPIC_CHARACTER_STATUS | LOGIN | Character logged in - add to registry |
| EVENT_TOPIC_CHARACTER_STATUS | LOGOUT | Character logged out - update registry |
| EVENT_TOPIC_CHARACTER_STATUS | CHANNEL_CHANGED | Character changed channel - update registry |

## Scaling Limitations

This service uses in-memory storage for messenger state. This design decision has the following implications:

1. **Single-pod deployment only** - Messenger state is not shared across instances
2. **State lost on restart** - All active messengers are disbanded when service restarts
3. **No horizontal scaling** - Cannot scale out without migrating to external state store (Redis/DB)

These limitations are acceptable for the ephemeral nature of messenger groups. If horizontal scaling becomes necessary, consider migrating to Redis for state storage.

## Business Rules

- Maximum 3 members per messenger
- Messenger is automatically disbanded when the last member leaves
- Characters can only be in one messenger at a time
- Messenger IDs start at 1,000,000,000 and increment per tenant
