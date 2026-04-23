# Merchant Notification Gaps - Task Checklist

Last Updated: 2026-03-18

## Phase 1: Packet Infrastructure (LEAVE + ENTER body)
- [x] 1.1 Add `CharacterInteractionModeLeave = "LEAVE"` constant to `libs/atlas-packet/interaction/clientbound/interaction_body.go`
- [x] 1.2 Add `InteractionLeave` struct (mode, slot, status bytes) to `libs/atlas-packet/interaction/clientbound/interaction.go`
- [x] 1.3 Add `CharacterInteractionLeaveBody(slot, status)` body function to `libs/atlas-packet/interaction/clientbound/interaction_body.go`
- [x] 1.4 Add `CharacterInteractionEnterBody(visitor)` body function to `libs/atlas-packet/interaction/clientbound/interaction_body.go`
- [x] 1.5 Add `"LEAVE": 10` to CharacterInteraction operations in `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`
- [x] 1.6 Add round-trip test for InteractionLeave
- [x] 1.7 Build + test atlas-packet

## Phase 2: Visitor Sorted Set Migration (Slot Stability)
- [x] 2.1 Checked atlas-redis Index — only supports regular sets, using raw Redis commands
- [x] 2.2 Migrate `AddVisitor` to ZADD with timestamp score in `visitor/registry.go`
- [x] 2.3 Migrate `GetVisitors` to ZRANGEBYSCORE (insertion-ordered) in `visitor/registry.go`
- [x] 2.4 Migrate `RemoveVisitor` to ZREM in `visitor/registry.go`
- [x] 2.5 Migrate `RemoveAllVisitors` to ZRANGEBYSCORE + DEL in `visitor/registry.go`
- [x] 2.6 Migrate `GetVisitorCount` to ZCARD in `visitor/registry.go`
- [x] 2.7 Build + test atlas-merchant

## Phase 3: Visitor Exit Notification (Gap 1) + Visitor Enter Broadcast (Gap 6)
- [x] 3.1 Add `Slot byte` field to `StatusEventVisitorBody` in atlas-merchant kafka message
- [x] 3.2 Add `Slot byte` field to `StatusEventVisitorBody` in atlas-channel kafka message
- [x] 3.3 Update `StatusEventVisitorExitedProvider` to accept `slot byte` in `shop/producer.go`
- [x] 3.4 In `ExitShop()`: call `GetVisitors()` before `RemoveVisitor()`, compute slot, pass to provider
- [x] 3.5 Update `StatusEventVisitorEnteredProvider` to accept `slot byte` in `shop/producer.go`
- [x] 3.6 In `EnterShop()`: after `AddVisitor()`, call `GetVisitors()`, compute slot, pass to provider
- [x] 3.7 Handle VISITOR_EXITED in atlas-channel consumer: send LEAVE(slot, 0) to exiting visitor + broadcast LEAVE to owner/remaining viewers
- [x] 3.8 Handle VISITOR_ENTERED: broadcast `CharacterInteractionEnterBody(visitor)` to existing viewers with avatar
- [x] 3.9 Add `visitorSlot` helper function and `emitEjectionEvents` helper
- [x] 3.10 Build + test atlas-merchant
- [x] 3.11 Build + test atlas-channel

## Phase 4: Ejection Events During Close/Maintenance (Gap 2)
- [x] 4.1 Update `StatusEventVisitorEjectedProvider` to accept `slot byte` in `shop/producer.go`
- [x] 4.2 In `CloseShop()`: get visitors → eject → emit VISITOR_EJECTED for each with slot via `mb.Put()` (mb confirmed in scope)
- [x] 4.3 In `PurchaseBundle()` sold-out close: same pattern (mb confirmed in scope)
- [x] 4.4 In `EnterMaintenance()`: same pattern (mb confirmed in scope)
- [x] 4.5 Update VISITOR_EJECTED handler in atlas-channel: send LEAVE(slot, 0) to ejected visitor
- [x] 4.6 Build + test atlas-merchant
- [x] 4.7 Build + test atlas-channel

## Phase 5: Shop Close Map Broadcast (Gap 5)
- [x] 5.1 Add WorldId, ChannelId, MapId, InstanceId to atlas-channel merchant RestModel and Model
- [x] 5.2 In `handleShopClosedEvent()`: fetch shop via REST, build field model
- [x] 5.3 Replace single-session despawn with `ForSessionsInMap()` broadcast
- [x] 5.4 Build + test atlas-channel

## Phase 6: Mini-Room Visitor Count (Gap 3)
- [x] 6.1 Add `VisitorCount byte` field to `MiniRoomBase` in `libs/atlas-packet/interaction/mini_room.go`
- [x] 6.2 Update `Spawn()` encoding to use `VisitorCount` instead of `len(VisitorList)`
- [x] 6.3 Set `VisitorCount: 0` in `handleShopOpenedEvent()` (merchant consumer)
- [x] 6.4 Set `VisitorCount: byte(len(m.Visitors()))` in `spawnMerchantsForSession()` (map consumer)
- [x] 6.5 Fix shop type: use `m.ShopType()` to set correct MiniRoomType in `spawnMerchantsForSession()`
- [x] 6.6 Build atlas-packet
- [x] 6.7 Build + test atlas-channel

## Phase 7: Maintenance Exit Refresh (Gap 4)
- [x] 7.1 In MAINTENANCE_EXITED handler: fetch shop via REST to get shop type
- [x] 7.2 For hired merchant (type 2): send `CharacterInteractionLeaveBody(0, 0)` to owner
- [x] 7.3 For personal shop (type 1): send `CharacterInteractionEnterResultSuccessBody(room)` with fresh room data
- [x] 7.4 Build + test atlas-channel

## Final Verification
- [x] Docker build atlas-merchant
- [x] Docker build atlas-channel
- [x] Docker build atlas-configurations
