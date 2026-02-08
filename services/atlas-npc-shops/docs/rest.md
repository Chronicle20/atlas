# REST

## Endpoints

### GET /api/shops

Retrieves all shops for the current tenant.

**Parameters**

| Name    | In    | Type   | Description                              |
|---------|-------|--------|------------------------------------------|
| include | query | string | Optional. "commodities" to include items |

**Request Headers**

| Name          | Type   | Required | Description           |
|---------------|--------|----------|-----------------------|
| TENANT_ID     | string | Yes      | Tenant identifier     |
| REGION        | string | Yes      | Region code           |
| MAJOR_VERSION | string | Yes      | Major version number  |
| MINOR_VERSION | string | Yes      | Minor version number  |

**Response Model**

JSON:API response with array of shop resources.

| Field      | Type   | Description                    |
|------------|--------|--------------------------------|
| type       | string | "shops"                        |
| id         | string | Shop identifier ("shop-{npcId}") |
| attributes | object | Shop attributes                |

**Attributes**

| Field     | Type   | Description                         |
|-----------|--------|-------------------------------------|
| npcId     | uint32 | NPC template identifier             |
| recharger | bool   | Whether shop supports recharging    |

**Relationships**

| Name        | Type          | Description                    |
|-------------|---------------|--------------------------------|
| commodities | []commodities | Included when ?include=commodities |

**Error Conditions**

| Status | Description            |
|--------|------------------------|
| 500    | Internal server error  |

---

### DELETE /api/shops

Deletes all shops and their commodities for the current tenant.

**Request Headers**

| Name          | Type   | Required | Description           |
|---------------|--------|----------|-----------------------|
| TENANT_ID     | string | Yes      | Tenant identifier     |
| REGION        | string | Yes      | Region code           |
| MAJOR_VERSION | string | Yes      | Major version number  |
| MINOR_VERSION | string | Yes      | Minor version number  |

**Error Conditions**

| Status | Description            |
|--------|------------------------|
| 204    | Success (no content)   |
| 500    | Internal server error  |

---

### GET /api/npcs/{npcId}/shop

Retrieves shop information for a specific NPC.

**Parameters**

| Name    | In    | Type   | Description                              |
|---------|-------|--------|------------------------------------------|
| npcId   | path  | uint32 | NPC template identifier                  |
| include | query | string | Optional. "commodities" to include items |

**Request Headers**

| Name          | Type   | Required | Description           |
|---------------|--------|----------|-----------------------|
| TENANT_ID     | string | Yes      | Tenant identifier     |
| REGION        | string | Yes      | Region code           |
| MAJOR_VERSION | string | Yes      | Major version number  |
| MINOR_VERSION | string | Yes      | Minor version number  |

**Response Model**

JSON:API response with shop resource.

| Field      | Type   | Description                    |
|------------|--------|--------------------------------|
| type       | string | "shops"                        |
| id         | string | Shop identifier ("shop-{npcId}") |
| attributes | object | Shop attributes                |

**Attributes**

| Field     | Type   | Description                         |
|-----------|--------|-------------------------------------|
| npcId     | uint32 | NPC template identifier             |
| recharger | bool   | Whether shop supports recharging    |

**Relationships**

| Name        | Type          | Description                    |
|-------------|---------------|--------------------------------|
| commodities | []commodities | Included when ?include=commodities |

**Commodity Attributes** (when included)

| Field           | Type    | Description                              |
|-----------------|---------|------------------------------------------|
| templateId      | uint32  | Item template identifier                 |
| mesoPrice       | uint32  | Price in mesos                           |
| discountRate    | byte    | Discount percentage                      |
| tokenTemplateId | uint32  | Alternative currency item identifier     |
| tokenPrice      | uint32  | Price in alternative currency            |
| period          | uint32  | Time limit on purchase in minutes        |
| levelLimit      | uint32  | Minimum level required                   |
| unitPrice       | float64 | Unit price for rechargeable items        |
| slotMax         | uint32  | Maximum stack size for the item          |

**Error Conditions**

