# Mini-Game REST API

All routes are served under the `/api/` base path. Collection endpoints return a JSON:API paginated envelope (`meta.page` + `first`/`prev`/`next`/`last` links); paging uses `page[number]`/`page[size]` (default and max page size 250).

## Endpoints

### GET /characters/{characterId}/game-records

Retrieves the character's win/tie/loss record for every mini-game type, zero-filled for any type never played. Paginated; results are sorted ascending by `gameType`.

**Parameters:**
- `characterId` (path, uint32): the character id.
- `page[number]` (query, int, optional): 1-based page number. Default 1.
- `page[size]` (query, int, optional): page size. Default and max 250.

**Request model:** none.

**Response model:** paginated array of `game-records` resources (`RestModel`). The resource `id` is a synthetic `"{characterId}-{gameType}"` string (a character has one record per game type, so there is no single natural resource id).

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character id |
| gameType | string | `OMOK` or `MATCH_CARDS` |
| wins | uint32 | Win count |
| ties | uint32 | Tie count |
| losses | uint32 | Loss count |

**Error conditions:**
- 400 Bad Request: `page[number]` or `page[size]` is invalid.
- 500 Internal Server Error: failure retrieving records or building the REST model.

### GET /worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/games

Retrieves every mini-game room currently registered in the given field. Paginated; results are sorted ascending by room id (owner character id).

**Parameters:**
- `worldId` (path, world.Id): the world id.
- `channelId` (path, channel.Id): the channel id.
- `mapId` (path, _map.Id): the map id.
- `instanceId` (path, uuid.UUID): the instance id.
- `page[number]` (query, int, optional): 1-based page number. Default 1.
- `page[size]` (query, int, optional): page size. Default and max 250.

**Request model:** none.

**Response model:** paginated array of `games` resources (`RestModel`). The resource `id` is the room id (owner character id) rendered as a decimal string.

| Field | Type | Description |
|-------|------|-------------|
| ownerId | uint32 | Room owner's character id |
| roomType | byte | 1 for Omok, 2 for Match Cards |
| title | string | Room title |
| private | bool | Whether the room is private |
| hasPassword | bool | Whether the room is private and has a non-empty password |
| pieceType | byte | Omok piece set, or Match Cards difficulty tier |
| occupancy | byte | 2 when a visitor is seated, 1 otherwise |
| inProgress | bool | Whether a game is currently running |

**Error conditions:**
- 400 Bad Request: `page[number]` or `page[size]` is invalid.
- 500 Internal Server Error: failure retrieving rooms or building the REST model.

### GET /characters/{characterId}/games

Retrieves the (0-or-1) room the character is currently seated in, as owner or visitor. A collection response (not a single resource), so the empty case is an empty list rather than a 404.

**Parameters:**
- `characterId` (path, uint32): the character id.

**Request model:** none.

**Response model:** array (0 or 1 elements) of `games` resources (`RestModel`), same shape as the field-listing endpoint above.

**Error conditions:**
- 500 Internal Server Error: failure building the REST model.
