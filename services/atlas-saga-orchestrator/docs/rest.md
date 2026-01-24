# REST API

## Endpoints

### GET /api/sagas

Returns all sagas for the current tenant.

#### Parameters

None.

#### Request Model

None.

#### Response Model

JSON:API collection of saga resources.

```json
{
  "data": [
    {
      "type": "sagas",
      "id": "uuid-string",
      "attributes": {
        "transactionId": "uuid-string",
        "sagaType": "inventory_transaction",
        "initiatedBy": "string",
        "steps": [
          {
            "stepId": "string",
            "status": "pending",
            "action": "award_asset",
            "payload": {},
            "createdAt": "2023-01-01T00:00:00Z",
            "updatedAt": "2023-01-01T00:00:00Z"
          }
        ]
      }
    }
  ]
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 500 | Failed to retrieve sagas |

---

### GET /api/sagas/{transactionId}

Returns a saga by its transaction ID.

#### Parameters

| Parameter | Type | Location | Required | Description |
|-----------|------|----------|----------|-------------|
| transactionId | uuid | path | yes | Transaction ID of the saga |

#### Request Model

None.

#### Response Model

JSON:API resource representing a saga.

```json
{
  "data": {
    "type": "sagas",
    "id": "uuid-string",
    "attributes": {
      "transactionId": "uuid-string",
      "sagaType": "inventory_transaction",
      "initiatedBy": "string",
      "steps": [
        {
          "stepId": "string",
          "status": "completed",
          "action": "award_asset",
          "payload": {
            "characterId": 12345,
            "item": {
              "templateId": 2000,
              "quantity": 1
            }
          },
          "createdAt": "2023-01-01T00:00:00Z",
          "updatedAt": "2023-01-01T00:00:00Z"
        }
      ]
    }
  }
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 500 | Failed to retrieve saga |
| 500 | Saga not found |

---

### POST /api/sagas

Creates a new saga.

#### Parameters

None.

#### Request Model

JSON:API resource representing a saga.

```json
{
  "data": {
    "type": "sagas",
    "attributes": {
      "transactionId": "uuid-string (optional, auto-generated if omitted)",
      "sagaType": "inventory_transaction",
      "initiatedBy": "string",
      "steps": [
        {
          "stepId": "step_1",
          "status": "pending",
          "action": "award_asset",
          "payload": {
            "characterId": 12345,
            "item": {
              "templateId": 2000,
              "quantity": 1
            }
          }
        }
      ]
    }
  }
}
```

#### Response Model

JSON:API resource representing the created saga.

```json
{
  "data": {
    "type": "sagas",
    "id": "uuid-string",
    "attributes": {
      "transactionId": "uuid-string",
      "sagaType": "inventory_transaction",
      "initiatedBy": "string",
      "steps": [
        {
          "stepId": "step_1",
          "status": "pending",
          "action": "award_asset",
          "payload": {
            "characterId": 12345,
            "item": {
              "templateId": 2000,
              "quantity": 1
            }
          },
          "createdAt": "2023-01-01T00:00:00Z",
          "updatedAt": "2023-01-01T00:00:00Z"
        }
      ]
    }
  }
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Failed to extract saga from request |
| 500 | Failed to create saga |
| 500 | Failed to retrieve created saga |
| 500 | Failed to transform saga |

## Required Headers

All endpoints require tenant identification headers:

| Header | Description |
|--------|-------------|
| TENANT_ID | UUID identifying the tenant |
| REGION | Region code (e.g., GMS) |
| MAJOR_VERSION | Major version number |
| MINOR_VERSION | Minor version number |
