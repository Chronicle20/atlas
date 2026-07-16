# REST

All endpoints are served under the base path `/api/` (`main.go:52`, `main.go:114`) and are registered read-only (`GET`). Shop routes are registered by `shop.InitializeRoutes` (`shop/resource.go:23-46`) and the Frederick route by `frederick.InitializeRoutes` (`frederick/resource.go`). Path/query ids are parsed by the shared parsers in `rest/handler.go`. Sparse fieldsets are honored via `jsonapi.ParseQueryFields`.

## Endpoints

### GET /api/merchants

Returns all Open shops, each populated with a listing count. Handler `handleGetMerchants` (`shop/resource.go:129`).

**Parameters**

None.

**Response Model**

JSON:API collection of `merchants` resources (`shop/rest.go` `RestModel`, `GetName()` = `merchants`).

```
RestModel {
  id: string (uuid)
  characterId: uint32
  shopType: byte
  state: byte
  title: string
  worldId: byte
  channelId: byte
  mapId: uint32
  instanceId: string (uuid)
  x: int16
  y: int16
  permitItemId: uint32
  closeReason: byte
  mesoBalance: uint32
  createdAt: time
  listingCount: int64
  visitors: []uint32 (omitempty)
  messages: []MessageRestModel (omitempty)
  // relationship: listings -> listings resources (populated only by GET /api/merchants/{shopId})
}

MessageRestModel {
  characterId: uint32
  content: string
  sentAt: time
}
```

**Error Conditions**

| Status | Condition |
|---|---|
| 500 | Retrieval or listing-count failure (`server.WriteErrorResponse`) |

---

### GET /api/merchants/search/listings

Searches Open/Maintenance shop listings for an item. Handler `handleSearchListings` (`shop/resource.go:167`).

**Parameters**

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| itemId | query | uint32 | yes | Item template id to search for |
| worldId | query | uint8 | no | Restrict to a world; omitted searches tenant-wide |
| order | query | string | no | `desc` sorts by price descending; any other value sorts ascending (default) |

Results are ordered by `pricePerBundle` and capped at 200 rows.

**Response Model**

JSON:API collection of `listing-search-results` resources (`shop/rest.go` `ListingSearchRestModel`, `GetName()` = `listing-search-results`).

```
ListingSearchRestModel {
  id: string
  shopId: string (uuid)
  shopTitle: string
  worldId: byte
  channelId: byte
  mapId: uint32
  ownerId: uint32
  shopType: byte
  state: byte
  itemId: uint32
  itemType: byte
  quantity: uint16
  bundleSize: uint16
  bundlesRemaining: uint16
  pricePerBundle: uint32
  itemSnapshot: asset.AssetData
}
```

**Error Conditions**

| Status | Condition |
|---|---|
| 400 | `itemId` missing, or `itemId`/`worldId` not parseable |
| 500 | Search or marshal failure |

---

### GET /api/merchants/{shopId}

Returns a single shop with its listings, current visitors, and persisted chat messages. Handler `handleGetMerchant` (`shop/resource.go:48`).

**Parameters**

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| shopId | path | uuid | yes | Shop identifier |

**Response Model**

JSON:API single `merchants` resource. The `listings` relationship is included; `visitors` and `messages` attributes are populated. Visitor-retrieval failure is non-fatal (visitors omitted); message-retrieval failure is non-fatal (messages omitted).

**Error Conditions**

| Status | Condition |
|---|---|
| 404 | Shop not found (`ErrNotFound` on `GetById`) |
| 500 | Listing retrieval or REST-model transform failure |

---

### GET /api/merchants/{shopId}/relationships/listings

Returns the listings for a shop. Handler `handleGetMerchantListings` (`shop/resource.go:101`).

**Parameters**

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| shopId | path | uuid | yes | Shop identifier |

**Response Model**

