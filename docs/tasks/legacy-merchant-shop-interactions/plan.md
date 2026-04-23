# Merchant Shop Interactions Plan

Last Updated: 2026-03-18

## Executive Summary

Three gaps in the merchant shop system prevent functional player-to-player commerce:
1. Visitors entering a shop receive no shop interior data (empty room)
2. After a purchase, no shop refresh is sent to viewers (stale listings)
3. Personal shops (type 4) are completely unimplemented (handlers only log)

This plan addresses all three gaps across `libs/atlas-packet`, `services/atlas-channel`, and `services/atlas-merchant`.

## Current State Analysis

### What Works
- Shop placement, opening, closing via Kafka commands
- MiniRoom spawn/despawn on map (shops visible to other players)
- Listing CRUD during Draft/Maintenance states
- Purchase flow with meso deduction/credit and inventory transfer
- Chat broadcast to shop viewers
- Visitor enter/exit/eject tracking
- Frederick storage, notifications, cleanup
- Character disconnect auto-close

### What's Missing

**#1 - Shop Interior on Enter**
- `handleVisitorEvent` in `atlas-channel/kafka/consumer/merchant/consumer.go` only broadcasts a chat message to existing viewers
- The entering visitor never receives `InteractionEnterResultSuccess` with room data
- `InteractionEnterResultSuccess` exists in `libs/atlas-packet` but no body builder function exists for it
- The channel-side `RestModel` doesn't include listings, meso balance, or state — only basic shop metadata

**#2 - Shop Refresh After Purchase**
- `handleListingPurchasedEvent` only logs the event (line 256)
- No UPDATE_MERCHANT packet type exists in `interaction_writer_body.go`
- The Java reference shows operation code 0x19 with: meso (int), item count (byte), items array

**#3 - Personal Shop Support**
- `PERSONAL_STORE_*` handlers (lines 228-262 of `character_interaction.go`) only parse and log
- `PlaceShop` only called for `MerchantShopMiniRoomType`, not `PersonalShopMiniRoomType`
- `PersonalShopMiniRoom` struct exists in socket model but is never constructed from real data
- atlas-merchant already has `CharacterShop` (type 1) with correct behavior (immediate meso, no expiration, disconnect close)

## Proposed Future State

### #1 - Shop Interior on Enter
When `VISITOR_ENTERED` event is received for a characterId:
1. Fetch full shop data via REST (`GET /merchants/{shopId}` — returns listings, visitors, meso balance)
2. Resolve character names/avatars for owner + all visitors via `character.NewProcessor`
3. Build `MerchantShopMiniRoom` or `PersonalShopMiniRoom` with full data
4. Send `InteractionEnterResultSuccess` packet to the entering visitor

The same flow is needed when an **owner** enters their own shop after placement or enters maintenance mode.

### #2 - Shop Refresh After Purchase
When `LISTING_PURCHASED` event is received:
1. Fetch updated shop data via REST
2. Build UPDATE_MERCHANT packet (operation 0x19) for each viewer
3. For owner: meso = shop's meso balance; for visitors: meso = 0 (client uses local meso)
4. Send to all viewers (owner + visitors)

### #3 - Personal Shop Support
Wire `PERSONAL_STORE_*` handlers to call `merchant.NewProcessor` methods, using the same commands as merchant shops. The key differences are handled at the packet layer:
- `PersonalShopMiniRoomType` (4) uses `PersonalShopMiniRoom.Enter()` encoding
- All visitors are `MiniRoomVisitorBase` (avatar + name), including owner
- No meso balance field, no messages
- Shop type byte maps to `CharacterShop` (1) in atlas-merchant

## Implementation Phases

### Phase 1: Channel-Side REST Model Enhancement
Enhance `atlas-channel`'s merchant REST model and processor to fetch full shop details (listings, meso balance, state, shopType) needed to build room packets.

### Phase 2: Enter Result Success Body Builder
Add `CharacterInteractionEnterResultSuccessBody` to `interaction_writer_body.go` so atlas-channel can send room data via the standard announce pattern.

### Phase 3: Visitor Enter Response
Wire `handleVisitorEvent` to send `InteractionEnterResultSuccess` to the entering visitor with full room data (visitors resolved with avatars from character service).

### Phase 4: UPDATE_MERCHANT Packet
Add the UPDATE_MERCHANT operation to `interaction_writer_body.go` and wire `handleListingPurchasedEvent` to broadcast the updated shop state to all viewers.

### Phase 5: Personal Shop Handlers
Wire `PERSONAL_STORE_*` handlers to call merchant processor methods and handle personal shop creation/spawn with `PersonalShopMiniRoomType`.

### Phase 6: Owner Enter Response
When an owner places a shop (SHOP_OPENED event for the owner's characterId), also send the room enter packet so the owner enters the shop editing interface.

### Phase 7: Build & Test
Build all affected services, run tests, verify Docker builds.

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| Channel-side REST model doesn't parse JSON:API included listings | Blocks #1, #2 | Need to verify JSON:API relationship parsing or add a dedicated request for shop+listings |
| Character avatar resolution adds latency to visitor enter | Medium | Resolve in parallel; cache is typically warm |
| Personal shop and merchant shop share backend but client expects different packet encoding | Medium | Handle routing in atlas-channel based on shopType from REST response |
| VISITOR_ENTERED event doesn't carry shopId in the right place | Low | Event body already has ShopId field |

## Success Metrics
- Player can enter a merchant/personal shop and see all listings, visitors, owner info
- After a purchase, all viewers see updated listing quantities
- Personal shops can be created, entered, and used for buying/selling
- All builds pass, Docker images build successfully

## Dependencies
- `libs/atlas-packet` — new body builders
- `services/atlas-channel` — consumer handlers, REST model, processor
- `services/atlas-merchant` — REST endpoint already returns needed data (no changes expected)
- `character.NewProcessor` — already available for name/avatar resolution
