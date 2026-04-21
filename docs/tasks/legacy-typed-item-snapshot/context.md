# Typed ItemSnapshot — Key Files, Decisions, Dependencies

Last Updated: 2026-03-18

## Key Files

### atlas-merchant — AssetData Definition
- `services/atlas-merchant/atlas.com/merchant/kafka/message/asset/kafka.go` — canonical `AssetData` struct (31 fields)

### atlas-merchant — Listing Domain
- `services/atlas-merchant/atlas.com/merchant/listing/model.go` — `Model.itemSnapshot json.RawMessage`
- `services/atlas-merchant/atlas.com/merchant/listing/entity.go` — `Entity.ItemSnapshot []byte` (JSONB)
- `services/atlas-merchant/atlas.com/merchant/listing/builder.go` — `ModelBuilder.itemSnapshot json.RawMessage`
- `services/atlas-merchant/atlas.com/merchant/listing/processor.go` — `Processor.Create()` takes `json.RawMessage`

### atlas-merchant — Frederick Domain
- `services/atlas-merchant/atlas.com/merchant/frederick/model.go` — `ItemModel.itemSnapshot json.RawMessage`
- `services/atlas-merchant/atlas.com/merchant/frederick/entity.go` — `ItemEntity.ItemSnapshot []byte` (JSONB)

### atlas-merchant — Shop Processor (Unmarshal Sites)
- `services/atlas-merchant/atlas.com/merchant/shop/processor.go:495-496` — `AddListing` signature
- `services/atlas-merchant/atlas.com/merchant/shop/processor.go:732-742` — PurchaseBundle unmarshal
- `services/atlas-merchant/atlas.com/merchant/shop/processor.go:908-929` — RetrieveFrederick unmarshal
- `services/atlas-merchant/atlas.com/merchant/shop/processor.go:1038-1064` — `listingSnapshot` + `acceptItemToBuffer` unmarshal
- `services/atlas-merchant/atlas.com/merchant/shop/processor.go:51,68` — Processor interface

### atlas-merchant — Mock Processor
- `services/atlas-merchant/atlas.com/merchant/shop/mock/processor.go:164,297` — mock signatures

### atlas-merchant — Kafka Messages
- `services/atlas-merchant/atlas.com/merchant/kafka/message/merchant/kafka.go:63-74` — `CommandAddListingBody.ItemSnapshot`
- `services/atlas-merchant/atlas.com/merchant/kafka/message/compartment/kafka.go:32-36` — `AcceptCommandBody` embeds `AssetData`

### atlas-merchant — Consumer
- `services/atlas-merchant/atlas.com/merchant/kafka/consumer/merchant/consumer.go:133-158` — `handleAddListingCommand` unmarshal

### atlas-merchant — Producer
- `services/atlas-merchant/atlas.com/merchant/shop/producer.go:185-199` — `AcceptAssetCommandProvider` takes typed `AssetData`

### atlas-merchant — Tests
- `services/atlas-merchant/atlas.com/merchant/shop/processor_test.go` — ~20 test functions use `json.Marshal(map[string]interface{}{"flag": 0})` for snapshot creation

### atlas-channel — Listing Model
- `services/atlas-channel/atlas.com/channel/merchant/listing.go` — `ListingRestModel` and `ListingModel` with `json.RawMessage`

### atlas-channel — Consumer (Unmarshal Site)
- `services/atlas-channel/atlas.com/channel/kafka/consumer/merchant/consumer.go:417-483` — local `itemSnapshot` struct + `assetFromSnapshot()`

### atlas-channel — Kafka Messages
- `services/atlas-channel/atlas.com/channel/kafka/message/merchant/kafka.go:73-83` — `CommandAddListingBody` (no ItemSnapshot field)

## Key Decisions

1. **No shared lib** — each service defines its own `AssetData` struct (user requirement)
2. **Scanner/Valuer on AssetData** — keeps JSONB storage, eliminates `[]byte` intermediary in entity
3. **WithQuantity method** — replaces the 3 mutation sites with a clean functional pattern
4. **atlas-channel struct is a subset** — only needs fields used for packet building (can keep fewer fields)
5. **AssetData stays in `kafka/message/asset/`** — already the right conceptual location in atlas-merchant; both Kafka messages and domain models reference it

## Dependencies

- No external library changes required
- No database migration required (JSONB format unchanged)
- No Kafka wire format changes (JSON tags identical)
- atlas-channel and atlas-merchant changes are independent (can be done in parallel)
