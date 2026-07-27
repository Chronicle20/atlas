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

Returns a page of notes in the tenant.

**Parameters:**
- page[number] (query, optional): int, default 1
- page[size] (query, optional): int, default 50, max 250

The legacy `limit` query parameter is rejected.

**Response Model:** Array of RestModel

JSON:API `meta` block:

| Field | Type | Description |
|-------|------|-------------|
| total | int | Total count of matching notes across all pages |
| page.number | int | Current page number |
| page.size | int | Current page size |
| page.last | int | Last page number |

**Error Conditions:**
- 400: Invalid page[number]/page[size] (non-integer, out of range, or legacy limit param used)
- 500: Internal server error

---

### GET /api/characters/{characterId}/notes

Returns a page of notes for a character.

**Parameters:**
- characterId (path, required): uint32
- page[number] (query, optional): int, default 1
- page[size] (query, optional): int, default 50, max 250

The legacy `limit` query parameter is rejected.

**Response Model:** Array of RestModel

JSON:API `meta` block:

| Field | Type | Description |
|-------|------|-------------|
| total | int | Total count of matching notes across all pages |
| page.number | int | Current page number |
| page.size | int | Current page size |
| page.last | int | Last page number |

**Error Conditions:**
- 400: Invalid characterId, or invalid page[number]/page[size] (non-integer, out of range, or legacy limit param used)
- 500: Internal server error

---

### GET /api/notes/{noteId}

Returns a specific note.

**Parameters:**
- noteId (path, required): uint32

**Response Model:** RestModel

**Error Conditions:**
- 400: Invalid noteId
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
- 400: Invalid request body, invalid noteId, or note ID mismatch
- 500: Internal server error

---

### DELETE /api/notes/{noteId}

Deletes a note.

**Parameters:**
- noteId (path, required): uint32

**Response Model:** None (204 No Content)

**Error Conditions:**
- 400: Invalid noteId
- 500: Internal server error

---

### DELETE /api/characters/{characterId}/notes

Deletes all notes for a character.

**Parameters:**
- characterId (path, required): uint32

**Response Model:** None (204 No Content)

**Error Conditions:**
- 400: Invalid characterId
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
