# REST API

## Endpoints

### GET /api/reactors/actions

Returns a page of reactor scripts for the current tenant.

**Parameters:**

| Name | Location | Type | Description |
|------|----------|------|-------------|
| page[number] | query | int | Page number (optional) |
| page[size] | query | int | Page size (optional) |

**Response Model:** Paginated array of `reactor-scripts`

**Error Conditions:**
- 400: Invalid page[number]/page[size]
- 500: Internal server error

---

### GET /api/reactors/actions/{scriptId}

Returns a reactor script by ID.

**Parameters:**

| Name | Location | Type | Description |
|------|----------|------|-------------|
| scriptId | path | uuid | Script identifier |

**Response Model:** `reactor-scripts`

**Error Conditions:**
- 400: Invalid script ID format
- 404: Script not found
- 500: Internal server error

---

### GET /api/reactors/{reactorId}/actions

Returns a reactor script by reactor classification ID.

**Parameters:**

| Name | Location | Type | Description |
|------|----------|------|-------------|
| reactorId | path | string | Reactor classification ID |

**Response Model:** `reactor-scripts`

**Error Conditions:**
- 400: Missing reactor ID
- 404: Script not found
- 500: Internal server error

---

### POST /api/reactors/actions

Creates a new reactor script.

**Request Model:** `reactor-scripts`

**Response Model:** `reactor-scripts`

**Error Conditions:**
- 400: Invalid request body or missing reactorId
- 500: Internal server error

---

### PATCH /api/reactors/actions/{scriptId}

Updates an existing reactor script.

**Parameters:**

| Name | Location | Type | Description |
|------|----------|------|-------------|
| scriptId | path | uuid | Script identifier |

**Request Model:** `reactor-scripts`

**Response Model:** `reactor-scripts`

**Error Conditions:**
- 400: Invalid script ID or request body
- 500: Internal server error

---

### DELETE /api/reactors/actions/{scriptId}

Deletes a reactor script (soft delete).

**Parameters:**

| Name | Location | Type | Description |
|------|----------|------|-------------|
| scriptId | path | uuid | Script identifier |

**Response Model:** None (204 No Content)

**Error Conditions:**
- 400: Invalid script ID format
- 500: Internal server error

---

### POST /api/reactors/actions/seed

Starts an asynchronous seed of reactor scripts from the seed catalog for the current tenant. Deletes all existing scripts for the tenant, then reloads them from the catalog in the background.

**Parameters:** None

**Response Model:** None (202 Accepted)

**Error Conditions:**
- 400: Missing tenant headers

---

### GET /api/reactors/actions/seed/status

Returns the seed status for the current tenant's reactor-actions group.

**Parameters:** None

**Response Model:**

```json
{
  "groupName": "reactor-actions",
  "subdomains": {
    "reactor-actions": {
      "count": 0,
      "updatedAt": null
    }
  },
  "updatedAt": null,
  "catalogRevision": "",
  "tenantSeededRevision": null,
  "tenantSeededAt": null
}
```

**Error Conditions:**
- 400: Missing tenant headers
- 500: Internal server error

---

## Resource Type: reactor-scripts

JSON:API resource for reactor scripts.

```json
{
  "type": "reactor-scripts",
  "id": "uuid",
  "attributes": {
    "reactorId": "2000",
    "description": "Maple Island Box",
    "hitRules": [],
    "actRules": []
  }
}
```

### Attributes

| Field | Type | Description |
|-------|------|-------------|
| reactorId | string | Reactor classification ID (required) |
| description | string | Human-readable description |
| hitRules | array | Rules for hit events |
| actRules | array | Rules for trigger events |

### Rule Structure

```json
{
  "id": "rule_id",
  "conditions": [],
  "operations": []
}
```

### Condition Structure

```json
{
  "type": "reactor_state",
  "operator": "=",
  "value": "1",
  "referenceId": "",
  "step": ""
}
```

| Field | Type | Description |
|-------|------|-------------|
| type | string | Condition type (`reactor_state`, `pq_custom_data`) |
| operator | string | Comparison operator (`=`, `!=`, `>`, `<`, `>=`, `<=`) |
| value | string | Expected value |
| referenceId | string | Reference identifier |
| step | string | Custom data key name (used by `pq_custom_data` condition type) |

### Operation Structure

```json
{
  "type": "drop_items",
  "params": {
    "meso": "true",
    "mesoMin": "2",
    "mesoMax": "8"
  }
}
```

---

## External API Consumption

The service makes REST API calls to the following services via `requests.RootUrl`.

### atlas-party-quests

#### GET /party-quests/instances/character/{characterId}

Retrieves the party quest instance for a character. Used by `pq_custom_data` condition evaluation and by `update_pq_state`, `broadcast_pq_message`, `stage_clear_attempt` operation execution.

**Parameters**

| Name | Type | Location | Description |
|------|------|----------|-------------|
| characterId | uint32 | path | Character ID |

**Response Model**

Resource type: `instances`

| Field | Type | Description |
|-------|------|-------------|
| id | uuid.UUID | Party quest instance ID |
| stageState | object | Stage state data |
| stageState.customData | map[string]any | Custom key-value data for the current stage |
