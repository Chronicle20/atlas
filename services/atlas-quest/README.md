# atlas-quest
Mushroom game Quest Service

## Overview

A RESTful resource which provides quest state management and progress tracking services. The service handles quest lifecycle (start, complete, forfeit) and tracks progress for various quest objectives such as monster kills, item collection, and map visits.

## Environment

- JAEGER_HOST_PORT - Jaeger [host]:[port]
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- DB_USER - Postgres user name
- DB_PASSWORD - Postgres user password
- DB_HOST - Postgres Database host
- DB_PORT - Postgres Database port
- DB_NAME - Postgres Database name
- BOOTSTRAP_SERVERS - Kafka [host]:[port]
- BASE_SERVICE_URL - [scheme]://[host]:[port]/api/
- COMMAND_TOPIC_QUEST - Kafka Topic for receiving quest commands.
- EVENT_TOPIC_QUEST_STATUS - Kafka Topic for transmitting quest status events.
- EVENT_TOPIC_MONSTER_STATUS - Kafka Topic for receiving monster status events (kill tracking).
- EVENT_TOPIC_ASSET_STATUS - Kafka Topic for receiving asset/item status events (item collection tracking).
- EVENT_TOPIC_CHARACTER_STATUS - Kafka Topic for receiving character status events (map visit tracking).

## API

### Header

All RESTful requests require the supplied header information to identify the server instance.

```
TENANT_ID:083839c6-c47c-42a6-9585-76492795d123
REGION:GMS
MAJOR_VERSION:83
MINOR_VERSION:1
```

### Requests

#### [GET] Get All Quests for Character

```/api/characters/{characterId}/quests```

Example Response:
```json
{
  "data": [
    {
      "type": "quest-status",
      "id": "1",
      "attributes": {
        "characterId": 12345,
        "questId": 1000,
        "state": 1,
        "startedAt": "2024-01-15T10:30:00Z",
        "progress": [
          {
            "infoNumber": 100100,
            "progress": "5"
          }
        ]
      }
    }
  ]
}
```

#### [GET] Get Started Quests for Character

```/api/characters/{characterId}/quests/started```

Returns all quests in the STARTED state for the specified character.

#### [GET] Get Completed Quests for Character

```/api/characters/{characterId}/quests/completed```

Returns all quests in the COMPLETED state for the specified character.

#### [GET] Get Specific Quest for Character

```/api/characters/{characterId}/quests/{questId}```

Example Response:
```json
{
  "data": {
    "type": "quest-status",
    "id": "1",
    "attributes": {
      "characterId": 12345,
      "questId": 1000,
      "state": 1,
      "startedAt": "2024-01-15T10:30:00Z",
      "progress": [
        {
          "infoNumber": 100100,
          "progress": "5"
        },
        {
          "infoNumber": 100101,
          "progress": "3"
        }
      ]
    }
  }
}
```

#### [POST] Start Quest

```/api/characters/{characterId}/quests/{questId}/start```

Starts a quest for the specified character. Returns the created quest record.

Response: 200 OK with quest data

#### [POST] Complete Quest

```/api/characters/{characterId}/quests/{questId}/complete```

Completes a quest for the specified character. The quest must be in the STARTED state.

Response: 204 No Content

#### [POST] Forfeit Quest

```/api/characters/{characterId}/quests/{questId}/forfeit```

Forfeits/abandons a quest for the specified character. The quest must be in the STARTED state.

Response: 204 No Content

#### [GET] Get Quest Progress

```/api/characters/{characterId}/quests/{questId}/progress```

Returns all progress entries for a specific quest.

Example Response:
```json
{
  "data": [
    {
      "type": "progress",
      "id": "1",
      "attributes": {
        "infoNumber": 100100,
        "progress": "5"
      }
    },
    {
      "type": "progress",
      "id": "2",
      "attributes": {
        "infoNumber": 100101,
        "progress": "3"
      }
    }
  ]
}
```

#### [PATCH] Update Quest Progress

```/api/characters/{characterId}/quests/{questId}/progress```

Updates the progress for a specific info number in a quest.

Example Request:
```json
{
  "data": {
    "type": "progress",
    "attributes": {
      "infoNumber": 100100,
      "progress": "10"
    }
  }
}
```

