# REST

## Endpoints

### GET /api/characters/{characterId}/buddy-list

Retrieves a character's buddy list.

#### Parameters
| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | yes | Character ID |

#### Request Model
None.

#### Response Model
JSON:API resource type: `buddy-list`

```json
{
  "data": {
    "type": "buddy-list",
    "id": "uuid",
    "attributes": {
      "characterId": 12345,
      "capacity": 50,
      "buddies": [
        {
          "characterId": 67890,
          "group": "Friends",
          "characterName": "BuddyName",
          "channelId": 1,
          "inShop": false,
          "pending": false
        }
      ]
    }
  }
}
```

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 404 Not Found | Buddy list does not exist for character |
| 500 Internal Server Error | Database or transformation error |

---

### POST /api/characters/{characterId}/buddy-list

Creates a buddy list for a character.

#### Parameters
| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | yes | Character ID |

#### Request Model
JSON:API resource type: `buddy-list`

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

#### Response Model
None. Returns 202 Accepted.

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 500 Internal Server Error | Failed to publish Kafka command |

---

### GET /api/characters/{characterId}/buddy-list/buddies

Retrieves all buddies in a character's buddy list. Results are paginated and sorted by `characterId` ascending.

#### Parameters
| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | yes | Character ID |
| page[number] | query | integer | no | Page number, minimum 1. Default 1. |
| page[size] | query | integer | no | Page size, 1 to 250. Default 250. |

The legacy `limit` query parameter is rejected.

#### Request Model
None.

#### Response Model
JSON:API resource type: `buddies`

```json
{
  "data": [
    {
      "type": "buddies",
      "id": "67890",
      "attributes": {
        "characterId": 67890,
        "group": "Friends",
        "characterName": "BuddyName",
        "channelId": 1,
        "inShop": false,
        "pending": false
      }
    }
  ],
  "meta": {
    "total": 1,
    "page": {
      "number": 1,
      "size": 250,
      "last": 1
    }
  },
  "links": {
    "self": "...",
    "first": "...",
    "last": "..."
  }
}
```

`links` also includes `prev` when not on the first page, and `next` when not on the last page.

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 400 Bad Request | `page[number]` or `page[size]` is non-integer, out of range, or the `limit` parameter is present |
| 404 Not Found | Buddy list does not exist for character |
| 500 Internal Server Error | Database or transformation error |

