# REST — atlas-map-actions

Base path: `/api/`

Resource type: `map-scripts`

## Endpoints

### GET /maps/actions

Returns all map scripts for the current tenant.

**Parameters:** None.

**Response model:** `[]RestModel` (JSON:API)

**Error conditions:**
- `500` — Internal error retrieving scripts.

---

### POST /maps/actions

Creates a new map script.

**Parameters:** None.

**Request model:** `RestModel` (JSON:API)

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `scriptName` | `string` | Yes | Script identifier |
| `scriptType` | `string` | Yes | `"onFirstUserEnter"` or `"onUserEnter"` |
| `description` | `string` | No | Human-readable description |
| `rules` | `[]RestRuleModel` | Yes | Ordered list of rules |

**RestRuleModel:**

| Field | Type | Description |
|-------|------|-------------|
| `id` | `string` | Rule identifier |
| `conditions` | `[]RestConditionModel` | Conditions (AND logic) |
| `operations` | `[]RestOperationModel` | Operations to execute |

**RestConditionModel:**

| Field | Type | Description |
|-------|------|-------------|
| `type` | `string` | Condition type |
| `operator` | `string` | Comparison operator |
| `value` | `string` | Value to compare against |
| `referenceId` | `string` | Reference identifier (optional) |

**RestOperationModel:**

| Field | Type | Description |
|-------|------|-------------|
| `type` | `string` | Operation type |
| `params` | `map[string]string` | Operation parameters (optional) |

**Response model:** `RestModel` (JSON:API)

**Error conditions:**
- `400` — Invalid or missing `scriptName` or `scriptType`.
- `500` — Internal error creating script.

---

### GET /maps/actions/{scriptId}

Returns a single map script by UUID.

**Parameters:**

| Name | In | Type | Description |
|------|----|------|-------------|
| `scriptId` | path | `uuid.UUID` | Script UUID |

**Response model:** `RestModel` (JSON:API)

**Error conditions:**
- `400` — Invalid UUID format.
- `404` — Script not found.
- `500` — Internal error retrieving script.

---

### PATCH /maps/actions/{scriptId}

Updates an existing map script.

**Parameters:**

| Name | In | Type | Description |
|------|----|------|-------------|
| `scriptId` | path | `uuid.UUID` | Script UUID |

**Request model:** `RestModel` (JSON:API). Same structure as POST.

**Response model:** `RestModel` (JSON:API)

**Error conditions:**
- `400` — Invalid UUID or missing required fields.
- `500` — Internal error updating script.

---

### DELETE /maps/actions/{scriptId}

Soft-deletes a map script.

**Parameters:**

| Name | In | Type | Description |
|------|----|------|-------------|
| `scriptId` | path | `uuid.UUID` | Script UUID |

**Error conditions:**
- `400` — Invalid UUID format.
- `500` — Internal error deleting script.

Returns `204 No Content` on success.

---

### GET /maps/{scriptName}/actions

Returns all map scripts matching the given script name (across all types).

**Parameters:**

| Name | In | Type | Description |
|------|----|------|-------------|
| `scriptName` | path | `string` | Script name identifier |

**Response model:** `[]RestModel` (JSON:API)

**Error conditions:**
- `400` — Empty script name.
- `500` — Internal error retrieving scripts.

---

### POST /maps/actions/seed

Deletes all existing scripts for the current tenant and re-creates them from JSON files on disk.

**Parameters:** None.

**Request model:** None.

**Response model:** `SeedResult` (JSON, not JSON:API)

```json
{
  "deletedCount": 10,
  "createdCount": 8,
  "failedCount": 2,
  "errors": ["error message 1", "error message 2"]
}
```

**Error conditions:**
- `500` — Internal error during seed operation.
