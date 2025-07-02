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
- Guild leader validation (0 = not a leader, 1 = is a leader)

## Environment

- JAEGER_HOST - Jaeger [host]:[port]
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace

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
          "type": "mapId",
          "operator": "=",
          "value": 2000
        },
        {
          "type": "fame",
          "operator": ">=",
          "value": 50
        },
        {
          "type": "gender",
          "operator": "=",
          "value": 0
        },
        {
          "type": "guildLeader",
          "operator": "=",
          "value": 1
        },
        {
          "type": "item",
          "operator": ">=",
          "value": 10,
          "itemId": 2000001
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
        "Passed: Map ID = 2000",
        "Passed: Fame >= 50",
        "Passed: Gender = 0",
        "Passed: Guild Leader = 1",
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
          "description": "Map ID = 2000",
          "type": "mapId",
          "operator": "=",
          "value": 2000,
          "actualValue": 2000
        },
        {
          "passed": true,
          "description": "Fame >= 50",
          "type": "fame",
          "operator": ">=",
          "value": 50,
          "actualValue": 75
        },
        {
          "passed": true,
          "description": "Gender = 0",
          "type": "gender",
          "operator": "=",
          "value": 0,
          "actualValue": 0
        },
        {
          "passed": true,
          "description": "Guild Leader = 1",
          "type": "guildLeader",
          "operator": "=",
          "value": 1,
          "actualValue": 1
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
| Guild Leader    | guildLeader=1             | Guild Service (guild.IsLeader) - 0=not a leader, 1=is a leader|
| Inventory Item  | item[2000001]>=10         | Inventory Service (quantity of item with template ID 2000001) |

**Supported Operators:**
- `=` (equals)
- `>` (greater than)
- `<` (less than)
- `>=` (greater than or equal to)
- `<=` (less than or equal to)

**Error Responses:**
- 400 Bad Request: Invalid condition format or unsupported condition type
- 500 Internal Server Error: Failed to retrieve character data or other server errors
