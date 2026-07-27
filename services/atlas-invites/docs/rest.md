# REST API

## Endpoints

### GET /api/characters/{characterId}/invites

Retrieves all pending invites for a character. Results are paginated and sorted by invite `id` ascending.

#### Parameters

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | Yes | Character identifier |
| page[number] | query | integer | No | Page number, minimum 1. Default 1. |
| page[size] | query | integer | No | Page size, 1 to 250. Default 250. |

The legacy `limit` query parameter is rejected.

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

Paginated responses include a `meta` block (`total`, `page.number`, `page.size`, `page.last`) and a `links` block (`self`, `first`, `last`, plus `prev` when not on the first page and `next` when not on the last page).

#### Error Conditions

| Status Code | Condition |
|-------------|-----------|
| 400 | Invalid characterId path parameter |
| 400 | `page[number]` or `page[size]` is non-integer, out of range, or the `limit` parameter is present |
| 500 | Internal error retrieving invites |
