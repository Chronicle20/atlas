# REST API

## Resource Types

### keys

JSON:API resource type for key bindings. Resource ID is the key identifier.

| Field  | Type  | JSON Field | Description                        |
|--------|-------|------------|------------------------------------|
| Key    | int32 | -          | Used as resource ID                |
| Type   | int8  | type       | Type of key binding                |
| Action | int32 | action     | Action associated with the binding |

## Endpoints

### GET /api/characters/{characterId}/keys

Retrieves all key bindings for a character.

**Parameters**

| Name         | Location | Type   | Required | Description                                        |
|--------------|----------|--------|----------|-----------------------------------------------------|
| characterId  | path     | uint32 | yes      | Character identifier                                |
| page[number] | query    | int    | no       | Page number, 1-based. Default 1. Must be >= 1.       |
| page[size]   | query    | int    | no       | Page size. Default 250. Must be between 1 and 250.   |

**Request Headers**

| Header        | Required | Description              |
|---------------|----------|--------------------------|
| TENANT_ID     | yes      | Tenant identifier        |
| REGION        | yes      | Region code              |
| MAJOR_VERSION | yes      | Major version number     |
| MINOR_VERSION | yes      | Minor version number     |

**Response Model**

JSON:API formatted, paginated array of `keys` resources, sorted by key ascending. Includes pagination `meta` and `links`.

| Field  | Type  | Description                        |
|--------|-------|------------------------------------|
| id     | string | Key identifier                    |
| type   | int8  | Type of key binding                |
| action | int32 | Action associated with the binding |

**Error Conditions**

| Status Code | Condition                              |
|-------------|------------------------------------------|
| 400         | Invalid characterId                       |
| 400         | Invalid page[number] or page[size]        |
| 500         | Database error                            |

---

### DELETE /api/characters/{characterId}/keys

Resets all key bindings for a character to default values.

**Parameters**

| Name        | Location | Type   | Required | Description          |
|-------------|----------|--------|----------|----------------------|
| characterId | path     | uint32 | yes      | Character identifier |

**Request Headers**

| Header        | Required | Description              |
|---------------|----------|--------------------------|
| TENANT_ID     | yes      | Tenant identifier        |
| REGION        | yes      | Region code              |
| MAJOR_VERSION | yes      | Major version number     |
| MINOR_VERSION | yes      | Minor version number     |

**Response Model**

Empty response body.

**Error Conditions**

| Status Code | Condition                      |
|-------------|--------------------------------|
| 200         | Success                        |
| 400         | Invalid characterId            |
| 500         | Database error                 |

---

### PATCH /api/characters/{characterId}/keys/{keyId}

Updates a specific key binding for a character.

**Parameters**

| Name        | Location | Type   | Required | Description          |
|-------------|----------|--------|----------|----------------------|
| characterId | path     | uint32 | yes      | Character identifier |
| keyId       | path     | int32  | yes      | Key identifier       |

**Request Headers**

| Header        | Required | Description              |
|---------------|----------|--------------------------|
| TENANT_ID     | yes      | Tenant identifier        |
| REGION        | yes      | Region code              |
| MAJOR_VERSION | yes      | Major version number     |
| MINOR_VERSION | yes      | Minor version number     |

**Request Model**

JSON:API formatted `keys` resource.

| Field  | Type  | Required | Description                        |
|--------|-------|----------|------------------------------------|
| type   | int8  | yes      | Type of key binding                |
| action | int32 | yes      | Action associated with the binding |

**Response Model**

Empty response body.

**Error Conditions**

| Status Code | Condition                      |
|-------------|--------------------------------|
| 200         | Success                        |
| 400         | Invalid characterId or keyId, or malformed request body |
| 500         | Database error                 |
