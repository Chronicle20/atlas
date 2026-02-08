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
      "instance": "uuid",
      "itemId": 0,
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
      "mod": 0,
      "strength": 0,
      "dexterity": 0,
      "intelligence": 0,
      "luck": 0,
      "hp": 0,
      "mp": 0,
      "weaponAttack": 0,
      "magicAttack": 0,
      "weaponDefense": 0,
      "magicDefense": 0,
      "accuracy": 0,
      "avoidability": 0,
      "hands": 0,
      "speed": 0,
      "jump": 0,
      "slots": 0
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

### GET /api/worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/drops

Retrieves all drops for a specific map instance.

#### Parameters

| Name | In | Type | Required | Description |
|------|-----|------|----------|-------------|
| worldId | path | int | Yes | World identifier |
| channelId | path | int | Yes | Channel identifier |
| mapId | path | int | Yes | Map identifier |
| instanceId | path | uuid | Yes | Instance identifier (use 00000000-0000-0000-0000-000000000000 for non-instanced maps) |

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
        "instance": "uuid",
        "itemId": 0,
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
        "mod": 0,
        "strength": 0,
        "dexterity": 0,
        "intelligence": 0,
        "luck": 0,
        "hp": 0,
        "mp": 0,
        "weaponAttack": 0,
        "magicAttack": 0,
        "weaponDefense": 0,
        "magicDefense": 0,
        "accuracy": 0,
        "avoidability": 0,
        "hands": 0,
        "speed": 0,
        "jump": 0,
        "slots": 0
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
