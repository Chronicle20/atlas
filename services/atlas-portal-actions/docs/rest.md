# Portal Actions REST API

## Endpoints

### GET /api/portals/scripts

Retrieves one page of portal scripts.

**Parameters:**

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| page[number] | query | integer | no | Page number (default 1) |
| page[size] | query | integer | no | Page size (default 50, max 250) |

**Response Model:**

```json
{
  "data": [
    {
      "type": "portal-scripts",
      "id": "uuid",
      "attributes": {
        "portalId": "string",
        "mapId": 100000000,
        "description": "string",
        "rules": []
      }
    }
  ],
  "meta": {
    "total": 0,
    "page": {
      "number": 1,
      "size": 50,
      "last": 1
    }
  }
}
```

**Error Conditions:**

| Status | Condition |
|--------|-----------|
| 400 | Invalid page[number]/page[size] parameter |
| 500 | Database error |

---

### GET /api/portals/scripts/{scriptId}

Retrieves a portal script by UUID.

**Parameters:**

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| scriptId | path | uuid | yes | Script UUID |

**Response Model:**

```json
{
  "data": {
    "type": "portal-scripts",
    "id": "uuid",
    "attributes": {
      "portalId": "string",
      "mapId": 100000000,
      "description": "string",
      "rules": []
    }
  }
}
```

**Error Conditions:**

| Status | Condition |
|--------|-----------|
| 404 | Script not found |
| 500 | Database error |

---

### GET /api/portals/{portalId}/scripts

Retrieves a portal script by portal ID.

**Parameters:**

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| portalId | path | string | yes | Portal identifier |

**Response Model:**

```json
{
  "data": {
    "type": "portal-scripts",
    "id": "uuid",
    "attributes": {
      "portalId": "string",
      "mapId": 100000000,
      "description": "string",
      "rules": []
    }
  }
}
```

**Error Conditions:**

| Status | Condition |
|--------|-----------|
| 404 | Script not found for portal |
| 500 | Database error |

---

### POST /api/portals/scripts

Creates a new portal script.

**Parameters:** None

**Request Model:**

```json
{
  "data": {
    "type": "portal-scripts",
    "attributes": {
      "portalId": "string",
      "mapId": 100000000,
      "description": "string",
      "rules": [
        {
          "id": "string",
          "conditions": [
            {
              "type": "string",
              "operator": "string",
              "value": "string",
              "referenceId": "string"
            }
          ],
          "onMatch": {
            "allow": true,
            "operations": [
              {
                "type": "string",
                "params": {}
              }
            ]
          }
        }
      ]
    }
  }
}
```

**Validation Rules:**

| Field | Rule |
|-------|------|
| portalId | Required |

**Response Model:** Same as GET /api/portals/scripts/{scriptId}

**Error Conditions:**

| Status | Condition |
|--------|-----------|
| 400 | Invalid request body or missing portalId |
| 500 | Database error |

---

### PATCH /api/portals/scripts/{scriptId}

Updates an existing portal script.

**Parameters:**

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| scriptId | path | uuid | yes | Script UUID |

**Request Model:** Same as POST /api/portals/scripts

**Response Model:** Same as GET /api/portals/scripts/{scriptId}

**Error Conditions:**

| Status | Condition |
|--------|-----------|
| 400 | Invalid request body |
| 500 | Database error or script not found |

---

### DELETE /api/portals/scripts/{scriptId}

Deletes a portal script.

**Parameters:**

| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| scriptId | path | uuid | yes | Script UUID |

**Response Model:** None (204 No Content)

**Error Conditions:**

| Status | Condition |
|--------|-----------|
| 500 | Database error |

---

### POST /api/portals/scripts/seed

Triggers an asynchronous seed of portal scripts from the configured catalog source (`SEED_CATALOG_ROOT`). Existing portal scripts for the tenant are deleted and replaced with the catalog contents in a background task.

**Parameters:** None

**Request Model:** None

**Response Model:** None (202 Accepted; seeding runs in the background)

**Error Conditions:**

| Status | Condition |
|--------|-----------|
| 400 | Missing or invalid tenant headers |

---

### GET /api/portals/scripts/seed/status

Retrieves the current seed status for the portal script catalog.

**Parameters:** None

**Response Model:**

```json
{
  "groupName": "portal-actions",
  "subdomains": {
    "portal-actions": {
      "count": 0,
      "updatedAt": "2024-01-01T00:00:00Z"
    }
  },
  "updatedAt": "2024-01-01T00:00:00Z",
  "catalogRevision": "string",
  "tenantSeededRevision": "string",
  "tenantSeededAt": "2024-01-01T00:00:00Z"
}
```

**Error Conditions:**

| Status | Condition |
|--------|-----------|
| 400 | Missing or invalid tenant headers |
| 500 | Failed to read seed status |
