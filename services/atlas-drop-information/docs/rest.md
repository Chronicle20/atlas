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
  - `item_id` (uint32)
  - `minimum_quantity` (uint32)
  - `maximum_quantity` (uint32)
  - `quest_id` (uint32)
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
- Attributes:
  - `drops` (array of drop objects)
- Relationships:
  - `drops` - embedded drop resources

**Drop Attributes**
- `item_id` (uint32)
- `minimum_quantity` (uint32)
- `maximum_quantity` (uint32)
- `quest_id` (uint32)
- `chance` (uint32)

**Error Conditions**
- `500 Internal Server Error` - Database or processing error

---

### POST /api/drops/seed

Seeds the database with drop data from JSON files on the filesystem.

**Parameters**

None

**Request Headers**
- Tenant header (required)

**Request Model**

None

**Response Model**
```json
{
  "monsterDrops": {
    "deletedCount": 0,
    "createdCount": 0,
    "failedCount": 0,
    "errors": []
  },
  "continentDrops": {
    "deletedCount": 0,
    "createdCount": 0,
    "failedCount": 0,
    "errors": []
  }
}
```

**Error Conditions**
- `500 Internal Server Error` - Seed operation failed
