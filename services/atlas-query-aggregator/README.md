# atlas-query-aggregator
Mushroom game query-aggregator Service

## Overview

A RESTful service that provides composite character state validation in the Atlas ecosystem. This service queries dependent services directly to validate conditions against character state.

### Features

- Validates character state against specified conditions
- Supports various comparison operators (=, >, <, >=, <=)
- Returns detailed validation results with pass/fail status
- JSON:API-compliant API design

### Supported Validations

- Job ID validation
- Meso (currency) validation
- Map ID validation
- Fame validation
- Gender validation (0 = male, 1 = female)
- Level validation
- Rebirth count validation
- Dojo points validation
- Vanquisher kills validation
- GM level validation
- Guild membership validation
- Guild rank validation
- Quest status validation
- Quest progress validation
- Marriage gift validation
- Character stats validation (strength, dexterity, intelligence, luck)

## Environment

- JAEGER_HOST - Jaeger [host]:[port]
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace

### External Service Dependencies

- **Character Service**: Provides character state data including level, stats, guild information, and other character attributes
- **Quest Service**: Provides quest status and progress information for quest-based validations
- **Marriage Service**: Provides marriage gift information for marriage-related validations
- **Inventory Service**: Provides item quantity information for inventory-based validations

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

#### POST /api/validations

Validates a set of conditions against a character's state.

**Request Body (Structured Format - Recommended):**
```json
{
  "data": {
    "id": "56",
    "type": "validations",
    "attributes": {
      "conditions": [
        {
          "type": "jobId",
          "operator": "=",
          "value": 100
        },
        {
          "type": "meso",
          "operator": ">=",
          "value": 10000
        },
        {
          "type": "level",
          "operator": ">=",
          "value": 30
        },
        {
          "type": "reborns",
          "operator": ">=",
          "value": 1
        },
        {
          "type": "dojoPoints",
          "operator": ">=",
          "value": 1000
        },
        {
          "type": "questStatus",
          "operator": "=",
          "value": 2,
          "referenceId": 1001
        },
        {
          "type": "guildId",
          "operator": "=",
          "value": 12345
        },
        {
          "type": "item",
          "operator": ">=",
          "value": 10,
          "referenceId": 2000001
        }
      ]
    }
  }
}
```

**Response:**
```json
{
  "data": {
    "type": "validations",
    "id": "56",
    "attributes": {
      "passed": true,
      "details": [
        "Passed: Job ID = 100",
        "Passed: Meso >= 10000",
        "Passed: Level >= 30",
        "Passed: Reborns >= 1",
        "Passed: Dojo Points >= 1000",
        "Passed: Quest 1001 Status = 2",
        "Passed: Guild ID = 12345",
        "Passed: Item 2000001 quantity >= 10"
      ],
      "results": [
        {
          "passed": true,
          "description": "Job ID = 100",
          "type": "jobId",
          "operator": "=",
          "value": 100,
          "actualValue": 100
        },
        {
          "passed": true,
          "description": "Meso >= 10000",
          "type": "meso",
          "operator": ">=",
          "value": 10000,
          "actualValue": 15000
        },
        {
          "passed": true,
          "description": "Level >= 30",
          "type": "level",
          "operator": ">=",
          "value": 30,
          "actualValue": 45
        },
        {
          "passed": true,
          "description": "Reborns >= 1",
          "type": "reborns",
          "operator": ">=",
          "value": 1,
          "actualValue": 2
        },
        {
          "passed": true,
          "description": "Dojo Points >= 1000",
          "type": "dojoPoints",
          "operator": ">=",
          "value": 1000,
          "actualValue": 1500
        },
        {
          "passed": true,
          "description": "Quest 1001 Status = 2",
          "type": "questStatus",
          "operator": "=",
          "value": 2,
          "actualValue": 2
        },
        {
          "passed": true,
          "description": "Guild ID = 12345",
          "type": "guildId",
          "operator": "=",
          "value": 12345,
          "actualValue": 12345
        },
        {
          "passed": true,
          "description": "Item 2000001 quantity >= 10",
          "type": "item",
          "operator": ">=",
          "value": 10,
          "itemId": 2000001,
          "actualValue": 15
        }
      ]
    }
  }
}
```

**Supported Conditions:**

| Condition       | Expression Format Example | Source                                                        |
|-----------------|---------------------------|---------------------------------------------------------------|
| Job             | jobId=100                 | Character Service (character.JobId)                           |
| Meso (Currency) | meso>=10000               | Character Service (character.Meso)                            |
| Map             | mapId=2000                | Character Service (character.MapId)                           |
| Fame            | fame>=50                  | Character Service (character.Fame)                            |
| Gender          | gender=0                  | Character Service (character.Gender) - 0=male, 1=female       |
| Level           | level>=30                 | Character Service (character.Level)                           |
| Reborns         | reborns>=1                | Character Service (character.Reborns)                         |
| Dojo Points     | dojoPoints>=1000          | Character Service (character.DojoPoints)                      |
| Vanquisher Kills| vanquisherKills>=10       | Character Service (character.VanquisherKills)                |
| GM Level        | gmLevel>=1                | Character Service (character.GmLevel)                         |
| Guild ID        | guildId=12345             | Character Service (character.Guild.Id)                        |
| Guild Rank      | guildRank>=2              | Character Service (character.Guild.Rank)                      |
| Quest Status    | questStatus=2             | Quest Service (quest.Status) - 0=UNDEFINED, 1=NOT_STARTED, 2=STARTED, 3=COMPLETED |
| Quest Progress  | questProgress>=5          | Quest Service (quest.Progress) - requires referenceId and step |
| Marriage Gifts  | hasUnclaimedMarriageGifts=1 | Marriage Service (marriage.HasUnclaimedGifts) - 0=false, 1=true |
| Strength        | strength>=100             | Character Service (character.Strength)                        |
| Dexterity       | dexterity>=100            | Character Service (character.Dexterity)                       |
| Intelligence    | intelligence>=100         | Character Service (character.Intelligence)                    |
| Luck            | luck>=100                 | Character Service (character.Luck)                            |
| Inventory Item  | item[2000001]>=10         | Inventory Service (quantity of item with template ID) - use referenceId |

**Supported Operators:**
- `=` (equals)
- `>` (greater than)
- `<` (less than)
- `>=` (greater than or equal to)
- `<=` (less than or equal to)

**Additional Parameters:**
- `referenceId` (uint32): Required for quest and item validations. Specifies the quest ID or item template ID to validate against.
- `step` (string): Required for quest progress validations. Specifies the specific quest step to check progress for.

**Quest Status Values:**
- `0` = UNDEFINED
- `1` = NOT_STARTED  
- `2` = STARTED
- `3` = COMPLETED

**Examples of Advanced Validations:**

```json
{
  "type": "questStatus",
  "operator": "=",
  "value": 2,
  "referenceId": 1001
}
```

```json
{
  "type": "questProgress", 
  "operator": ">=",
  "value": 5,
  "referenceId": 1001,
  "step": "mobsKilled"
}
```

```json
{
  "type": "item",
  "operator": ">=", 
  "value": 10,
  "referenceId": 2000001
}
```

**Error Responses:**
- 400 Bad Request: Invalid condition format or unsupported condition type
- 500 Internal Server Error: Failed to retrieve character data or other server errors
