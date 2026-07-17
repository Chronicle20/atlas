# REST

All endpoints are served under the base path `/api/` (`main.go:50`, `main.go:105`) and are registered read-only (`GET`). Shop routes are registered by `shop.InitializeRoutes` (`shop/resource.go:24-47`) and the Frederick route by `frederick.InitializeRoutes` (`frederick/resource.go:14-23`). Path/query ids are parsed by the shared parsers in `rest/handler.go`. Sparse fieldsets are honored via `jsonapi.ParseQueryFields`.

## Pagination

List endpoints accept JSON:API `page[number]` / `page[size]` query parameters (`libs/atlas-rest/server/paginate.ParseParams`). `page[number]` defaults to 1; `page[size]` defaults to a per-endpoint value (see each endpoint below) and is capped at 250 (`paginate.MaxPageSize`). A non-integer, `page[number]` < 1, or `page[size]` outside `[1, 250]` is a 400. The legacy `limit` query parameter is rejected outright (400) — paging is expressed only via `page[*]`.

Paginated responses carry a JSON:API `meta` block (`{"total": int, "page": {"number": int, "size": int, "last": int}}`) and `links` (`self`, `first`, `last`, and `prev`/`next` where applicable).

## Endpoints

### GET /api/merchants

Returns Open shops, each populated with a listing count. Handler `handleGetMerchants` (`shop/resource.go:136`).

**Parameters**

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| page[number] | query | int | no | Page number (default 1) |
| page[size] | query | int | no | Page size (default 50, max 250) |

**Response Model**

Paginated JSON:API collection of `merchants` resources (`shop/rest.go` `RestModel`, `GetName()` = `merchants`). See Pagination above for the envelope.

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
| 400 | Invalid `page[number]`/`page[size]` |
| 500 | Retrieval or listing-count failure (`server.WriteErrorResponse`) |

---

### GET /api/merchants/search/listings

Searches Open/Maintenance shop listings for an item. Handler `handleSearchListings` (`shop/resource.go:180`).

**Parameters**

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| itemId | query | uint32 | yes | Item template id to search for |
| worldId | query | uint8 | no | Restrict to a world; omitted searches tenant-wide |
| order | query | string | no | `desc` sorts by price descending; any other value sorts ascending (default) |
| page[number] | query | int | no | Page number (default 1) |
| page[size] | query | int | no | Page size (default 200, max 250) |

Results are ordered by `pricePerBundle`.

**Response Model**

Paginated JSON:API collection of `listing-search-results` resources (`shop/rest.go` `ListingSearchRestModel`, `GetName()` = `listing-search-results`). See Pagination above for the envelope.

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
| 400 | `itemId` missing, `itemId`/`worldId` not parseable, or invalid `page[number]`/`page[size]` |
| 500 | Search or marshal failure |

---

### GET /api/merchants/{shopId}

Returns a single shop with its listings, current visitors, and persisted chat messages. Handler `handleGetMerchant` (`shop/resource.go:49`). Not paginated.

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

Returns the listings for a shop. Handler `handleGetMerchantListings` (`shop/resource.go:102`).

**Parameters**

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| shopId | path | uuid | yes | Shop identifier |
| page[number] | query | int | no | Page number (default 1) |
| page[size] | query | int | no | Page size (default 250, max 250) |

**Response Model**

Paginated JSON:API collection of `listings` resources (`listing/rest.go` `RestModel`, `GetName()` = `listings`). See Pagination above for the envelope.

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
| 400 | Invalid `page[number]`/`page[size]` |
| 500 | Listing retrieval or marshal failure |

---

### GET /api/merchants/{shopId}/blacklist

Returns the shop's blacklist (banned character names). Handler `handleGetMerchantBlacklist` (`shop/resource.go:331`). Read-only; blacklist mutation is performed via Kafka commands, not REST.

**Parameters**

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| shopId | path | uuid | yes | Shop identifier |
| page[number] | query | int | no | Page number (default 1) |
| page[size] | query | int | no | Page size (default 250, max 250) |

