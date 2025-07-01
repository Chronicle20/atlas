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

#### POST /api/qas/validations

Validates a set of conditions against a character's state.

**Request Body:**
```json
{
  "data": {
    "type": "validations",
    "attributes": {
      "characterId": 123,
      "conditions": [
        "jobId=100",
        "meso>=10000",
        "mapId=2000",
        "fame>=50"
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
    "id": "123",
    "attributes": {
      "characterId": 123,
      "passed": true,
      "details": [
        "Passed: Job ID = 100",
        "Passed: Meso >= 10000",
        "Passed: Map ID = 2000",
        "Passed: Fame >= 50"
      ]
    }
  }
}
```

**Supported Conditions:**

| Condition      | Expression Format Example | Source                         |
|----------------|--------------------------|---------------------------------|
| Job            | jobId=100                | Character Service (character.JobId) |
| Meso (Currency)| meso>=10000              | Character Service (character.Meso) |
| Map            | mapId=2000               | Character Service (character.MapId) |
| Fame           | fame>=50                 | Character Service (character.Fame) |
| Inventory Item | item[2000001]>=10        | Inventory Service (quantity of item with template ID 2000001) |

**Supported Operators:**
- `=` (equals)
- `>` (greater than)
- `<` (less than)
- `>=` (greater than or equal to)
- `<=` (less than or equal to)

**Error Responses:**
- 400 Bad Request: Invalid condition format or unsupported condition type
- 500 Internal Server Error: Failed to retrieve character data or other server errors
