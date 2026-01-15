# REST

## Endpoints

### GET /api/accounts/{accountId}/wallet

Retrieves a wallet for an account.

#### Parameters
| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| accountId | path | uint32 | yes | Account ID |

#### Request Model
None.

#### Response Model
JSON:API resource type: `wallets`

```json
{
  "data": {
    "type": "wallets",
    "id": "uuid",
    "attributes": {
      "accountId": 12345,
      "credit": 1000,
      "points": 500,
      "prepaid": 200
    }
  }
}
```

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 404 Not Found | Wallet does not exist |
| 500 Internal Server Error | Database error |

---

### POST /api/accounts/{accountId}/wallet

Creates a wallet for an account.

#### Parameters
| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| accountId | path | uint32 | yes | Account ID |

#### Request Model
JSON:API resource type: `wallets`

```json
{
  "data": {
    "type": "wallets",
    "attributes": {
      "credit": 1000,
      "points": 500,
      "prepaid": 200
    }
  }
}
```

#### Response Model
JSON:API resource type: `wallets`

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 500 Internal Server Error | Failed to create wallet |

---

### PATCH /api/accounts/{accountId}/wallet

Updates a wallet for an account.

#### Parameters
| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| accountId | path | uint32 | yes | Account ID |

#### Request Model
JSON:API resource type: `wallets`

```json
{
  "data": {
    "type": "wallets",
    "attributes": {
      "credit": 900,
      "points": 500,
      "prepaid": 200
    }
  }
}
```

#### Response Model
JSON:API resource type: `wallets`

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 500 Internal Server Error | Failed to update wallet |

---

### GET /api/characters/{characterId}/cash-shop/wishlist

Retrieves wishlist items for a character.

#### Parameters
| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | yes | Character ID |

#### Request Model
None.

#### Response Model
JSON:API resource type: `items`

```json
{
  "data": [
    {
      "type": "items",
      "id": "uuid",
      "attributes": {
        "characterId": 12345,
        "serialNumber": 67890
      }
    }
  ]
}
```

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 404 Not Found | No wishlist found |
| 500 Internal Server Error | Database error |

---

### POST /api/characters/{characterId}/cash-shop/wishlist

Adds an item to a character's wishlist.

#### Parameters
| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | yes | Character ID |

#### Request Model
JSON:API resource type: `items`

```json
{
  "data": {
    "type": "items",
    "attributes": {
      "serialNumber": 67890
    }
  }
}
```

#### Response Model
JSON:API resource type: `items`

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 500 Internal Server Error | Failed to add item |

---

### DELETE /api/characters/{characterId}/cash-shop/wishlist

Clears all items from a character's wishlist.

#### Parameters
| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | yes | Character ID |

#### Request Model
None.

#### Response Model
None. Returns 204 No Content.

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 500 Internal Server Error | Failed to clear wishlist |

---

### DELETE /api/characters/{characterId}/cash-shop/wishlist/{itemId}

Removes a specific item from a character's wishlist.

#### Parameters
| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | yes | Character ID |
| itemId | path | uuid | yes | Wishlist item ID |

#### Request Model
None.

#### Response Model
None. Returns 204 No Content.

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 500 Internal Server Error | Failed to remove item |

---

### GET /api/accounts/{accountId}/cash-shop/inventory

Retrieves cash inventory for an account.

#### Parameters
| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| accountId | path | uint32 | yes | Account ID |

#### Request Model
None.

#### Response Model
JSON:API resource type: `cash-inventories`

```json
{
  "data": {
    "type": "cash-inventories",
    "id": "uuid",
    "attributes": {
      "accountId": 12345
    },
    "relationships": {
      "compartments": {
        "data": [
          { "type": "compartments", "id": "uuid" }
        ]
      }
    }
  },
  "included": [
    {
      "type": "compartments",
      "id": "uuid",
      "attributes": {
        "accountId": 12345,
        "type": 1,
        "capacity": 55
      }
    }
  ]
}
```

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 404 Not Found | Inventory does not exist |
| 500 Internal Server Error | Database error |

---

### POST /api/accounts/{accountId}/cash-shop/inventory

Creates a cash inventory for an account.

#### Parameters
| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| accountId | path | uint32 | yes | Account ID |

#### Request Model
JSON:API resource type: `cash-inventories`

#### Response Model
JSON:API resource type: `cash-inventories`

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 500 Internal Server Error | Failed to create inventory |

---

### GET /api/accounts/{accountId}/cash-shop/inventory/compartments

Retrieves cash compartments for an account.

#### Parameters
| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| accountId | path | uint32 | yes | Account ID |
| type | query | int | no | Compartment type (1=Explorer, 2=Cygnus, 3=Legend) |

#### Request Model
None.

#### Response Model
JSON:API resource type: `compartments`

If `type` is specified, returns a single compartment. Otherwise returns all compartments.

```json
{
  "data": {
    "type": "compartments",
    "id": "uuid",
    "attributes": {
      "accountId": 12345,
      "type": 1,
      "capacity": 55
    },
    "relationships": {
      "assets": {
        "data": [
          { "type": "assets", "id": "uuid" }
        ]
      }
    }
  }
}
```

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 400 Bad Request | Invalid type parameter |
| 500 Internal Server Error | Database error |

---

### GET /api/accounts/{accountId}/cash-shop/inventory/compartments/{compartmentId}/assets/{assetId}

Retrieves a specific asset by ID.

#### Parameters
| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| accountId | path | uint32 | yes | Account ID |
| compartmentId | path | uuid | yes | Compartment ID |
| assetId | path | uuid | yes | Asset ID |

#### Request Model
None.

#### Response Model
JSON:API resource type: `assets`

```json
{
  "data": {
    "type": "assets",
    "id": "uuid",
    "attributes": {
      "compartmentId": "uuid"
    },
    "relationships": {
      "item": {
        "data": { "type": "items", "id": "12345" }
      }
    }
  }
}
```

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 404 Not Found | Asset not found or belongs to different compartment |

---

### GET /api/cash-shop/items/{itemId}

Retrieves a cash item by ID.

#### Parameters
| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| itemId | path | uint32 | yes | Item ID |

#### Request Model
None.

#### Response Model
JSON:API resource type: `items`

```json
{
  "data": {
    "type": "items",
    "id": "12345",
    "attributes": {
      "cashId": "67890",
      "templateId": 5000,
      "quantity": 1,
      "flag": 0,
      "purchasedBy": 12345
    }
  }
}
```

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 404 Not Found | Item does not exist |
| 500 Internal Server Error | Database error |

---

### POST /api/cash-shop/items

Creates a new cash item.

#### Parameters
None.

#### Request Model
JSON:API resource type: `items`

```json
{
  "data": {
    "type": "items",
    "attributes": {
      "templateId": 5000,
      "quantity": 1,
      "purchasedBy": 12345
    }
  }
}
```

#### Response Model
JSON:API resource type: `items`

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 400 Bad Request | Invalid input |
| 500 Internal Server Error | Failed to create item |
