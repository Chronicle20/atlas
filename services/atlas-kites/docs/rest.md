# Kite REST API

All routes are served under the `/api/` base path. Responses use the JSON:API format with resource type `kites`.

## Endpoints

### GET /kites/{characterId}

Retrieves the kite owned by a character.

**Parameters:**
- `characterId` (path, uint32): the owning character id.

**Request model:** none.

**Response model:** `kites` resource (`RestModel`).

| Field | Type | Description |
|-------|------|-------------|
| id | uint32 | Kite wire id (JSON:API resource id; not the character id) |
| characterId | uint32 | Owning character id |
| name | string | Owner's character name |
| templateId | uint32 | Kite template/item id |
| message | string | Kite message text |
| x | int16 | X coordinate |
| y | int16 | Y coordinate |
| worldId | world.Id | World id |
| channelId | channel.Id | Channel id |
| mapId | _map.Id | Map id |
| instanceId | uuid.UUID | Field instance |
| createdAt | time.Time | Placement time |

The resource `id` is the kite wire id rendered as a decimal string.

**Error conditions:**
- 404 Not Found: the character has no kite placed.
- 500 Internal Server Error: failure retrieving the kite or building the REST model.

### GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/kites

Retrieves all kites placed in the given field. Paginated; results are sorted ascending by kite id before pagination.

**Parameters:**
- `worldId` (path, world.Id): the world id.
- `channelId` (path, channel.Id): the channel id.
- `mapId` (path, _map.Id): the map id.
- `instanceId` (path, uuid.UUID): the instance id.
- `page[number]` (query, int, optional): 1-based page number. Default 1. Must be >= 1.
- `page[size]` (query, int, optional): page size. Default 250. Must be between 1 and 250.
- `limit` (query): rejected outright; use `page[size]` instead.

**Request model:** none.

**Response model:** paginated array of `kites` resources (`RestModel`).

**Error conditions:**
- 400 Bad Request: `page[number]` or `page[size]` is non-integer, out of range, or the legacy `limit` param is present.
- 500 Internal Server Error: failure retrieving kites or building the REST model.
