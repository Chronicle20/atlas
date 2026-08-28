# Dragon REST API

All routes are served under the `/api/` base path. Responses use the JSON:API format with resource type `dragons`.

## Endpoints

### GET /dragons/{characterId}

Retrieves the dragon owned by a character.

**Parameters:**
- `characterId` (path, uint32): the owner character id.

**Request model:** none.

**Response model:** `dragons` resource (`RestModel`).

| Field | Type | Description |
|-------|------|-------------|
| ownerCharacterId | uint32 | Owner character id |
| x | int32 | Dragon X coordinate |
| y | int32 | Dragon Y coordinate |
| stance | byte | Dragon stance |
| jobId | uint16 | Owner's job id |
| worldId | world.Id | World id |
| channelId | channel.Id | Channel id |
| mapId | _map.Id | Field map id |
| instance | uuid.UUID | Field instance |

The resource `id` is the owner character id rendered as a decimal string.

**Error conditions:**
- 404 Not Found: the character has no dragon.
- 500 Internal Server Error: a retrieval failure other than "no dragon", or failure building the REST model.

### GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/dragons

Retrieves all dragons whose field matches the given world, channel, map, and instance. Paginated; results are sorted ascending by `ownerCharacterId`.

**Parameters:**
- `worldId` (path, world.Id): the world id.
- `channelId` (path, channel.Id): the channel id.
- `mapId` (path, _map.Id): the map id.
- `instanceId` (path, uuid.UUID): the instance id.
- `page[number]` (query, int, optional): 1-based page number. Default 1. Must be >= 1.
- `page[size]` (query, int, optional): page size. Default 250. Must be between 1 and 250.
- `limit` (query): rejected outright; use `page[size]` instead.

**Request model:** none.

**Response model:** paginated array of `dragons` resources (`RestModel`).

**Error conditions:**
- 400 Bad Request: `page[number]` or `page[size]` is non-integer, out of range, or the legacy `limit` param is present.
- 500 Internal Server Error: failure retrieving dragons or building the REST model.
