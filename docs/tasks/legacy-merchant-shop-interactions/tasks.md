# Merchant Shop Interactions - Tasks

Last Updated: 2026-03-18

## Phase 1: Channel-Side REST Model Enhancement (M)

Enhance atlas-channel's merchant model and REST layer to include full shop data needed for room packets. The atlas-merchant endpoint returns listings as JSON:API `included` resources. The channel-side RestModel must implement `api2go` relationship interfaces to parse them (following the pattern in `atlas-inventory/inventory/rest.go`).

- [x] 1.1 Create `atlas-channel/merchant/listing.go` — define `ListingRestModel` with fields: Id, ShopId, ItemId, ItemType, Quantity, BundleSize, BundlesRemaining, PricePerBundle, ItemSnapshot (json.RawMessage), DisplayOrder. Implement `GetID()`, `SetID()`, `GetName()` (returns "listings")
- [x] 1.2 Define `ListingModel` value type with getters in same file
- [x] 1.3 Add fields to `atlas-channel/merchant/rest.go` RestModel: `MesoBalance uint32`, `State byte`, `Listings []ListingRestModel` (json:"-" tag)
- [x] 1.4 Implement JSON:API relationship interfaces on RestModel:
  - `GetReferences()` — returns listings reference
  - `GetReferencedIDs()` — returns listing IDs
  - `GetReferencedStructs()` — returns listing objects
  - `SetToManyReferenceIDs(name, IDs)` — initializes Listings slice with IDs
  - `SetReferencedStructs(references)` — populates Listings from included data using `jsonapi.ProcessIncludeData`
- [x] 1.5 Add corresponding fields to `atlas-channel/merchant/model.go` Model with getters: `MesoBalance()`, `State()`, `Listings()` (returns []ListingModel)
- [x] 1.6 Update `Extract()` in `rest.go` to map listings, mesoBalance, state into Model
- [x] 1.7 Build atlas-channel, verify compilation

## Phase 2: Enter Result Success Body Builder (S)

Add body builder functions to atlas-packet for sending room enter success and update merchant packets.

- [x] 2.1 Add `CharacterInteractionEnterResultSuccessBody(room Room)` to `interaction_writer_body.go` — resolves ENTER_RESULT operation code, returns `InteractionEnterResultSuccess` encoded bytes
- [x] 2.2 Add `CharacterInteractionUpdateMerchantBody(meso uint32, items []RoomShopItem)` to `interaction_writer_body.go` — new operation mode "UPDATE_MERCHANT", encodes meso (int) + item count (byte) + items array (short perBundle, short quantity, int price, asset)
- [x] 2.3 Add `InteractionUpdateMerchant` struct to `interaction_writer.go` with Encode method
- [x] 2.4 Build libs/atlas-packet, verify compilation

## Phase 3: Visitor Enter Response (M)

Wire the VISITOR_ENTERED event handler to send full shop interior to the entering visitor.

- [x] 3.1 Add `buildShopRoom` helper function in merchant consumer that fetches shop via REST, resolves owner/visitor characters, builds appropriate room type
- [x] 3.2 Add `buildMerchantShopRoom` — MerchantVisitor owner (permitItemId, title), BaseVisitor entries for visitors, items from listings, mesoBalance
- [x] 3.3 Add `buildPersonalShopRoom` — BaseVisitor owner (avatar + name), BaseVisitor entries for visitors, items from listings
- [x] 3.4 Add `buildShopItems` and `assetFromSnapshot` helpers to convert listing ItemSnapshot JSON to packet model Assets
- [x] 3.5 In `handleVisitorEvent` VISITOR_ENTERED case: send `InteractionEnterResultSuccessBody(room)` to the entering visitor
- [x] 3.6 Build atlas-channel, verify compilation

## Phase 4: UPDATE_MERCHANT Packet on Purchase (M)

Wire the LISTING_PURCHASED event handler to broadcast updated shop state to all viewers.

- [x] 4.1 In `handleListingPurchasedEvent`: fetch updated shop data via REST
- [x] 4.2 Convert shop listings to `[]RoomShopItem`
- [x] 4.3 For each viewer: send `CharacterInteractionUpdateMerchantBody`. Owner receives meso=shop.MesoBalance(), visitors receive meso=0
- [x] 4.4 Build atlas-channel, verify compilation

## Phase 5: Personal Shop Handlers (M)

Wire PERSONAL_STORE_* handlers to call merchant processor methods.

- [x] 5.1 In `character_interaction.go` CREATE handler: also call `PlaceShop` for `PersonalShopMiniRoomType` with shopType=1 (CharacterShop)
- [x] 5.2 Wire `PERSONAL_STORE_PUT_ITEM` handler to call `mp.AddListing()`
- [x] 5.3 Wire `PERSONAL_STORE_BUY` handler to call `mp.PurchaseBundle()`
- [x] 5.4 Wire `PERSONAL_STORE_REMOVE_ITEM` handler to call `mp.RemoveListing()`
- [x] 5.5 In `handleShopOpenedEvent`: check `shopType` from event body, spawn correct MiniRoomType (PersonalShop vs MerchantShop)

## Phase 6: Owner Enter Response (S)

When an owner creates a shop, send them the room enter packet so they enter the shop editing interface.

- [x] 6.1 In `handleShopOpenedEvent`: after spawning MiniRoom on the map, send `InteractionEnterResultSuccessBody(room)` to the owner's session

## Phase 7: Maintenance Enter Response (S)

When an owner enters maintenance, send them the room enter packet with current shop state.

- [x] 7.1 In `handleMaintenanceEvent` MAINTENANCE_ENTERED case: fetch shop data, build room, send `InteractionEnterResultSuccessBody` to the owner

## Phase 8: Build & Verify (S)

- [x] 8.1 Run `go test ./...` in atlas-channel — all pass
- [x] 8.2 Run `go build` in atlas-channel — passes
- [x] 8.3 Run `go test ./...` in atlas-packet — all pass
- [x] 8.4 Run `go build` in atlas-packet — passes
- [x] 8.5 Docker build verification for atlas-channel
