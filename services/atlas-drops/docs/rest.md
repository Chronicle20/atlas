# REST API

## Endpoints

### GET /api/drops/{id}

Retrieves a drop by ID.

#### Parameters

| Name | In | Type | Required | Description |
|------|-----|------|----------|-------------|
| id | path | uint32 | Yes | Drop identifier |

#### Request Headers

| Header | Required | Description |
|--------|----------|-------------|
| TENANT_ID | Yes | Tenant UUID |
| REGION | Yes | Region identifier |
| MAJOR_VERSION | Yes | Major version number |
| MINOR_VERSION | Yes | Minor version number |

#### Response Model

```json
{
  "data": {
    "type": "drops",
    "id": "1000000001",
    "attributes": {
      "worldId": 0,
      "channelId": 0,
      "mapId": 0,
      "itemId": 0,
      "equipmentId": 0,
      "quantity": 0,
      "meso": 0,
      "type": 0,
      "x": 0,
      "y": 0,
      "ownerId": 0,
      "ownerPartyId": 0,
      "dropTime": "2024-01-01T00:00:00Z",
      "dropperId": 0,
      "dropperX": 0,
      "dropperY": 0,
      "characterDrop": false,
      "mod": 0
    }
  }
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid drop ID format |
| 404 | Drop not found |
| 500 | Internal server error |

---

### GET /api/worlds/{worldId}/channels/{channelId}/maps/{mapId}/drops

Retrieves all drops for a specific map.

#### Parameters

| Name | In | Type | Required | Description |
|------|-----|------|----------|-------------|
| worldId | path | uint8 | Yes | World identifier |
| channelId | path | uint8 | Yes | Channel identifier |
| mapId | path | uint32 | Yes | Map identifier |

#### Request Headers

| Header | Required | Description |
|--------|----------|-------------|
| TENANT_ID | Yes | Tenant UUID |
| REGION | Yes | Region identifier |
| MAJOR_VERSION | Yes | Major version number |
| MINOR_VERSION | Yes | Minor version number |

#### Response Model

```json
{
  "data": [
    {
      "type": "drops",
      "id": "1000000001",
      "attributes": {
        "worldId": 0,
        "channelId": 0,
        "mapId": 0,
        "itemId": 0,
        "equipmentId": 0,
        "quantity": 0,
        "meso": 0,
        "type": 0,
        "x": 0,
        "y": 0,
        "ownerId": 0,
        "ownerPartyId": 0,
        "dropTime": "2024-01-01T00:00:00Z",
        "dropperId": 0,
        "dropperX": 0,
        "dropperY": 0,
        "characterDrop": false,
        "mod": 0
      }
    }
  ]
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid path parameter format |
| 404 | No drops found |
| 500 | Internal server error |
