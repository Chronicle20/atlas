# Door REST API

All routes are served under the `/api/` base path. Responses use the JSON:API format with resource type `doors`.

## Endpoints

### GET /doors/{doorId}

Retrieves a single door by its area door id.

**Parameters:**
- `doorId` (path, uint32): the area door id.

**Request model:** none.

**Response model:** `doors` resource (`RestModel`).

| Field | Type | Description |
|-------|------|-------------|
| areaDoorId | uint32 | Area-side door object id |
| townDoorId | uint32 | Town-side door object id |
| pairId | uint32 | Pair id (equals areaDoorId) |
| ownerCharacterId | character.Id | Owner character id |
| partyId | uint32 | Owner's party id (0 when solo) |
| worldId | world.Id | World id |
| channelId | channel.Id | Channel id |
| mapId | _map.Id | Area field map id |
| instance | uuid.UUID | Area field instance |
| townMapId | _map.Id | Resolved return-town map |
| slot | byte | Party door slot |
| townPortalId | uint32 | Wire town-portal id (0x80 + slot) |
| areaX | point.X | Area-door X coordinate |
| areaY | point.Y | Area-door Y coordinate |
| townX | point.X | Town-door X coordinate |
| townY | point.Y | Town-door Y coordinate |
| skillId | skill.Id | Casting skill id |
| skillLevel | byte | Casting skill level |
| expiresAt | time.Time | Door expiry time |

The resource `id` is the area door id rendered as a decimal string.

**Error conditions:**
- 404 Not Found: no door with the given id exists for the tenant in context.
- 500 Internal Server Error: failure building the REST model.

### GET /characters/{characterId}/doors

Retrieves all doors owned by a character for the tenant in context. Paginated; results are sorted ascending by `pairId` (areaDoorId).

**Parameters:**
- `characterId` (path, character.Id): the owner character id.
- `page[number]` (query, int, optional): 1-based page number. Default 1. Must be >= 1.
- `page[size]` (query, int, optional): page size. Default 50. Must be between 1 and 250.
- `limit` (query): rejected outright; use `page[size]` instead.

**Request model:** none.

**Response model:** paginated array of `doors` resources (`RestModel`). The JSON:API document includes a `meta` block (`total`, `page.number`, `page.size`, `page.last`) and `links` (`self`, `first`, `last`, and `prev`/`next` where applicable).

**Error conditions:**
- 400 Bad Request: `page[number]` or `page[size]` is non-integer, out of range, or the legacy `limit` param is present.
- 500 Internal Server Error: failure retrieving doors or building the REST model.

### GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/doors

Retrieves all doors whose area field matches the given world, channel, map, and instance. Paginated; results are sorted ascending by `pairId` (areaDoorId).

**Parameters:**
- `worldId` (path, world.Id): the world id.
- `channelId` (path, channel.Id): the channel id.
- `mapId` (path, _map.Id): the map id.
- `instanceId` (path, uuid.UUID): the instance id.
- `page[number]` (query, int, optional): 1-based page number. Default 1. Must be >= 1.
- `page[size]` (query, int, optional): page size. Default 50. Must be between 1 and 250.
- `limit` (query): rejected outright; use `page[size]` instead.

**Request model:** none.

**Response model:** paginated array of `doors` resources (`RestModel`). The JSON:API document includes a `meta` block (`total`, `page.number`, `page.size`, `page.last`) and `links` (`self`, `first`, `last`, and `prev`/`next` where applicable).

**Error conditions:**
- 400 Bad Request: `page[number]` or `page[size]` is non-integer, out of range, or the legacy `limit` param is present.
- 500 Internal Server Error: failure retrieving doors or building the REST model.
