# atlas-marriages
Mushroom game marriages Service

## Overview

The `atlas-marriages` service manages the complete lifecycle of character marriages within the game world. It handles proposals, engagement tracking, ceremony coordination, and historical data storage following Atlas microservice architecture patterns.

### Key Features

- **Marriage Lifecycle Management**: Proposals, engagements, ceremonies, and divorces
- **Eligibility Validation**: Level requirements, relationship constraints, and cooldown management
- **Ceremony Orchestration**: Scheduling, invitee management, and state transitions
- **Event-Driven Architecture**: Kafka messaging for inter-service communication
- **Historical Tracking**: Complete audit trail of all marriage-related activities

## Environment Variables

- `JAEGER_HOST` - Jaeger [host]:[port]
- `LOG_LEVEL` - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- `COMMAND_TOPIC_MARRIAGE` - Kafka topic for marriage commands
- `EVENT_TOPIC_MARRIAGE_STATUS` - Kafka topic for marriage events

## API Documentation

### Headers

All RESTful requests require the following header information to identify the server instance:

```
Content-Type: application/json
TENANT_ID: 083839c6-c47c-42a6-9585-76492795d123
REGION: GMS
MAJOR_VERSION: 83
MINOR_VERSION: 1
```

## REST API Endpoints

### GET /api/characters/{characterId}/marriage

Returns the current marriage information for a character.

**Parameters:**
- `characterId` (path, required): The character ID to query

**Response (200 OK):**
```json
{
  "data": {
    "id": "12345",
    "type": "marriage",
    "attributes": {
      "id": 12345,
      "characterId1": 1001,
      "characterId2": 1002,
      "status": "married",
      "proposedAt": "2023-07-15T10:30:00Z",
      "engagedAt": "2023-07-15T11:45:00Z",
      "marriedAt": "2023-07-16T14:20:00Z",
      "createdAt": "2023-07-15T10:30:00Z",
      "updatedAt": "2023-07-16T14:20:00Z",
      "partner": {
        "characterId": 1002
      },
      "ceremony": {
        "id": 5678,
        "status": "completed",
        "scheduledAt": "2023-07-16T14:00:00Z",
        "startedAt": "2023-07-16T14:00:00Z",
        "completedAt": "2023-07-16T14:20:00Z",
        "inviteeCount": 8
      }
    }
  }
}
```

**Response (404 Not Found):**
```json
{
  "error": {
    "status": 404,
    "title": "Not Found",
    "detail": "Character is not married"
  }
}
```

**Marriage Status Values:**
- `proposed` - Proposal has been sent but not yet accepted
- `engaged` - Proposal accepted, ceremony not yet completed
- `married` - Ceremony completed, marriage is active
- `divorced` - Marriage has been ended by divorce

### GET /api/characters/{characterId}/marriage/history

Returns the complete marriage history for a character, including past marriages and proposals.

