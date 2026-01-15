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
