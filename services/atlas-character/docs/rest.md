# REST

## Endpoints

### GET /characters

Retrieves characters based on query parameters.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| accountId | query | uint32 | conditional | Account ID (requires worldId) |
| worldId | query | uint32 | conditional | World ID |
| mapId | query | uint32 | conditional | Map ID (requires worldId) |
| name | query | string | conditional | Character name |
| include | query | string | no | Related resources to include |

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
        "mapId": 0,
        "instance": "00000000-0000-0000-0000-000000000000",
        "spawnPoint": 0,
        "gm": 0,
        "x": 0,
        "y": 0,
        "stance": 0
      }
    }
  ]
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid accountId or worldId format |
| 500 | Database error |

---

### GET /characters/{characterId}

Retrieves a character by ID.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | yes | Character ID |
| include | query | string | no | Related resources to include |

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
      "mapId": 0,
      "spawnPoint": 0,
      "gm": 0,
      "x": 0,
      "y": 0,
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
      "mapId": 0,
      "spawnPoint": 0,
      "gm": 0,
      "x": 0,
      "y": 0,
      "stance": 0
    }
  }
}
```

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
      "mapId": 0,
      "gm": 0
    }
  }
}
```

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

Retrieves session history for a character.

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
  ]
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid characterId or since format |
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
