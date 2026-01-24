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
        "templateId": 2000000,
        "expiration": "0001-01-01T00:00:00Z",
        "referenceId": 0,
        "referenceType": "consumable",
        "referenceData": {
          "ownerId": 0,
          "quantity": 100,
          "flag": 0,
          "rechargeable": 0
        }
      }
    }
  ]
}
```

**Error Conditions**
- `500 Internal Server Error`: Database or decoration error

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
- `409 Conflict`: Storage already exists for account and world
- `500 Internal Server Error`: Database error

---

### GET /api/storage/accounts/{accountId}/assets

Retrieves all assets for an account's storage.

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
        "referenceId": 0,
        "referenceType": "consumable",
        "referenceData": {
          "ownerId": 0,
          "quantity": 100,
          "flag": 0,
          "rechargeable": 0
        }
      }
    }
  ]
}
```

**Error Conditions**
- `500 Internal Server Error`: Database or decoration error

---

### GET /api/storage/accounts/{accountId}/assets/{assetId}

Retrieves a specific asset.

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
      "storage_id": "uuid",
      "inventory_type": 2,
      "slot": 0,
      "template_id": 2000000,
      "expiration": "0001-01-01T00:00:00Z",
      "reference_id": 0,
      "reference_type": "consumable",
      "quantity": 100,
      "owner_id": 0,
      "flag": 0
    }
  }
}
```

**Error Conditions**
- `404 Not Found`: Asset does not exist
- `500 Internal Server Error`: Database error

---

### GET /api/storage/projections/{characterId}

Retrieves the storage projection for a character.

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
        "equip": [],
        "use": [
          {
            "id": 1,
            "slot": 0,
            "templateId": 2000000,
            "expiration": "0001-01-01T00:00:00Z",
            "referenceId": 0,
            "referenceType": "consumable",
            "referenceData": {
              "ownerId": 0,
              "quantity": 100,
              "flag": 0,
              "rechargeable": 0
            }
          }
        ],
        "setup": [],
        "etc": [],
        "cash": []
      }
    }
  }
}
```

**Error Conditions**
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
      "referenceId": 0,
      "referenceType": "consumable",
      "referenceData": {
        "ownerId": 0,
        "quantity": 100,
        "flag": 0,
        "rechargeable": 0
      }
    }
  }
}
```

**Error Conditions**
- `400 Bad Request`: Invalid compartment type
- `404 Not Found`: Projection does not exist or asset not found at slot
- `500 Internal Server Error`: Transform error
