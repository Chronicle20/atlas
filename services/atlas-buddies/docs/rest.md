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

Retrieves all buddies in a character's buddy list.

#### Parameters
| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | yes | Character ID |

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
  ]
}
```

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 404 Not Found | Buddy list does not exist for character |
| 500 Internal Server Error | Database or transformation error |

---

### POST /api/characters/{characterId}/buddy-list/buddies

Adds a buddy to a character's buddy list.

#### Parameters
| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | yes | Character ID |

#### Request Model
JSON:API resource type: `buddies`

```json
{
  "data": {
    "type": "buddies",
    "attributes": {
      "characterId": 67890,
      "group": "Friends",
      "characterName": "BuddyName",
      "channelId": 1,
      "inShop": false,
      "pending": true
    }
  }
}
```

#### Response Model
None. Returns 202 Accepted.

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 500 Internal Server Error | Failed to process request |