JSON:API collection of `listings` resources (`listing/rest.go` `RestModel`, `GetName()` = `listings`).

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
  itemSnapshot: asset.AssetData
  displayOrder: uint16
}
```

**Error Conditions**

| Status | Condition |
|---|---|
| 500 | Listing retrieval or marshal failure |

---

### GET /api/merchants/{shopId}/blacklist

Returns the shop's blacklist (banned character names). Handler `handleGetMerchantBlacklist` (`shop/resource.go:301`). Read-only; blacklist mutation is performed via Kafka commands, not REST.

**Parameters**

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| shopId | path | uuid | yes | Shop identifier |

**Response Model**

JSON:API collection of `merchant-blacklist` resources (`shop/resource.go:282-289`). The resource `id` is the name.

```
BlacklistRestModel {
  id: string   // set to name
  name: string
}
```

**Error Conditions**

| Status | Condition |
|---|---|
| 500 | Blacklist retrieval failure |

---

### GET /api/merchants/{shopId}/visits

Returns the shop's visit list (visitor name and cumulative visit count). Handler `handleGetMerchantVisits` (`shop/resource.go:321`). Read-only.

**Parameters**

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| shopId | path | uuid | yes | Shop identifier |

**Response Model**

JSON:API collection of `merchant-visits` resources (`shop/resource.go:291-299`). The resource `id` is the name.

```
VisitRestModel {
  id: string    // set to name
  name: string
  count: uint32
}
```

**Error Conditions**

| Status | Condition |
|---|---|
| 500 | Visit-list retrieval failure |

---

### GET /api/characters/{characterId}/merchants

Returns shops owned by a character. Handler `handleGetCharacterMerchants` (`shop/resource.go:216`).

**Parameters**

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| characterId | path | uint32 | yes | Character identifier |

**Response Model**

JSON:API collection of `merchants` resources (no listing count populated).

**Error Conditions**

| Status | Condition |
|---|---|
| 500 | Retrieval or marshal failure |

---

### GET /api/characters/{characterId}/visiting

Returns the single shop a character is currently occupying (as visitor or owner). Handler `handleGetCharacterVisiting` (`shop/resource.go:243`).

**Parameters**

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| characterId | path | uint32 | yes | Character identifier |

**Response Model**

JSON:API single `merchants` resource.

**Error Conditions**

| Status | Condition |
|---|---|
| 404 | Character is not occupying a shop, or the resolved shop no longer loads (`ErrNotFound`) |
| 500 | Retrieval or transform failure |

---

### GET /api/worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/merchants

Returns shops in a specific field (world/channel/map/instance), each with a listing count. Handler `handleGetFieldMerchants` (`shop/resource.go:341`).

**Parameters**

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| worldId | path | byte | yes | World id |
| channelId | path | byte | yes | Channel id |
| mapId | path | uint32 | yes | Map id |
| instanceId | path | uuid | yes | Field instance id |

**Response Model**

JSON:API collection of `merchants` resources.

**Error Conditions**

| Status | Condition |
|---|---|
| 500 | Retrieval, listing-count, or marshal failure |

---

### GET /api/worlds/{worldId}/shop-searches/top

Returns the top 10 most-searched item ids for the world, ranked by search count. The limit is fixed at 10 in the handler. Handler `handleGetTopShopSearches` (`shop/resource.go:387`).

**Parameters**

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| worldId | path | byte | yes | World id |

**Response Model**

JSON:API collection of `shop-search-counts` resources (`searchcount/rest.go` `RestModel`, `GetName()` = `shop-search-counts`). The resource `id` is the decimal item id.

```
RestModel {
  id: string   // decimal itemId
  itemId: uint32
  count: uint64
}
```

**Error Conditions**

| Status | Condition |
|---|---|
| 500 | Ranking or marshal failure (`WriteHeader(http.StatusInternalServerError)` directly) |

---

### GET /api/characters/{characterId}/frederick

Returns whether a character has items or mesos pending at Frederick. Handler `handleGetCharacterFrederick` (`frederick/resource.go`).

**Parameters**

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| characterId | path | uint32 | yes | Character identifier |

**Response Model**

JSON:API single `frederick-status` resource (`frederick/rest.go` `StatusRestModel`, `GetName()` = `frederick-status`). The resource `id` is the decimal character id.

```
StatusRestModel {
  id: string        // decimal characterId
  hasPending: bool
}
```

**Error Conditions**

| Status | Condition |
|---|---|
| 500 | Pending-check or transform failure |

---

### GET /debug/consumers

Non-JSON:API debug endpoint mounted via `server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())` (`main.go:118`). Returns Kafka consumer-manager introspection output; not a domain resource.
</content>
