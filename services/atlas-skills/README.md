# atlas-skills
Mushroom game Skills Service

## Overview

A RESTful resource which provides skills services, including skill management, cooldowns, and macros.

## Environment Variables

- `REST_PORT` - Port for the REST server
- `JAEGER_HOST_PORT` - Jaeger host and port in format [host]:[port]
- `LOG_LEVEL` - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- `DB_USER` - Database username
- `DB_PASSWORD` - Database password
- `DB_HOST` - Database host
- `DB_PORT` - Database port
- `DB_NAME` - Database name

## API

### Header

All RESTful requests require the supplied header information to identify the server instance.

```
TENANT_ID:083839c6-c47c-42a6-9585-76492795d123
REGION:GMS
MAJOR_VERSION:83
MINOR_VERSION:1
```

### Endpoints

#### Skills

##### Get All Skills for a Character
- **URL**: `/characters/{characterId}/skills`
- **Method**: GET
- **URL Parameters**:
  - `characterId`: ID of the character
- **Response**: Array of skill objects
  ```json
  {
    "data": [
      {
        "type": "skills",
        "id": "1234",
        "attributes": {
          "level": 10,
          "masterLevel": 20,
          "expiration": "2023-01-01T00:00:00Z",
          "cooldownExpiresAt": "2023-01-01T00:00:00Z"
        }
      }
    ]
  }
  ```

##### Create a Skill for a Character
- **URL**: `/characters/{characterId}/skills`
- **Method**: POST
- **URL Parameters**:
  - `characterId`: ID of the character
- **Request Body**:
  ```json
  {
    "data": {
      "type": "skills",
      "id": "1234",
      "attributes": {
        "level": 10,
        "masterLevel": 20,
        "expiration": "2023-01-01T00:00:00Z"
      }
    }
  }
  ```
- **Response**: Status 202 Accepted

##### Get a Specific Skill for a Character
- **URL**: `/characters/{characterId}/skills/{skillId}`
- **Method**: GET
- **URL Parameters**:
  - `characterId`: ID of the character
  - `skillId`: ID of the skill
- **Response**: Skill object
  ```json
  {
    "data": {
      "type": "skills",
      "id": "1234",
      "attributes": {
        "level": 10,
        "masterLevel": 20,
        "expiration": "2023-01-01T00:00:00Z",
        "cooldownExpiresAt": "2023-01-01T00:00:00Z"
      }
    }
  }
  ```

##### Update a Skill for a Character
- **URL**: `/characters/{characterId}/skills/{skillId}`
- **Method**: PATCH
- **URL Parameters**:
  - `characterId`: ID of the character
  - `skillId`: ID of the skill
- **Request Body**:
  ```json
  {
    "data": {
      "type": "skills",
      "id": "1234",
      "attributes": {
        "level": 15,
        "masterLevel": 25,
        "expiration": "2023-02-01T00:00:00Z"
      }
    }
  }
  ```
- **Response**: Status 202 Accepted

#### Macros

##### Get All Macros for a Character
- **URL**: `/characters/{characterId}/macros`
- **Method**: GET
- **URL Parameters**:
  - `characterId`: ID of the character
- **Response**: Array of macro objects
  ```json
  {
    "data": [
      {
        "type": "macros",
        "id": "1",
        "attributes": {
          "name": "Attack Combo",
          "shout": true,
          "skillId1": 1000,
          "skillId2": 2000,
          "skillId3": 3000
        }
      }
    ]
  }
  ```
