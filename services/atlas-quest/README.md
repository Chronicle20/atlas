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
- COMMAND_TOPIC_SAGA - Kafka Topic for sending saga commands (rewards processing).
- EVENT_TOPIC_QUEST_STATUS - Kafka Topic for transmitting quest status events.
- EVENT_TOPIC_MONSTER_STATUS - Kafka Topic for receiving monster status events (kill tracking).
- EVENT_TOPIC_ASSET_STATUS - Kafka Topic for receiving asset/item status events (item collection tracking).
- EVENT_TOPIC_CHARACTER_STATUS - Kafka Topic for receiving character status events (map visit tracking).
- DATA_BASE_URL - atlas-data service URL for quest definitions.
- QUERY_AGGREGATOR_BASE_URL - query-aggregator service URL for validation.

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
        "expirationTime": "2024-01-15T11:30:00Z",
        "completedCount": 0,
        "forfeitCount": 0,
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
      "completedAt": null,
      "expirationTime": "2024-01-15T11:30:00Z",
      "completedCount": 2,
      "forfeitCount": 1,
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

Starts a quest for the specified character. By default, validates start requirements via query-aggregator and processes start actions.

Request:
```json
{
  "data": {
    "type": "start-quest-input",
    "attributes": {
      "skipValidation": false
    }
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| skipValidation | boolean | If true, skips requirement validation (default: false) |

Validates (unless skipValidation is true):
- Level (min/max)
- Job
- Fame
- Meso (min/max)
- Required items
- Prerequisite quest states

Processes start actions:
- Items consumed/awarded on start
- Experience awarded on start
- Meso awarded on start

Response: 200 OK with quest data

Error Response (422 Unprocessable Entity) when requirements not met:
```json
{
  "data": {
    "type": "validation-failed",
    "id": "0",
    "attributes": {
      "failedConditions": ["level", "item"]
    }
  }
}
```

#### [POST] Complete Quest

```/api/characters/{characterId}/quests/{questId}/complete```

Completes a quest for the specified character. By default, validates end requirements and processes rewards via saga-orchestrator. The quest must be in the STARTED state and not expired.

Request:
```json
{
  "data": {
    "type": "complete-quest-input",
    "attributes": {
      "worldId": 0,
      "channelId": 1,
      "skipValidation": false
    }
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| worldId | byte | World ID (required for rewards) |
| channelId | byte | Channel ID (required for rewards) |
| skipValidation | boolean | If true, skips requirement validation (default: false) |

Validates end requirements (unless skipValidation is true):
- Required items (with count validation)

Processes rewards:
- Experience
- Meso
- Fame
- Items (award/consume)
- Skills

Response: 204 No Content (or 200 with `nextQuestId` if quest chain)

#### [POST] Forfeit Quest

```/api/characters/{characterId}/quests/{questId}/forfeit```

Forfeits/abandons a quest for the specified character. The quest must be in the STARTED state. Increments `forfeitCount`.

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

#### [DELETE] Delete All Quests for Character

```/api/characters/{characterId}/quests```

Deletes all quest progress and statuses for the specified character. This is a destructive operation that removes all quest data including progress entries.

Response: 204 No Content

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
| NOT_STARTED | 0 | Quest has not been started or was forfeited |
| STARTED | 1 | Quest is currently active |
| COMPLETED | 2 | Quest has been completed |

## Advanced Features

### Repeatable Quests

Quests with `startRequirements.interval > 0` can be repeated after the specified interval (in minutes) has elapsed since the last completion.

- `completedCount` tracks how many times the quest has been completed
- `forfeitCount` tracks how many times the quest has been forfeited
- Attempting to start before interval elapses returns `400 Bad Request`

### Time-Limited Quests

Quests with `timeLimit` or `timeLimit2` (in seconds) will have an expiration time set when started.

- `expirationTime` is set to `startedAt + timeLimit`
- Attempting to complete an expired quest returns `410 Gone`
- Progress tracking continues but completion is blocked

### Quest Chains

Quests with `endActions.nextQuest` automatically trigger the next quest in the chain upon completion.

- Complete response includes `nextQuestId` when a chain exists
- Chained quests bypass interval checks (can start immediately)
- Progress is initialized for the new quest automatically

### Medal Quests (Map Visit Tracking)

Quests with `endRequirements.fieldEnter` track map visits.

- Progress entries are initialized with "0" (not visited) on quest start
- When character enters a tracked map, progress is updated to "1"
- Used for explorer/medal type quests

### Auto-Start Quests

Quests with `autoStart=true` are automatically started when:
- Character enters a map matching the quest's area
- Character meets start requirements
- Quest hasn't been started or interval has elapsed (for repeatables)

### Auto-Complete Quests

Quests with `autoComplete=true` are automatically completed when:
- All mob kill requirements are met (tracked via progress)
- All map visit requirements are met (tracked via progress)
- Note: Item requirements are validated externally

## Kafka Commands

The quest service supports several Kafka commands for server-to-server communication.

### START Command

Starts a quest for a character. Skips validation (validation should be done by the caller).

**Topic:** `COMMAND_TOPIC_QUEST`

**Command Structure:**
```json
{
  "worldId": 0,
  "channelId": 1,
  "characterId": 12345,
  "type": "START",
  "body": {
    "questId": 1000,
    "npcId": 9000000
  }
}
```

### COMPLETE Command

Completes a quest for a character. Handles quest chains and rewards automatically. Skips validation (validation should be done by the caller).

**Topic:** `COMMAND_TOPIC_QUEST`

**Command Structure:**
```json
{
  "worldId": 0,
  "channelId": 1,
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
  "channelId": 1,
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
  "channelId": 1,
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
- `INTERVAL_NOT_ELAPSED`: Repeatable quest cooldown not finished
- `QUEST_EXPIRED`: Time-limited quest has expired
- `UNKNOWN_ERROR`: Unexpected system error occurred

## Automatic Progress Tracking

The quest service automatically tracks progress by consuming events from other services:

### Monster Kills
- Listens to `EVENT_TOPIC_MONSTER_STATUS` for `KILLED` events
- Updates progress for quests tracking the killed monster's ID as an infoNumber
- Checks for auto-complete after progress update

### Item Collection
- Listens to `EVENT_TOPIC_ASSET_STATUS` for `CREATED` events
- Updates progress for quests tracking the collected item's template ID as an infoNumber

### Map Visits
- Listens to `EVENT_TOPIC_CHARACTER_STATUS` for `MAP_CHANGED` events
- Updates progress for quests tracking the visited map's ID as an infoNumber
- Triggers auto-start quest checks for the new map
- Checks for auto-complete after progress update

## Service Dependencies

| Service | Purpose |
|---------|---------|
| atlas-data | Quest definitions (requirements, actions, rewards) |
| query-aggregator | Character state validation (level, job, items, etc.) |
| saga-orchestrator | Rewards distribution (exp, meso, items, skills) |

## Database Schema

### quest_statuses
| Column | Type | Description |
|--------|------|-------------|
| id | uint32 | Primary key |
| tenant_id | uuid | Tenant identifier |
| character_id | uint32 | Character ID |
| quest_id | uint32 | Quest definition ID |
| state | int | Quest state (0/1/2) |
| started_at | timestamp | When quest was started |
| completed_at | timestamp | When quest was completed |
| expiration_time | timestamp | When quest expires (time-limited) |
| completed_count | uint32 | Times completed (repeatable) |
| forfeit_count | uint32 | Times forfeited |

### quest_progress
| Column | Type | Description |
|--------|------|-------------|
| id | uint32 | Primary key |
| quest_status_id | uint32 | Foreign key to quest_statuses |
| info_number | uint32 | Mob ID or Map ID being tracked |
| progress | string | Progress value (kill count or "0"/"1" for maps) |
