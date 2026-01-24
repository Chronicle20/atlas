# Portal Actions REST API

## Endpoints

### GET /api/portals/scripts

Retrieves all portal scripts.

**Parameters:** None

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
  ]
}
```

**Error Conditions:**

| Status | Condition |
|--------|-----------|
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

Seeds portal scripts from the filesystem.

**Parameters:** None

**Request Model:** None

**Response Model:**

```json
{
  "deletedCount": 0,
  "createdCount": 0,
  "failedCount": 0,
  "errors": []
}
```

**Error Conditions:**

| Status | Condition |
|--------|-----------|
| 500 | Failed to clear existing scripts or read scripts directory |
