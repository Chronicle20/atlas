# atlas-character-factory

Mushroom game character-factory Service

## Overview

A RESTful resource which provides character-factory services using saga-based orchestration for character creation. The service validates character creation requests and delegates the actual creation process to the Atlas Saga Orchestrator.

## Environment

- JAEGER_HOST - Jaeger [host]:[port]
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- BOOTSTRAP_SERVERS - Kafka [host]:[port]
- BASE_SERVICE_URL - [scheme]://[host]:[port]/api/
- EVENT_TOPIC_CHARACTER_STATUS - Kafka Topic for transmitting character status events
- EVENT_TOPIC_INVENTORY_CHANGED - Kafka Topic for transmitting inventory change events
- COMMAND_TOPIC_SAGA - Kafka Topic for transmitting saga commands to orchestrator

## Character Creation Process

The character creation process utilizes saga-based orchestration:

1. **Validation Phase**: The service validates character creation requests against configured templates
2. **Saga Construction**: A `character_creation` saga is built with sequential steps:
   - `create_character` - Creates the base character entity
   - `award_asset` - Awards template-defined starting items
   - `create_and_equip_asset` - Creates and equips starting equipment (Top, Bottom, Shoes, Weapon)
   - `create_skill` - Creates starting skills for the character
3. **Saga Emission**: The saga is sent to the Atlas Saga Orchestrator via Kafka
4. **Response**: The client receives a `202 Accepted` response with a transaction ID for tracking

The orchestrator handles all cross-service coordination, ensuring atomicity and fault tolerance.

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

#### POST /api/characters/seed

Creates a new character using saga-based orchestration.

**Request Body:**
```json
{
  "accountId": 12345,
  "worldId": 0,
  "name": "TestCharacter",
  "gender": 0,
  "jobIndex": 1,
  "subJobIndex": 0,
  "face": 20000,
  "hair": 30000,
  "hairColor": 0,
  "skinColor": 0,
  "top": 1040002,
  "bottom": 1060002,
  "shoes": 1072001,
  "weapon": 1302000
}
```

**Response:**
- **Status**: `202 Accepted`
- **Body**:
```json
{
  "transactionId": "123e4567-e89b-12d3-a456-426614174000"
}
```

**Error Responses:**
- `400 Bad Request` - Invalid input data or template validation failure
- `500 Internal Server Error` - Configuration errors or saga creation failure

**Notes:**
- The transaction ID can be used to track saga progress (requires separate orchestrator API)
- All validation is performed against configured character templates
- Equipment and items are validated against job/gender-specific templates

## Configuration Notes

### GMS v12

| Job Index | Sub Job Index |        Job |
|-----------|:-------------:|-----------:|
| 1         |       0       | Adventurer |

### GMS v83

| Job Index | Sub Job Index |        Job |
|-----------|:-------------:|-----------:|
| 0         |       0       |     Cygnus |
| 1         |       0       | Adventurer |
| 2         |       0       |       Aran |

### GMS v87

| Job Index | Sub Job Index |        Job |
|-----------|:-------------:|-----------:|
| 0         |       0       |     Cygnus |
| 1         |       0       | Adventurer |
| 2         |       0       |       Aran |
| 3         |       0       |       Evan |

### GMS v92

| Job Index | Sub Job Index |        Job |
|-----------|:-------------:|-----------:|
| 0         |       0       |     Cygnus |
| 1         |       0       | Adventurer |
| 1         |       1       | Dual Blade |
| 2         |       0       |       Aran |
| 3         |       0       |       Evan |

### JMS v185

| Job Index | Sub Job Index |        Job |
|-----------|:-------------:|-----------:|
| 0         |       0       |     Cygnus |
| 1         |       0       | Adventurer |
| 1         |       1       | Dual Blade |
| 2         |       0       |       Aran |
| 3         |       0       |       Evan |