**Parameters:**
- `characterId` (path, required): The character ID to query

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "12345",
      "type": "marriage",
      "attributes": {
        "id": 12345,
        "characterId1": 1001,
        "characterId2": 1002,
        "status": "divorced",
        "proposedAt": "2023-07-15T10:30:00Z",
        "engagedAt": "2023-07-15T11:45:00Z",
        "marriedAt": "2023-07-16T14:20:00Z",
        "divorcedAt": "2023-08-01T09:15:00Z",
        "createdAt": "2023-07-15T10:30:00Z",
        "updatedAt": "2023-08-01T09:15:00Z"
      }
    }
  ]
}
```

**Response (200 OK - Empty History):**
```json
{
  "data": []
}
```

### GET /api/characters/{characterId}/marriage/proposals

Returns all pending proposals for a character (both sent and received).

**Parameters:**
- `characterId` (path, required): The character ID to query

**Response (200 OK):**
```json
{
  "data": [
    {
      "id": "9876",
      "type": "proposal",
      "attributes": {
        "id": 9876,
        "proposerId": 1001,
        "targetId": 1003,
        "status": "pending",
        "proposedAt": "2023-07-16T08:30:00Z",
        "expiresAt": "2023-07-17T08:30:00Z",
        "rejectionCount": 0,
        "createdAt": "2023-07-16T08:30:00Z",
        "updatedAt": "2023-07-16T08:30:00Z"
      }
    }
  ]
}
```

**Proposal Status Values:**
- `pending` - Proposal is active and awaiting response
- `accepted` - Proposal has been accepted (leads to engagement)
- `declined` - Proposal has been declined by the target
- `expired` - Proposal has expired (24 hours without response)
- `cancelled` - Proposal has been cancelled by the proposer

### Error Responses

All endpoints may return the following error responses:

**400 Bad Request:**
```json
{
  "error": {
    "status": 400,
    "title": "Bad Request",
    "detail": "Invalid character ID"
  }
}
```

**500 Internal Server Error:**
```json
{
  "error": {
    "status": 500,
    "title": "Internal Server Error",
    "detail": "An unexpected error occurred"
  }
}
```

## Kafka Events & Commands

### Command Topics

Commands are sent to the `COMMAND_TOPIC_MARRIAGE` topic with the following structure:

```json
{
  "characterId": 1001,
  "type": "COMMAND_TYPE",
  "body": {
    // Command-specific payload
  }
}
```

### Available Commands

#### Marriage Proposal Commands

**PROPOSE** - Initiate a marriage proposal
```json
{
  "characterId": 1001,
  "type": "PROPOSE",
  "body": {
    "targetCharacterId": 1002
  }
}
```

**ACCEPT** - Accept a marriage proposal
```json
{
  "characterId": 1002,
  "type": "ACCEPT",
  "body": {
    "proposalId": 9876
  }
}
```

**DECLINE** - Decline a marriage proposal
```json
{
  "characterId": 1002,
  "type": "DECLINE",
  "body": {
    "proposalId": 9876
  }
}
```

**CANCEL** - Cancel a sent proposal
```json
{
  "characterId": 1001,
  "type": "CANCEL",
  "body": {
    "proposalId": 9876
  }
}
```

#### Marriage Commands

**DIVORCE** - Initiate divorce proceedings
```json
{
  "characterId": 1001,
  "type": "DIVORCE",
  "body": {
    "marriageId": 12345
  }
}
```

#### Ceremony Commands

**SCHEDULE_CEREMONY** - Schedule a wedding ceremony
```json
{
  "characterId": 1001,
  "type": "SCHEDULE_CEREMONY",
  "body": {
    "marriageId": 12345,
    "scheduledAt": "2023-07-16T14:00:00Z",
    "invitees": [1003, 1004, 1005]
  }
}
```

**START_CEREMONY** - Begin a scheduled ceremony
```json
{
  "characterId": 1001,
  "type": "START_CEREMONY",
  "body": {
    "ceremonyId": 5678
  }
}
```

**COMPLETE_CEREMONY** - Complete a ceremony
```json
{
  "characterId": 1001,
  "type": "COMPLETE_CEREMONY",
  "body": {
    "ceremonyId": 5678
  }
}
```

**CANCEL_CEREMONY** - Cancel a ceremony
```json
{
  "characterId": 1001,
  "type": "CANCEL_CEREMONY",
  "body": {
    "ceremonyId": 5678
  }
}
```

**POSTPONE_CEREMONY** - Postpone a ceremony
```json
{
  "characterId": 1001,
  "type": "POSTPONE_CEREMONY",
  "body": {
    "ceremonyId": 5678
  }
}
```

**RESCHEDULE_CEREMONY** - Reschedule a ceremony
```json
{
  "characterId": 1001,
  "type": "RESCHEDULE_CEREMONY",
  "body": {
    "ceremonyId": 5678,
    "scheduledAt": "2023-07-17T16:00:00Z"
  }
}
```

**ADD_INVITEE** - Add an invitee to a ceremony
```json
{
  "characterId": 1001,
  "type": "ADD_INVITEE",
  "body": {
    "ceremonyId": 5678,
    "characterId": 1006
  }
}
```

**REMOVE_INVITEE** - Remove an invitee from a ceremony
```json
{
  "characterId": 1001,
  "type": "REMOVE_INVITEE",
  "body": {
    "ceremonyId": 5678,
    "characterId": 1006
  }
}
```

### Event Topics

Events are published to the `EVENT_TOPIC_MARRIAGE_STATUS` topic with the following structure:

```json
{
  "characterId": 1001,
  "type": "EVENT_TYPE",
  "body": {
    // Event-specific payload
  }
}
```

### Available Events

#### Proposal Events

**PROPOSAL_CREATED** - A new proposal has been created
```json
{
  "characterId": 1001,
  "type": "PROPOSAL_CREATED",
  "body": {
    "proposalId": 9876,
    "proposerId": 1001,
    "targetCharacterId": 1002,
    "proposedAt": "2023-07-16T08:30:00Z",
    "expiresAt": "2023-07-17T08:30:00Z"
  }
}
```

**PROPOSAL_ACCEPTED** - A proposal has been accepted
```json
{
  "characterId": 1002,
  "type": "PROPOSAL_ACCEPTED",
  "body": {
    "proposalId": 9876,
    "proposerId": 1001,
    "targetCharacterId": 1002,
    "acceptedAt": "2023-07-16T09:15:00Z"
  }
}
```

**PROPOSAL_DECLINED** - A proposal has been declined
```json
{
  "characterId": 1002,
  "type": "PROPOSAL_DECLINED",
  "body": {
    "proposalId": 9876,
    "proposerId": 1001,
    "targetCharacterId": 1002,
    "declinedAt": "2023-07-16T09:15:00Z",
    "rejectionCount": 1,
    "cooldownUntil": "2023-07-17T09:15:00Z"
  }
}
```

**PROPOSAL_EXPIRED** - A proposal has expired
```json
{
  "characterId": 1001,
  "type": "PROPOSAL_EXPIRED",
  "body": {
    "proposalId": 9876,
    "proposerId": 1001,
    "targetCharacterId": 1002,
    "expiredAt": "2023-07-17T08:30:00Z"
  }
}
```

#### Marriage Events

**MARRIAGE_CREATED** - A new marriage has been created
```json
{
  "characterId": 1001,
  "type": "MARRIAGE_CREATED",
  "body": {
    "marriageId": 12345,
    "characterId1": 1001,
    "characterId2": 1002,
    "marriedAt": "2023-07-16T14:20:00Z"
  }
}
```

**MARRIAGE_DIVORCED** - A marriage has been divorced
```json
{
  "characterId": 1001,
  "type": "MARRIAGE_DIVORCED",
  "body": {
    "marriageId": 12345,
    "characterId1": 1001,
    "characterId2": 1002,
    "divorcedAt": "2023-08-01T09:15:00Z",
    "initiatedBy": 1001
  }
}
```

#### Ceremony Events

**CEREMONY_SCHEDULED** - A ceremony has been scheduled
```json
{
  "characterId": 1001,
  "type": "CEREMONY_SCHEDULED",
  "body": {
    "ceremonyId": 5678,
    "marriageId": 12345,
    "characterId1": 1001,
    "characterId2": 1002,
    "scheduledAt": "2023-07-16T14:00:00Z",
    "invitees": [1003, 1004, 1005]
  }
}
```

**CEREMONY_STARTED** - A ceremony has started
```json
{
  "characterId": 1001,
  "type": "CEREMONY_STARTED",
  "body": {
    "ceremonyId": 5678,
    "marriageId": 12345,
    "characterId1": 1001,
    "characterId2": 1002,
    "startedAt": "2023-07-16T14:00:00Z"
  }
}
```

**CEREMONY_COMPLETED** - A ceremony has been completed
```json
{
  "characterId": 1001,
  "type": "CEREMONY_COMPLETED",
  "body": {
    "ceremonyId": 5678,
    "marriageId": 12345,
    "characterId1": 1001,
    "characterId2": 1002,
    "completedAt": "2023-07-16T14:20:00Z"
  }
}
```

#### Error Events

**MARRIAGE_ERROR** - An error occurred during marriage operations
```json
{
  "characterId": 1001,
  "type": "MARRIAGE_ERROR",
  "body": {
    "errorType": "PROPOSAL_ERROR",
    "errorCode": "ALREADY_MARRIED",
    "message": "Character is already married",
    "characterId": 1001,
    "context": "Marriage proposal validation",
    "timestamp": "2023-07-16T08:30:00Z"
  }
}
```

### Error Codes

Common error codes that may be returned in error events:

- `ALREADY_MARRIED` - Character is already married
- `ALREADY_ENGAGED` - Character is already engaged
- `INSUFFICIENT_LEVEL` - Character does not meet level requirements
- `SELF_PROPOSAL` - Cannot propose to oneself
- `GLOBAL_COOLDOWN` - Global proposal cooldown is active
- `TARGET_COOLDOWN` - Target-specific cooldown is active
- `PROPOSAL_EXPIRED` - The proposal has expired
- `PROPOSAL_NOT_FOUND` - Proposal does not exist
- `MARRIAGE_NOT_FOUND` - Marriage does not exist
- `CEREMONY_NOT_FOUND` - Ceremony does not exist
- `INVALID_STATE` - Invalid state transition
- `INVITEE_LIMIT_EXCEEDED` - Too many invitees (max 15)
- `PARTNER_DISCONNECTED` - Partner has disconnected during ceremony
- `CEREMONY_TIMEOUT` - Ceremony timed out due to inactivity
- `TENANT_MISMATCH` - Characters are not in the same tenant

## Business Rules

### Eligibility Requirements

- Characters must be **level 10 or higher** to marry
- A character may only be in **one relationship** at a time
- Both characters must be in the **same tenant/world**
- **No gender restrictions** apply

### Proposal Constraints

- Cannot propose if currently married or engaged
- Cannot receive proposals if already engaged
- Proposals may be sent to offline characters
- Proposals expire after **24 hours**
- **Global cooldown**: 4 hours between any proposals by the same character
- **Per-target cooldown**: Starts at 24 hours, doubles on each successive rejection

### Ceremony Rules

- Must be scheduled after engagement
- Maximum of **15 invitees** allowed
- Ceremony is postponed if either partner is offline for **5+ minutes**
- Ceremony must be restarted from the beginning after postponement

### Divorce

- Either party may initiate divorce unilaterally
- Divorce cost enforcement is handled by external services
- Marriage is automatically ended if a character is deleted
