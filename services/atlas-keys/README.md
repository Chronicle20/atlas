# atlas-keys
Mushroom game keys Service

## Overview

A RESTful resource which provides key binding services for characters. This service manages the key mappings for character controls.

## Environment Variables

### Database Configuration
- `DB_USER` - Postgres user name
- `DB_PASSWORD` - Postgres user password
- `DB_HOST` - Postgres Database host
- `DB_PORT` - Postgres Database port
- `DB_NAME` - Postgres Database name

### Kafka Configuration
- `BOOTSTRAP_SERVERS` - Kafka [host]:[port]
- `EVENT_TOPIC_CHARACTER_STATUS` - Kafka Topic for transmitting character status events

### Observability
- `JAEGER_HOST` - Jaeger [host]:[port]
- `LOG_LEVEL` - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace

## REST API

### Header

All RESTful requests require the supplied header information to identify the server instance.

```
TENANT_ID:083839c6-c47c-42a6-9585-76492795d123
REGION:GMS
MAJOR_VERSION:83
MINOR_VERSION:1
```

### Endpoints

#### [GET] Get Keys

```
GET /api/characters/{characterId}/keys
```

Retrieves all key bindings for a character.

**Response**: JSON:API formatted list of key bindings

#### [DELETE] Reset Keys

```
DELETE /api/characters/{characterId}/keys
```

Resets all key bindings for a character to default values.

**Response**: 200 OK on success

#### [PATCH] Update Key

```
PATCH /api/characters/{characterId}/keys/{keyId}
```

Updates a specific key binding for a character.

**Request Body**:
```json
{
  "type": 4,
  "action": 10
}
```

**Response**: 200 OK on success

## Kafka Message API

### Consumed Events

#### Character Status Events

**Topic**: Defined by `EVENT_TOPIC_CHARACTER_STATUS` environment variable

**Event Structure**:
```json
{
  "transactionId": "uuid-string",
  "characterId": 123,
  "type": "CREATED|DELETED",
  "worldId": 0,
  "body": {
    "name": "Example field for CREATED events"
  }
}
```

**Handled Event Types**:
- `CREATED`: Creates default key bindings for a new character
- `DELETED`: Deletes all key bindings for a deleted character
