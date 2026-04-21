# Merchant Notification Gaps - Implementation Plan

Last Updated: 2026-03-18

## Executive Summary

Five merchant shop notification gaps prevent proper client-server synchronization during shop interactions. Visitors receive no UI updates when exiting shops or being ejected during close/maintenance. Shop closures aren't broadcast to the map, leaving stale shop sprites. Mini-room spawn packets show 0 visitors regardless of actual occupancy. Maintenance exit leaves the owner's UI in a stale state.

Additionally, two structural issues were discovered: visitor enter only sends a chat message to existing viewers (no avatar added), and Redis set-backed visitor storage has non-deterministic ordering making slot assignment fragile.

All gaps are primarily in **atlas-channel's Kafka consumer** and **atlas-packet's interaction library**, with structural fixes needed in **atlas-merchant** (emitting ejection events, slot computation, sorted set migration). No database changes required.

## Current State Analysis

### What Works
- Shop open broadcasts mini-room spawn to all map sessions via `ForSessionsInMap()`
- Visitor enter sends full room (ENTER_RESULT) to entering visitor
- Listing purchase sends UPDATE_MERCHANT refresh to all viewers (owner + visitors)
- Visitor ejected handler exists in atlas-channel but is **never triggered** (provider defined but never called)

### What's Broken

| Gap | Location | Root Cause |
|-----|----------|------------|
| 1. Visitor exit no packets | atlas-channel consumer line 177-178 | VISITOR_EXITED only logs, sends nothing |
| 2. Shop close no visitor notify | atlas-merchant processor line 440, 721, 324 | `EjectAllVisitors()` removes from Redis only, emits no Kafka events |
| 3. Mini-room 0 visitors | atlas-channel map consumer line 443 | `VisitorList` hardcoded as `[]MiniRoomVisitor{}` |
| 4. Maintenance exit no refresh | atlas-channel consumer line 204-205 | MAINTENANCE_EXITED only logs, sends nothing |
| 5. Shop close no map broadcast | atlas-channel consumer line 141-142 | Despawn sent only to owner session, not to map |

### Additional Issues Discovered

| Issue | Location | Root Cause |
|-------|----------|------------|
| 6. Visitor enter no avatar for existing viewers | atlas-channel consumer line 168-176 | Only sends chat message, not ENTER packet with avatar |
| 7. Non-deterministic slot ordering | atlas-merchant visitor/registry.go | Redis set (`SMEMBERS`) has no guaranteed order |

### Key Structural Issue: Missing LEAVE Packet

The CharacterInteraction protocol requires a **LEAVE mode (opcode 10)** to notify clients when a visitor leaves a room. This mode is not defined in the template or atlas-packet. Format: `[mode=10][slot byte][status byte]`.

The client handles LEAVE as:
- If slot == viewer's own slot → close room dialog
- If slot != viewer's slot → remove that visitor from room display

### Key Structural Issue: Missing ENTER Body Function

The `InteractionEnter` struct exists (interaction_writer.go:86) with `CharacterInteractionModeEnter = "ENTER"` (opcode 4), but there is **no `CharacterInteractionEnterBody` body builder function**. Only `CharacterInteractionEnterResultSuccessBody` and `CharacterInteractionEnterResultErrorBody` exist. A body function is needed to announce new visitors to existing room viewers.

### Key Structural Issue: Silent Ejection

`EjectAllVisitors()` (processor.go:807-812) calls `vr.RemoveAllVisitors()` which removes visitors from Redis, but emits **zero Kafka events**. The `StatusEventVisitorEjectedProvider` exists (producer.go:101) but is **never called** anywhere in the codebase. This means shop close, maintenance enter, and sold-out close all silently drop visitors with no notification.

### Verified: `mb` Available in CloseShop

The message buffer `mb` is the parameter of `CloseShop(mb *message.Buffer)` and is captured by the returned closure (line 385). It is in scope at line 440 where `EjectAllVisitors` is called, and is already used at lines 448 and 452 via `mb.Put()`. No structural change needed to emit ejection events.

## Proposed Future State

After implementation:
1. **Visitor enter** → Existing viewers see the new visitor's avatar appear in the room (ENTER packet).
2. **Visitor exit** → Exiting visitor receives LEAVE (closes room UI). Owner + remaining visitors receive LEAVE (removes avatar from slot).
3. **Shop close** → All visitors receive LEAVE before shop closure. All map sessions receive mini-room despawn.
4. **Map enter** → Mini-room spawn shows correct visitor count.
5. **Maintenance exit** → Hired merchant owner receives LEAVE (closes management UI). Personal shop owner receives full room refresh.
6. **Shop close broadcast** → All characters on the map see the shop sprite disappear.
7. **Slot ordering** → Redis sorted set guarantees consistent insertion-order slots.

