# Typed ItemSnapshot — Task Checklist

Last Updated: 2026-03-18

## Phase 1: atlas-merchant — AssetData Enhancement [S]

- [x] 1.1 Add `Scanner`/`Valuer` methods to `AssetData` (`kafka/message/asset/kafka.go`)
- [x] 1.2 Add `WithQuantity(uint32) AssetData` convenience method
- [x] 1.3 Add roundtrip JSONB test (marshal→unmarshal preserves all fields)

## Phase 2: atlas-merchant — Listing Domain [M]

- [x] 2.1 `listing/entity.go` — change `ItemSnapshot []byte` to `ItemSnapshot asset.AssetData`
- [x] 2.2 `listing/model.go` — change `itemSnapshot json.RawMessage` to `itemSnapshot asset.AssetData`, update getter return type
- [x] 2.3 `listing/builder.go` — change `itemSnapshot json.RawMessage` to `itemSnapshot asset.AssetData`, update setter param type
- [x] 2.4 `listing/processor.go` — update `Processor` interface and `Create()` to take `asset.AssetData`
- [x] 2.5 Remove `encoding/json` import from listing package files where no longer needed
- [x] 2.6 `listing/rest.go` — change `RestModel.ItemSnapshot` from `json.RawMessage` to `asset.AssetData`

## Phase 3: atlas-merchant — Frederick Domain [S]

- [x] 3.1 `frederick/entity.go` — change `ItemSnapshot []byte` to `ItemSnapshot asset.AssetData`
- [x] 3.2 `frederick/model.go` — change `itemSnapshot json.RawMessage` to `itemSnapshot asset.AssetData`, update getter return type
- [x] 3.3 Remove `encoding/json` import from frederick package files where no longer needed
- [x] 3.4 `frederick/processor.go` — change `StoredItem.ItemSnapshot` from `[]byte` to `asset.AssetData`

## Phase 4: atlas-merchant — Kafka Messages [S]

- [x] 4.1 `kafka/message/merchant/kafka.go` — change `CommandAddListingBody.ItemSnapshot` from `json.RawMessage` to `asset.AssetData`

## Phase 5: atlas-merchant — Shop Processor [L]

- [x] 5.1 Update `Processor` interface — `AddListing` and `AddListingAndEmit` signatures: remove `flag` param, use `asset.AssetData`
- [x] 5.2 Update `ProcessorImpl.AddListing` — change param type, access `itemSnapshot.Flag` directly
- [x] 5.3 Update `ProcessorImpl.AddListingAndEmit` — change param type
- [x] 5.4 Remove unmarshal in `PurchaseBundle` — use `result.ItemSnapshot` directly + `WithQuantity()`
- [x] 5.5 Remove unmarshal in `RetrieveFrederick` — use `fi.ItemSnapshot()` directly + `WithQuantity()`
- [x] 5.6 Update `listingSnapshot` struct — `ItemSnapshot json.RawMessage` → `ItemSnapshot asset.AssetData`
- [x] 5.7 Remove unmarshal in `acceptItemToBuffer` — use `ls.ItemSnapshot` directly + `WithQuantity()`
- [x] 5.8 Update `PurchaseResult.ItemSnapshot` from `json.RawMessage` to `asset.AssetData`
- [x] 5.9 Remove `encoding/json` import from processor.go

## Phase 6: atlas-merchant — Mock Processor [S]

- [x] 6.1 Update `ProcessorMock.AddListing` signature and func field
- [x] 6.2 Update `ProcessorMock.AddListingAndEmit` signature and func field

## Phase 7: atlas-merchant — Consumer [S]

- [x] 7.1 Update `handleAddListingCommand` — remove manual unmarshal, pass `e.Body.ItemSnapshot` directly
- [x] 7.2 Remove unused `asset` and `encoding/json` imports

## Phase 8: atlas-merchant — Tests [M]

- [x] 8.1 Update all snapshot creation in processor_test.go to use `asset.AssetData{}`
- [x] 8.2 Update mock_test.go snapshot creation and call signatures
- [x] 8.3 Update listing/builder_test.go snapshot creation
- [x] 8.4 Update frederick/processor_test.go snapshot creation
- [x] 8.5 `go test ./... -count=1` — all tests pass
- [x] 8.6 `go build` — service builds

## Phase 9: atlas-channel — AssetData Definition [S]

- [x] 9.1 Create `merchant/asset_data.go` with channel-local `AssetData` struct

## Phase 10: atlas-channel — Listing Model [S]

- [x] 10.1 `merchant/listing.go` — change `ListingRestModel.ItemSnapshot` from `json.RawMessage` to `AssetData`
- [x] 10.2 `merchant/listing.go` — change `ListingModel.itemSnapshot` from `json.RawMessage` to `AssetData`, update getter
- [x] 10.3 Remove `encoding/json` import from listing.go

## Phase 11: atlas-channel — Consumer Cleanup [M]

- [x] 11.1 Remove local `itemSnapshot` struct from `kafka/consumer/merchant/consumer.go`
- [x] 11.2 Update `assetFromSnapshot` — accept typed `AssetData` instead of `json.RawMessage`, remove unmarshal
- [x] 11.3 Remove unused `encoding/json` and `time` imports
- [x] 11.4 `go test ./... -count=1` — all tests pass
- [x] 11.5 `go build` — service builds

## Phase 12: Final Verification [S]

- [x] 12.1 `go build` atlas-merchant
- [x] 12.2 `go build` atlas-channel
- [x] 12.3 `go test ./... -count=1` atlas-merchant
- [x] 12.4 `go test ./... -count=1` atlas-channel
- [x] 12.5 Grep for remaining `json.RawMessage` related to ItemSnapshot — zero found
- [x] 12.6 Docker build atlas-merchant
- [x] 12.7 Docker build atlas-channel
