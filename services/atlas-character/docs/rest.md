# REST

## Endpoints

### GET /characters

Retrieves characters based on query parameters. `accountId` and `worldId` must both be supplied to filter by account in world; `name` filters by name independently; if neither pairing is supplied, all characters (for the tenant) are returned. Results are paginated.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| accountId | query | uint32 | conditional | Account ID (requires worldId; ignored unless worldId is also present) |
| worldId | query | uint32 | conditional | World ID (requires accountId; ignored unless accountId is also present) |
| name | query | string | conditional | Character name (case-insensitive exact match) |
| page[number] | query | int | no | Page number, 1-based (default 1) |
| page[size] | query | int | no | Page size (default 50, max 250) |
| include | query | string | no | Accepted, currently has no effect on the response |

#### Request Model
None

#### Response Model
```json
{
  "data": [
    {
      "type": "characters",
      "id": "string",
      "attributes": {
        "accountId": 0,
        "worldId": 0,
        "name": "string",
        "level": 0,
        "experience": 0,
        "gachaponExperience": 0,
        "strength": 0,
        "dexterity": 0,
        "intelligence": 0,
        "luck": 0,
        "hp": 0,
        "maxHp": 0,
        "mp": 0,
        "maxMp": 0,
        "meso": 0,
        "hpMpUsed": 0,
        "jobId": 0,
        "skinColor": 0,
        "gender": 0,
        "fame": 0,
        "hair": 0,
        "face": 0,
        "ap": 0,
        "sp": "string",
        "spawnPoint": 0,
        "gm": 0,
        "x": 0,
        "y": 0,
        "fh": 0,
        "stance": 0
      }
    }
  ],
  "meta": {
    "total": 0,
    "page": { "number": 0, "size": 0, "last": 0 }
  },
  "links": {
    "self": "string",
    "first": "string",
    "prev": "string",
    "next": "string",
    "last": "string"
  }
}
```

`mapId` and `instance` are not part of the response; atlas-maps owns character location state.

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid page[number] or page[size] |
| 500 | Database error |

---

### GET /characters/name-validity

Checks whether a character name is valid (length, character set) and unique within a world. Not JSON:API enveloped.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| name | query | string | yes | Candidate character name |
| worldId | query | uint8 | yes | World ID to check uniqueness against |

#### Request Model
None

#### Response Model
```json
{
  "valid": true,
  "reason": "string",
  "detail": "string"
}
```

`reason` is one of `length`, `regex`, `duplicate`; omitted when `valid` is true.

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Missing or invalid name or worldId |
| 500 | Lookup error |

---

### GET /characters/{characterId}

Retrieves a character by ID.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | yes | Character ID |
| include | query | string | no | Accepted, currently has no effect on the response |

#### Request Model
None