## Implementation Phases

### Phase 1: Packet Infrastructure (LEAVE + ENTER body)

Add the LEAVE packet type and ENTER body function to atlas-packet and the configuration template. This is the foundation for all notification fixes.

**Changes:**

#### 1.1 Add LEAVE mode constant to atlas-packet
- **File:** `libs/atlas-packet/interaction/interaction_writer_body.go`
- **Change:** Add `CharacterInteractionModeLeave CharacterInteractionMode = "LEAVE"` to constants
- **Effort:** S

#### 1.2 Add InteractionLeave struct to atlas-packet
- **File:** `libs/atlas-packet/interaction/interaction_writer.go`
- **Change:** Add new `InteractionLeave` struct with fields: `mode byte`, `slot byte`, `status byte`
- Implement `Operation()`, `String()`, `Encode()` methods following existing patterns
- Encode format: `[mode byte][slot byte][status byte]`
- **Effort:** S

#### 1.3 Add CharacterInteractionLeaveBody function
- **File:** `libs/atlas-packet/interaction/interaction_writer_body.go`
- **Change:** Add body function:
  ```go
  func CharacterInteractionLeaveBody(slot byte, status byte) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte
  ```
- Resolves LEAVE mode from template operations, creates `InteractionLeave`
- Status is hardcoded 0 for now. TODO: verify status byte values with client testing.
- **Effort:** S

#### 1.4 Add CharacterInteractionEnterBody function
- **File:** `libs/atlas-packet/interaction/interaction_writer_body.go`
- **Change:** Add body function:
  ```go
  func CharacterInteractionEnterBody(visitor Visitor) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte
  ```
- Resolves ENTER mode (opcode 4) from template operations, creates `InteractionEnter`
- The `InteractionEnter` struct already exists (interaction_writer.go:86-113), just needs a body builder
- **Effort:** S

#### 1.5 Add LEAVE to CharacterInteraction template
- **File:** `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`
- **Change:** Add `"LEAVE": 10` to the CharacterInteraction operations map
- **Effort:** S

**Phase 1 Acceptance Criteria:**
- `CharacterInteractionLeaveBody(slot, 0)` produces correct packet bytes
- `CharacterInteractionEnterBody(visitor)` produces correct packet bytes
- Template includes LEAVE opcode
- atlas-packet builds cleanly

---

### Phase 2: Visitor Sorted Set Migration (Slot Stability)

Migrate visitor storage from Redis set to sorted set to guarantee insertion-order slot stability.

**Problem:** Redis `SMEMBERS` returns members in no particular order. When visitor A enters at slot 1 and visitor B at slot 2, a subsequent `SMEMBERS` call might return them in reversed order. This means the slot a visitor sees when they enter (via `buildShopRoom`) could differ from the slot computed at exit time.

**Solution:** Use Redis sorted set (`ZADD` with timestamp score) so `ZRANGEBYSCORE` always returns visitors in insertion order.

#### 2.1 Migrate AddVisitor to ZADD
- **File:** `services/atlas-merchant/atlas.com/merchant/visitor/registry.go`
- **Change:** Replace set-based `AddVisitor` with sorted set add using `time.Now().UnixNano()` as score
- **Effort:** M

#### 2.2 Migrate GetVisitors to ZRANGEBYSCORE
- **File:** `services/atlas-merchant/atlas.com/merchant/visitor/registry.go`
- **Change:** Replace `SMEMBERS` with `ZRANGEBYSCORE 0 +inf` to return visitors in insertion order
- **Effort:** S

#### 2.3 Migrate RemoveVisitor to ZREM
- **File:** `services/atlas-merchant/atlas.com/merchant/visitor/registry.go`
- **Change:** Replace set remove with sorted set `ZREM`
- **Effort:** S

#### 2.4 Migrate RemoveAllVisitors to ZRANGEBYSCORE + DEL
- **File:** `services/atlas-merchant/atlas.com/merchant/visitor/registry.go`
- **Change:** Get ordered members first (for return value), then delete key
- **Effort:** S

#### 2.5 Migrate GetVisitorCount to ZCARD
- **File:** `services/atlas-merchant/atlas.com/merchant/visitor/registry.go`
- **Change:** Replace `SCARD` with `ZCARD`
- **Effort:** S

#### 2.6 Verify atlas-redis Index supports sorted sets (or use raw commands)
- Check if `atlas.Index` uses sets or sorted sets internally. If it only supports sets, use direct Redis commands via the registry's Redis client.
- **Effort:** S-M (depends on atlas-redis API)

