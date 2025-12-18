# atlas-notes
Mushroom game notes Service

## Overview

A RESTful resource which provides notes services.

## Environment

### Logging and Tracing
- JAEGER_HOST - Jaeger [host]:[port]
- LOG_LEVEL - Logging level - Panic / Fatal / Error / Warn / Info / Debug / Trace

### Database
- DB_USER - Database user
- DB_PASSWORD - Database password
- DB_HOST - Database host
- DB_PORT - Database port
- DB_NAME - Database name

### Kafka
- BOOTSTRAP_SERVERS - Kafka bootstrap servers
- EVENT_TOPIC_NOTE_STATUS - Topic for note status events
- COMMAND_TOPIC_CHARACTER_NOTE - Topic for character note commands

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

#### Get All Notes

```
GET /api/notes
```

Returns all notes in the system.

#### Get Notes for a Character

```
GET /api/characters/{characterId}/notes
```

Returns all notes for a specific character.

#### Get a Specific Note

```
GET /api/notes/{noteId}
```

Returns a specific note by ID.

#### Create a Note

```
POST /api/notes
```

Creates a new note. The request body should be a JSON:API document with the following attributes:

```json
{
  "data": {
    "type": "notes",
    "attributes": {
      "characterId": "123",
      "senderId": "456",
      "message": "This is a note message",
      "flag": "0"
    }
  }
}
```

#### Update a Note

```
PATCH /api/notes/{noteId}
```

Updates an existing note. The request body should be a JSON:API document with the attributes to update.

#### Delete a Note

```
DELETE /api/notes/{noteId}
```

Deletes a specific note by ID.

#### Delete All Notes for a Character

```
DELETE /api/characters/{characterId}/notes
```

Deletes all notes for a specific character.
