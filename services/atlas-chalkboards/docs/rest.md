# REST API

## Endpoints

### GET /chalkboards/{characterId}

Retrieves chalkboard message for a specific character.

#### Parameters

| Name | Location | Type | Required |
|------|----------|------|----------|
| characterId | path | uint32 | yes |

#### Request Model

None.

#### Response Model

Single Chalkboard resource.

| Field | Type | JSON Key |
|-------|------|----------|
| Id | uint32 | (resource id) |
| Message | string | message |

Resource type: `chalkboards`

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 200 OK | Chalkboard retrieved |
| 404 Not Found | Character has no chalkboard message |
| 500 Internal Server Error | Transformation error |

---

### GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/chalkboards

Retrieves chalkboard messages for characters present in a specific map instance, paginated.

#### Parameters

| Name | Location | Type | Required |
|------|----------|------|----------|
| worldId | path | world.Id | yes |
| channelId | path | channel.Id | yes |
| mapId | path | map.Id | yes |
| instanceId | path | uuid.UUID | yes |
| page[number] | query | integer | no (default 1) |
| page[size] | query | integer | no (default 250, max 250) |

`limit` is not accepted; its presence is rejected.

#### Request Model

None.

#### Response Model

Paginated array of Chalkboard resources (see GET /chalkboards/{characterId}), sorted ascending by characterId. Characters present in the map without an active chalkboard message are excluded.

JSON:API meta block: `total` (filtered count), `page.number`, `page.size`, `page.last`.

JSON:API links: `self`, `first`, `last`, and `prev`/`next` where applicable.

#### Error Conditions

| Status | Condition |
|--------|-----------|
| 200 OK | Chalkboards retrieved |
| 400 Bad Request | Invalid page[number]/page[size] (non-integer, out of range, or `limit` param present) |
| 500 Internal Server Error | Transformation error |