| Status | Description            |
|--------|------------------------|
| 404    | Shop not found         |
| 500    | Internal server error  |

---

### POST /api/npcs/{npcId}/shop

Creates a new shop for a specific NPC.

**Parameters**

| Name  | In   | Type   | Description             |
|-------|------|--------|-------------------------|
| npcId | path | uint32 | NPC template identifier |

**Request Headers**

| Name          | Type   | Required | Description           |
|---------------|--------|----------|-----------------------|
| TENANT_ID     | string | Yes      | Tenant identifier     |
| REGION        | string | Yes      | Region code           |
| MAJOR_VERSION | string | Yes      | Major version number  |
| MINOR_VERSION | string | Yes      | Minor version number  |

**Request Model**

JSON:API request with shop resource and included commodities.

| Field     | Type | Description                         |
|-----------|------|-------------------------------------|
| recharger | bool | Whether shop supports recharging    |

**Response Model**

JSON:API response with created shop resource.

**Error Conditions**

| Status | Description            |
|--------|------------------------|
| 201    | Created                |
| 400    | Bad request            |
| 500    | Internal server error  |

---

### PUT /api/npcs/{npcId}/shop

Updates an existing shop for a specific NPC. Replaces all commodities.

**Parameters**

| Name  | In   | Type   | Description             |
|-------|------|--------|-------------------------|
| npcId | path | uint32 | NPC template identifier |

**Request Headers**

| Name          | Type   | Required | Description           |
|---------------|--------|----------|-----------------------|
| TENANT_ID     | string | Yes      | Tenant identifier     |
| REGION        | string | Yes      | Region code           |
| MAJOR_VERSION | string | Yes      | Major version number  |
| MINOR_VERSION | string | Yes      | Minor version number  |

**Request Model**

JSON:API request with shop resource and included commodities.

| Field     | Type | Description                         |
|-----------|------|-------------------------------------|
| recharger | bool | Whether shop supports recharging    |

**Response Model**

JSON:API response with updated shop resource.

**Error Conditions**

| Status | Description            |
|--------|------------------------|
| 200    | Success                |
| 400    | Bad request            |
| 500    | Internal server error  |

---

### GET /api/npcs/{npcId}/shop/characters

Retrieves characters currently in a shop.

**Parameters**

| Name  | In   | Type   | Description             |
|-------|------|--------|-------------------------|
| npcId | path | uint32 | NPC template identifier |

**Request Headers**

| Name          | Type   | Required | Description           |
|---------------|--------|----------|-----------------------|
| TENANT_ID     | string | Yes      | Tenant identifier     |
| REGION        | string | Yes      | Region code           |
| MAJOR_VERSION | string | Yes      | Major version number  |
| MINOR_VERSION | string | Yes      | Minor version number  |

**Response Model**

JSON:API response with array of character identifiers.

| Field | Type   | Description           |
|-------|--------|-----------------------|
| type  | string | "characters"          |
| id    | string | Character identifier  |

**Error Conditions**

| Status | Description            |
|--------|------------------------|
| 500    | Internal server error  |

---

### POST /api/npcs/{npcId}/shop/relationships/commodities

Adds a new commodity to an NPC's shop.

**Parameters**

| Name  | In   | Type   | Description             |
|-------|------|--------|-------------------------|
| npcId | path | uint32 | NPC template identifier |

**Request Headers**

| Name          | Type   | Required | Description           |
|---------------|--------|----------|-----------------------|
| TENANT_ID     | string | Yes      | Tenant identifier     |
| REGION        | string | Yes      | Region code           |
| MAJOR_VERSION | string | Yes      | Major version number  |
| MINOR_VERSION | string | Yes      | Minor version number  |

**Request Model**

JSON:API request with commodity resource.

| Field           | Type   | Description                              |
|-----------------|--------|------------------------------------------|
| templateId      | uint32 | Item template identifier                 |
| mesoPrice       | uint32 | Price in mesos                           |
| discountRate    | byte   | Discount percentage                      |
| tokenTemplateId | uint32 | Alternative currency item identifier     |
| tokenPrice      | uint32 | Price in alternative currency            |
| period          | uint32 | Time limit on purchase in minutes        |
| levelLimit      | uint32 | Minimum level required                   |

