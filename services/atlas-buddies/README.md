# atlas-buddies
Mushroom game buddies Service

## Overview

A RESTful resource which provides buddies services.

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
- COMMAND_TOPIC_BUDDY_LIST - Kafka Topic for transmitting buddy list commands.
- COMMAND_TOPIC_INVITE - Kafka Topic for transmitting invite commands.
- EVENT_TOPIC_BUDDY_LIST_STATUS - Kafka Topic for transmitting buddy list status events.
- EVENT_TOPIC_CASH_SHOP_STATUS - Kafka Topic for receiving cash shop status events.
- EVENT_TOPIC_CHARACTER_STATUS - Kafka Topic for receiving character status events.
- EVENT_TOPIC_INVITE_STATUS - Kafka Topic for receiving invite status events.

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

#### [GET] Get Characters Buddy List

```/api/characters/{characterId}/buddy-list```

Example Response:
```json
{
  "data": {
    "type": "buddy-list",
    "id": "1",
    "attributes": {
      "characterId": 12345,
      "capacity": 50,
      "buddies": [
        {
          "characterId": 67890,
          "group": "Friends",
          "characterName": "MapleHero",
          "channelId": 1,
          "inShop": false,
          "pending": false
        },
        {
          "characterId": 54321,
          "group": "Guild",
          "characterName": "MapleWarrior",
          "channelId": 2,
          "inShop": true,
          "pending": false
        }
      ]
    }
  }
}
```

#### [POST] Create Characters Buddy List

```/api/characters/{characterId}/buddy-list```

Example Request:
```json
{
  "data": {
    "type": "buddy-list",
    "attributes": {
      "capacity": 50
    }
  }
}
```

Response: 202 Accepted (No content)

#### [GET] Get Buddies in Character's Buddy List

```/api/characters/{characterId}/buddy-list/buddies```

Example Response:
```json
{
  "data": [
    {
      "type": "buddies",
      "id": "67890",
      "attributes": {
        "characterId": 67890,
        "group": "Friends",
        "characterName": "MapleHero",
        "channelId": 1,
        "inShop": false,
        "pending": false
      }
    },
    {
      "type": "buddies",
      "id": "54321",
      "attributes": {
        "characterId": 54321,
        "group": "Guild",
        "characterName": "MapleWarrior",
        "channelId": 2,
        "inShop": true,
        "pending": false
      }
    }
  ]
}
```

#### [POST] Add Buddy to Character's Buddy List

```/api/characters/{characterId}/buddy-list/buddies```

Example Request:
```json
{
  "data": {
    "type": "buddies",
    "attributes": {
      "characterId": 67890,
      "group": "Friends",
      "characterName": "MapleHero",
      "channelId": 1,
      "inShop": false,
      "pending": true
    }
  }
}
```

Response: 202 Accepted (No content)

## Kafka Commands

The buddy service supports several Kafka commands for server-to-server communication and administrative operations.

### INCREASE_CAPACITY Command

Increases a character's buddy list capacity. This command is typically used for premium features or administrative purposes.

**Topic:** `COMMAND_TOPIC_BUDDY_LIST`

**Command Structure:**
```json
{
  "worldId": 0,
  "characterId": 12345,
  "type": "INCREASE_CAPACITY",
  "body": {
    "newCapacity": 100
  }
}
```

**Parameters:**
- `worldId` (byte): The world/server ID where the character resides
- `characterId` (uint32): The ID of the character whose capacity should be increased
- `newCapacity` (byte): The new capacity value (must be greater than current capacity)

**Validation Rules:**
- The character must have an existing buddy list
- The new capacity must be strictly greater than the current capacity
- The character must exist in the system

**Status Events Emitted:**

**Success - CAPACITY_CHANGE Event:**
```json
{
  "worldId": 0,
  "characterId": 12345,
  "type": "CAPACITY_CHANGE",
  "body": {
    "capacity": 100
  }
}
```

**Failure - ERROR Events:**
```json
{
  "worldId": 0,
  "characterId": 12345,
  "type": "ERROR",
  "body": {
    "error": "INVALID_CAPACITY"  // or "CHARACTER_NOT_FOUND" or "UNKNOWN_ERROR"
  }
}
```

**Error Types:**
- `INVALID_CAPACITY`: New capacity is not greater than current capacity
- `CHARACTER_NOT_FOUND`: Character's buddy list does not exist
- `UNKNOWN_ERROR`: Unexpected system error occurred

**Usage Example:**
This command would typically be triggered by:
- Cash shop purchases for buddy list expansions
- Administrative tools for customer support
- Game events that reward increased buddy capacity
- Premium account benefits