**Response Model**

Paginated JSON:API collection of `merchant-blacklist` resources (`shop/resource.go:312-319`). The resource `id` is the name. See Pagination above for the envelope.

```
BlacklistRestModel {
  id: string   // set to name
  name: string
}
```

**Error Conditions**

| Status | Condition |
|---|---|
| 400 | Invalid `page[number]`/`page[size]` |
| 500 | Blacklist retrieval failure |

---

### GET /api/merchants/{shopId}/visits

Returns the shop's visit list (visitor name and cumulative visit count). Handler `handleGetMerchantVisits` (`shop/resource.go:360`). Read-only.

**Parameters**

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| shopId | path | uuid | yes | Shop identifier |
| page[number] | query | int | no | Page number (default 1) |
| page[size] | query | int | no | Page size (default 250, max 250) |

**Response Model**

Paginated JSON:API collection of `merchant-visits` resources (`shop/resource.go:321-329`). The resource `id` is the name. See Pagination above for the envelope.

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
| 400 | Invalid `page[number]`/`page[size]` |
| 500 | Visit-list retrieval failure |

---

### GET /api/characters/{characterId}/merchants

Returns shops owned by a character. Handler `handleGetCharacterMerchants` (`shop/resource.go:240`).

**Parameters**

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| characterId | path | uint32 | yes | Character identifier |
| page[number] | query | int | no | Page number (default 1) |
| page[size] | query | int | no | Page size (default 250, max 250) |

**Response Model**

Paginated JSON:API collection of `merchants` resources (no listing count populated). See Pagination above for the envelope.

**Error Conditions**

| Status | Condition |
|---|---|
| 400 | Invalid `page[number]`/`page[size]` |
| 500 | Retrieval or marshal failure |

---

### GET /api/characters/{characterId}/visiting

Returns the single shop a character is currently occupying (as visitor or owner). Handler `handleGetCharacterVisiting` (`shop/resource.go:273`). Not paginated.

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

Returns shops in a specific field (world/channel/map/instance), each with a listing count. Handler `handleGetFieldMerchants` (`shop/resource.go:390`).

**Parameters**

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| worldId | path | byte | yes | World id |
| channelId | path | byte | yes | Channel id |
| mapId | path | uint32 | yes | Map id |
| instanceId | path | uuid | yes | Field instance id |
| page[number] | query | int | no | Page number (default 1) |
| page[size] | query | int | no | Page size (default 250, max 250) |

**Response Model**

Paginated JSON:API collection of `merchants` resources. See Pagination above for the envelope.

**Error Conditions**

| Status | Condition |
|---|---|
| 400 | Invalid `page[number]`/`page[size]` |
| 500 | Retrieval, listing-count, or marshal failure |

---

### GET /api/worlds/{worldId}/shop-searches/top

Returns the most-searched item ids for the world, ranked by search count. The underlying ranking is fixed at the top 10 (`GetTop(worldId, 10)`); `page[number]`/`page[size]` page over that top-10 result set. Handler `handleGetTopShopSearches` (`shop/resource.go:442`).

**Parameters**

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| worldId | path | byte | yes | World id |
| page[number] | query | int | no | Page number (default 1) |
| page[size] | query | int | no | Page size (default 250, max 250) |

**Response Model**

Paginated JSON:API collection of `shop-search-counts` resources (`searchcount/rest.go` `RestModel`, `GetName()` = `shop-search-counts`). The resource `id` is the decimal item id. `meta.total` is at most 10. See Pagination above for the envelope.

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
| 400 | Invalid `page[number]`/`page[size]` |
| 500 | Ranking or marshal failure |

---

### GET /api/characters/{characterId}/frederick

Returns whether a character has items or mesos pending at Frederick. Handler `handleGetCharacterFrederick` (`frederick/resource.go:25`). Not paginated.

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

Non-JSON:API debug endpoint mounted via `server.MountHandler("/debug/consumers", consumer.GetManager().DebugHandler())` (`main.go:109`). Returns Kafka consumer-manager introspection output; not a domain resource.