#### Response Model
```json
{
  "data": {
    "type": "characters",
    "id": "string",
    "attributes": {
      "accountId": 0,
      "worldId": 0,
      "name": "string",
      "level": 0,
      "experience": 0,
      "gachaponExperience": 0,
      "strength": 0,
      "dexterity": 0,
      "intelligence": 0,
      "luck": 0,
      "hp": 0,
      "maxHp": 0,
      "mp": 0,
      "maxMp": 0,
      "meso": 0,
      "hpMpUsed": 0,
      "jobId": 0,
      "skinColor": 0,
      "gender": 0,
      "fame": 0,
      "hair": 0,
      "face": 0,
      "ap": 0,
      "sp": "string",
      "spawnPoint": 0,
      "gm": 0,
      "x": 0,
      "y": 0,
      "fh": 0,
      "stance": 0
    }
  }
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid characterId format |
| 404 | Character not found |
| 500 | Database error |

---

### POST /characters

Creates a new character.

#### Parameters
None

#### Request Model
```json
{
  "data": {
    "type": "characters",
    "attributes": {
      "accountId": 0,
      "worldId": 0,
      "name": "string",
      "level": 0,
      "strength": 0,
      "dexterity": 0,
      "intelligence": 0,
      "luck": 0,
      "maxHp": 0,
      "maxMp": 0,
      "jobId": 0,
      "skinColor": 0,
      "gender": 0,
      "hair": 0,
      "face": 0,
      "mapId": 0,
      "gm": 0
    }
  }
}
```

#### Response Model
```json
{
  "data": {
    "type": "characters",
    "id": "string",
    "attributes": {
      "accountId": 0,
      "worldId": 0,
      "name": "string",
      "level": 0,
      "experience": 0,
      "gachaponExperience": 0,
      "strength": 0,
      "dexterity": 0,
      "intelligence": 0,
      "luck": 0,
      "hp": 0,
      "maxHp": 0,
      "mp": 0,
      "maxMp": 0,
      "meso": 0,
      "hpMpUsed": 0,
      "jobId": 0,
      "skinColor": 0,
      "gender": 0,
      "fame": 0,
      "hair": 0,
      "face": 0,
      "ap": 0,
      "sp": "string",
      "spawnPoint": 0,
      "gm": 0,
      "x": 0,
      "y": 0,
      "fh": 0,
      "stance": 0
    }
  }
}
```

`mapId` on the request is the spawn map forwarded to atlas-maps on character creation; it is create-time input only and is not part of the response.

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Blocked name or invalid level |
| 500 | Database error |

---

### PATCH /characters/{characterId}

Updates a character.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | yes | Character ID |

#### Request Model
```json
{
  "data": {
    "type": "characters",
    "id": "string",
    "attributes": {
      "name": "string",
      "hair": 0,
      "face": 0,
      "gender": 0,
      "skinColor": 0,
      "gm": 0
    }
  }
}
```

Only the fields above are applied; other RestModel attributes present in the body are ignored. `name`/`hair`/`face`/`skinColor` values of `""`/`0` are treated as "no change requested" for that field (gender has no such sentinel: any value differing from the current gender is validated and applied). `gm` is nullable; omitting it (`null`) means no change, while an explicit `0` demotes.

#### Response Model
None (204 No Content)

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid or duplicate name |
| 400 | Invalid hair ID |
| 400 | Invalid face ID |
| 400 | Invalid gender value |
| 400 | Invalid skin color value |
| 400 | Invalid GM value |
| 404 | Character not found |
| 500 | Database error |

---

### DELETE /characters/{characterId}

Deletes a character.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | yes | Character ID |

#### Request Model
None

#### Response Model
None (204 No Content)

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid characterId format |
| 500 | Database error |

---

### GET /characters/{characterId}/sessions

Retrieves session history for a character. Results are paginated.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | yes | Character ID |
| since | query | string | no | Start time (unix timestamp or RFC3339). Defaults to 24 hours ago |
| page[number] | query | int | no | Page number, 1-based (default 1) |
| page[size] | query | int | no | Page size (default 50, max 250) |

#### Request Model
None

#### Response Model
```json
{
  "data": [
    {
      "type": "sessions",
      "id": "string",
      "attributes": {
        "characterId": 0,
        "worldId": 0,
        "channelId": 0,
        "loginTime": "2006-01-02T15:04:05Z",
        "logoutTime": "2006-01-02T15:04:05Z"
      }
    }
  ],
  "meta": {
    "total": 0,
    "page": { "number": 0, "size": 0, "last": 0 }
  },
  "links": {
    "self": "string",
    "first": "string",
    "prev": "string",
    "next": "string",
    "last": "string"
  }
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid characterId or since format |
| 400 | Invalid page[number] or page[size] |
| 500 | Database error |

---

### GET /characters/{characterId}/sessions/playtime

Computes total playtime for a character.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | yes | Character ID |
| since | query | string | no | Start time (unix timestamp or RFC3339). Defaults to 24 hours ago |

#### Request Model
None

#### Response Model
```json
{
  "data": {
    "type": "playtime",
    "id": "string",
    "attributes": {
      "characterId": 0,
      "totalSeconds": 0,
      "formattedTime": "00:00:00"
    }
  }
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid characterId or since format |
| 500 | Database error |

---

### GET /characters/{characterId}/locations/{type}

Retrieves a saved location by type.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | yes | Character ID |
| type | path | string | yes | Location type |

#### Request Model
None

#### Response Model
```json
{
  "data": {
    "type": "saved-locations",
    "id": "string",
    "attributes": {
      "characterId": 0,
      "locationType": "string",
      "mapId": 0,
      "portalId": 0
    }
  }
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid characterId or missing type |
| 404 | Location not found |
| 500 | Database error |

---

### PUT /characters/{characterId}/locations/{type}

Creates or updates a saved location.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | yes | Character ID |
| type | path | string | yes | Location type |

#### Request Model
```json
{
  "data": {
    "type": "saved-locations",
    "attributes": {
      "mapId": 0,
      "portalId": 0
    }
  }
}
```

#### Response Model
```json
{
  "data": {
    "type": "saved-locations",
    "id": "string",
    "attributes": {
      "characterId": 0,
      "locationType": "string",
      "mapId": 0,
      "portalId": 0
    }
  }
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid characterId or missing type |
| 500 | Database error |

---

### DELETE /characters/{characterId}/locations/{type}

Deletes a saved location.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | yes | Character ID |
| type | path | string | yes | Location type |

#### Request Model
None

#### Response Model
```json
{
  "data": {
    "type": "saved-locations",
    "id": "string",
    "attributes": {
      "characterId": 0,
      "locationType": "string",
      "mapId": 0,
      "portalId": 0
    }
  }
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid characterId or missing type |
| 404 | Location not found |
| 500 | Database error |

---

### GET /characters/{characterId}/teleport-rock-maps

Retrieves a character's saved teleport-rock map lists (regular and VIP), unpadded, plus each list's capacity.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | yes | Character ID |

#### Request Model
None

#### Response Model
```json
{
  "data": {
    "type": "teleport-rock-maps",
    "id": "string",
    "attributes": {
      "regular": [0],
      "vip": [0],
      "regularCapacity": 0,
      "vipCapacity": 0
    }
  }
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid characterId |
| 500 | Database error |

---

### POST /characters/{characterId}/teleport-rock-maps

Registers the character's current map on the given list (`regular` or `vip`). Returns the updated lists.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | yes | Character ID |

#### Request Model
```json
{
  "data": {
    "type": "teleport-rock-maps",
    "attributes": {
      "list": "string",
      "mapId": 0
    }
  }
}
```

#### Response Model
```json
{
  "data": {
    "type": "teleport-rock-maps",
    "id": "string",
    "attributes": {
      "regular": [0],
      "vip": [0],
      "regularCapacity": 0,
      "vipCapacity": 0
    }
  }
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid characterId, unknown `list`, or map ineligible for registration |
| 409 | List is full, or map already present on the list |
| 500 | Database error |

---

### DELETE /characters/{characterId}/teleport-rock-maps/{list}/{mapId}

Removes a map from the given list (`regular` or `vip`) and compacts the remaining entries. Returns the updated lists.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | yes | Character ID |
| list | path | string | yes | `regular` or `vip` |
| mapId | path | uint32 | yes | Map ID to remove |

#### Request Model
None

#### Response Model
```json
{
  "data": {
    "type": "teleport-rock-maps",
    "id": "string",
    "attributes": {
      "regular": [0],
      "vip": [0],
      "regularCapacity": 0,
      "vipCapacity": 0
    }
  }
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid characterId, unknown `list`, or invalid mapId |
| 404 | Map not present on the list |
| 500 | Database error |
