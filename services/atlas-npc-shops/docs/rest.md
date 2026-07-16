# REST

## Endpoints

### GET /api/shops

Retrieves all shops for the current tenant. Paginated.

**Parameters**

| Name         | In    | Type   | Description                                |
|--------------|-------|--------|---------------------------------------------|
| include      | query | string | Optional. "commodities" to include items    |
| page[number] | query | int    | Optional. Page number, 1-based (default 1)  |
| page[size]   | query | int    | Optional. Page size (default 50, max 250)   |

**Request Headers**

| Name          | Type   | Required | Description           |
|---------------|--------|----------|-----------------------|
| TENANT_ID     | string | Yes      | Tenant identifier     |
| REGION        | string | Yes      | Region code           |
| MAJOR_VERSION | string | Yes      | Major version number  |
| MINOR_VERSION | string | Yes      | Minor version number  |

**Response Model**

Paginated JSON:API response with array of shop resources.

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

| Status | Description                        |
|--------|-------------------------------------|
| 400    | Invalid page[number] or page[size]  |
| 500    | Internal server error               |

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

Retrieves characters currently in a shop. Registry-backed (in-memory, not a database query). Paginated.

**Parameters**

| Name         | In    | Type   | Description                                 |
|--------------|-------|--------|-----------------------------------------------|
| npcId        | path  | uint32 | NPC template identifier                       |
| page[number] | query | int    | Optional. Page number, 1-based (default 1)    |
| page[size]   | query | int    | Optional. Page size (default 250, max 250)    |

**Request Headers**

| Name          | Type   | Required | Description           |
|---------------|--------|----------|-----------------------|
| TENANT_ID     | string | Yes      | Tenant identifier     |
| REGION        | string | Yes      | Region code           |
| MAJOR_VERSION | string | Yes      | Major version number  |
| MINOR_VERSION | string | Yes      | Minor version number  |

**Response Model**

Paginated JSON:API response with array of character identifiers, stable-sorted ascending by character ID.

| Field | Type   | Description           |
|-------|--------|-----------------------|
| type  | string | "characters"          |
| id    | string | Character identifier  |

**Error Conditions**

| Status | Description                        |
|--------|-------------------------------------|
| 400    | Invalid page[number] or page[size]  |
| 500    | Internal server error               |

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

Triggers an asynchronous seed of shop and commodity data from JSON files on disk (catalog root: SEED_CATALOG_ROOT, default `./deploy/seed`, under `npc-shops/shops`). The request returns immediately; the seed itself runs in the background. Deletes all existing commodities then all existing shops for the tenant before bulk-creating from file data.

**Request Headers**

| Name          | Type   | Required | Description           |
|---------------|--------|----------|-----------------------|
| TENANT_ID     | string | Yes      | Tenant identifier     |
| REGION        | string | Yes      | Region code           |
| MAJOR_VERSION | string | Yes      | Major version number  |
| MINOR_VERSION | string | Yes      | Minor version number  |

**Response Model**

No response body.

**Error Conditions**

| Status | Description                          |
|--------|----------------------------------------|
| 202    | Accepted; seed started in the background |
| 400    | Missing/invalid tenant headers          |

---

### GET /api/shops/seed/status

Retrieves the status of the most recent seed operation for the tenant, along with current row counts.

**Request Headers**

| Name          | Type   | Required | Description           |
|---------------|--------|----------|-----------------------|
| TENANT_ID     | string | Yes      | Tenant identifier     |
| REGION        | string | Yes      | Region code           |
| MAJOR_VERSION | string | Yes      | Major version number  |
| MINOR_VERSION | string | Yes      | Minor version number  |

**Response Model**

JSON response (not JSON:API).

| Field                | Type                        | Description                                                        |
|----------------------|------------------------------|---------------------------------------------------------------------|
| groupName            | string                       | Seed group name ("npc-shops")                                       |
| subdomains           | map[string]SubdomainStatus   | Per-subdomain status, keyed by "npc-shops" and "commodities"        |
| updatedAt            | timestamp \| null            | Most recent subdomain update timestamp                              |
| catalogRevision      | string                       | Revision identifier of the on-disk seed catalog                     |
| tenantSeededRevision | string \| null               | Catalog revision the tenant was last seeded with                    |
| tenantSeededAt       | timestamp \| null            | Timestamp of the tenant's last completed seed                       |

**SubdomainStatus**

| Field     | Type               | Description                          |
|-----------|--------------------|----------------------------------------|
| count     | int64              | Current row count for the subdomain    |
| updatedAt | timestamp \| null  | Most recent row update timestamp       |

**Error Conditions**

| Status | Description                     |
|--------|-----------------------------------|
| 200    | Success                           |
| 400    | Missing/invalid tenant headers    |
| 500    | Internal server error             |

---

### GET /api/commodities/items/{itemId}

Retrieves all commodities across shops that reference the given item template ID. Paginated.

**Parameters**

| Name         | In    | Type   | Description                                 |
|--------------|-------|--------|-----------------------------------------------|
| itemId       | path  | uint32 | Item template identifier                      |
| page[number] | query | int    | Optional. Page number, 1-based (default 1)    |
| page[size]   | query | int    | Optional. Page size (default 250, max 250)    |

**Request Headers**

| Name          | Type   | Required | Description           |
|---------------|--------|----------|-----------------------|
| TENANT_ID     | string | Yes      | Tenant identifier     |
| REGION        | string | Yes      | Region code           |
| MAJOR_VERSION | string | Yes      | Major version number  |
| MINOR_VERSION | string | Yes      | Minor version number  |

**Response Model**

Paginated JSON:API response with array of commodity resources.

| Field      | Type   | Description        |
|------------|--------|---------------------|
| type       | string | "commodities"       |
| id         | string | Commodity identifier |
| attributes | object | Commodity attributes |

**Attributes**

| Field           | Type   | Description                              |
|-----------------|--------|-------------------------------------------|
| npcId           | uint32 | NPC template identifier                   |
| templateId      | uint32 | Item template identifier                  |
| mesoPrice       | uint32 | Price in mesos                            |
| discountRate    | byte   | Discount percentage                       |
| tokenTemplateId | uint32 | Alternative currency item identifier      |
| tokenPrice      | uint32 | Price in alternative currency             |
| period          | uint32 | Time limit on purchase in minutes         |
| levelLimit      | uint32 | Minimum level required                    |

**Error Conditions**

| Status | Description                        |
|--------|-------------------------------------|
| 400    | Invalid itemId, page[number], or page[size] |
| 500    | Internal server error               |
