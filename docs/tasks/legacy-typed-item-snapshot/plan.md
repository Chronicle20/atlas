# Typed ItemSnapshot — Replace json.RawMessage with Structured Type

Last Updated: 2026-03-18

## Executive Summary

Replace all `json.RawMessage` / `[]byte` usage for ItemSnapshot across atlas-merchant and atlas-channel with a typed `AssetData` struct. Each service defines its own copy of the struct (no shared lib). GORM entities use `Scanner`/`Valuer` for JSONB persistence. Manual `json.Unmarshal` boilerplate is eliminated at all 5 consumption sites.

## Current State Analysis

### atlas-merchant

ItemSnapshot flows through the service as opaque `json.RawMessage`:

| Layer | Type | File |
|-------|------|------|
| `listing.Model` | `json.RawMessage` | listing/model.go:19 |
| `listing.Entity` | `[]byte` + `gorm:"type:jsonb"` | listing/entity.go:21 |
| `listing.ModelBuilder` | `json.RawMessage` | listing/builder.go:24 |
| `listing.Processor.Create` | `json.RawMessage` param | listing/processor.go:53 |
| `frederick.ItemModel` | `json.RawMessage` | frederick/model.go:15 |
| `frederick.ItemEntity` | `[]byte` + `gorm:"type:jsonb"` | frederick/entity.go:18 |
| `CommandAddListingBody` | `json.RawMessage` | kafka/message/merchant/kafka.go:73 |
| `shop.Processor` interface | `json.RawMessage` param | shop/processor.go:51,68 |
| `shop.ProcessorImpl.AddListing` | `json.RawMessage` param | shop/processor.go:495-496 |
| `shop.ProcessorMock.AddListing` | `json.RawMessage` param | shop/mock/processor.go:164,297 |
| `listingSnapshot` helper struct | `json.RawMessage` field | shop/processor.go:1042 |

**Manual unmarshal sites (4):**
1. `kafka/consumer/merchant/consumer.go:145-150` — extract Flag for validation
2. `shop/processor.go:732-742` — PurchaseBundle: update Quantity, send to buyer
3. `shop/processor.go:908-929` — RetrieveFrederick: update Quantity, return items
4. `shop/processor.go:1045-1064` — acceptItemToBuffer: update Quantity on shop close

The canonical struct already exists: `kafka/message/asset/kafka.go` defines `AssetData` with all 31 fields.

### atlas-channel

| Layer | Type | File |
|-------|------|------|
| `ListingRestModel` | `json.RawMessage` | merchant/listing.go:14 |
| `ListingModel` | `json.RawMessage` | merchant/listing.go:40 |

**Manual unmarshal site (1):**
- `kafka/consumer/merchant/consumer.go:417-483` — local `itemSnapshot` struct + `assetFromSnapshot()` for packet building

### Key Observations

1. `AssetData` is the deserialization target everywhere — the "raw" treatment is an illusion
2. atlas-channel duplicates the struct definition locally (28 fields, slightly different from the 31-field original — missing `CreatedAt`, `OwnerId`, `CommodityId`, `PurchaseBy`, `EquippedSince`)
3. Every consumer does nil-check + unmarshal + error handling boilerplate
4. Quantity is mutated post-unmarshal in 3 separate places (purchase, Frederick, shop close)
5. The `AcceptAssetCommandProvider` already takes `asset.AssetData` as a typed param — so the unmarshal→mutate→pass pattern is pure overhead

## Proposed Future State

### atlas-merchant

- **Promote `AssetData`** from `kafka/message/asset/` to a domain-level package `asset/` (or keep in current location — it's already well-positioned)
- Add `Scanner`/`Valuer` methods to `AssetData` for direct GORM JSONB persistence
- Add `WithQuantity(uint32) AssetData` method to eliminate mutation boilerplate
- Replace `json.RawMessage` with `AssetData` in:
  - `listing.Model`, `listing.Entity`, `listing.ModelBuilder`, `listing.Processor`
  - `frederick.ItemModel`, `frederick.ItemEntity`
  - `CommandAddListingBody`
  - `shop.Processor` interface + all implementations
  - `listingSnapshot` helper struct
- Remove all manual `json.Unmarshal` calls — access fields directly
- Extract Flag directly: `e.Body.ItemSnapshot.Flag` instead of unmarshal→check

### atlas-channel

- Define own `AssetData` struct in `merchant/` package (matching JSON tags)
- Replace `json.RawMessage` in `ListingRestModel` and `ListingModel`
- Remove local `itemSnapshot` struct from consumer
- Simplify `assetFromSnapshot` to accept typed `AssetData` directly

### Database Compatibility

No schema migration needed. GORM `Scanner`/`Valuer` on the typed struct serializes to the same JSONB format. Existing rows deserialize correctly.

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| JSONB format mismatch after type change | Low | High | Scanner/Valuer uses same json tags; write roundtrip test |
| Nil snapshot handling changes | Medium | Medium | Typed struct zero-value replaces nil check; verify all nil-guard sites |
| Kafka wire format change | Low | High | JSON tags unchanged; struct fields match exactly |
| Test snapshot creation pattern changes | Certain | Low | Tests use `map[string]interface{}{"flag": 0}` → switch to typed struct |

## Success Metrics

- Zero `json.RawMessage` references related to ItemSnapshot in atlas-merchant and atlas-channel
- Zero manual `json.Unmarshal` calls for ItemSnapshot data
- All existing tests pass
- Docker builds succeed for both services
- JSONB roundtrip test validates backward compatibility
