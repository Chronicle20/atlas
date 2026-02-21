# REST API

## Endpoints

### GET /api/monsters/{monsterId}/drops

Retrieves all drop entries for a specific monster.

**Parameters**
- `monsterId` (path, required) - Monster identifier (uint32)

**Request Headers**
- Tenant header (required)

**Response Model**
- Resource type: `drops`
- Attributes:
  - `itemId` (uint32)
  - `minimumQuantity` (uint32)
  - `maximumQuantity` (uint32)
  - `questId` (uint32)
  - `chance` (uint32)

**Error Conditions**
- `400 Bad Request` - Invalid monsterId format
- `404 Not Found` - No drops found for monster
- `500 Internal Server Error` - Database or processing error

---

### GET /api/continents/drops

Retrieves all continent-wide drop entries grouped by continent.

**Parameters**

None

**Request Headers**
- Tenant header (required)

**Response Model**
- Resource type: `continents`
- Relationships:
  - `drops` - included drop resources

**Drop Attributes**
- Resource type: `drops`
- `itemId` (uint32)
- `minimumQuantity` (uint32)
- `maximumQuantity` (uint32)
- `questId` (uint32)
- `chance` (uint32)

**Error Conditions**
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

Seeds the database with drop data from JSON files on the filesystem. Returns immediately with 202 Accepted and processes seeding in the background.

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
