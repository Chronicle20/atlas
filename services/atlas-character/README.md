# atlas-character
Mushroom game Character Service

## Overview

A RESTful resource which provides character services.

## Environment

- JAEGER_HOST - Jaeger [host]:[port]
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace
- DB_USER - Postgres user name
- DB_PASSWORD - Postgres user password
- DB_HOST - Postgres Database host
- DB_PORT - Postgres Database port
- DB_NAME - Postgres Database name
- BASE_SERVICE_URL - [scheme]://[host]:[port]/api/
- BOOTSTRAP_SERVERS - Kafka [host]:[port]
- COMMAND_TOPIC_CHARACTER - Kafka Topic for transmitting character commands
- COMMAND_TOPIC_EQUIP_ITEM - Kafka Topic for transmitting equip item commands
- COMMAND_TOPIC_UNEQUIP_ITEM - Kafka Topic for transmitting unequip item commands
- COMMAND_TOPIC_CHARACTER_MOVEMENT - Kafka Topic for transmitting character movement commands
- COMMAND_TOPIC_DROP - Kafka Topic for transmitting drop commands
- COMMAND_TOPIC_SKILL - Kafka Topic for transmitting skill commands
- COMMAND_TOPIC_INVENTORY - Kafka Topic for transmitting inventory commands
- EVENT_TOPIC_CHARACTER_STATUS - Kafka Topic for transmitting character status events
- EVENT_TOPIC_INVENTORY_CHANGED - Kafka Topic for transmitting inventory change events
- EVENT_TOPIC_SESSION_STATUS - Kafka Topic for capturing session events
- EVENT_TOPIC_CHARACTER_MOVEMENT - Kafka Topic for transmitting character movement events
- EVENT_TOPIC_DROP_STATUS - Kafka Topic for transmitting drop status events
- REST_PORT - Port number for the REST server

## API

### Header

All RESTful requests require the supplied header information to identify the server instance.

```
TENANT_ID:083839c6-c47c-42a6-9585-76492795d123
REGION:GMS
MAJOR_VERSION:83
MINOR_VERSION:1
```

### Character APIs

#### [GET] Get Characters - By Account and World
```/api/cos/characters?accountId={accountId}&worldId={worldId}```

#### [GET] Get Characters - By World and Map
```/api/cos/characters?worldId={worldId}&mapId={mapId}```

#### [GET] Get Characters - By Name
```/api/cos/characters?name={name}```

#### [GET] Get Character - By Id
```/api/cos/characters/{characterId}```

#### [POST] Create Character
```/api/cos/characters```

#### [DELETE] Delete Character
```/api/cos/characters/{characterId}```

#### [PATCH] Update Character
```/api/cos/characters/{characterId}```

### Inventory APIs

#### [GET] Get Item By Slot
```/api/cos/characters/{characterId}/inventories/{inventoryType}/items?slot={slot}```

#### [POST] Create Item
```/api/cos/characters/{characterId}/inventories/{inventoryType}/items```

### Equipment APIs

#### [POST] Equip Item
```/api/cos/characters/{characterId}/equipment/{slotType}/equipable```

#### [DELETE] Unequip Item
```/api/cos/characters/{characterId}/equipment/{slotType}/equipable```

### Response Format

All responses follow the JSON:API specification format. The service supports:
- Resource relationships
- Sparse fieldsets
- Includes for related resources
- Pagination

## Kafka Integration

The atlas-character service supports asynchronous character operations via Kafka messaging. This enables distributed workflows and event-driven architecture patterns.

### Character Creation Command

#### Topic
Characters can be created asynchronously by publishing to the `COMMAND_TOPIC_CHARACTER` topic.

#### Command Structure
```json
{
  "transactionId": "123e4567-e89b-12d3-a456-426614174000",
  "worldId": 1,
  "characterId": 0,
  "type": "CREATE_CHARACTER",
  "body": {
    "accountId": 12345,
    "worldId": 1,
    "name": "NewCharacter",
    "level": 1,
    "strength": 12,
    "dexterity": 5,
    "intelligence": 4,
    "luck": 4,
    "maxHp": 50,
    "maxMp": 25,
    "jobId": 0,
    "gender": 0,
    "hair": 30000,
    "face": 20000,
    "skinColor": 0,
    "mapId": 100000000
  }
}
```

