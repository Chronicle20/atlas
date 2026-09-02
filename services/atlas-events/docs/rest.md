# Event REST API

All routes are served under the `/api/` base path. JSON:API endpoints use resource type `event-definitions` (definitions) or `event-occurrences` (occurrences), with `event-occurrence-transitions` as an included relationship.

## Endpoints

### GET /events/definitions

Retrieves a paged, optionally filtered list of event definitions.

**Parameters:**
- `page[number]` (query, int, optional): 1-based page number. Default 1.
- `page[size]` (query, int, optional): page size. Default and max per `paginate.DefaultPageSize`/`paginate.MaxPageSize`.
- `filter[type]` (query, string, optional): restricts to definitions of one event type.
- `filter[enabled]` (query, `"true"`/`"false"`, optional): restricts to enabled or disabled definitions of the filtered type. Requires `filter[type]` — using it without a type filter is a 400.

**Request model:** none.

**Response model:** paginated array of `event-definitions` resources (`RestModel`).

| Field | Type | Description |
|-------|------|-------------|
| type | string | Event type |
| name | string | Definition name |
| enabled | bool | Whether the definition is active |
| configuration | json.RawMessage | Opaque, handler-interpreted configuration |
| singleOccurrence | bool | Whether this type's registered handler declares a constant concurrency key (at most one occurrence can ever exist) |
| createdAt | time.Time | Creation timestamp |
| updatedAt | time.Time | Last-update timestamp |

**Error conditions:**
- 400 Bad Request: invalid `page[number]`/`page[size]`, or `filter[enabled]` supplied without `filter[type]`.
- 500 Internal Server Error: failure retrieving definitions or building the REST model.

### GET /events/definitions/{definitionId}

Retrieves a single event definition by id.

**Parameters:**
- `definitionId` (path, uuid.UUID): the definition id.

**Request model:** none.

**Response model:** `event-definitions` resource (`RestModel`, same shape as above).

**Error conditions:**
- 404 Not Found: no definition with the given id exists for the tenant in context.
- 500 Internal Server Error: failure retrieving the definition or building the REST model.

### PATCH /events/definitions/{definitionId}

Updates a definition's `enabled` flag. This is the only attribute this route may ever change; a request body naming any other attribute, or more than one attribute, is rejected.

**Parameters:**
- `definitionId` (path, uuid.UUID): the definition id.

**Request model:** JSON:API `event-definitions` resource with a single `enabled` (bool) attribute.

**Response model:** `event-definitions` resource (`RestModel`, same shape as GET).

**Error conditions:**
- 400 Bad Request: the attributes object contains anything other than exactly one `enabled` boolean attribute.
- 404 Not Found: no definition with the given id exists.
- 500 Internal Server Error: failure updating the definition or building the REST model.

Note: on a false→true transition, this route's write also schedules one generic `TRIGGER_EVALUATION` work row (see `docs/domain.md`'s `orchestration` package) as part of the same call.

### POST /events/definitions/seed

Asynchronously clears all event definitions for the tenant and re-creates them from the seed catalog (`SEED_CATALOG_ROOT`, resolved via the shared/all catalog root, files matching `event-*.json` under `events/definitions`). Seeding runs in the background after the response is sent; progress is polled via the seed status endpoint.

**Parameters:** none.

**Request model:** none.

**Response model:** none.

**Error conditions:** none synchronously (seed failures are recorded asynchronously and surfaced via the status endpoint).

**Success:** 202 Accepted.

### GET /events/definitions/seed/status

Returns the current seed catalog status for the tenant.

**Parameters:** none.

**Request model:** none.

**Response model:** `seeder.Status` (JSON, not JSON:API).

**Error conditions:**
- 500 Internal Server Error: failure retrieving status.

### GET /events/occurrences

Retrieves a paged, filtered list of event occurrences.

**Parameters:**
- `page[number]` (query, int, optional): 1-based page number. Default 1.
- `page[size]` (query, int, optional): page size. Default and max per `paginate.DefaultPageSize`/`paginate.MaxPageSize`.
- `filter[definitionId]` (query, uuid.UUID, optional)
- `filter[type]` (query, string, optional)
- `filter[state]` (query, string, optional): one of `ACTIVE`, `COMPLETED`, `CANCELLED`, `FAILED`.
- `filter[worldId]` (query, int, optional)
- `filter[channelId]` (query, int, optional)
- `filter[mapId]` (query, int, optional): joins against the occurrence's map-scope rows.
- `filter[voyageId]` (query, uuid.UUID, optional)
- `filter[startedAt][from]` / `filter[startedAt][to]` (query, RFC3339 timestamp, optional)

**Request model:** none.

**Response model:** paginated array of `event-occurrences` resources (`RestModel`), without transition history.

| Field | Type | Description |
|-------|------|-------------|
| type | string | Event type |
| state | string | ACTIVE / COMPLETED / CANCELLED / FAILED |
| stage | string | Handler-defined stage |
| context | json.RawMessage | Opaque, handler-interpreted context |
| startedAt | time.Time | Start timestamp |
| nextTransitionAt | *time.Time | omitted when nil |
| completedAt | *time.Time | omitted when nil |
| completionReason | string | omitted when empty |

Relationship: `definition` (to-one, `event-definitions`).

**Error conditions:**
- 400 Bad Request: invalid `page[number]`/`page[size]`, an unparseable filter id/timestamp, or a malformed `filter[startedAt][from]`/`filter[startedAt][to]`.
- 500 Internal Server Error: failure retrieving occurrences or building the REST model.

### GET /events/occurrences/{occurrenceId}

Retrieves a single event occurrence, including its full transition history as an included relationship.

**Parameters:**
- `occurrenceId` (path, uuid.UUID): the occurrence id.

**Request model:** none.

**Response model:** `event-occurrences` resource (`RestModel`, same fields as the collection listing), with the `transitions` relationship (to-many, `event-occurrence-transitions`) populated and included.

Each included `event-occurrence-transitions` resource:

| Field | Type | Description |
|-------|------|-------------|
| occurrenceId | uuid.UUID | Owning occurrence |
| fromStage | string | Prior stage; empty for the creation row |
| toStage | string | Stage reached |
| occurredAt | time.Time | When the transition occurred |
| triggerType | string | What caused the transition |
| triggerReference | string | Reference tied to the trigger |

**Error conditions:**
- 404 Not Found: no occurrence with the given id exists.
- 500 Internal Server Error: failure retrieving the occurrence, its transitions, or building the REST model.

### GET /events/worlds/{worldId}/channels/{channelId}/maps/{mapId}/visuals

Retrieves the active event visuals for one map, for the channel to render. A narrow, game-capped projection (paginated with `paginate.MaxPageSize` as both default and max), not the full occurrence shape.

**Parameters:**
- `worldId` (path, world.Id)
- `channelId` (path, channel.Id)
- `mapId` (path, _map.Id)
- `page[number]` / `page[size]` (query, int, optional)

**Request model:** none.

**Response model:** paginated array of `event-visuals` resources (`VisualRestModel`).

| Field | Type | Description |
|-------|------|-------------|
| occurrenceId | string | Owning occurrence id |
| visual | string | Visual name, read out of the occurrence's context |
| bgm | string | Background music key, read out of the occurrence's context |

**Error conditions:**
- 400 Bad Request: invalid path ids or `page[number]`/`page[size]`.
- 500 Internal Server Error: failure retrieving visuals or building the REST model.
