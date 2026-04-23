# Merchant Shop Interactions - Context

Last Updated: 2026-03-18

## Key Files

### libs/atlas-packet (packet definitions)
- `interaction/interaction_writer.go` — `InteractionEnterResultSuccess` struct (line 115), `NewInteractionEnterResultSuccess(mode, room)`, encodes mode byte + room data
- `interaction/interaction_writer_body.go` — Body builders: `CharacterInteractionChatBody`, `CharacterInteractionEnterResultErrorBody`. Missing: enter success body, update merchant body
- `interaction/room.go` — `Room` struct with `NewPersonalShopRoom()`, `NewMerchantShopRoom()`. RoomType 4=PersonalShop, 5=MerchantShop. `RoomShopItem{PerBundle, Quantity, Price, Asset}`
- `interaction/visitor.go` — `NewBaseVisitor(slot, avatar, name)`, `NewMerchantVisitor(itemId, merchantName)`. BaseVisitorType=0, MerchantVisitorType=2
- `interaction/mini_room.go` — MiniRoomType enum, MiniRoom interface with `Spawn()`, `Despawn()`, `Enter()`

### services/atlas-channel (client-facing)
- `merchant/processor.go` — `GetShop(shopId)`, `GetVisitingShop(characterId)`, `PlaceShop()`, `EnterShop()`, etc.
- `merchant/model.go` — Channel-side Model: id, characterId, shopType, title, x, y, permitItemId, listingCount, visitors. **Missing**: listings, mesoBalance, state
- `merchant/rest.go` — Channel-side RestModel: same fields as Model. **Missing**: listings, mesoBalance, state, shopType detail
- `merchant/requests.go` — `requestShop(shopId)` → `GET /merchants/{shopId}`. Returns basic RestModel without listings
- `merchant/producer.go` — 10 Kafka command providers (PlaceShop, OpenShop, CloseShop, EnterShop, ExitShop, EnterMaintenance, ExitMaintenance, AddListing, RemoveListing, PurchaseBundle, SendMessage)
- `kafka/consumer/merchant/consumer.go` — Event handlers: handleShopOpenedEvent (spawns MiniRoom), handleVisitorEvent (chat broadcast only), handleListingPurchasedEvent (log only)
- `kafka/message/merchant/kafka.go` — All command/event type definitions. StatusEventVisitorBody has ShopId + CharacterId
- `socket/handler/character_interaction.go` — All handler modes. PERSONAL_STORE_* (lines 228-262) only log. MERCHANT_* (lines 276-369) call merchant processor
- `socket/model/mini_room.go` — `MerchantShopMiniRoom` (lines 261-338), `PersonalShopMiniRoom` (lines 189-259), constructors, Enter/ToPacketRoom methods
- `socket/model/avatar.go` — `NewFromCharacter(c, mega)` builds avatar from character model
- `character/processor.go` — `GetById()` with decorators, `InventoryDecorator`, `PetAssetEnrichmentDecorator`

### services/atlas-merchant (backend)
- `shop/rest.go` — Full RestModel: CharacterId, ShopType, State, MesoBalance, Visitors, Listings (JSON:API relationship)
- `shop/resource.go` — `GET /merchants/{shopId}` returns shop with listings + visitors via `TransformWithListingsAndVisitors`
- `listing/rest.go` — ListingRestModel: ItemId, ItemType, Quantity, BundleSize, BundlesRemaining, PricePerBundle, ItemSnapshot, DisplayOrder
- `shop/state.go` — ShopType: CharacterShop=1, HiredMerchant=2. State: Draft=1, Open=2, Maintenance=3, Closed=4

## Key Decisions

1. **Atlas-channel resolves character names/avatars** (not atlas-merchant). Character service is already available in atlas-channel via `character.NewProcessor`.

2. **UPDATE_MERCHANT packet** (operation 0x19): Sends meso (int) + items array. Owner sees shop meso balance, visitors see their own meso (0 from server, client substitutes local).

3. **Personal shops route through atlas-merchant** as `ShopType=CharacterShop` (1). Same backend logic, different packet encoding handled in atlas-channel.

4. **Channel-side REST model needs enhancement** to include listings, mesoBalance, state, and shopType. The atlas-merchant endpoint already returns this data via JSON:API includes.

## Data Flow for Shop Enter

```
Client → VISIT packet → character_interaction.go
  → merchant.EnterShop() → Kafka ENTER_SHOP command
  → atlas-merchant validates, adds visitor to Redis
  → emits VISITOR_ENTERED event

atlas-channel consumer receives VISITOR_ENTERED:
  → For entering visitor:
    1. GET /merchants/{shopId} (full shop with listings + visitors)
    2. Resolve owner character (name, avatar) via character.GetById
    3. Resolve visitor characters (names, avatars) via character.GetById
    4. Build MerchantShopMiniRoom or PersonalShopMiniRoom
    5. Send InteractionEnterResultSuccess to entering visitor
  → For existing viewers:
    1. Broadcast chat "Visitor [name] has entered"
```

## Data Flow for Purchase Refresh

```
Client → MERCHANT_BUY packet → character_interaction.go
  → merchant.PurchaseBundle() → Kafka PURCHASE_BUNDLE command
  → atlas-merchant validates, updates listing, handles meso/inventory
  → emits LISTING_PURCHASED event

atlas-channel consumer receives LISTING_PURCHASED:
  1. GET /merchants/{shopId} (updated listings)
  2. For each viewer (owner + visitors):
     Build UPDATE_MERCHANT packet (0x19):
       - Owner: meso = shop.mesoBalance
       - Visitor: meso = 0
       - Items: current listing array
     Send to viewer
```

## Packet Format Reference

### InteractionEnterResultSuccess (existing)
```
byte  mode          // ENTER_RESULT operation code (0x05)
byte  roomType      // 4=PersonalShop, 5=MerchantShop
byte  capacity      // 4
visitor[]            // encoded visitors, terminated with 0xFF
[type-specific data] // see below
```

### MerchantShop room data
```
short messageCount  // 0 for non-owners
[messages]          // string + byte slot (owner only)
string ownerName
byte   maxItemCount // 16
int    meso
byte   itemCount
[items]             // short perBundle, short quantity, int price, asset
```

### PersonalShop room data
```
string title
byte   maxItemCount // 16
byte   itemCount
[items]             // short perBundle, short quantity, int price, asset
```

### UPDATE_MERCHANT (new, operation 0x19)
```
byte  mode          // UPDATE_MERCHANT operation code
int   meso          // owner=shop balance, visitor=0
byte  itemCount
[items]             // short perBundle, short quantity, int price, asset
```