#### Success Response
When character creation succeeds, a success event is emitted to `EVENT_TOPIC_CHARACTER_STATUS`:

```json
{
  "transactionId": "123e4567-e89b-12d3-a456-426614174000",
  "worldId": 1,
  "characterId": 1001,
  "type": "CREATED",
  "body": {
    "name": "NewCharacter"
  }
}
```

#### Error Response
When character creation fails, an error event is emitted to `EVENT_TOPIC_CHARACTER_STATUS`:

```json
{
  "transactionId": "123e4567-e89b-12d3-a456-426614174000",
  "worldId": 1,
  "characterId": 0,
  "type": "CREATION_FAILED",
  "body": {
    "name": "NewCharacter",
    "message": "Character name already exists"
  }
}
```

#### Error Scenarios
The `CREATION_FAILED` event is emitted in the following scenarios:

1. **Invalid Character Name**
   - Name is too long (exceeds character limit)
   - Name contains forbidden characters
   - Name is reserved or blocked

2. **Name Already Exists**
   - Character name is already taken in the specified world

3. **Invalid Level**
   - Level is below minimum (typically 1)
   - Level exceeds maximum allowed

4. **Database Persistence Failure**
   - Database connection issues
   - Transaction rollback
   - Constraint violations

5. **Validation Errors**
   - Invalid account ID
   - Invalid world ID
   - Invalid job ID
   - Invalid stat values (strength, dexterity, intelligence, luck)
   - Invalid appearance values (hair, face, skinColor)
   - Invalid map ID

### Event-Driven Architecture Benefits

- **Asynchronous Processing**: Character creation doesn't block the calling service
- **Observability**: Success and failure events provide comprehensive audit trails
- **Saga Support**: Enables distributed transaction patterns and compensation logic
- **Scalability**: Kafka enables horizontal scaling of character creation workloads
- **Resilience**: Failed operations are clearly identified with detailed error messages

## Character Update API

The character update API allows synchronous modification of character properties via a JSON:API-compliant PATCH request.

### [PATCH] Update Character

```
PATCH /api/cos/characters/{characterId}
```

Updates specific character properties. Only provided fields will be modified; unchanged fields remain unaffected.

#### Request Headers
All standard headers are required:
```
TENANT_ID:083839c6-c47c-42a6-9585-76492795d123
REGION:GMS
MAJOR_VERSION:83
MINOR_VERSION:1
Content-Type: application/json
```

#### Request Body
```json
{
  "data": {
    "type": "characters",
    "id": "1001",
    "attributes": {
      "name": "UpdatedName",
      "hair": 30100,
      "face": 20100,
      "gender": 1,
      "skinColor": 0,
      "mapId": 110000000,
      "gm": 1
    }
  }
}
```

#### Updatable Fields

| Field       | Type    | Description                                    | Validation                           |
|-------------|---------|------------------------------------------------|-------------------------------------|
| `name`      | string  | Character name                                 | Must be unique and valid format     |
| `hair`      | uint32  | Hair style ID                                 | Must be in valid hair ID range      |
| `face`      | uint32  | Face ID                                       | Must be in valid face ID range      |
| `gender`    | byte    | Character gender (0 = male, 1 = female)      | Must be 0 or 1                     |
| `skinColor` | byte    | Skin color ID                                 | Must be a valid skin color value    |
| `mapId`     | uint32  | Character's current map location              | Must be a valid map ID and accessible to character |
| `gm`        | int     | GM level (0 = not GM, 1+ = GM level)         | Must be non-negative integer        |

#### Response

**Success (204 No Content)**
```
HTTP/1.1 204 No Content
```

**Error (400 Bad Request)**
```json
{
  "error": {
    "status": 400,
    "title": "Bad Request",
    "detail": "Invalid character name format"
  }
}
```

