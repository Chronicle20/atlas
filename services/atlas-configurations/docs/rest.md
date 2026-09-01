# REST API

## Endpoints

All endpoints below accept an optional `ENVIRONMENT` request header naming the caller's execution environment. When absent, requests are treated as legacy (unscoped) callers. When present, it must name a known environment that is not `DEACTIVATING` or `DELETED`, or the request is rejected with 400. A write that targets a row owned by a different environment than the caller's is rejected with 403 (JSON:API `errors` array; entry has `status`, `title`, `detail`).

### GET /api/configurations/templates

Retrieves all configuration templates, paginated.

**Parameters**

| Name | Type | Location | Required |
|------|------|----------|----------|
| page[number] | int | query | no (default 1) |
| page[size] | int | query | no (default 50, max 250) |

The legacy `limit` query parameter is rejected.

**Response Model**

Paginated array of `templates` resources, with a `meta` block (`total`, `page.number`, `page.size`, `page.last`) and `self`/`first`/`last`/`prev`/`next` links. Each resource's attributes include `shippedRevision`, `storedRevision`, and `seedDrift` in addition to the template's own fields.

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

Single `templates` resource, including `shippedRevision`, `storedRevision`, and `seedDrift`.

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

Single `templates` resource, including `shippedRevision`, `storedRevision`, and `seedDrift`.

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
- `mapleLife` (object)
- `diagnostics` (object) - currently holds only `tracePackets` (boolean, default `false`); enabling it writes plaintext credentials to the serving pod's logs

**Response Model**

Created `templates` resource, read back through the same view used by GET (includes `shippedRevision`, `storedRevision`, `seedDrift`).

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 400 | Invalid JSON or deserialization error |
| 400 | Socket document validation failed (JSON:API `errors` array; each entry has `status`, `title`, `detail`, and `meta.path`) |
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
| 400 | Character preset or socket document validation failed (JSON:API `errors` array; each entry has `status`, `title`, `detail`, and `meta.path`) |
| 403 | Write targets a template owned by another environment |
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
| 403 | Delete targets a template owned by another environment |
| 500 | Database error or record not found |

---

### POST /api/configurations/templates/{templateId}/reseed

Replaces a template's stored content with the file this image ships for its region/version.

**Parameters**

| Name | Type | Location | Required |
|------|------|----------|----------|
| templateId | UUID | path | yes |

**Request Model**

None

**Response Model**

None (204 No Content on success)

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 400 | Invalid UUID format |
| 400 | The shipped seed file for the row's region/version fails write-path validation (JSON:API `errors` array; each entry has `status`, `title`, `detail`, and `meta.path`) |
| 404 | No template exists with the given ID (JSON:API `errors` array; entry has `status`, `title`, `detail`) |
| 409 | This image ships no seed file for the template's region and version (JSON:API `errors` array; entry has `status`, `title`, `detail`) |
| 500 | Database error |

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
- `mapleLife` (object)
- `diagnostics` (object) - currently holds only `tracePackets` (boolean, default `false`); enabling it writes plaintext credentials to the serving pod's logs

**Response Model**

Created `tenants` resource

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 400 | Invalid JSON or deserialization error |
| 400 | Socket document validation failed (JSON:API `errors` array; each entry has `status`, `title`, `detail`, and `meta.path`) |
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
| 400 | Character preset or socket document validation failed (JSON:API `errors` array; each entry has `status`, `title`, `detail`, and `meta.path`) |
| 403 | Write targets a tenant owned by another environment |
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
| 403 | Delete targets a tenant owned by another environment |
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
| 403 | Write targets a service configuration owned by another environment |
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
| 403 | Delete targets a service configuration owned by another environment |
| 500 | Database error or record not found |

---

### GET /api/configurations/environments

Retrieves all environments, paginated.

**Parameters**

| Name | Type | Location | Required |
|------|------|----------|----------|
| page[number] | int | query | no (default 1) |
| page[size] | int | query | no (default 50, max 250) |

**Response Model**

Paginated array of `environments` resources, with a `meta` block (`total`, `page.number`, `page.size`, `page.last`) and `self`/`first`/`last`/`prev`/`next` links

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 400 | Invalid page[number] or page[size] |
| 500 | Database error |

---

### GET /api/configurations/environments/{name}

Retrieves an environment by name.

**Parameters**

| Name | Type | Location | Required |
|------|------|----------|----------|
| name | string | path | yes |

**Response Model**

Single `environments` resource

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 500 | Database error or record not found |

---

### POST /api/configurations/environments

Creates a new environment.

**Parameters**

None

**Request Model**

JSON:API `environments` resource with attributes:
- `name` (string, required, must be a well-formed environment id)
- `baseline` (string)
- `namespace` (string)
- `tenant` (string)
- `overrides` (object, map of service name to namespace)
- `phase` (string, required - must be `PROVISIONING`, `ACTIVE`, `DEACTIVATING`, or `DELETED`)

**Response Model**

Created `environments` resource

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 400 | `name` missing or not well-formed, or `phase` not one of the valid values |
| 500 | Database error |

---

### PATCH /api/configurations/environments/{name}

Updates an existing environment. Fields omitted from the request body retain their previously stored value.

**Parameters**

| Name | Type | Location | Required |
|------|------|----------|----------|
| name | string | path | yes |

**Request Model**

JSON:API `environments` resource with attributes to update

**Response Model**

Updated `environments` resource

**Error Conditions**

| Status | Condition |
|--------|-----------|
| 400 | `name` not well-formed, `phase` not one of the valid values, or the phase transition is not legal (must be a no-op or advance exactly one step along `PROVISIONING` -> `ACTIVE` -> `DEACTIVATING` -> `DELETED`) |
| 500 | Database error or record not found |
