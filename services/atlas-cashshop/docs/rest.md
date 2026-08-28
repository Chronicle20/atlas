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
        "createdAt": "2025-05-01T00:00:00Z",
        "giftFrom": "",
        "giftMessage": ""
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
      "createdAt": "2025-05-01T00:00:00Z",
      "giftFrom": "",
      "giftMessage": ""
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
      "createdAt": "2025-05-01T00:00:00Z",
      "giftFrom": "",
      "giftMessage": ""
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

---

### GET /api/accounts/{accountId}/purchaseRecords/{serialNumber}

Answers whether an account has ever purchased a given commodity serial number, and how many times. A never-purchased serial returns 200 with `purchased: false`, never 404.

#### Parameters
| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| accountId | path | uint32 | yes | Account ID |
| serialNumber | path | uint32 | yes | Commodity serial number |

#### Request Model
None.

#### Response Model
JSON:API resource type: `purchaseRecords`

```json
{
  "data": {
    "type": "purchaseRecords",
    "id": "67890",
    "attributes": {
      "purchased": true,
      "count": 3
    }
  }
}
```

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 500 Internal Server Error | Database error |

---

### GET /api/rings

Retrieves ring pair halves for a character. `filter[characterId]` is required; there is no unfiltered listing.

#### Parameters
| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| filter[characterId] | query | uint32 | yes | Character ID |
| page[number] | query | int | no | Page number, default 1, must be >= 1 |
| page[size] | query | int | no | Page size, default 250, must be between 1 and 250 |

#### Request Model
None.

#### Response Model
JSON:API resource type: `rings`

```json
{
  "data": [
    {
      "type": "rings",
      "id": "uuid",
      "attributes": {
        "pairId": "uuid",
        "characterId": 12345,
        "partnerCharacterId": 54321,
        "assetId": 42,
        "itemTemplateId": 5000,
        "ringType": "COUPLE",
        "state": "ACTIVE",
        "createdAt": "2025-05-01T00:00:00Z",
        "cashId": 12345,
        "partnerCashId": 67890,
        "partnerName": "Partner"
      }
    }
  ],
  "meta": {
    "total": 1,
    "page": { "number": 1, "size": 250, "last": 1 }
  }
}
```

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 400 Bad Request | `filter[characterId]` missing or not a positive integer, or invalid `page[number]`/`page[size]` |
| 500 Internal Server Error | Database error |

---

### GET /api/rings/{ringId}

Retrieves a single ring pair half by ID.

#### Parameters
| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| ringId | path | uuid | yes | Ring half ID |

#### Request Model
None.

#### Response Model
JSON:API resource type: `rings` (same attribute shape as GET /api/rings)

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 404 Not Found | No such ring (including a ring belonging to another tenant) |
| 500 Internal Server Error | Database error |

---

### GET /api/coupons

Lists coupons for the tenant, with optional filters.

#### Parameters
| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| filter[code] | query | string | no | Exact code match (normalized before comparison) |
| filter[active] | query | bool | no | `true` or `false` |
| filter[batchId] | query | uuid | no | Coupons belonging to a batch |
| filter[expiresBefore] | query | RFC3339 timestamp | no | Coupons expiring before this time |
| filter[expiresAfter] | query | RFC3339 timestamp | no | Coupons expiring after this time |
| page[number] | query | int | no | Page number, default 1, must be >= 1 |
| page[size] | query | int | no | Page size, default 250, must be between 1 and 250 |

#### Request Model
None.

#### Response Model
JSON:API resource type: `coupons`

