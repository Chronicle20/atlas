# REST

## Endpoints

### GET /api/party-quests/definitions

Returns all party quest definitions for the tenant.

- **Parameters**: None
- **Request model**: None
- **Response model**: `[]definition.RestModel` (JSON:API resource type: `definitions`)
- **Error conditions**:
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

Clears all definitions for the tenant and re-creates them from JSON definition files on disk.

- **Parameters**: None
- **Request model**: None
- **Response model**: `definition.SeedResult` (JSON, not JSON:API)
  - `deletedCount` — `int`
  - `createdCount` — `int`
  - `failedCount` — `int`
  - `errors` — `[]string` (optional)
- **Error conditions**:
  - `500` — Internal error during seeding

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

Returns all active party quest instances for the tenant.

- **Parameters**: None
- **Request model**: None
- **Response model**: `[]instance.RestModel` (JSON:API resource type: `instances`)
- **Error conditions**:
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

### GET /api/party-quests/instances/{instanceId}/stage

Returns the current stage information for an instance.

- **Parameters**: `instanceId` (path, UUID)
- **Request model**: None
- **Response model**: `instance.RestModel` (JSON:API resource type: `instances`)
- **Error conditions**:
  - `400` — Invalid UUID format
  - `404` — Instance not found