**Phase 2 Acceptance Criteria:**
- Visitors returned in insertion order consistently
- Slot assignment is deterministic: first entrant = slot 1, second = slot 2, etc.
- Existing tests pass

---

### Phase 3: Visitor Exit Notification (Gap 1) + Visitor Enter Broadcast (Gap 6)

When a visitor voluntarily exits a shop, notify all parties. Also fix visitor enter to broadcast avatar to existing viewers.

**Design Decision — Slot Computation:**
The LEAVE packet requires the exiting visitor's slot number. Slots are assigned based on position in the visitor array (slot 0 = owner, slot 1+ = visitors in insertion order). The VISITOR_EXITED event only carries `ShopId` and `CharacterId` — no slot.

**Solution:** Enrich the `StatusEventVisitorBody` with a `Slot byte` field. atlas-merchant computes the slot before removing the visitor from the registry. With Phase 2's sorted set, slot order is now deterministic.

#### 3.1 Add Slot field to StatusEventVisitorBody (atlas-merchant)
- **File:** `services/atlas-merchant/atlas.com/merchant/kafka/message/merchant/kafka.go`
- **Change:** Add `Slot byte \`json:"slot"\`` to `StatusEventVisitorBody`
- **Effort:** S

#### 3.2 Add Slot field to StatusEventVisitorBody (atlas-channel)
- **File:** `services/atlas-channel/atlas.com/channel/kafka/message/merchant/kafka.go`
- **Change:** Mirror the same `Slot byte` field addition
- **Effort:** S

#### 3.3 Compute slot in ExitShop and pass to provider
- **File:** `services/atlas-merchant/atlas.com/merchant/shop/processor.go`
- **Change:** In `ExitShop()` (line 793), before calling `vr.RemoveVisitor()`:
  1. Call `vr.GetVisitors()` to get current visitor list (sorted by insertion)
  2. Find `characterId` position in list → slot = position + 1
  3. Pass slot to updated provider
- **Effort:** M

#### 3.4 Update StatusEventVisitorExitedProvider with slot
- **File:** `services/atlas-merchant/atlas.com/merchant/shop/producer.go`
- **Change:** Update `StatusEventVisitorExitedProvider` to accept and populate `Slot byte`
- **Effort:** S

#### 3.5 Compute slot in EnterShop and pass to provider
- **File:** `services/atlas-merchant/atlas.com/merchant/shop/processor.go`
- **Change:** In `EnterShop()`, after calling `vr.AddVisitor()`:
  1. Call `vr.GetVisitors()` to get updated list
  2. Find `characterId` position → slot = position + 1
  3. Pass slot to updated provider
- **Effort:** S

#### 3.6 Update StatusEventVisitorEnteredProvider with slot
- **File:** `services/atlas-merchant/atlas.com/merchant/shop/producer.go`
- **Change:** Update `StatusEventVisitorEnteredProvider` to accept and populate `Slot byte`
- **Effort:** S

#### 3.7 Handle VISITOR_EXITED in atlas-channel consumer
- **File:** `services/atlas-channel/atlas.com/channel/kafka/consumer/merchant/consumer.go`
- **Change:** Replace the log-only handler (line 177-178) with:
  1. Send LEAVE(slot, 0) to the exiting visitor (closes their room UI)
  2. Broadcast LEAVE(slot, 0) to owner + remaining visitors (removes avatar)
  3. Use `broadcastToShopViewers()` pattern, excluding the exiting visitor
- **Effort:** M

#### 3.8 Handle VISITOR_ENTERED — broadcast ENTER to existing viewers
- **File:** `services/atlas-channel/atlas.com/channel/kafka/consumer/merchant/consumer.go`
- **Change:** In the VISITOR_ENTERED case (line 156-176), after sending ENTER_RESULT to the entering visitor:
  1. Build a `BaseVisitor` for the entering visitor (slot from event, avatar from character data, name)
  2. Broadcast `CharacterInteractionEnterBody(visitor)` to owner + existing visitors (excluding the entrant)
  3. Replace or supplement the existing chat message notification
- **Effort:** M

#### 3.9 Update mock processor if signatures changed
- **File:** `services/atlas-merchant/atlas.com/merchant/shop/mock/processor.go`
- **Change:** Update if ExitShop/EnterShop mock signatures change
- **Effort:** S

**Phase 3 Acceptance Criteria:**
- Exiting visitor's room UI closes
- Owner and remaining visitors see the exiting visitor's avatar removed
- Entering visitor's avatar appears for existing viewers
- Slot numbers are correct and deterministic (1-indexed for visitors, insertion order)