**Response Model**

JSON:API response with created commodity resource.

**Error Conditions**

| Status | Description            |
|--------|------------------------|
| 201    | Created                |
| 500    | Internal server error  |

---

### PUT /api/npcs/{npcId}/shop/relationships/commodities/{commodityId}

Updates an existing commodity in a shop.

**Parameters**

| Name        | In   | Type      | Description             |
|-------------|------|-----------|-------------------------|
| npcId       | path | uint32    | NPC template identifier |
| commodityId | path | uuid.UUID | Commodity identifier    |

**Request Headers**

| Name          | Type   | Required | Description           |
|---------------|--------|----------|-----------------------|
| TENANT_ID     | string | Yes      | Tenant identifier     |
| REGION        | string | Yes      | Region code           |
| MAJOR_VERSION | string | Yes      | Major version number  |
| MINOR_VERSION | string | Yes      | Minor version number  |

**Request Model**

JSON:API request with commodity resource.

**Response Model**

JSON:API response with updated commodity resource.

**Error Conditions**

| Status | Description            |
|--------|------------------------|
| 200    | Success                |
| 500    | Internal server error  |

---

### DELETE /api/npcs/{npcId}/shop/relationships/commodities/{commodityId}

Removes a commodity from a shop.

**Parameters**

| Name        | In   | Type      | Description             |
|-------------|------|-----------|-------------------------|
| npcId       | path | uint32    | NPC template identifier |
| commodityId | path | uuid.UUID | Commodity identifier    |

**Request Headers**

| Name          | Type   | Required | Description           |
|---------------|--------|----------|-----------------------|
| TENANT_ID     | string | Yes      | Tenant identifier     |
| REGION        | string | Yes      | Region code           |
| MAJOR_VERSION | string | Yes      | Major version number  |
| MINOR_VERSION | string | Yes      | Minor version number  |

**Error Conditions**

| Status | Description            |
|--------|------------------------|
| 204    | Success (no content)   |
| 500    | Internal server error  |

---

### DELETE /api/npcs/{npcId}/shop/relationships/commodities

Deletes all commodities for an NPC's shop.

**Parameters**

| Name  | In   | Type   | Description             |
|-------|------|--------|-------------------------|
| npcId | path | uint32 | NPC template identifier |

**Request Headers**

| Name          | Type   | Required | Description           |
|---------------|--------|----------|-----------------------|
| TENANT_ID     | string | Yes      | Tenant identifier     |
| REGION        | string | Yes      | Region code           |
| MAJOR_VERSION | string | Yes      | Major version number  |
| MINOR_VERSION | string | Yes      | Minor version number  |

**Error Conditions**

| Status | Description            |
|--------|------------------------|
| 204    | Success (no content)   |
| 500    | Internal server error  |

---

### POST /api/shops/seed

Seeds the database with shop data from JSON files on disk. Deletes all existing shops and commodities for the tenant before seeding.

**Request Headers**

| Name          | Type   | Required | Description           |
|---------------|--------|----------|-----------------------|
| TENANT_ID     | string | Yes      | Tenant identifier     |
| REGION        | string | Yes      | Region code           |
| MAJOR_VERSION | string | Yes      | Major version number  |
| MINOR_VERSION | string | Yes      | Minor version number  |

**Response Model**

JSON response (not JSON:API).

| Field              | Type     | Description                      |
|--------------------|----------|----------------------------------|
| deletedShops       | int      | Number of shops deleted          |
| deletedCommodities | int      | Number of commodities deleted    |
| createdShops       | int      | Number of shops created          |
| createdCommodities | int      | Number of commodities created    |
| failedCount        | int      | Number of failed operations      |
| errors             | []string | Error messages (omitted if none) |

**Error Conditions**

| Status | Description            |
|--------|------------------------|
| 200    | Success                |
| 500    | Internal server error  |
