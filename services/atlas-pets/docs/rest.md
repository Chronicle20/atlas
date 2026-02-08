# REST API

## Endpoints

### GET /api/pets/{petId}

Retrieves a pet by identifier. Includes temporal position data (x, y, stance, fh) from the in-memory registry.

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
      "cashId": 7000000,
      "templateId": 5000017,
      "name": "Mr. Roboto",
      "level": 10,
      "closeness": 100,
      "fullness": 100,
      "expiration": "2023-12-31T23:59:59Z",
      "ownerId": 1,
      "slot": 0,
      "x": 100,
      "y": 200,
      "stance": 0,
      "fh": 5,
      "excludes": [
        { "itemId": 2059000 }
      ],
      "flag": 0,
      "purchaseBy": 1
    }
  }
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid petId path parameter |
| 500 | Pet not found or internal error |

---

### GET /api/characters/{characterId}/pets

Retrieves all pets for a character. Includes temporal position data for each pet.

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
        "cashId": 7000000,
        "templateId": 5000017,
        "name": "Mr. Roboto",
        "level": 10,
        "closeness": 100,
        "fullness": 100,
        "expiration": "2023-12-31T23:59:59Z",
        "ownerId": 1,
        "slot": 0,
        "x": 100,
        "y": 200,
        "stance": 0,
        "fh": 5,
        "excludes": [],
        "flag": 0,
        "purchaseBy": 1
      }
    }
  ]
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid characterId path parameter |
| 500 | Internal error |

---

### POST /api/pets

Creates a new pet. The pet is persisted to the database and a CREATED status event is emitted via Kafka.

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
      "cashId": 7000000,
      "templateId": 5000017,
      "name": "Mr. Roboto",
      "level": 1,
      "closeness": 0,
      "fullness": 100,
      "expiration": "2023-12-31T23:59:59Z",
      "ownerId": 1,
      "slot": -1,
      "excludes": [],
      "flag": 0,
      "purchaseBy": 1
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
      "cashId": 7000000,
      "templateId": 5000017,
      "name": "Mr. Roboto",
      "level": 1,
      "closeness": 0,
      "fullness": 100,
      "expiration": "2023-12-31T23:59:59Z",
      "ownerId": 1,
      "slot": -1,
      "x": 0,
      "y": 0,
      "stance": 0,
      "fh": 1,
      "excludes": [],
      "flag": 0,
      "purchaseBy": 1
    }
  }
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid input model (JSON parse failure) |
| 500 | Internal error (creation failed or transform failed) |

---

### POST /api/characters/{characterId}/pets

Creates a new pet for a character. Behaves identically to `POST /api/pets`; the characterId path parameter is available but the ownerId from the request body is used.

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
      "cashId": 7000000,
      "templateId": 5000017,
      "name": "Mr. Roboto",
      "level": 1,
      "closeness": 0,
      "fullness": 100,
      "expiration": "2023-12-31T23:59:59Z",
      "ownerId": 1,
      "slot": -1,
      "excludes": [],
      "flag": 0,
      "purchaseBy": 1
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
      "cashId": 7000000,
      "templateId": 5000017,
      "name": "Mr. Roboto",
      "level": 1,
      "closeness": 0,
      "fullness": 100,
      "expiration": "2023-12-31T23:59:59Z",
      "ownerId": 1,
      "slot": -1,
      "x": 0,
      "y": 0,
      "stance": 0,
      "fh": 1,
      "excludes": [],
      "flag": 0,
      "purchaseBy": 1
    }
  }
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid input model (JSON parse failure) |
| 500 | Internal error (creation failed or transform failed) |
