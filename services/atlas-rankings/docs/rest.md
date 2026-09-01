# Rankings REST API

All routes are served under the `/api/` base path. Responses use the JSON:API format with resource type `rankings`.

## Endpoints

### GET /rankings

Retrieves a paginated per-world leaderboard, ordered overall or restricted to a job category.

**Parameters:**
- `filter[worldId]` (query, world.Id, required): the world id.
- `filter[jobCategory]` (query, uint16, optional): restricts the leaderboard to a job category and orders by job rank instead of overall rank.
- `page[number]` (query, int, optional): 1-based page number. Default 1. Must be >= 1.
- `page[size]` (query, int, optional): page size. Default 50. Must be between 1 and 250.
- `limit` (query): rejected outright; use `page[size]` instead.

**Request model:** none.

**Response model:** paginated array of `rankings` resources (`LeaderboardRestModel`).

| Field | Type | Description |
|-------|------|-------------|
| characterId | uint32 | Character id (also the resource `id`) |
| name | string | Character name |
| worldId | world.Id | World id |
| jobId | job.Id | Job id |
| jobCategory | uint16 | Job category (`jobId / 100`) |
| level | byte | Character level |
| rank | uint32 | Overall rank within the world |
| rankMove | int32 | Change from the previous cycle's overall rank (positive = moved up) |
| jobRank | uint32 | Rank within the world and job category |
| jobRankMove | int32 | Change from the previous cycle's job rank (positive = moved up) |
| computedAt | time.Time | Timestamp of the cycle that produced this rank |

The resource `id` is the character id rendered as a decimal string. The JSON:API document includes a `meta` block (`total`, `page.number`, `page.size`, `page.last`) and `links` (`self`, `first`, `last`, and `prev`/`next` where applicable).

**Error conditions:**
- 400 Bad Request: `filter[worldId]` is missing or not a valid world id; `filter[jobCategory]` is present but not a valid job category; `page[number]` or `page[size]` is non-integer, out of range, or the legacy `limit` param is present.
- 500 Internal Server Error: failure retrieving the leaderboard or building the REST model.

### GET /rankings/characters?ids={id},{id},…

Bulk fetch of rankings for a comma-separated set of character ids. Blank segments (leading, trailing, or duplicate commas) are skipped. One `rankings` resource is returned per requested id that has an entry; unknown ids are omitted from the response.

**Parameters:**
- `ids` (query, comma-separated uint32 list, required): character ids to fetch.

**Request model:** none.

**Response model:** array of `rankings` resources (`RestModel`).

| Field | Type | Description |
|-------|------|-------------|
| worldId | world.Id | World id |
| rank | uint32 | Overall rank within the world |
| rankMove | int32 | Change from the previous cycle's overall rank (positive = moved up) |
| jobRank | uint32 | Rank within the world and job category |
| jobRankMove | int32 | Change from the previous cycle's job rank (positive = moved up) |
| computedAt | time.Time | Timestamp of the cycle that produced this rank |

The resource `id` is the character id rendered as a decimal string.

**Error conditions:**
- 400 Bad Request: `ids` is absent, empty, all-blank, or contains a non-numeric segment.
- 500 Internal Server Error: failure retrieving rankings or building the REST model.

### GET /rankings/characters/{characterId}

Retrieves a single character's ranking.

**Parameters:**
- `characterId` (path, uint32): the character id.

**Request model:** none.

**Response model:** `rankings` resource (`RestModel`).

| Field | Type | Description |
|-------|------|-------------|
| worldId | world.Id | World id |
| rank | uint32 | Overall rank within the world |
| rankMove | int32 | Change from the previous cycle's overall rank (positive = moved up) |
| jobRank | uint32 | Rank within the world and job category |
| jobRankMove | int32 | Change from the previous cycle's job rank (positive = moved up) |
| computedAt | time.Time | Timestamp of the cycle that produced this rank |

The resource `id` is the character id rendered as a decimal string.

**Error conditions:**
- 400 Bad Request: `characterId` path segment is not a valid integer.
- 404 Not Found: no ranking entry exists for the character in the tenant in context.
- 500 Internal Server Error: failure retrieving the ranking or building the REST model.
