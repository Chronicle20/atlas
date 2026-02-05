# REST API

## Endpoints

### GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/characters

Returns character IDs present in the specified map instance.

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

JSON:API array of character resources.

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
| 400 | Invalid worldId, channelId, or mapId |
| 500 | Failed to retrieve characters from registry |
