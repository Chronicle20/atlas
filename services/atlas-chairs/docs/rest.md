# REST API

## Endpoints

### GET /chairs/{characterId}

Retrieves chair for a specific character.

#### Parameters

| Name | Location | Type | Required |
|------|----------|------|----------|
| characterId | path | uint32 | yes |

#### Request Model

None.

#### Response Model

Single Chair resource.

| Field | Type | JSON Key |
|-------|------|----------|
| Id | uint32 | (resource id) |
| Type | string | type |
| CharacterId | uint32 | characterId |

Resource type: `chairs`

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 200 OK | Chair retrieved |
| 404 Not Found | Character not sitting on chair |
| 500 Internal Server Error | Transformation error |

---

### GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/chairs

Retrieves all chairs in use in a specific map instance. Paginated.

#### Parameters

| Name | Location | Type | Required |
|------|----------|------|----------|
| worldId | path | world.Id | yes |
| channelId | path | channel.Id | yes |
| mapId | path | map.Id | yes |
| instanceId | path | uuid.UUID | yes |
| page[number] | query | int | no (default 1) |
| page[size] | query | int | no (default 250, max 250) |

The legacy `limit` query parameter is rejected.

#### Request Model

None.

#### Response Model

Array of Chair resources (see GET /chairs/{characterId}), sorted by characterId ascending.

JSON:API `meta` block:

| Field | Type | Description |
|-------|------|--------------|
| total | int | Total count of matching chairs across all pages |
| page.number | int | Current page number |
| page.size | int | Current page size |
| page.last | int | Last page number |

JSON:API `links` block: `self`, `first`, `last`, and `prev`/`next` where applicable.

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 200 OK | Chairs retrieved |
| 400 Bad Request | Invalid page[number]/page[size] (non-integer, out of range, or legacy limit param used) |
| 500 Internal Server Error | Transformation error |
