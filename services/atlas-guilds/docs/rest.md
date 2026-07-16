# REST

## Endpoints

### GET /api/guilds

Retrieves all guilds. Supports at most one of the `filter[members.id]` or `filter[name]` query parameters; if neither is supplied, all guilds are returned.

**Parameters**

| Name | Location | Required | Description |
|------|----------|----------|-------------|
| `filter[members.id]` | query | No | Filter by member character ID |
| `filter[name]` | query | No | Filter by guild name (substring match) |
| `page[number]` | query | No | Page number (default 1) |
| `page[size]` | query | No | Page size (default 50, max 250) |

The legacy `limit` query parameter is rejected.

**Request Model**

None.

**Response Model**

JSON:API response with resource type `guilds`.

```json
{
  "data": [
    {
      "type": "guilds",
      "id": "123",
      "attributes": {
        "worldId": 0,
        "name": "GuildName",
        "notice": "Guild notice text",
        "points": 0,
        "capacity": 30,
        "logo": 0,
        "logoColor": 0,
        "logoBackground": 0,
        "logoBackgroundColor": 0,
        "leaderId": 456,
        "members": [
          {
            "characterId": 456,
            "name": "CharacterName",
            "jobId": 100,
            "level": 50,
            "title": 1,
            "online": true,
            "allianceTitle": 5
          }
        ],
        "titles": [
          {
            "name": "Master",
            "index": 1
          }
        ]
      }
    }
  ],
  "meta": {},
  "links": {}
}
```

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 400 | `filter[members.id]` value is not an integer |
| 400 | `filter[name]` value is empty |
| 400 | Invalid `page[number]`/`page[size]` (non-integer, out of range, or legacy `limit` param used) |
| 500 | Database error |

---

### GET /api/guilds/{guildId}

Retrieves a specific guild by ID.

**Parameters**

| Name | Location | Required | Description |
|------|----------|----------|-------------|
| `guildId` | path | Yes | Guild identifier |

**Request Model**

None.

**Response Model**

JSON:API response with resource type `guilds`.

```json
{
  "data": {
    "type": "guilds",
    "id": "123",
    "attributes": {
      "worldId": 0,
      "name": "GuildName",
      "notice": "Guild notice text",
      "points": 0,
      "capacity": 30,
      "logo": 0,
      "logoColor": 0,
      "logoBackground": 0,
      "logoBackgroundColor": 0,
      "leaderId": 456,
      "members": [],
      "titles": []
    }
  }
}
```

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 400 | `guildId` is not an integer |
| 500 | Guild not found or database error |

---

### GET /api/guilds/{guildId}/threads

Retrieves all threads for a guild.

**Parameters**

| Name | Location | Required | Description |
|------|----------|----------|-------------|
| `guildId` | path | Yes | Guild identifier |
| `page[number]` | query | No | Page number (default 1) |
| `page[size]` | query | No | Page size (default 50, max 250) |

The legacy `limit` query parameter is rejected.

**Request Model**

None.

**Response Model**

JSON:API response with resource type `threads`.

```json
{
  "data": [
    {
      "type": "threads",
      "id": "1",
      "attributes": {
        "posterId": 456,
        "title": "Thread Title",
        "message": "Thread content",
        "emoticonId": 0,
        "notice": false,
        "replies": [
          {
            "id": 1,
            "posterId": 789,
            "message": "Reply content",
            "createdAt": "2024-01-01T00:00:00Z"
          }
        ],
        "createdAt": "2024-01-01T00:00:00Z"
      }
    }
  ],
  "meta": {},
  "links": {}
}
```

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 400 | `guildId` is not an integer |
| 400 | Invalid `page[number]`/`page[size]` (non-integer, out of range, or legacy `limit` param used) |
| 500 | Database error |

---

### GET /api/guilds/{guildId}/threads/{threadId}

Retrieves a specific thread by ID.

**Parameters**

| Name | Location | Required | Description |
|------|----------|----------|-------------|
| `guildId` | path | Yes | Guild identifier |
| `threadId` | path | Yes | Thread identifier |

**Request Model**

None.

**Response Model**

JSON:API response with resource type `threads`.

```json
{
  "data": {
    "type": "threads",
    "id": "1",
    "attributes": {
      "posterId": 456,
      "title": "Thread Title",
      "message": "Thread content",
      "emoticonId": 0,
      "notice": false,
      "replies": [],
      "createdAt": "2024-01-01T00:00:00Z"
    }
  }
}
```

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 400 | `guildId` or `threadId` is not an integer |
| 500 | Thread not found or database error |
