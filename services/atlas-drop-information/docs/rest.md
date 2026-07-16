# REST API

## Endpoints

### GET /api/monsters/{monsterId}/drops

Retrieves a page of drop entries for a specific monster.

**Parameters**
- `monsterId` (path, required) - Monster identifier (uint32)
- `page[number]` (query, optional) - Page number, default 1
- `page[size]` (query, optional) - Page size, default 50, max 250

**Request Headers**
- Tenant header (required)

**Response Model**
- Resource type: `drops`
- Attributes:
  - `monsterId` (uint32)
  - `itemId` (uint32)
  - `minimumQuantity` (uint32)
  - `maximumQuantity` (uint32)
  - `questId` (uint32)
  - `chance` (uint32)
- Paginated response envelope: `meta.total`, `meta.page.number`, `meta.page.size`, `meta.page.last`, and `links` (`self`, `first`, `prev`, `next`, `last`)

**Error Conditions**
- `400 Bad Request` - Invalid monsterId format, or invalid `page[number]`/`page[size]`
- `404 Not Found` - No drops found for monster
- `500 Internal Server Error` - Database or processing error

---

### GET /api/items/{itemId}/drops

Retrieves a page of monster-drop entries for a specific item.

**Parameters**
- `itemId` (path, required) - Item identifier (uint32)
- `page[number]` (query, optional) - Page number, default 1
- `page[size]` (query, optional) - Page size, default 50, max 250

**Request Headers**
- Tenant header (required)

**Response Model**
- Resource type: `drops`
- Attributes:
  - `monsterId` (uint32)
  - `itemId` (uint32)
  - `minimumQuantity` (uint32)
  - `maximumQuantity` (uint32)
  - `questId` (uint32)
  - `chance` (uint32)
- Paginated response envelope: `meta.total`, `meta.page.number`, `meta.page.size`, `meta.page.last`, and `links` (`self`, `first`, `prev`, `next`, `last`)

**Error Conditions**
- `400 Bad Request` - Invalid itemId format, or invalid `page[number]`/`page[size]`
- `404 Not Found` - No drops found for item
- `500 Internal Server Error` - Database or processing error

---

### GET /api/continents/drops

Retrieves a page of continent-wide drop entries grouped by continent.

**Parameters**
- `page[number]` (query, optional) - Page number, default 1
- `page[size]` (query, optional) - Page size, default 50, max 250

**Request Headers**
- Tenant header (required)

**Response Model**
- Resource type: `continents`
- Relationships:
  - `drops` - included drop resources
- Paginated response envelope: `meta.total`, `meta.page.number`, `meta.page.size`, `meta.page.last`, and `links` (`self`, `first`, `prev`, `next`, `last`)

**Drop Attributes**
- Resource type: `drops`
- `itemId` (uint32)
- `minimumQuantity` (uint32)
- `maximumQuantity` (uint32)
- `questId` (uint32)
- `chance` (uint32)

**Error Conditions**
- `400 Bad Request` - Invalid `page[number]`/`page[size]`
- `500 Internal Server Error` - Database or processing error

---

### GET /api/reactors/{reactorId}/drops

Retrieves all drop entries for a specific reactor.

**Parameters**
- `reactorId` (path, required) - Reactor identifier (uint32)

**Request Headers**
- Tenant header (required)

**Response Model**
- Resource type: `reactors`
- Relationships:
  - `drops` - embedded drop resources

**Drop Attributes**
- Resource type: `drops`
- `itemId` (uint32)
- `questId` (uint32, omitted if 0)
- `chance` (uint32)

**Error Conditions**
- `400 Bad Request` - Invalid reactorId format
- `500 Internal Server Error` - Database or processing error

---

### POST /api/drops/seed

Seeds the database with drop data from the configured seed catalog. Returns immediately with 202 Accepted and processes seeding in the background.

**Parameters**

None

**Request Headers**
- Tenant header (required)

**Request Model**

None

**Response**
- `202 Accepted` - Seed operation started in background

**Error Conditions**
- None (errors are logged server-side)

---

### GET /api/drops/seed/status

Reports the seed catalog status for the `drops` group (monster, continent, and reactor drop subdomains). Response is a plain JSON document, not a JSON:API resource.

**Parameters**

None

**Request Headers**
- Tenant header (required)

**Response Model**
- `groupName` (string) - always `"drops"`
- `subdomains` (object) - keyed by subdomain name (`monster-drop`, `continent-drop`, `reactor-drop`), each with:
  - `count` (int64)
  - `updatedAt` (timestamp, nullable)
- `updatedAt` (timestamp, nullable) - latest subdomain update time
- `catalogRevision` (string) - revision of the on-disk seed catalog
- `tenantSeededRevision` (string, nullable) - catalog revision recorded at last successful seed for the tenant
- `tenantSeededAt` (timestamp, nullable) - time of last successful seed for the tenant

**Error Conditions**
- `500 Internal Server Error` - Failure reading seed status
