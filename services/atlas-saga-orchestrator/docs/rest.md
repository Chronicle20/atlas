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

Creates a new saga. If `transactionId` is omitted or nil, a UUID is auto-generated.

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

The `payload` field is action-specific. The service unmarshals the payload based on the `action` field using registered unmarshalers. Actions without a registered unmarshaler pass the payload through as-is.

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

## REST Models

### RestModel (Saga)

| Field | Type | JSON Key | Description |
|-------|------|----------|-------------|
| TransactionID | uuid.UUID | transactionId | Unique ID for the transaction |
| SagaType | Type | sagaType | Type of saga |
| InitiatedBy | string | initiatedBy | Who initiated the saga |
| Steps | []StepRestModel | steps | Ordered list of steps |

### StepRestModel

| Field | Type | JSON Key | Description |
|-------|------|----------|-------------|
| StepID | string | stepId | Unique ID for the step within the saga |
| Status | Status | status | Step status (pending, completed, failed) |
| Action | Action | action | Action to perform |
| Payload | interface{} | payload | Action-specific data |
| CreatedAt | string | createdAt | Creation timestamp (RFC3339) |
| UpdatedAt | string | updatedAt | Last update timestamp (RFC3339) |

### Payload Unmarshalers

The following actions have registered payload unmarshalers for the POST endpoint. Payloads for these actions are deserialized into their typed structs during extraction:

| Action | Payload Type |
|--------|-------------|
| award_inventory | AwardItemActionPayload |
| award_experience | AwardExperiencePayload |
| award_level | AwardLevelPayload |
| award_mesos | AwardMesosPayload |
| warp_to_random_portal | WarpToRandomPortalPayload |
| warp_to_portal | WarpToPortalPayload |
| destroy_asset | DestroyAssetPayload |
| destroy_asset_from_slot | DestroyAssetFromSlotPayload |

Actions not listed above pass the raw payload through without typed deserialization.

## Saga Types

| Type | Value |
|------|-------|
| InventoryTransaction | inventory_transaction |
| QuestReward | quest_reward |
| TradeTransaction | trade_transaction |
| CharacterCreation | character_creation |
| StorageOperation | storage_operation |
| CharacterRespawn | character_respawn |
| GachaponTransaction | gachapon_transaction |

## Step Statuses

| Status | Value |
|--------|-------|
| Pending | pending |
| Completed | completed |
| Failed | failed |
