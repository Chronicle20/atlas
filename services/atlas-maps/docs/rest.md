# REST API

## Endpoints

### GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/characters

Returns character IDs present across all instances of the specified map.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| worldId | path | byte | yes | World identifier |
| channelId | path | byte | yes | Channel identifier |
| mapId | path | uint32 | yes | Map identifier |
| page[number] | query | int | no | Page number (default 1) |
| page[size] | query | int | no | Page size (default 250, max 250) |

#### Request Headers

| Name | Required | Description |
|------|----------|-------------|
| TENANT_ID | yes | Tenant UUID |
| REGION | yes | Region code |
| MAJOR_VERSION | yes | Major version |
| MINOR_VERSION | yes | Minor version |

#### Response Model

Paginated JSON:API array of character resources.

```
{
    "data": [
        {
            "type": "characters",
            "id": "<characterId>"
        }
    ]
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid worldId, channelId, mapId, or page[number]/page[size] |
| 500 | Failed to retrieve characters from registry |

### GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/characters

Returns character IDs present in the specified map instance.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| worldId | path | byte | yes | World identifier |
| channelId | path | byte | yes | Channel identifier |
| mapId | path | uint32 | yes | Map identifier |
| instanceId | path | uuid | yes | Instance identifier (use 00000000-0000-0000-0000-000000000000 for non-instanced maps) |
| page[number] | query | int | no | Page number (default 1) |
| page[size] | query | int | no | Page size (default 250, max 250) |

#### Request Headers

| Name | Required | Description |
|------|----------|-------------|
| TENANT_ID | yes | Tenant UUID |
| REGION | yes | Region code |
| MAJOR_VERSION | yes | Major version |
| MINOR_VERSION | yes | Minor version |

#### Response Model

Paginated JSON:API array of character resources.

```
{
    "data": [
        {
            "type": "characters",
            "id": "<characterId>"
        }
    ]
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid worldId, channelId, mapId, instanceId, or page[number]/page[size] |
| 500 | Failed to retrieve characters from registry |

### GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/weather

Returns the active weather effect in the specified map instance.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| worldId | path | byte | yes | World identifier |
| channelId | path | byte | yes | Channel identifier |
| mapId | path | uint32 | yes | Map identifier |
| instanceId | path | uuid | yes | Instance identifier (use 00000000-0000-0000-0000-000000000000 for non-instanced maps) |

#### Request Headers

| Name | Required | Description |
|------|----------|-------------|
| TENANT_ID | yes | Tenant UUID |
| REGION | yes | Region code |
| MAJOR_VERSION | yes | Major version |
| MINOR_VERSION | yes | Minor version |

#### Response Model

JSON:API single weather resource.

```
{
    "data": {
        "type": "weather",
        "id": "<itemId>",
        "attributes": {
            "itemId": 5120000,
            "message": "A storm is brewing..."
        }
    }
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid worldId, channelId, mapId, or instanceId |
| 404 | No active weather effect in map |
| 500 | Failed to create REST model |

### GET /characters/{characterId}/visits

Returns all map visit records for a character.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | yes | Character identifier |
| page[number] | query | int | no | Page number (default 1) |
| page[size] | query | int | no | Page size (default 50, max 250) |

#### Request Headers

| Name | Required | Description |
|------|----------|-------------|
| TENANT_ID | yes | Tenant UUID |
| REGION | yes | Region code |
| MAJOR_VERSION | yes | Major version |
| MINOR_VERSION | yes | Minor version |

#### Response Model

Paginated JSON:API array of visit resources.

```
{
    "data": [
        {
            "type": "visits",
            "id": "<mapId>",
            "attributes": {
                "characterId": 12345,
                "mapId": 100000000,
                "firstVisitedAt": "2025-01-01T00:00:00Z"
            }
        }
    ]
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid characterId or page[number]/page[size] |
| 500 | Failed to retrieve visits |

### GET /characters/{characterId}/visits/{mapId}

Returns a specific map visit record for a character.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | yes | Character identifier |
| mapId | path | uint32 | yes | Map identifier |

#### Request Headers

| Name | Required | Description |
|------|----------|-------------|
| TENANT_ID | yes | Tenant UUID |
| REGION | yes | Region code |
| MAJOR_VERSION | yes | Major version |
| MINOR_VERSION | yes | Minor version |

#### Response Model

JSON:API single visit resource.

```
{
    "data": {
        "type": "visits",
        "id": "<mapId>",
        "attributes": {
            "characterId": 12345,
            "mapId": 100000000,
            "firstVisitedAt": "2025-01-01T00:00:00Z"
        }
    }
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid characterId or mapId |
| 404 | Visit record not found |
| 500 | Failed to retrieve visit |

### GET /characters/{characterId}/location

Returns the character's persisted last-known field (world, channel, map, instance).

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | yes | Character identifier |

#### Request Headers

| Name | Required | Description |
|------|----------|-------------|
| TENANT_ID | yes | Tenant UUID |
| REGION | yes | Region code |
| MAJOR_VERSION | yes | Major version |
| MINOR_VERSION | yes | Minor version |

#### Response Model

JSON:API single character-locations resource.

```
{
    "data": {
        "type": "character-locations",
        "id": "<characterId>",
        "attributes": {
            "worldId": 0,
            "channelId": 0,
            "mapId": 100000000,
            "instance": "00000000-0000-0000-0000-000000000000"
        }
    }
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid characterId |
| 404 | No location record for character |
| 500 | Failed to retrieve location or create REST model |

### PATCH /characters/{characterId}/location

Warps a character to a different map. The destination channel is the character's currently stored channel and the spawn portal is 0; channelId and instance in the request body are ignored.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | yes | Character identifier |

#### Request Headers

| Name | Required | Description |
|------|----------|-------------|
| TENANT_ID | yes | Tenant UUID |
| REGION | yes | Region code |
| MAJOR_VERSION | yes | Major version |
| MINOR_VERSION | yes | Minor version |

#### Request Model

JSON:API character-locations resource. Only the mapId attribute is used.

```
{
    "data": {
        "type": "character-locations",
        "attributes": {
            "mapId": 100000000
        }
    }
}
```

#### Response Model

No content on success (204).

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid characterId, or target mapId does not exist |
| 404 | No location record for character |
| 500 | Failed to load current location, verify the target map, or perform the warp |
