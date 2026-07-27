# Summon REST API

## Endpoints

### GET /api/summons/{summonId}

Retrieves a summon by its id.

**Parameters:**

| Name | In | Type | Required | Description |
|------|-----|------|----------|-------------|
| summonId | path | uint32 | yes | Summon id |

**Headers:**

| Name | Required | Description |
|------|----------|-------------|
| TENANT_ID | yes | Tenant identifier |
| REGION | yes | Region code |
| MAJOR_VERSION | yes | Major version |
| MINOR_VERSION | yes | Minor version |

**Response Model:**

```json
{
  "data": {
    "type": "summons",
    "id": "1000000001",
    "attributes": {
      "ownerCharacterId": 0,
      "skillId": 0,
      "skillLevel": 0,
      "summonType": "PUPPET",
      "movementType": 0,
      "x": 0,
      "y": 0,
      "hp": 0,
      "maxHp": 0,
      "expiresAt": 0,
      "worldId": 0,
      "channelId": 0,
      "mapId": 0,
      "instance": "00000000-0000-0000-0000-000000000000"
    }
  }
}
```

`expiresAt` is in Unix milliseconds.

**Error Conditions:**

| Status | Condition |
|--------|-----------|
| 404 | Summon not found |
| 500 | Internal error building the response model |

---

### GET /api/worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/summons

Retrieves all summons in a map instance, paginated and sorted by ascending id.

**Parameters:**

| Name | In | Type | Required | Description |
|------|-----|------|----------|-------------|
| worldId | path | world.Id | yes | World identifier |
| channelId | path | channel.Id | yes | Channel identifier |
| mapId | path | \_map.Id | yes | Map identifier |
| instanceId | path | uuid | yes | Instance identifier |
| page[number] | query | int | no | Page number |
| page[size] | query | int | no | Page size, capped at the paginate package's max page size |

**Headers:**

| Name | Required | Description |
|------|----------|-------------|
| TENANT_ID | yes | Tenant identifier |
| REGION | yes | Region code |
| MAJOR_VERSION | yes | Major version |
| MINOR_VERSION | yes | Minor version |

**Response Model:**

```json
{
  "data": [
    {
      "type": "summons",
      "id": "1000000001",
      "attributes": {
        "ownerCharacterId": 0,
        "skillId": 0,
        "skillLevel": 0,
        "summonType": "PUPPET",
        "movementType": 0,
        "x": 0,
        "y": 0,
        "hp": 0,
        "maxHp": 0,
        "expiresAt": 0,
        "worldId": 0,
        "channelId": 0,
        "mapId": 0,
        "instance": "00000000-0000-0000-0000-000000000000"
      }
    }
  ]
}
```

**Error Conditions:**

| Status | Condition |
|--------|-----------|
| 400 | Invalid page[number]/page[size], or invalid path parameter format |
| 500 | Internal error retrieving summons or building the response model |
