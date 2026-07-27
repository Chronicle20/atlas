# REST API

## Endpoints

### GET /api/configurations/templates

Retrieves all configuration templates, paginated.

**Parameters**

| Name | Type | Location | Required |
|------|------|----------|----------|
| page[number] | int | query | no (default 1) |
| page[size] | int | query | no (default 50, max 250) |

The legacy `limit` query parameter is rejected.

**Response Model**

Paginated array of `templates` resources, with a `meta` block (`total`, `page.number`, `page.size`, `page.last`) and `self`/`first`/`last`/`prev`/`next` links

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 400 | Invalid page[number] or page[size], or legacy `limit` param supplied |
| 500 | Database error |

---

### GET /api/configurations/templates?region={region}&majorVersion={majorVersion}&minorVersion={minorVersion}

Retrieves a configuration template by region and version.

**Parameters**

| Name | Type | Location | Required |
|------|------|----------|----------|
| region | string | query | yes |
| majorVersion | uint16 | query | yes |
| minorVersion | uint16 | query | yes |

**Response Model**

Single `templates` resource

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 400 | Invalid majorVersion or minorVersion |
| 500 | Database error or record not found |

---

### GET /api/configurations/templates/{templateId}

Retrieves a configuration template by ID.

**Parameters**

| Name | Type | Location | Required |
|------|------|----------|----------|
| templateId | UUID | path | yes |

**Response Model**

Single `templates` resource

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 400 | Invalid UUID format |
| 500 | Database error or record not found |

---

### POST /api/configurations/templates

Creates a new configuration template.

**Parameters**

None

**Request Model**

JSON:API `templates` resource with attributes:
- `region` (string)
- `majorVersion` (uint16)
- `minorVersion` (uint16)
- `usesPin` (bool)
- `socket` (object)
- `characters` (object)
- `npcs` (array)
- `worlds` (array)
- `cashShop` (object)

**Response Model**

Created `templates` resource

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 400 | Invalid JSON or deserialization error |
| 500 | Database error |

---

### PATCH /api/configurations/templates/{templateId}

Updates an existing configuration template.

**Parameters**

| Name | Type | Location | Required |
|------|------|----------|----------|
| templateId | UUID | path | yes |

**Request Model**

JSON:API `templates` resource with attributes to update

**Response Model**

None (empty body on success)

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 400 | Invalid UUID format or JSON |
| 400 | Character preset validation failed (JSON:API `errors` array; each entry has `status`, `title`, `detail`, and `meta.path`) |
| 500 | Database error or record not found |

---

### DELETE /api/configurations/templates/{templateId}

Deletes a configuration template.

**Parameters**

| Name | Type | Location | Required |
|------|------|----------|----------|
| templateId | UUID | path | yes |

**Response Model**

None (empty body on success)

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 400 | Invalid UUID format |
| 500 | Database error or record not found |

---

### GET /api/configurations/tenants

Retrieves all configuration tenants, paginated.

**Parameters**

| Name | Type | Location | Required |
|------|------|----------|----------|
| page[number] | int | query | no (default 1) |
| page[size] | int | query | no (default 50, max 250) |

The legacy `limit` query parameter is rejected.

**Response Model**

Paginated array of `tenants` resources, with a `meta` block (`total`, `page.number`, `page.size`, `page.last`) and `self`/`first`/`last`/`prev`/`next` links

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 400 | Invalid page[number] or page[size], or legacy `limit` param supplied |
| 500 | Database error |

---

### GET /api/configurations/tenants/{tenantId}

Retrieves a configuration tenant by ID.

**Parameters**

| Name | Type | Location | Required |
|------|------|----------|----------|
| tenantId | UUID | path | yes |

**Response Model**

Single `tenants` resource

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 400 | Invalid UUID format |
| 500 | Database error or record not found |

---

### POST /api/configurations/tenants

Creates a new configuration tenant.

**Parameters**

None

**Request Model**

JSON:API `tenants` resource with attributes:
- `id` (string, optional - generated if not provided)
- `region` (string)
- `majorVersion` (uint16)
- `minorVersion` (uint16)
- `usesPin` (bool)
- `socket` (object)
- `characters` (object)
- `npcs` (array)
- `worlds` (array)
- `cashShop` (object)

**Response Model**

Created `tenants` resource

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 400 | Invalid JSON or deserialization error |
| 500 | Database error |

---

### PATCH /api/configurations/tenants/{tenantId}

Updates an existing configuration tenant. Creates a history record before updating.

**Parameters**

| Name | Type | Location | Required |
|------|------|----------|----------|
| tenantId | UUID | path | yes |

**Request Model**

JSON:API `tenants` resource with attributes to update

**Response Model**

None (empty body on success)

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 400 | Invalid UUID format or JSON |
| 400 | Character preset validation failed (JSON:API `errors` array; each entry has `status`, `title`, `detail`, and `meta.path`) |
| 500 | Database error or record not found |

---

### DELETE /api/configurations/tenants/{tenantId}

Deletes a configuration tenant. Creates a history record before deleting.

**Parameters**

| Name | Type | Location | Required |
|------|------|----------|----------|
| tenantId | UUID | path | yes |

**Response Model**

None (empty body on success)

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 400 | Invalid UUID format |
| 500 | Database error or record not found |

---

### GET /api/configurations/services

Retrieves all service configurations, paginated.

**Parameters**

| Name | Type | Location | Required |
|------|------|----------|----------|
| page[number] | int | query | no (default 1) |
| page[size] | int | query | no (default 50, max 250) |

The legacy `limit` query parameter is rejected.

**Response Model**

Paginated array of `services` resources, with a `meta` block (`total`, `page.number`, `page.size`, `page.last`) and `self`/`first`/`last`/`prev`/`next` links

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 400 | Invalid page[number] or page[size], or legacy `limit` param supplied |
| 500 | Database error |

---

### GET /api/configurations/services/{serviceId}

Retrieves a service configuration by ID.

**Parameters**

| Name | Type | Location | Required |
|------|------|----------|----------|
| serviceId | UUID | path | yes |

**Response Model**

Single `services` resource

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 400 | Invalid UUID format |
| 500 | Database error or record not found |

---

### POST /api/configurations/services

Creates a new service configuration.

**Parameters**

None

**Request Model**

JSON:API `services` resource with attributes:
- `id` (string, optional - generated if not provided)
- `type` (string, required - must be `login-service`, `channel-service`, or `drops-service`)
- `tasks` (array)
- `tenants` (object, optional - structure varies by service type)

**Response Model**

Created `services` resource

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 400 | Invalid service type |
| 500 | Database error |

---

### PATCH /api/configurations/services/{serviceId}

Updates an existing service configuration. Creates a history record before updating.

**Parameters**

| Name | Type | Location | Required |
|------|------|----------|----------|
| serviceId | UUID | path | yes |

**Request Model**

JSON:API `services` resource with attributes to update

**Response Model**

Updated `services` resource

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 400 | Invalid UUID format, invalid service type, or invalid JSON |
| 500 | Database error or record not found |

---

### DELETE /api/configurations/services/{serviceId}

Deletes a service configuration. Creates a history record before deleting.

**Parameters**

| Name | Type | Location | Required |
|------|------|----------|----------|
| serviceId | UUID | path | yes |

**Response Model**

None (204 No Content on success)

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 400 | Invalid UUID format |
| 500 | Database error or record not found |
