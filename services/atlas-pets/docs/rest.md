# REST API

## Endpoints

### GET /api/pets/{petId}

Retrieves a pet by identifier.

#### Parameters

| Name | In | Type | Required | Description |
|------|-----|------|----------|-------------|
| petId | path | uint32 | yes | Pet identifier |

#### Request Headers

| Name | Required | Description |
|------|----------|-------------|
| TENANT_ID | yes | Tenant identifier |
| REGION | yes | Region code |
| MAJOR_VERSION | yes | Major version |
| MINOR_VERSION | yes | Minor version |

#### Response Model

```json
{
  "data": {
    "type": "pets",
    "id": "1",
    "attributes": {
      "cashId": 5000123,
      "templateId": 5000,
      "name": "Fluffy",
      "level": 10,
      "closeness": 100,
      "fullness": 100,
      "expiration": "2023-12-31T23:59:59Z",
      "ownerId": 54321,
      "slot": 0,
      "x": 100,
      "y": 200,
      "stance": 0,
      "fh": 5,
      "excludes": [
        { "itemId": 1000 }
      ],
      "flag": 0,
      "purchaseBy": 54321
    }
  }
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 500 | Pet not found or internal error |

---

### GET /api/characters/{characterId}/pets

Retrieves all pets for a character.

#### Parameters

| Name | In | Type | Required | Description |
|------|-----|------|----------|-------------|
| characterId | path | uint32 | yes | Character identifier |

#### Request Headers

| Name | Required | Description |
|------|----------|-------------|
| TENANT_ID | yes | Tenant identifier |
| REGION | yes | Region code |
| MAJOR_VERSION | yes | Major version |
| MINOR_VERSION | yes | Minor version |

#### Response Model

```json
{
  "data": [
    {
      "type": "pets",
      "id": "1",
      "attributes": {
        "cashId": 5000123,
        "templateId": 5000,
        "name": "Fluffy",
        "level": 10,
        "closeness": 100,
        "fullness": 100,
        "expiration": "2023-12-31T23:59:59Z",
        "ownerId": 54321,
        "slot": 0,
        "x": 100,
        "y": 200,
        "stance": 0,
        "fh": 5,
        "excludes": [],
        "flag": 0,
        "purchaseBy": 54321
      }
    }
  ]
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 500 | Internal error |

---

### POST /api/pets

Creates a new pet.

#### Request Headers

| Name | Required | Description |
|------|----------|-------------|
| TENANT_ID | yes | Tenant identifier |
| REGION | yes | Region code |
| MAJOR_VERSION | yes | Major version |
| MINOR_VERSION | yes | Minor version |
| Content-Type | yes | application/json |

#### Request Model

```json
{
  "data": {
    "type": "pets",
    "attributes": {
      "cashId": 5000123,
      "templateId": 5000,
      "name": "Fluffy",
      "level": 1,
      "closeness": 0,
      "fullness": 100,
      "expiration": "2023-12-31T23:59:59Z",
      "ownerId": 54321,
      "slot": -1,
      "excludes": [],
      "flag": 0,
      "purchaseBy": 54321
    }
  }
}
```

#### Response Model

```json
{
  "data": {
    "type": "pets",
    "id": "1",
    "attributes": {
      "cashId": 5000123,
      "templateId": 5000,
      "name": "Fluffy",
      "level": 1,
      "closeness": 0,
      "fullness": 100,
      "expiration": "2023-12-31T23:59:59Z",
      "ownerId": 54321,
      "slot": -1,
      "x": 0,
      "y": 0,
      "stance": 0,
      "fh": 1,
      "excludes": [],
      "flag": 0,
      "purchaseBy": 54321
    }
  }
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid input model |
| 500 | Internal error |

---

### POST /api/characters/{characterId}/pets

Creates a new pet for a character.

#### Parameters

| Name | In | Type | Required | Description |
|------|-----|------|----------|-------------|
| characterId | path | uint32 | yes | Character identifier |

#### Request Headers

| Name | Required | Description |
|------|----------|-------------|
| TENANT_ID | yes | Tenant identifier |
| REGION | yes | Region code |
| MAJOR_VERSION | yes | Major version |
| MINOR_VERSION | yes | Minor version |
| Content-Type | yes | application/json |

#### Request Model

```json
{
  "data": {
    "type": "pets",
    "attributes": {
      "cashId": 5000123,
      "templateId": 5000,
      "name": "Fluffy",
      "level": 1,
      "closeness": 0,
      "fullness": 100,
      "expiration": "2023-12-31T23:59:59Z",
      "slot": -1,
      "excludes": [],
      "flag": 0,
      "purchaseBy": 54321
    }
  }
}
```

#### Response Model

```json
{
  "data": {
    "type": "pets",
    "id": "1",
    "attributes": {
      "cashId": 5000123,
      "templateId": 5000,
      "name": "Fluffy",
      "level": 1,
      "closeness": 0,
      "fullness": 100,
      "expiration": "2023-12-31T23:59:59Z",
      "ownerId": 54321,
      "slot": -1,
      "x": 0,
      "y": 0,
      "stance": 0,
      "fh": 1,
      "excludes": [],
      "flag": 0,
      "purchaseBy": 54321
    }
  }
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid input model |
| 500 | Internal error |