**Error (404 Not Found)**
```json
{
  "error": {
    "status": 404,
    "title": "Not Found",
    "detail": "Character not found"
  }
}
```

**Error (409 Conflict)**
```json
{
  "error": {
    "status": 409,
    "title": "Conflict",
    "detail": "Character name already exists"
  }
}
```

#### Example Usage

Update character name, appearance, and location:
```bash
curl -X PATCH \
  -H "Content-Type: application/json" \
  -H "TENANT_ID:083839c6-c47c-42a6-9585-76492795d123" \
  -H "REGION:GMS" \
  -H "MAJOR_VERSION:83" \
  -H "MINOR_VERSION:1" \
  -d '{
    "data": {
      "type": "characters",
      "id": "1001",
      "attributes": {
        "name": "NewCharacterName",
        "hair": 30150,
        "face": 20120,
        "mapId": 110000000,
        "gm": 1
      }
    }
  }' \
  https://api.example.com/api/cos/characters/1001
```

#### Business Rules

- **Name Uniqueness**: Character names must be unique within the tenant/world context
- **Validation**: All field values are validated against game rules and constraints
- **Transactional**: Updates are applied atomically - either all changes succeed or none are applied
- **Audit Trail**: Character updates may trigger audit events for tracking changes
- **Immutable Fields**: Some character properties (like characterId, accountId) cannot be modified via this endpoint

### Character Update Events

When character properties are updated via the PATCH endpoint, specific events are emitted to the `EVENT_TOPIC_CHARACTER_STATUS` topic to provide detailed audit trails and enable downstream services to react to specific changes. Each modified field triggers its own dedicated event with old and new values for tracking purposes.

#### Emitted Events by Field

| Field Changed | Event Type | Event Body Structure | Description |
|---------------|------------|---------------------|-------------|
| `name` | `NAME_CHANGED` | `{"oldName": "OldName", "newName": "NewName"}` | Character name change with audit trail |
| `hair` | `HAIR_CHANGED` | `{"oldHair": 30000, "newHair": 30100}` | Hair style ID change tracking |
| `face` | `FACE_CHANGED` | `{"oldFace": 20000, "newFace": 20100}` | Face ID change tracking |
| `gender` | `GENDER_CHANGED` | `{"oldGender": 0, "newGender": 1}` | Gender change (0=male, 1=female) |
| `skinColor` | `SKIN_COLOR_CHANGED` | `{"oldSkinColor": 0, "newSkinColor": 1}` | Skin color ID change tracking |
| `mapId` | `MAP_CHANGED` | `{"channelId": 1, "oldMapId": 100000000, "targetMapId": 110000000, "targetPortalId": 0}` | Map location change (reuses existing event type) |
| `gm` | `GM_CHANGED` | `{"oldGm": false, "newGm": true}` | GM status change tracking |

#### Event Structure

All character update events follow the standard status event structure:

```json
{
  "transactionId": "123e4567-e89b-12d3-a456-426614174000",
  "worldId": 1,
  "characterId": 1001,
  "type": "NAME_CHANGED",
  "body": {
    "oldName": "OldCharacterName",
    "newName": "NewCharacterName"
  }
}
```

#### Multiple Field Updates

When multiple fields are updated in a single PATCH request, **multiple specific events are emitted** - one for each changed field. For example, updating both name and hair in one request will emit both a `NAME_CHANGED` event and a `HAIR_CHANGED` event.

Example PATCH request updating name and hair:
```json
{
  "data": {
    "type": "characters",
    "id": "1001",
    "attributes": {
      "name": "NewName",
      "hair": 30150
    }
  }
}
```

Results in two separate events:
1. `NAME_CHANGED` event with old/new name values
2. `HAIR_CHANGED` event with old/new hair ID values

#### Benefits for Downstream Services

- **Granular Reactivity**: Services can subscribe to specific change types instead of processing generic "UPDATED" events
- **Audit Trail**: Complete old/new value tracking for compliance and debugging
- **Selective Processing**: Different services can react to different field changes without unnecessary processing
- **Event Sourcing**: Detailed change history enables event replay and state reconstruction