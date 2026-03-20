# REST

## Endpoints

### GET /api/merchants

Returns merchants for a map.

**Parameters**

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| mapId | query | uint32 | yes | Map ID to filter by |

**Response Model**

JSON:API collection of `merchants` resources.

```
RestModel {
  id: string (uuid)
  characterId: uint32
  shopType: byte
  state: byte
  title: string
  mapId: uint32
  x: int16
  y: int16
  permitItemId: uint32
  closeReason: byte
  mesoBalance: uint32
}
```

**Error Conditions**

| Status | Condition |
|---|---|
| 400 | Missing or invalid mapId query parameter |
| 500 | Internal error |

---

### GET /api/merchants/{shopId}

Returns a single merchant with included listings.

**Parameters**

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| shopId | path | uuid | yes | Shop identifier |

**Response Model**

JSON:API single `merchants` resource with `listings` relationship included.

**Error Conditions**

| Status | Condition |
|---|---|
| 400 | Invalid shopId format |
| 404 | Shop not found |
| 500 | Internal error |

---

### GET /api/merchants/{shopId}/relationships/listings

Returns listings for a merchant.

**Parameters**

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| shopId | path | uuid | yes | Shop identifier |

**Response Model**

JSON:API collection of `listings` resources.

```
RestModel {
  id: string (uuid)
  shopId: string (uuid)
  itemId: uint32
  itemType: byte
  quantity: uint16
  bundleSize: uint16
  bundlesRemaining: uint16
  pricePerBundle: uint32
  itemSnapshot: json.RawMessage
  displayOrder: uint16
}
```

**Error Conditions**

| Status | Condition |
|---|---|
| 400 | Invalid shopId format |
| 500 | Internal error |

---

### GET /api/characters/{characterId}/merchants

Returns merchants owned by a character.

**Parameters**

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| characterId | path | uint32 | yes | Character identifier |

**Response Model**

JSON:API collection of `merchants` resources.

**Error Conditions**

| Status | Condition |
|---|---|
| 400 | Invalid characterId format |
| 500 | Internal error |
