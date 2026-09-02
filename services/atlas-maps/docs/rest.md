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

### GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/jukebox

Returns the active jukebox entry in the specified map instance.

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

JSON:API single jukebox resource.

```
{
    "data": {
        "type": "jukebox",
        "id": "<itemId>",
        "attributes": {
            "itemId": 5150000,
            "playerName": "Bob"
        }
    }
}
```

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid worldId, channelId, mapId, or instanceId |
| 404 | No active jukebox entry in map instance |
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
            "instance": "00000000-0000-0000-0000-000000000000",
            "state": "IN_FIELD"
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

### GET /api/fields

Enumerates live field instances for the requesting tenant. A field is listed if and only if it currently holds at least one character. The `/api/` prefix is implied by the service's base path configuration; the route itself is `/fields`.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| filter[worldId] | query | uint32 | no | World identifier filter (optional, independent of other filters) |
| filter[channelId] | query | uint32 | no | Channel identifier filter (optional, independent of other filters) |
| filter[mapId] | query | uint32 | no | Map identifier filter (optional, independent of other filters) |
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

Paginated JSON:API collection of field resources. Each field resource carries its world, channel, map, and instance identifiers, along with the current count of characters in that instance.

```
{
    "data": [
        {
            "type": "fields",
            "id": "0:1:910340000:6f1c8a2e-0000-0000-0000-000000000000",
            "attributes": {
                "worldId": 0,
                "channelId": 1,
                "mapId": 910340000,
                "instanceId": "6f1c8a2e-0000-0000-0000-000000000000",
                "characterCount": 4
            }
        }
    ]
}
```

#### Liveness Rule

A field appears in the enumeration if and only if it currently holds at least one character. The character registry (`map[MapKey][]uint32`) is keyed by world, channel, map, and instance; when the last character leaves a field via `RemoveCharacter`, the registry slice is emptied but the key remains. A field is considered "live" only when its character slice is non-empty. Therefore, key existence alone does not determine liveness — the slice must contain at least one character ID.

#### Filtering and Ordering

Filters (`filter[worldId]`, `filter[channelId]`, `filter[mapId]`) are optional and independently combinable. Unknown filter keys are silently ignored; malformed filter values (non-numeric or out of range) cause a 400 error. When specified, a filter constrains the result set to fields matching that criteria; unspecified filters match all values.

Results are ordered by (worldId, channelId, mapId, instanceId) in ascending order before pagination is applied. This ensures that pages partition the result set consistently across repeated requests with the same filters.

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Malformed filter value (non-numeric or out of range), or invalid page[number]/page[size] |
| 500 | Failed to read the character registry |