```json
{
  "data": [
    {
      "type": "coupons",
      "id": "uuid",
      "attributes": {
        "batchId": "uuid",
        "code": "ABCD1234EF",
        "description": "Launch promo",
        "active": true,
        "startsAt": "2025-05-01T00:00:00Z",
        "expiresAt": "2025-06-01T00:00:00Z",
        "maxUses": 1,
        "redemptionCount": 0,
        "rewards": [{ "type": "CURRENCY", "currency": 1, "amount": 100 }],
        "createdAt": "2025-05-01T00:00:00Z",
        "updatedAt": "2025-05-01T00:00:00Z"
      }
    }
  ],
  "meta": {
    "total": 1,
    "page": { "number": 1, "size": 250, "last": 1 }
  }
}
```

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 400 Bad Request | Unparseable `filter[active]`, `filter[batchId]`, `filter[expiresBefore]`/`filter[expiresAfter]`, or invalid `page[number]`/`page[size]` |
| 500 Internal Server Error | Database error |

---

### POST /api/coupons

Creates a coupon. A blank `code` generates one.

#### Parameters
None.

#### Request Model
JSON:API resource type: `coupons`

```json
{
  "data": {
    "type": "coupons",
    "attributes": {
      "code": "",
      "description": "Launch promo",
      "active": true,
      "startsAt": null,
      "expiresAt": "2025-06-01T00:00:00Z",
      "maxUses": 1,
      "rewards": [{ "type": "CASH_ITEM", "serialNumber": 67890, "quantity": 1 }]
    }
  }
}
```

#### Response Model
JSON:API resource type: `coupons` (same shape as GET /api/coupons/{couponId})

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 422 Unprocessable Entity | Implausible code, malformed/empty reward bundle, `expiresAt` not after `startsAt`, `maxUses` below redemption count, or a `CASH_ITEM` reward referencing an unknown commodity serial |
| 409 Conflict | A coupon with this (normalized) code already exists |
| 500 Internal Server Error | Database error |

---

### GET /api/coupons/{couponId}

Retrieves a single coupon.

#### Parameters
| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| couponId | path | uuid | yes | Coupon ID |

#### Request Model
None.

#### Response Model
JSON:API resource type: `coupons`

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 404 Not Found | No such coupon |
| 500 Internal Server Error | Database error |

---

### PATCH /api/coupons/{couponId}

Partially updates a coupon. A field omitted from the body preserves its stored value; `startsAt`/`expiresAt`/`maxUses` may be explicitly set to `null` to clear them. `code`, `batchId`, `id`, and `redemptionCount` are not editable.

#### Parameters
| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| couponId | path | uuid | yes | Coupon ID |

#### Request Model
JSON:API resource type: `coupons`

```json
{
  "data": {
    "type": "coupons",
    "attributes": {
      "active": false
    }
  }
}
```

#### Response Model
JSON:API resource type: `coupons`

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 404 Not Found | No such coupon |
| 422 Unprocessable Entity | Malformed reward bundle, `expiresAt` not after `startsAt`, or `maxUses` below the stored redemption count |
| 500 Internal Server Error | Database error |

---

### DELETE /api/coupons/{couponId}

Deletes a coupon that has never been redeemed.

#### Parameters
| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| couponId | path | uuid | yes | Coupon ID |

#### Request Model
None.

#### Response Model
None. Returns 204 No Content.

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 404 Not Found | No such coupon |
| 409 Conflict | Coupon has redemptions and cannot be deleted |
| 500 Internal Server Error | Database error |

---

### GET /api/coupon-batches

Lists bulk coupon-generation batches.

#### Parameters
| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| page[number] | query | int | no | Page number, default 1, must be >= 1 |
| page[size] | query | int | no | Page size, default 250, must be between 1 and 250 |

#### Request Model
None.

#### Response Model
JSON:API resource type: `coupon-batches`

```json
{
  "data": [
    {
      "type": "coupon-batches",
      "id": "uuid",
      "attributes": {
        "description": "Anniversary giveaway",
        "requestedCount": 500,
        "generatedCount": 500,
        "redeemedCount": 12,
        "createdAt": "2025-05-01T00:00:00Z"
      }
    }
  ],
  "meta": {
    "total": 1,
    "page": { "number": 1, "size": 250, "last": 1 }
  }
}
```

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 400 Bad Request | Invalid `page[number]`/`page[size]` |
| 500 Internal Server Error | Database error |

