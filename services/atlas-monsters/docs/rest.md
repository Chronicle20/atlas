# Monster REST API

## Endpoints

### GET /api/monsters/{monsterId}

Retrieves a monster by its unique ID.

**Parameters:**

| Name | In | Type | Required | Description |
|------|-----|------|----------|-------------|
| monsterId | path | uint32 | yes | Monster unique ID |

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
    "type": "monsters",
    "id": "1000000001",
    "attributes": {
      "worldId": 0,
      "channelId": 0,
      "mapId": 100000000,
      "instance": "00000000-0000-0000-0000-000000000000",
      "monsterId": 100100,
      "controlCharacterId": 12345,
      "controllerHasAggro": true,
      "x": 100,
      "y": -50,
      "fh": 1,
      "stance": 5,
      "team": -1,
      "maxHp": 1000,
      "hp": 750,
      "maxMp": 100,
      "mp": 100,
      "damageEntries": [
        {
          "characterId": 12345,
          "damage": 250
        }
      ],
      "statusEffects": [
        {
          "sourceSkillId": 0,
          "sourceSkillLevel": 0,
          "statuses": {
            "STATUS_TYPE": 0
          },
          "expiresAt": 0
        }
      ],
      "nextEligibleRepickAtMs": 0
    }
  }
}
```

`expiresAt` is in Unix milliseconds. `nextEligibleRepickAtMs` is omitted when zero.

**Error Conditions:**

| Status | Condition |
|--------|-----------|
| 400 | Invalid monsterId format |
| 404 | Monster not found |

---

### GET /api/worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/monsters

Retrieves all monsters in a map instance, sorted by ascending uniqueId, paginated.

**Parameters:**

| Name | In | Type | Required | Description |
|------|-----|------|----------|-------------|
| worldId | path | byte | yes | World identifier |
| channelId | path | byte | yes | Channel identifier |
| mapId | path | uint32 | yes | Map identifier |
| instanceId | path | uuid | yes | Instance identifier |
| page[number] | query | int | no | Page number (default 1) |
| page[size] | query | int | no | Page size (default and max 250) |

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
      "type": "monsters",
      "id": "1000000001",
      "attributes": {
        "worldId": 0,
        "channelId": 0,
        "mapId": 100000000,
        "instance": "00000000-0000-0000-0000-000000000000",
        "monsterId": 100100,
        "controlCharacterId": 12345,
        "controllerHasAggro": true,
        "x": 100,
        "y": -50,
        "fh": 1,
        "stance": 5,
        "team": -1,
        "maxHp": 1000,
        "hp": 1000,
        "maxMp": 100,
        "mp": 100,
        "damageEntries": [],
        "statusEffects": [],
        "nextEligibleRepickAtMs": 0
      }
    }
  ],
  "meta": {
    "total": 1,
    "page": {
      "number": 1,
      "size": 250,
      "last": 1
    }
  }
}
```

The response also includes JSON:API `links` (self/first/last, plus prev/next where applicable) for pagination.

**Error Conditions:**

| Status | Condition |
|--------|-----------|
| 400 | Invalid path parameter format, or invalid page[number]/page[size] |
| 500 | Internal error retrieving monsters |

---

### GET /api/worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/monsters/in-rect

Retrieves monsters in a map instance whose position falls within an inclusive rectangle, sorted by ascending squared distance from the rectangle center, paginated.

**Parameters:**

| Name | In | Type | Required | Description |
|------|-----|------|----------|-------------|
| worldId | path | byte | yes | World identifier |
| channelId | path | byte | yes | Channel identifier |
| mapId | path | uint32 | yes | Map identifier |
| instanceId | path | uuid | yes | Instance identifier |
| x1 | query | int16 | yes | Rectangle corner X |
| y1 | query | int16 | yes | Rectangle corner Y |
| x2 | query | int16 | yes | Opposite rectangle corner X |
| y2 | query | int16 | yes | Opposite rectangle corner Y |
| limit | query | uint32 | no | Maximum results to return (default 0 = no cap) |
| page[number] | query | int | no | Page number (default 1) |
| page[size] | query | int | no | Page size (default and max 250) |

Corners may be supplied in any order.

**Headers:**

| Name | Required | Description |
|------|----------|-------------|
| TENANT_ID | yes | Tenant identifier |
| REGION | yes | Region code |
| MAJOR_VERSION | yes | Major version |
| MINOR_VERSION | yes | Minor version |

**Response Model:**

Same shape as GET .../monsters, including the `meta` pagination block and JSON:API `links`.

**Error Conditions:**

| Status | Condition |
|--------|-----------|
| 400 | Invalid path parameter format; missing/invalid x1, y1, x2, y2; invalid page[number]/page[size] |
| 500 | Internal error retrieving monsters |

---

### POST /api/worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/monsters

Creates a monster in a map instance.

**Parameters:**

| Name | In | Type | Required | Description |
|------|-----|------|----------|-------------|
| worldId | path | byte | yes | World identifier |
| channelId | path | byte | yes | Channel identifier |
| mapId | path | uint32 | yes | Map identifier |
| instanceId | path | uuid | yes | Instance identifier |

**Headers:**

| Name | Required | Description |
|------|----------|-------------|
| TENANT_ID | yes | Tenant identifier |
| REGION | yes | Region code |
| MAJOR_VERSION | yes | Major version |
| MINOR_VERSION | yes | Minor version |

**Request Model:**

```json
{
  "data": {
    "type": "monsters",
    "attributes": {
      "monsterId": 100100,
      "x": 100,
      "y": -50,
      "fh": 1,
      "team": -1
    }
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| monsterId | uint32 | yes | Monster type identifier |
| x | int16 | yes | X coordinate |
| y | int16 | yes | Y coordinate |
| fh | int16 | yes | Foothold |
| team | int8 | yes | Team assignment |

**Response Model:**

Returns the created monster in the same format as GET /api/monsters/{monsterId}.

**Error Conditions:**

| Status | Condition |
|--------|-----------|
| 400 | Invalid request body or path parameters; monster information not found |
| 500 | Internal error creating monster |

---

### DELETE /api/worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/monsters

Destroys all monsters in a map instance.

**Parameters:**

| Name | In | Type | Required | Description |
|------|-----|------|----------|-------------|
| worldId | path | byte | yes | World identifier |
| channelId | path | byte | yes | Channel identifier |
| mapId | path | uint32 | yes | Map identifier |
| instanceId | path | uuid | yes | Instance identifier |

**Headers:**

| Name | Required | Description |
|------|----------|-------------|
| TENANT_ID | yes | Tenant identifier |
| REGION | yes | Region code |
| MAJOR_VERSION | yes | Major version |
| MINOR_VERSION | yes | Minor version |

**Response Model:**

No response body.

**Error Conditions:**

| Status | Condition |
|--------|-----------|
| 202 | Monsters destroyed successfully |
| 400 | Invalid path parameter format |
| 500 | Internal error destroying monsters |
