# REST API

## Headers

All requests require tenant identification headers:

```
TENANT_ID: <uuid>
REGION: <string>
MAJOR_VERSION: <number>
MINOR_VERSION: <number>
```

## Endpoints

### GET /api/notes

Returns all notes in the tenant.

**Parameters:** None

**Response Model:** Array of RestModel

**Error Conditions:**
- 500: Internal server error

---

### GET /api/characters/{characterId}/notes

Returns all notes for a character.

**Parameters:**
- characterId (path, required): uint32

**Response Model:** Array of RestModel

**Error Conditions:**
- 500: Internal server error

---

### GET /api/notes/{noteId}

Returns a specific note.

**Parameters:**
- noteId (path, required): uint32

**Response Model:** RestModel

**Error Conditions:**
- 500: Internal server error

---

### POST /api/notes

Creates a new note.

**Request Model:** RestModel

```json
{
  "data": {
    "type": "notes",
    "attributes": {
      "characterId": 123,
      "senderId": 456,
      "message": "Note message",
      "flag": 0
    }
  }
}
```

**Response Model:** RestModel

**Error Conditions:**
- 400: Invalid request body or missing required fields
- 500: Internal server error

---

### PATCH /api/notes/{noteId}

Updates an existing note.

**Parameters:**
- noteId (path, required): uint32

**Request Model:** RestModel

**Response Model:** RestModel

**Error Conditions:**
- 400: Invalid request body or note ID mismatch
- 500: Internal server error

---

### DELETE /api/notes/{noteId}

Deletes a note.

**Parameters:**
- noteId (path, required): uint32

**Response Model:** None (204 No Content)

**Error Conditions:**
- 500: Internal server error

---

### DELETE /api/characters/{characterId}/notes

Deletes all notes for a character.

**Parameters:**
- characterId (path, required): uint32

**Response Model:** None (204 No Content)

**Error Conditions:**
- 500: Internal server error

## Resource Model

### RestModel

JSON:API resource type: `notes`

| Attribute | Type | Description |
|-----------|------|-------------|
| characterId | uint32 | ID of the character who owns the note |
| senderId | uint32 | ID of the character who sent the note |
| message | string | Note content |
| flag | byte | Note flag |
| timestamp | time.Time | When the note was created |