Response: 204 No Content

## Quest States

| State | Value | Description |
|-------|-------|-------------|
| NOT_STARTED | 0 | Quest has not been started |
| STARTED | 1 | Quest is currently active |
| COMPLETED | 2 | Quest has been completed |

## Kafka Commands

The quest service supports several Kafka commands for server-to-server communication.

### START Command

Starts a quest for a character.

**Topic:** `COMMAND_TOPIC_QUEST`

**Command Structure:**
```json
{
  "worldId": 0,
  "characterId": 12345,
  "type": "START",
  "body": {
    "questId": 1000,
    "npcId": 9000000
  }
}
```

### COMPLETE Command

Completes a quest for a character.

**Topic:** `COMMAND_TOPIC_QUEST`

**Command Structure:**
```json
{
  "worldId": 0,
  "characterId": 12345,
  "type": "COMPLETE",
  "body": {
    "questId": 1000,
    "npcId": 9000000,
    "selection": 0
  }
}
```

### FORFEIT Command

Forfeits a quest for a character.

**Topic:** `COMMAND_TOPIC_QUEST`

**Command Structure:**
```json
{
  "worldId": 0,
  "characterId": 12345,
  "type": "FORFEIT",
  "body": {
    "questId": 1000
  }
}
```

### UPDATE_PROGRESS Command

Updates progress for a specific quest objective.

**Topic:** `COMMAND_TOPIC_QUEST`

**Command Structure:**
```json
{
  "worldId": 0,
  "characterId": 12345,
  "type": "UPDATE_PROGRESS",
  "body": {
    "questId": 1000,
    "infoNumber": 100100,
    "progress": "5"
  }
}
```

## Kafka Status Events

The quest service emits status events for quest state changes.

**Topic:** `EVENT_TOPIC_QUEST_STATUS`

### STARTED Event

Emitted when a quest is successfully started.

```json
{
  "worldId": 0,
  "characterId": 12345,
  "type": "STARTED",
  "body": {
    "questId": 1000
  }
}
```

### COMPLETED Event

Emitted when a quest is successfully completed.

```json
{
  "worldId": 0,
  "characterId": 12345,
  "type": "COMPLETED",
  "body": {
    "questId": 1000
  }
}
```

### FORFEITED Event

Emitted when a quest is forfeited.

```json
{
  "worldId": 0,
  "characterId": 12345,
  "type": "FORFEITED",
  "body": {
    "questId": 1000
  }
}
```

### PROGRESS_UPDATED Event

Emitted when quest progress is updated.

```json
{
  "worldId": 0,
  "characterId": 12345,
  "type": "PROGRESS_UPDATED",
  "body": {
    "questId": 1000,
    "infoNumber": 100100,
    "progress": "5"
  }
}
```

### ERROR Event

Emitted when a quest operation fails.

```json
{
  "worldId": 0,
  "characterId": 12345,
  "type": "ERROR",
  "body": {
    "questId": 1000,
    "error": "QUEST_NOT_FOUND"
  }
}
```

**Error Types:**
- `QUEST_NOT_FOUND`: The specified quest does not exist
- `QUEST_ALREADY_ACTIVE`: Attempting to start a quest that is already active
- `QUEST_NOT_STARTED`: Attempting to complete/forfeit a quest that hasn't been started
- `QUEST_ALREADY_COMPLETED`: Attempting to modify an already completed quest
- `REQUIREMENTS_NOT_MET`: Quest requirements are not satisfied
- `UNKNOWN_ERROR`: Unexpected system error occurred

## Automatic Progress Tracking

The quest service automatically tracks progress by consuming events from other services:

### Monster Kills
- Listens to `EVENT_TOPIC_MONSTER_STATUS` for `KILLED` events
- Updates progress for quests tracking the killed monster's ID as an infoNumber

### Item Collection
- Listens to `EVENT_TOPIC_ASSET_STATUS` for `CREATED` events
- Updates progress for quests tracking the collected item's template ID as an infoNumber

### Map Visits
- Listens to `EVENT_TOPIC_CHARACTER_STATUS` for `MAP_CHANGED` events
- Updates progress for quests tracking the visited map's ID as an infoNumber
