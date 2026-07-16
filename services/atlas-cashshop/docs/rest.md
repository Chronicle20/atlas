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

Retrieves wishlist items for a character. Paginated.

#### Parameters
| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| characterId | path | uint32 | yes | Character ID |
| page[number] | query | int | no | Page number, default 1, must be >= 1 |
| page[size] | query | int | no | Page size, default 250, must be between 1 and 250 |

`limit` is rejected outright; paging is expressed only via `page[number]`/`page[size]`.

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
  ],
  "meta": {
    "total": 3,
    "page": { "number": 1, "size": 2, "last": 2 }
  },
  "links": {
    "self": "...",
    "first": "...",
    "last": "...",
    "next": "..."
  }
}
```

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 400 Bad Request | Invalid `page[number]`/`page[size]`, or `limit` supplied |
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
      },
      "relationships": {
        "assets": {
          "data": [
            { "type": "assets", "id": "42" }
          ]
        }
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

Retrieves cash compartments for an account. The route requires a `type` query parameter to be present (a request with `type` entirely absent does not match this route). When `type` has a non-empty value, returns a single compartment matching that type. When `type` is present with an empty value (`type=`), returns all compartments for the account, paginated.

#### Parameters
| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| accountId | path | uint32 | yes | Account ID |
| type | query | int | yes | Compartment type (1=Explorer, 2=Cygnus, 3=Legend). Empty value (`type=`) returns all compartments instead of one. |
| page[number] | query | int | no | Page number, default 1, must be >= 1. Only applies when `type` is empty. |
| page[size] | query | int | no | Page size, default 250, must be between 1 and 250. Only applies when `type` is empty. |

`limit` is rejected outright when listing all compartments; paging is expressed only via `page[number]`/`page[size]`.

#### Request Model
None.

#### Response Model
JSON:API resource type: `compartments`

When `type` has a non-empty value, returns a single compartment:

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
          { "type": "assets", "id": "42" }
        ]
      }
    }
  },
  "included": [
    {
      "type": "assets",
      "id": "42",
      "attributes": {
        "compartmentId": "uuid",
        "cashId": "12345",
        "templateId": 5000,
        "commodityId": 100,
        "quantity": 1,
        "flag": 0,
        "petId": 0,
        "purchasedBy": 67890,
        "expiration": "2025-06-01T00:00:00Z",
        "createdAt": "2025-05-01T00:00:00Z"
      }
    }
  ]
}
```

When `type` is empty, returns a paginated array of compartments, with `meta.total`/`meta.page` and JSON:API pagination `links`, matching the shape of the wishlist list response above.

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 400 Bad Request | Invalid (non-integer) type parameter |
| 400 Bad Request | Invalid `page[number]`/`page[size]`, or `limit` supplied (when `type` is empty) |
| 500 Internal Server Error | Database error |

---

### GET /api/accounts/{accountId}/cash-shop/inventory/compartments/{compartmentId}/assets/{assetId}

Retrieves a specific asset by ID. `accountId` and `compartmentId` are parsed and validated as well-formed but are not used to scope the lookup; the asset is fetched by `assetId` alone.

#### Parameters
| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| accountId | path | uint32 | yes | Account ID |
| compartmentId | path | uuid | yes | Compartment ID |
| assetId | path | uint32 | yes | Asset ID |

#### Request Model
None.

#### Response Model
JSON:API resource type: `assets`

```json
{
  "data": {
    "type": "assets",
    "id": "42",
    "attributes": {
      "compartmentId": "uuid",
      "cashId": "12345",
      "templateId": 5000,
      "commodityId": 100,
      "quantity": 1,
      "flag": 0,
      "petId": 0,
      "purchasedBy": 67890,
      "expiration": "2025-06-01T00:00:00Z",
      "createdAt": "2025-05-01T00:00:00Z"
    }
  }
}
```

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 404 Not Found | Asset not found |

---

### GET /api/cash-shop/assets/{assetId}

Retrieves an asset by ID.

#### Parameters
| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| assetId | path | uint32 | yes | Asset ID |

#### Request Model
None.

#### Response Model
JSON:API resource type: `assets`

```json
{
  "data": {
    "type": "assets",
    "id": "42",
    "attributes": {
      "compartmentId": "uuid",
      "cashId": "12345",
      "templateId": 5000,
      "commodityId": 100,
      "quantity": 1,
      "flag": 0,
      "petId": 0,
      "purchasedBy": 67890,
      "expiration": "2025-06-01T00:00:00Z",
      "createdAt": "2025-05-01T00:00:00Z"
    }
  }
}
```

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 404 Not Found | Asset does not exist |
| 500 Internal Server Error | Database error |

---

### POST /api/cash-shop/assets

Creates a new cash asset.

#### Parameters
None.

#### Request Model
JSON:API resource type: `assets`

```json
{
  "data": {
    "type": "assets",
    "attributes": {
      "compartmentId": "uuid",
      "templateId": 5000,
      "commodityId": 100,
      "quantity": 1,
      "petId": 0,
      "purchasedBy": 67890
    }
  }
}
```

#### Response Model
JSON:API resource type: `assets`

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 400 Bad Request | Invalid input |
| 500 Internal Server Error | Failed to create asset |

---

### PATCH /api/cash-shop/assets/{assetId}

Updates an asset's quantity.

#### Parameters
| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| assetId | path | uint32 | yes | Asset ID |

#### Request Model
JSON:API resource type: `assets`

```json
{
  "data": {
    "type": "assets",
    "attributes": {
      "quantity": 5
    }
  }
}
```

#### Response Model
None. Returns 204 No Content.

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 500 Internal Server Error | Failed to update asset |

---

### DELETE /api/cash-shop/assets/{assetId}

Deletes a cash asset.

#### Parameters
| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| assetId | path | uint32 | yes | Asset ID |

#### Request Model
None.

#### Response Model
None. Returns 204 No Content.

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 500 Internal Server Error | Failed to delete asset |
