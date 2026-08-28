# RPS REST API

All routes are served under the `/api/` base path. Responses use the JSON:API format with resource type `rps-games`.

## Endpoints

### POST /rps/games

Opens a new RPS session for a character at an NPC, disposing any stale prior session for that character.

**Parameters:** none.

**Request model:** `rps-games` resource (`RestModel`).

| Field | Type | Description |
|-------|------|--------------|
| characterId | uint32 | Player character id |
| worldId | world.Id | World id |
| channelId | channel.Id | Channel id |
| npcId | uint32 | NPC the session is opened at |

**Response model:** `rps-games` resource (`RestModel`). The resource `id` is the character id rendered as a decimal string. `prize` is never populated on this response (a freshly opened session is always at rung 0 with no prize).

| Field | Type | Description |
|-------|------|--------------|
| characterId | uint32 | Player character id |
| worldId | world.Id | World id |
| channelId | channel.Id | Channel id |
| npcId | uint32 | NPC id |
| rung | int | Current ladder rung |
| status | string | Session status (`OPEN`, `AWAITING_SELECT`, `AWAITING_DECISION`, `ENDED`) |
| prize | object, omitted | Not populated on this response |

**Error conditions:**
- 500 Internal Server Error: failure starting the session or building the REST model.

### GET /rps/games/{characterId}

Retrieves the character's active RPS session, with the prize currently resolved at its rung.

**Parameters:**
- `characterId` (path, uint32): the character id.

**Request model:** none.

**Response model:** `rps-games` resource (`RestModel`). The resource `id` is the character id rendered as a decimal string.

| Field | Type | Description |
|-------|------|--------------|
| characterId | uint32 | Player character id |
| worldId | world.Id | World id |
| channelId | channel.Id | Channel id |
| npcId | uint32 | NPC id |
| rung | int | Current ladder rung |
| status | string | Session status (`OPEN`, `AWAITING_SELECT`, `AWAITING_DECISION`, `ENDED`) |
| prize | object, optional | `itemId` (item.Id), `quantity` (uint32), `meso` (uint32); present only when a prize is resolved at the session's current rung |

**Error conditions:**
- 404 Not Found: no active session exists for the character.
- 500 Internal Server Error: failure retrieving the session or building the REST model.