---

### Phase 4: Ejection Events During Close/Maintenance (Gap 2)

`EjectAllVisitors()` silently removes visitors from Redis. Callers need to emit VISITOR_EJECTED events so atlas-channel can send LEAVE packets to each ejected visitor.

**Design Decision:** Rather than modifying `EjectAllVisitors()` itself (which would require passing the message buffer through), the callers emit events for each ejected visitor. `mb` is confirmed in scope at all three call sites.

#### 4.1 Update StatusEventVisitorEjectedProvider with slot
- **File:** `services/atlas-merchant/atlas.com/merchant/shop/producer.go`
- **Change:** Update `StatusEventVisitorEjectedProvider` to accept and populate `Slot byte`
- **Effort:** S

#### 4.2 Emit VISITOR_EJECTED events in CloseShop
- **File:** `services/atlas-merchant/atlas.com/merchant/shop/processor.go`
- **Change:** At line 440 (`CloseShop`), before calling `EjectAllVisitors()`:
  1. Call `GetVisitors()` to get current visitor list with positions (sorted)
  2. Call `EjectAllVisitors()` to remove from Redis
  3. For each ejected visitor, emit `StatusEventVisitorEjectedProvider` with slot (position + 1) via `mb.Put()`
- `mb` is in scope — confirmed as parameter of `CloseShop(mb *message.Buffer)`, used at lines 448 and 452
- **Effort:** M

#### 4.3 Emit VISITOR_EJECTED events in PurchaseBundle sold-out close
- **File:** `services/atlas-merchant/atlas.com/merchant/shop/processor.go`
- **Change:** At line 721 (sold-out close in `PurchaseBundle`), same pattern as 4.2
- Verify `mb` is in scope at this call site
- **Effort:** S

#### 4.4 Emit VISITOR_EJECTED events in EnterMaintenance
- **File:** `services/atlas-merchant/atlas.com/merchant/shop/processor.go`
- **Change:** At line 324 (`EnterMaintenance`), same pattern:
  1. Get visitors with positions before ejection
  2. Eject all
  3. Emit VISITOR_EJECTED for each with slot
- Verify `mb` is in scope at this call site
- **Effort:** S

#### 4.5 Update VISITOR_EJECTED handler in atlas-channel
- **File:** `services/atlas-channel/atlas.com/channel/kafka/consumer/merchant/consumer.go`
- **Change:** Replace line 179-181 (sends ENTER_RESULT error) with:
  1. Send LEAVE(slot, 0) to the ejected visitor (closes room UI)
  2. No need to broadcast to other viewers — they're all being ejected too (during close/maintenance, all visitors are ejected)
- **Effort:** S

**Phase 4 Acceptance Criteria:**
- When shop closes, all visitors receive LEAVE packet and their room UI closes
- When maintenance entered, all visitors receive LEAVE and are removed
- When shop sells out, all visitors are ejected with LEAVE
- `mb` confirmed in scope at all three call sites

---

### Phase 5: Shop Close Map Broadcast (Gap 5)

The mini-room despawn must be broadcast to all sessions on the map, not just the owner.

#### 5.1 Fetch shop data and broadcast despawn to map
- **File:** `services/atlas-channel/atlas.com/channel/kafka/consumer/merchant/consumer.go`
- **Change:** In `handleShopClosedEvent()` (line 128-144):
  1. Fetch shop via `merchant.NewProcessor(l, ctx).GetShop(e.Body.ShopId)` to get field data (mapId, instanceId)
  2. Build `field.Model` from shop's worldId/channelId/mapId/instanceId
  3. Replace single-session despawn with `_map.NewProcessor(l, ctx).ForSessionsInMap(f, session.Announce(...)(...)(mr.Despawn(e.CharacterId)))`
  4. Follow the exact pattern from `handleShopOpenedEvent` (line 113)
- **Note:** Shop data remains available via REST even after closing (DB record persists)
- **Effort:** M

**Phase 5 Acceptance Criteria:**
- All characters on the map see the shop sprite disappear when it closes
- Works for all close reasons (manual, disconnect, expired, sold out, empty)

---

### Phase 6: Mini-Room Visitor Count (Gap 3)

#### 6.1 Add VisitorCount field to MiniRoomBase
- **File:** `libs/atlas-packet/interaction/mini_room.go`
- **Change:** Add `VisitorCount byte` field to `MiniRoomBase`. In `Spawn()`, encode this instead of `byte(len(mr.VisitorList))`
- **Effort:** S

