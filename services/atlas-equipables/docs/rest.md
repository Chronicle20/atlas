# REST API

## Endpoints

### POST /api/equipables

Creates an equipable with provided statistics. If all stat values are zero, fetches template stats from atlas-data service.

**Parameters**

None.

**Request Headers**

| Header | Description |
|--------|-------------|
| TENANT_ID | Tenant identifier (UUID) |

**Request Model**

JSON:API resource type: `equipables`

| Field | Type | Required |
|-------|------|----------|
| itemId | uint32 | Yes |
| strength | uint16 | No |
| dexterity | uint16 | No |
| intelligence | uint16 | No |
| luck | uint16 | No |
| hp | uint16 | No |
| mp | uint16 | No |
| weaponAttack | uint16 | No |
| magicAttack | uint16 | No |
| weaponDefense | uint16 | No |
| magicDefense | uint16 | No |
| accuracy | uint16 | No |
| avoidability | uint16 | No |
| hands | uint16 | No |
| speed | uint16 | No |
| jump | uint16 | No |
| slots | uint16 | No |

**Response Model**

JSON:API resource type: `equipables`

Returns full equipable model with assigned ID.

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 400 | Invalid request body |
| 500 | Creation failed |

---

### POST /api/equipables?random=true

Creates an equipable with randomized statistics based on template values.

**Parameters**

| Parameter | Type | Description |
|-----------|------|-------------|
| random | string | Must be "true" |

**Request Headers**

| Header | Description |
|--------|-------------|
| TENANT_ID | Tenant identifier (UUID) |

**Request Model**

JSON:API resource type: `equipables`

| Field | Type | Required |
|-------|------|----------|
| itemId | uint32 | Yes |

**Response Model**

JSON:API resource type: `equipables`

Returns full equipable model with randomized stats and assigned ID.

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 500 | Creation failed or template lookup failed |

---

### GET /api/equipables/{equipmentId}

Retrieves an equipable by ID.

**Parameters**

| Parameter | Type | Description |
|-----------|------|-------------|
| equipmentId | uint32 | Equipable ID (path) |

**Request Headers**

| Header | Description |
|--------|-------------|
| TENANT_ID | Tenant identifier (UUID) |

**Response Model**

JSON:API resource type: `equipables`

Returns full equipable model.

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 404 | Equipable not found |

---

### PATCH /api/equipables/{equipmentId}

Updates an equipable.

**Parameters**

| Parameter | Type | Description |
|-----------|------|-------------|
| equipmentId | uint32 | Equipable ID (path) |

**Request Headers**

| Header | Description |
|--------|-------------|
| TENANT_ID | Tenant identifier (UUID) |

**Request Model**

JSON:API resource type: `equipables`

All fields from the equipable model are accepted. Only changed fields are persisted.

**Response Model**

JSON:API resource type: `equipables`

Returns updated equipable model.

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 400 | Invalid request body |
| 404 | Equipable not found |

---

### DELETE /api/equipables/{equipmentId}

Deletes an equipable.

**Parameters**

| Parameter | Type | Description |
|-----------|------|-------------|
| equipmentId | uint32 | Equipable ID (path) |

**Request Headers**

| Header | Description |
|--------|-------------|
| TENANT_ID | Tenant identifier (UUID) |

**Response Model**

None.

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 204 | Success (no content) |
| 404 | Equipable not found |
