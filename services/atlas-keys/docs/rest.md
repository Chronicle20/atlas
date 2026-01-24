# REST API

## Endpoints

### GET /api/characters/{characterId}/keys

Retrieves all key bindings for a character.

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

JSON:API formatted array of `keys` resources.

| Field  | Type  | Description                        |
|--------|-------|------------------------------------|
| id     | string | Key identifier                    |
| type   | int8  | Type of key binding                |
| action | int32 | Action associated with the binding |

**Error Conditions**

| Status Code | Condition                      |
|-------------|--------------------------------|
| 400         | Invalid characterId            |
| 500         | Database error                 |

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