#### 6.2 Set visitor count 0 in shop open handler
- **File:** `services/atlas-channel/atlas.com/channel/kafka/consumer/merchant/consumer.go`
- **Change:** In `handleShopOpenedEvent()`, set `VisitorCount: 0` (new shops have no visitors)
- **Effort:** S

#### 6.3 Populate visitor count in map enter spawn
- **File:** `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go`
- **Change:** In `spawnMerchantsForSession()` (line 438-445), set `VisitorCount: byte(len(m.Visitors()))`
- **Effort:** S

#### 6.4 Fix MiniRoomType for personal shops in map enter
- **File:** `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go`
- **Change:** `spawnMerchantsForSession` hardcodes `MerchantShopMiniRoomType`. Should check `m.ShopType()` and use `PersonalShopMiniRoomType` for personal shops, matching the pattern in `handleShopOpenedEvent`.
- **Effort:** S

**Phase 6 Acceptance Criteria:**
- Characters entering a map see correct visitor counts on shop bubbles
- Personal shops display correct mini-room type

---

### Phase 7: Maintenance Exit Refresh (Gap 4)

#### 7.1 Handle MAINTENANCE_EXITED based on shop type
- **File:** `services/atlas-channel/atlas.com/channel/kafka/consumer/merchant/consumer.go`
- **Change:** In `handleMaintenanceEvent()` MAINTENANCE_EXITED case (line 204-205):
  1. Fetch shop via REST to determine shop type
  2. For **hired merchants** (shopType == 2): Send LEAVE(slot=0, status=0) to owner — closes management UI, shop continues autonomously
  3. For **personal shops** (shopType == 1): Send full ENTER_RESULT room refresh to owner — they stay seated behind the counter with updated listings
- **Effort:** M

**Phase 7 Acceptance Criteria:**
- Hired merchant owner's management UI closes when exiting maintenance
- Personal shop owner sees refreshed room with updated listings after maintenance

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| LEAVE status byte value incorrect | Medium | Low | Hardcode 0 with TODO; verify during client testing |
| Sorted set migration breaks existing visitor flows | Low | High | Phase 2 isolated; run full test suite after migration |
| REST call in SHOP_CLOSED handler returns stale/missing data | Low | Medium | Shop records persist in DB after close; REST endpoint returns all states |
| Visitor ejection events flood Kafka during mass close | Low | Low | Max 3 visitors per shop; negligible message volume |
| MiniRoom visitor count stale between events | Low | Low | Count is best-effort on map enter; real-time accuracy not critical for map sprites |
| ENTER packet for visitor join requires avatar data not available in event | Medium | Medium | atlas-channel fetches character avatar via existing REST patterns when building visitor |

## Success Metrics

1. **Visitor enter**: New visitor's avatar appears in room for existing viewers
2. **Visitor exit**: Room UI closes for exiting visitor; avatar removed for remaining viewers
3. **Shop close**: All visitors' room UIs close; shop sprite disappears for all map characters
4. **Map enter**: Shop bubbles show actual visitor count (0-3)
5. **Maintenance exit**: Owner UI correctly transitions based on shop type
6. **Slot stability**: Slots are deterministic across enter/exit lifecycle
7. **No regressions**: Existing shop open, purchase, and chat flows unchanged

## Dependencies

- **atlas-packet** (Phase 1) must be completed before Phases 3-7
- **Phase 2** (sorted set) should be completed before Phase 3 (slot stability)
- **Phase 3** (slot enrichment) must be completed before Phase 4 (same pattern)
- **Phase 5** (map broadcast) is independent after Phase 1
- **Phase 6** (visitor count) is fully independent
- **Phase 7** (maintenance exit) is independent after Phase 1

```
Phase 1 (LEAVE + ENTER packets)
  │
  ├── Phase 2 (sorted set migration)
  │     └── Phase 3 (visitor exit + enter broadcast) ──→ Phase 4 (ejection events)
  │
  ├── Phase 5 (map broadcast)
  └── Phase 7 (maintenance exit)

Phase 6 (visitor count) — independent
```

## Affected Services

| Service | Changes | Build/Test Required |
|---------|---------|---------------------|
| libs/atlas-packet | LEAVE packet type, ENTER body function, MiniRoom VisitorCount | `go build`, `go test ./...` |
| atlas-configurations | Template update | Seed data only |
| atlas-merchant | Sorted set migration, slot computation, ejection events | `go build`, `go test ./...` |
| atlas-channel | Consumer handlers, map spawn, visitor enter broadcast | `go build`, `go test ./...` |

All four must be Docker-verified after changes.
