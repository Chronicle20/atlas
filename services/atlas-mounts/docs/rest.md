# REST API

## Endpoints

### GET /api/characters/{characterId}/mount

Retrieves the mount progression for a character. If no mount record exists yet, a default record (level 1, exp 0, tiredness 0) is created and returned. The JSON:API resource type is `mounts`; the resource id is the mount's own UUID.

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
  "data": {
    "type": "mounts",
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "attributes": {
      "characterId": 12345,
      "level": 1,
      "exp": 0,
      "tiredness": 0,
      "lastTirednessTickAt": null
    }
  }
}
```

The `lastTirednessTickAt` attribute is `null` when the mount has not ticked yet.

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 400 | Invalid characterId path parameter |
| 500 | Internal error (retrieval or transform failed) |
