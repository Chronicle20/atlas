# REST API

## Endpoints

### GET /api/characters/{characterId}/invites

Retrieves all pending invites for a character.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | Yes | Character identifier |

#### Request Headers

| Header | Required | Description |
|--------|----------|-------------|
| TENANT_ID | Yes | Tenant identifier |
| REGION | Yes | Region code |
| MAJOR_VERSION | Yes | Major version |
| MINOR_VERSION | Yes | Minor version |

#### Request Model

None.

#### Response Model

JSON:API response with resource type `invites`.

| Attribute | Type | Description |
|-----------|------|-------------|
| type | string | Invite type category |
| referenceId | uint32 | Reference entity identifier |
| originatorId | uint32 | Character who sent the invite |
| targetId | uint32 | Character who receives the invite |
| age | time.Time | Creation timestamp |

#### Error Conditions

| Status Code | Condition |
|-------------|-----------|
| 400 | Invalid characterId path parameter |
| 500 | Internal error retrieving invites |
