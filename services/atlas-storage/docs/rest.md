# REST

## Endpoints

### GET /api/storage/accounts/{accountId}

Retrieves storage for an account. Creates storage if it does not exist.

**Parameters**
- `accountId` (path): Account identifier (uint32)
- `worldId` (query): World identifier (byte, required)

**Request Model**

None.

**Response Model**

```json
{
  "data": {
    "type": "storages",
    "id": "uuid",
    "attributes": {
      "world_id": 0,
      "account_id": 12345,
      "capacity": 4,
      "mesos": 0
    },
    "relationships": {
      "assets": {
        "data": [
          { "type": "storage_assets", "id": "1" }
        ]
      }
    }
  },
  "included": [
    {
      "type": "storage_assets",
      "id": "1",
      "attributes": {
        "id": 1,
        "slot": 0,
        "templateId": 1322005,
        "expiration": "0001-01-01T00:00:00Z",
        "quantity": 0,
        "ownerId": 0,
        "flag": 0,
        "rechargeable": 0,
        "strength": 15,
        "dexterity": 10,
        "intelligence": 0,
        "luck": 0,
        "hp": 0,
        "mp": 0,
        "weaponAttack": 0,
        "magicAttack": 0,
        "weaponDefense": 0,
        "magicDefense": 0,
        "accuracy": 0,
        "avoidability": 0,
        "hands": 0,
        "speed": 0,
        "jump": 0,
        "slots": 7,
        "levelType": 0,
        "level": 0,
        "experience": 0,
        "hammersApplied": 0,
        "cashId": "0",
        "commodityId": 0,
        "purchaseBy": 0,
        "petId": 0
      }
    }
  ]
}
```

**Error Conditions**
- `400 Bad Request`: Invalid accountId or missing/invalid worldId query parameter
- `500 Internal Server Error`: Database or transform error

---

### POST /api/storage/accounts/{accountId}

Creates storage for an account.

**Parameters**
- `accountId` (path): Account identifier (uint32)
- `worldId` (query): World identifier (byte, required)

**Request Model**

None.

**Response Model**

Same as GET response.

**Error Conditions**
- `400 Bad Request`: Invalid accountId or missing/invalid worldId query parameter
- `409 Conflict`: Storage already exists for account and world
- `500 Internal Server Error`: Database error

---

### GET /api/storage/accounts/{accountId}/assets

Retrieves all assets for an account's storage. Creates the storage if it does not exist.

**Parameters**
- `accountId` (path): Account identifier (uint32)
- `worldId` (query): World identifier (byte, required)

**Request Model**

None.

**Response Model**

```json
{
  "data": [
    {
      "type": "storage_assets",
      "id": "1",
      "attributes": {
        "id": 1,
        "slot": 0,
        "templateId": 2000000,
        "expiration": "0001-01-01T00:00:00Z",
        "quantity": 100,
        "ownerId": 0,
        "flag": 0,
        "rechargeable": 0,
        "strength": 0,
        "dexterity": 0,
        "intelligence": 0,
        "luck": 0,
        "hp": 0,
        "mp": 0,
        "weaponAttack": 0,
        "magicAttack": 0,
        "weaponDefense": 0,
        "magicDefense": 0,
        "accuracy": 0,
        "avoidability": 0,
        "hands": 0,
        "speed": 0,
        "jump": 0,
        "slots": 0,
        "levelType": 0,
        "level": 0,
        "experience": 0,
        "hammersApplied": 0,
        "cashId": "0",
        "commodityId": 0,
        "purchaseBy": 0,
        "petId": 0
      }
    }
  ]
}
```

**Error Conditions**
- `400 Bad Request`: Invalid accountId or missing/invalid worldId query parameter
- `500 Internal Server Error`: Database or transform error

---

### GET /api/storage/accounts/{accountId}/assets/{assetId}

Retrieves a specific asset by ID.

**Parameters**
- `accountId` (path): Account identifier (uint32)
- `assetId` (path): Asset identifier (uint32)
- `worldId` (query): World identifier (byte, required)

