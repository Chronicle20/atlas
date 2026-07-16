# REST

## Endpoints

### GET /api/party-quests/definitions

Returns a page of party quest definitions for the tenant.

- **Parameters**: `page[number]` (query, optional, default `1`), `page[size]` (query, optional, default `50`, max `250`)
- **Request model**: None
- **Response model**: `[]definition.RestModel` (JSON:API resource type: `definitions`, paginated)
- **Error conditions**:
  - `400` — Invalid `page[number]`/`page[size]` (non-integer, `page[number]<1`, `page[size]<1`, `page[size]>250`, or use of the legacy `limit` param)
  - `500` — Internal error during retrieval or transformation

### GET /api/party-quests/definitions/{definitionId}

Returns a single definition by UUID.

- **Parameters**: `definitionId` (path, UUID)
- **Request model**: None
- **Response model**: `definition.RestModel` (JSON:API resource type: `definitions`)
- **Error conditions**:
  - `400` — Invalid UUID format
  - `404` — Definition not found
  - `500` — Internal error

### GET /api/party-quests/definitions/quest/{questId}

Returns a single definition by quest ID string.

- **Parameters**: `questId` (path, string)
- **Request model**: None
- **Response model**: `definition.RestModel` (JSON:API resource type: `definitions`)
- **Error conditions**:
  - `400` — Empty quest ID
  - `404` — Definition not found
  - `500` — Internal error

### POST /api/party-quests/definitions

Creates a new definition.

- **Parameters**: None
- **Request model**: `definition.RestModel` (JSON:API)
- **Response model**: `definition.RestModel` (JSON:API resource type: `definitions`)
- **Error conditions**:
  - `400` — Invalid input or missing required fields (`questId`, `name`)
  - `500` — Internal error during creation

### PATCH /api/party-quests/definitions/{definitionId}

Updates an existing definition.

- **Parameters**: `definitionId` (path, UUID)
- **Request model**: `definition.RestModel` (JSON:API)
- **Response model**: `definition.RestModel` (JSON:API resource type: `definitions`)
- **Error conditions**:
  - `400` — Invalid UUID or input
  - `500` — Internal error during update

### DELETE /api/party-quests/definitions/{definitionId}

Soft-deletes a definition.

- **Parameters**: `definitionId` (path, UUID)
- **Request model**: None
- **Response model**: None
- **Error conditions**:
  - `400` — Invalid UUID
  - `500` — Internal error during deletion
- **Success**: `204 No Content`

### POST /api/party-quests/definitions/seed

Asynchronously clears all definitions for the tenant and re-creates them from the seed catalog (`SEED_CATALOG_ROOT`, files matching `party-quests/definitions/party-quest-*.json`). Seeding runs in the background after the response is sent; progress is polled via the seed status endpoint.

- **Parameters**: None
- **Request model**: None
- **Response model**: None
- **Error conditions**: None (seed failures are recorded asynchronously and surfaced via the status endpoint)
- **Success**: `202 Accepted`

### GET /api/party-quests/definitions/seed/status

Returns the current seed catalog status for the tenant, including per-subdomain row counts and the last completed seed revision.

- **Parameters**: None
- **Request model**: None
- **Response model**: `seeder.Status` (JSON, not JSON:API)
  - `groupName` — `string`
  - `subdomains` — `map[string]{count int64, updatedAt *time.Time}`
  - `updatedAt` — `*time.Time`
  - `catalogRevision` — `string`
  - `tenantSeededRevision` — `*string`
  - `tenantSeededAt` — `*time.Time`
- **Error conditions**:
  - `500` — Internal error retrieving status

### POST /api/party-quests/definitions/validate

Validates all JSON definition files on disk without persisting.

- **Parameters**: None
- **Request model**: None
- **Response model**: `[]definition.ValidationResult` (JSON, not JSON:API)
  - `valid` — `bool`
  - `questId` — `string`
  - `name` — `string`
  - `errors` — `[]string` (optional)
  - `warnings` — `[]string` (optional)
- **Error conditions**: None (validation errors returned in response body)

### GET /api/party-quests/instances

Returns a page of active party quest instances for the tenant, sorted by instance ID.

- **Parameters**: `page[number]` (query, optional, default `1`), `page[size]` (query, optional, default `50`, max `250`)
- **Request model**: None
- **Response model**: `[]instance.RestModel` (JSON:API resource type: `instances`, paginated)
- **Error conditions**:
  - `400` — Invalid `page[number]`/`page[size]` (non-integer, `page[number]<1`, `page[size]<1`, `page[size]>250`, or use of the legacy `limit` param)
  - `500` — Internal error during transformation

### GET /api/party-quests/instances/{instanceId}

Returns a single active instance by UUID.

- **Parameters**: `instanceId` (path, UUID)
- **Request model**: None
- **Response model**: `instance.RestModel` (JSON:API resource type: `instances`)
- **Error conditions**:
  - `400` — Invalid UUID format
  - `404` — Instance not found

### GET /api/party-quests/instances/character/{characterId}

Returns the active instance containing the specified character.

- **Parameters**: `characterId` (path, uint32)
- **Request model**: None
- **Response model**: `instance.RestModel` (JSON:API resource type: `instances`)
- **Error conditions**:
  - `400` — Invalid character ID format
  - `404` — No instance found for character

### GET /api/party-quests/instances/character/{characterId}/timer

Returns the remaining timer duration for the character's active instance.

- **Parameters**: `characterId` (path, uint32)
- **Request model**: None
- **Response model**: `instance.TimerRestModel` (JSON:API resource type: `timers`)
  - `duration` — `uint64`, remaining seconds
- **Error conditions**:
  - `400` — Invalid character ID format
  - `404` — No instance found for character or no timer configured

### GET /api/party-quests/instances/field/{fieldInstance}

Returns the active instance associated with the specified field instance UUID.

- **Parameters**: `fieldInstance` (path, UUID)
- **Request model**: None
- **Response model**: `instance.RestModel` (JSON:API resource type: `instances`)
- **Error conditions**:
  - `400` — Invalid UUID format
  - `404` — No instance found for field instance
