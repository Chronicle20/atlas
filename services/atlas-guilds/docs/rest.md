# REST

## Endpoints

### GET /api/guilds

Retrieves all guilds.

**Parameters**

| Name | Location | Required | Description |
|------|----------|----------|-------------|
| `filter[members.id]` | query | No | Filter by member character ID |

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
  ]
}
```

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 400 | Invalid member ID filter value |
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
| 500 | Guild not found or database error |

---

### GET /api/guilds/{guildId}/threads

Retrieves all threads for a guild.

**Parameters**

| Name | Location | Required | Description |
|------|----------|----------|-------------|
| `guildId` | path | Yes | Guild identifier |

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
  ]
}
```

**Error Conditions**

| Status | Condition |
|--------|-----------|
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
| 500 | Thread not found or database error |