**Request Model**

None.

**Response Model**

```json
{
  "data": {
    "type": "storage_assets",
    "id": "1",
    "attributes": {
      "id": 1,
      "slot": 0,
      "templateId": 2000000,
      "expiration": "0001-01-01T00:00:00Z",
      "quantity": 100,
      "ownerId": 0,
      "flag": 0,
      "rechargeable": 0,
      "strength": 0,
      "dexterity": 0,
      "intelligence": 0,
      "luck": 0,
      "hp": 0,
      "mp": 0,
      "weaponAttack": 0,
      "magicAttack": 0,
      "weaponDefense": 0,
      "magicDefense": 0,
      "accuracy": 0,
      "avoidability": 0,
      "hands": 0,
      "speed": 0,
      "jump": 0,
      "slots": 0,
      "levelType": 0,
      "level": 0,
      "experience": 0,
      "hammersApplied": 0,
      "cashId": "0",
      "commodityId": 0,
      "purchaseBy": 0,
      "petId": 0
    }
  }
}
```

**Error Conditions**
- `400 Bad Request`: Invalid accountId, assetId, or missing/invalid worldId query parameter
- `404 Not Found`: Asset does not exist
- `500 Internal Server Error`: Transform error

---

### GET /api/storage/projections/{characterId}

Retrieves the in-memory storage projection for an active character session.

**Parameters**
- `characterId` (path): Character identifier (uint32)

**Request Model**

None.

**Response Model**

```json
{
  "data": {
    "type": "storage_projections",
    "id": "12345",
    "attributes": {
      "characterId": 12345,
      "accountId": 67890,
      "worldId": 0,
      "storageId": "uuid",
      "capacity": 4,
      "mesos": 1000,
      "npcId": 9030000,
      "compartments": {
        "equip": [
          {
            "id": 1,
            "slot": 0,
            "templateId": 1322005,
            "expiration": "0001-01-01T00:00:00Z",
            "quantity": 0,
            "ownerId": 0,
            "flag": 0,
            "rechargeable": 0,
            "strength": 15,
            "dexterity": 10,
            "slots": 7
          }
        ],
        "use": [],
        "setup": [],
        "etc": [],
        "cash": []
      }
    }
  }
}
```

**Error Conditions**
- `400 Bad Request`: Invalid characterId
- `404 Not Found`: Projection does not exist (storage not open)
- `500 Internal Server Error`: Transform error

---

### GET /api/storage/projections/{characterId}/compartments/{compartmentType}/assets/{slot}

Retrieves a specific asset from a projection compartment by slot.

**Parameters**
- `characterId` (path): Character identifier (uint32)
- `compartmentType` (path): Inventory type (1-5 or name: equip, use, setup, etc, cash)
- `slot` (path): Slot index (int16)

**Request Model**

None.

**Response Model**

```json
{
  "data": {
    "type": "storage_assets",
    "id": "1",
    "attributes": {
      "id": 1,
      "slot": 0,
      "templateId": 2000000,
      "expiration": "0001-01-01T00:00:00Z",
      "quantity": 100,
      "ownerId": 0,
      "flag": 0,
      "rechargeable": 0,
      "strength": 0,
      "dexterity": 0,
      "intelligence": 0,
      "luck": 0,
      "hp": 0,
      "mp": 0,
      "weaponAttack": 0,
      "magicAttack": 0,
      "weaponDefense": 0,
      "magicDefense": 0,
      "accuracy": 0,
      "avoidability": 0,
      "hands": 0,
      "speed": 0,
      "jump": 0,
      "slots": 0,
      "levelType": 0,
      "level": 0,
      "experience": 0,
      "hammersApplied": 0,
      "cashId": "0",
      "commodityId": 0,
      "purchaseBy": 0,
      "petId": 0
    }
  }
}
```

**Error Conditions**
- `400 Bad Request`: Invalid characterId, compartment type, or slot
- `404 Not Found`: Projection does not exist or asset not found at slot
- `500 Internal Server Error`: Transform error
