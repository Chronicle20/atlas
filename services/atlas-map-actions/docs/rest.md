# REST — atlas-map-actions

Base path: `/api/`

Resource type: `map-scripts`

## Endpoints

### GET /maps/actions

Returns one page of map scripts for the current tenant.

**Parameters:**

| Name | In | Type | Description |
|------|----|------|-------------|
| `page[number]` | query | int | Page number (default `1`) |
| `page[size]` | query | int | Page size (default `50`, max `250`) |

**Response model:** `[]RestModel` (JSON:API, paginated — includes `meta.total`, `meta.page.number`, `meta.page.size`, `meta.page.last`, and `self`/`first`/`last`/`prev`/`next` links)

**Error conditions:**
- `400` — Invalid `page[number]`/`page[size]`.
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

Returns one page of map scripts matching the given script name (across all types).

**Parameters:**

| Name | In | Type | Description |
|------|----|------|-------------|
| `scriptName` | path | `string` | Script name identifier |
| `page[number]` | query | int | Page number (default `1`) |
| `page[size]` | query | int | Page size (default `50`, max `250`) |

**Response model:** `[]RestModel` (JSON:API, paginated — includes `meta.total`, `meta.page.number`, `meta.page.size`, `meta.page.last`, and `self`/`first`/`last`/`prev`/`next` links)

**Error conditions:**
- `400` — Empty script name, or invalid `page[number]`/`page[size]`.
- `500` — Internal error retrieving scripts.

---

### POST /maps/actions/seed

Starts an asynchronous seed of map scripts for the current tenant from the `map-actions` catalog group (`onFirstUserEnter` and `onUserEnter` subdomains). Existing scripts of each subdomain's script type are hard-deleted and replaced with entries built from catalog files matching `map-<name>.json`. Seeding runs in a background goroutine; the request returns before it completes.

**Parameters:** None.

**Request model:** None.

**Response model:** None.

**Error conditions:**
- `400` — Missing tenant headers (`TENANT_ID`, `REGION`, `MAJOR_VERSION`, `MINOR_VERSION`).

Returns `202 Accepted` on success (request accepted; seed runs asynchronously).

---

### GET /maps/actions/seed/status

Returns the current seed status for the `map-actions` catalog group for the current tenant.

**Parameters:** None.

**Response model:** JSON (not JSON:API)

```json
{
  "groupName": "map-actions",
  "subdomains": {
    "onFirstUserEnter": { "count": 8, "updatedAt": "2026-07-16T00:00:00Z" },
    "onUserEnter": { "count": 5, "updatedAt": "2026-07-16T00:00:00Z" }
  },
  "updatedAt": "2026-07-16T00:00:00Z",
  "catalogRevision": "abc123",
  "tenantSeededRevision": "abc123",
  "tenantSeededAt": "2026-07-16T00:00:00Z"
}
```

**Error conditions:**
- `400` — Missing tenant headers (`TENANT_ID`, `REGION`, `MAJOR_VERSION`, `MINOR_VERSION`).
- `500` — Internal error reading seed status.