---

### POST /api/coupon-batches

Generates a batch of coupon codes (whole batch or none). The generated plaintext codes appear only in this response.

#### Parameters
None.

#### Request Model
JSON:API resource type: `coupon-batches`

```json
{
  "data": {
    "type": "coupon-batches",
    "attributes": {
      "description": "Anniversary giveaway",
      "count": 500,
      "prefix": "ANNIV-",
      "length": 10,
      "startsAt": null,
      "expiresAt": "2025-06-01T00:00:00Z",
      "rewards": [{ "type": "CURRENCY", "currency": 2, "amount": 500 }]
    }
  }
}
```

#### Response Model
JSON:API resource type: `coupon-batches`

```json
{
  "data": {
    "type": "coupon-batches",
    "id": "uuid",
    "attributes": {
      "description": "Anniversary giveaway",
      "requestedCount": 500,
      "generatedCount": 500,
      "redeemedCount": 0,
      "createdAt": "2025-05-01T00:00:00Z",
      "codes": ["ANNIV-A1B2C3D4E5", "..."]
    }
  }
}
```

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 422 Unprocessable Entity | Zero/invalid count, prefix+length exceeding the code length limit, malformed reward bundle, `expiresAt` not after `startsAt`, or an unknown commodity serial |
| 500 Internal Server Error | Database error |

---

### GET /api/coupon-batches/{batchId}

Retrieves a single batch.

#### Parameters
| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| batchId | path | uuid | yes | Batch ID |

#### Request Model
None.

#### Response Model
JSON:API resource type: `coupon-batches` (no `codes` field on a GET; codes are returned only on creation)

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 404 Not Found | No such coupon batch |
| 500 Internal Server Error | Database error |

---

### GET /api/coupons/{couponId}/redemptions

Retrieves the redemption audit trail for a single coupon. Read-only.

#### Parameters
| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| couponId | path | uuid | yes | Coupon ID |
| page[number] | query | int | no | Page number, default 1, must be >= 1 |
| page[size] | query | int | no | Page size, default 250, must be between 1 and 250 |

#### Request Model
None.

#### Response Model
JSON:API resource type: `coupon-redemptions`

```json
{
  "data": [
    {
      "type": "coupon-redemptions",
      "id": "uuid",
      "attributes": {
        "couponId": "uuid",
        "accountId": 12345,
        "characterId": 67890,
        "transactionId": "uuid",
        "rewardsGranted": [{ "type": "CURRENCY", "currency": 1, "amount": 100 }],
        "redeemedAt": "2025-05-01T00:00:00Z"
      }
    }
  ],
  "meta": {
    "total": 1,
    "page": { "number": 1, "size": 250, "last": 1 }
  }
}
```

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 400 Bad Request | Invalid `page[number]`/`page[size]` |
| 500 Internal Server Error | Database error |

---

### GET /api/coupon-redemptions

Retrieves the redemption audit trail for an account. `filter[accountId]` is required; there is no unfiltered listing.

#### Parameters
| Name | Location | Type | Required | Description |
|------|----------|------|----------|-------------|
| filter[accountId] | query | uint32 | yes | Account ID |
| page[number] | query | int | no | Page number, default 1, must be >= 1 |
| page[size] | query | int | no | Page size, default 250, must be between 1 and 250 |

#### Request Model
None.

#### Response Model
JSON:API resource type: `coupon-redemptions` (same shape as GET /api/coupons/{couponId}/redemptions)

#### Error Conditions
| Status | Condition |
|--------|-----------|
| 400 Bad Request | `filter[accountId]` missing or not a positive integer, or invalid `page[number]`/`page[size]` |
| 500 Internal Server Error | Database error |
