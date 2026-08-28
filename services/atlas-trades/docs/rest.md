# Trade REST API

All routes are served under the `/api/` base path. Responses use the
JSON:API format. Every route documented here is read-only: trade rooms are
created and mutated exclusively by Kafka commands, and ledger rows are
immutable once written.

## Endpoints

### GET /trades/rooms

Retrieves the request's tenant's live trade rooms (the process-local
registry of *this* pod, which is why atlas-trades runs single-replica),
filtered, deterministically ordered, and paged.

**Parameters:**
- `filter[characterId]` (query, character.Id, optional): matches a room where
  the character occupies either side.
- `filter[worldId]` (query, world.Id, optional).
- `filter[channelId]` (query, channel.Id, optional).
- `filter[mapId]` (query, map.Id, optional).
- `page[number]` (query, int, optional): 1-based page number.
- `page[size]` (query, int, optional): page size, capped at 100.

**Request model:** none.

**Response model:** paginated array of `rooms` resources (`RestModel`),
sorted ascending by room id.

| Field | Type | Description |
|-------|------|-------------|
| roomType | byte | miniroom type byte. |
| worldId | world.Id | |
| channelId | channel.Id | |
| mapId | map.Id | |
| state | string | Room lifecycle state. |
| participants | []ParticipantRestModel | |
| createdAt | time.Time | |

`ParticipantRestModel`:

| Field | Type | Description |
|-------|------|-------------|
| characterId | character.Id | |
| position | byte | 0 = owner, 1 = visitor. |
| confirmed | bool | |
| mesoStaged | uint32 | |
| items | []StagedItemRestModel | |

`StagedItemRestModel`:

| Field | Type | Description |
|-------|------|-------------|
| tradeSlot | byte | Client dialog slot (1..9), not the source inventory slot. |
| itemId | item.Id | |
| quantity | asset.Quantity | |
| assetId | asset.Id | |

The resource `id` is the room's uuid. `handle` (the wire serial) is
deliberately not projected: it is meaningful only inside the trade protocol.

**Error conditions:**
- 400 Bad Request: an unparseable `filter[characterId]`/`filter[worldId]`/
  `filter[channelId]`/`filter[mapId]`, or an invalid `page[number]`/
  `page[size]`.
- 500 Internal Server Error: failure building the REST model.

### GET /trades/rooms/{roomId}

Retrieves a single live trade room by id.

**Parameters:**
- `roomId` (path, uuid.UUID): the room id.

**Request model:** none.

**Response model:** `rooms` resource (`RestModel`), same shape as above.

**Error conditions:**
- 404 Not Found: no such room in the tenant's live registry. A settled or
  cancelled room has already been removed from the registry, so it 404s
  rather than serving a stale snapshot.
- 500 Internal Server Error: failure building the REST model.

### GET /trades/ledger

Retrieves settled trades in which the given character appears as either
side, within an optional time window, newest-relevant window first
(ordered by the underlying provider), paged in SQL.

**Parameters:**
- `filter[characterId]` (query, character.Id, **required**): 400 if absent.
- `filter[from]` (query, RFC3339 timestamp, optional): defaults to the zero
  time.
- `filter[to]` (query, RFC3339 timestamp, optional): defaults to a
  far-future sentinel (9999-12-31T23:59:59Z).
- `page[number]` (query, int, optional).
- `page[size]` (query, int, optional): default `paginate.DefaultPageSize`,
  capped at 100.

**Request model:** none.

**Response model:** paginated array of `ledgerEntries` resources
(`RestModel`).

| Field | Type | Description |
|-------|------|-------------|
| transactionId | string | The settlement saga's transaction id. |
| worldId | world.Id | |
| channelId | channel.Id | |
| mapId | map.Id | |
| roomType | byte | miniroom type byte. |
| settledAt | time.Time | |
| sides | []SideRestModel | Exactly two. |

`SideRestModel`:

| Field | Type | Description |
|-------|------|-------------|
| characterId | character.Id | |
| characterName | string | Denormalised at settlement time. |
| mesoStaged | uint32 | What this side gave. |
| mesoTax | uint32 | Destroyed from this side's staged meso. |
| mesoDelivered | uint32 | What this side received (from the counterparty). |
| items | []ItemRestModel | |

`ItemRestModel`:

| Field | Type | Description |
|-------|------|-------------|
| itemId | item.Id | |
| quantity | asset.Quantity | |
| assetId | *asset.Id | Null for an asset with no identity of its own (a plain stackable). |
| referenceId | *uint32 | Null unless the asset carries an equip/pet/cash reference. |

`sides` ordering is a determinism guarantee, not a role guarantee: match a
side by `characterId`, never by array position.

**Error conditions:**
- 400 Bad Request: missing `filter[characterId]`, or an unparseable
  `filter[characterId]`/`filter[from]`/`filter[to]`/`page[number]`/
  `page[size]`.
- 500 Internal Server Error: failure reading or projecting entries.

### GET /trades/ledger/{entryId}

Retrieves a single settled-trade ledger entry by id, scoped to the request's
tenant.

**Parameters:**
- `entryId` (path, uuid.UUID): the ledger entry id.

**Request model:** none.

**Response model:** `ledgerEntries` resource (`RestModel`), same shape as
above.

**Error conditions:**
- 404 Not Found: no such entry for the tenant in context (including another
  tenant's entry).
- 500 Internal Server Error: failure reading or projecting the entry.
