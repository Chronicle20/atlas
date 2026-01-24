# REST

## Endpoints

### GET /api/portals/blocked

Returns blocked portals for a character.

**Parameters**

| Name | In | Type | Required | Description |
|------|-----|------|----------|-------------|
| characterId | query | uint32 | Yes | Character identifier |
| Tenant-Id | header | uuid | Yes | Tenant identifier |

**Request Model**

None.

**Response Model**

JSON:API response with `blocked-portals` resource type.

| Field | Type | Description |
|-------|------|-------------|
| id | string | Composite key `{mapId}:{portalId}` |
| characterId | uint32 | Character identifier |
| mapId | uint32 | Map identifier |
| portalId | uint32 | Portal identifier |

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 400 | Missing or invalid characterId |
| 500 | Internal error transforming blocked portals |
