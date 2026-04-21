# Merchant Notification Gaps - Context

Last Updated: 2026-03-18

## Key Files

### atlas-packet (interaction library)
| File | Purpose | Key Lines |
|------|---------|-----------|
| `libs/atlas-packet/interaction/interaction_writer.go` | Packet structs (InteractionEnter, InteractionChat, etc.) | 86-113 (InteractionEnter — exists, needs body func) |
| `libs/atlas-packet/interaction/interaction_writer_body.go` | Body builder functions + mode constants | 15-47 (constants), 49-91 (body funcs — missing LEAVE + ENTER bodies) |
| `libs/atlas-packet/interaction/mini_room.go` | MiniRoomBase Spawn/Despawn encoding | 68-94 (Spawn encodes visitorCount as len(VisitorList)) |
| `libs/atlas-packet/interaction/visitor.go` | Visitor types (Base, Game, Merchant) | 38-48 (constructors), 62-78 (encoding) |
| `libs/atlas-packet/interaction/room.go` | Room encoding for merchant/personal shops | Full file |

### atlas-channel (consumer + map)
| File | Purpose | Key Lines |
|------|---------|-----------|
| `services/atlas-channel/.../kafka/consumer/merchant/consumer.go` | All merchant event handlers | 128-144 (SHOP_CLOSED), 146-184 (visitor events), 186-208 (maintenance), 285-318 (listing purchased), 320-331 (broadcastToShopViewers) |
| `services/atlas-channel/.../kafka/consumer/map/consumer.go` | Map enter merchant spawn | 218 (enterMap call), 433-450 (spawnMerchantsForSession) |
| `services/atlas-channel/.../kafka/message/merchant/kafka.go` | Kafka message types (channel-side) | Full file |
| `services/atlas-channel/.../merchant/processor.go` | REST client for atlas-merchant | Full file (87 lines) |
| `services/atlas-channel/.../merchant/model.go` | Merchant model with Visitors() | Full file |
| `services/atlas-channel/.../merchant/requests.go` | REST endpoint URLs | Full file |

### atlas-merchant (backend)
| File | Purpose | Key Lines |
|------|---------|-----------|
| `services/atlas-merchant/.../shop/processor.go` | Core business logic | 324 (maintenance eject), 385-453 (CloseShop), 440 (close eject), 721 (sold-out eject), 765-790 (EnterShop), 793-805 (ExitShop), 807-813 (EjectAllVisitors) |
| `services/atlas-merchant/.../shop/producer.go` | Kafka event producers | 36-47 (closed), 62-73 (maint exited), 75-86 (visitor entered), 88-99 (visitor exited), 101-112 (visitor ejected — never called) |
| `services/atlas-merchant/.../kafka/message/merchant/kafka.go` | Event body structs | 145-153 (ShopClosedBody, VisitorBody) |
| `services/atlas-merchant/.../visitor/registry.go` | Redis visitor management (currently set-based) | Full file |
| `services/atlas-merchant/.../shop/mock/processor.go` | Mock for tests | Lines 37, 207-212 |

### Configuration
| File | Purpose |
|------|---------|
| `services/atlas-configurations/seed-data/templates/template_gms_83_1.json` | CharacterInteraction operation codes (needs LEAVE: 10) |

## Key Design Decisions

### 1. LEAVE Packet Status Byte
- Hardcoded to `0` for all cases
- TODO for user to verify against client behavior when testing
- Client reads: `[mode=10][slot byte][status byte]`

### 2. ENTER Packet for Visitor Join
- `InteractionEnter` struct already exists (interaction_writer.go:86-113)
- Needs only a body builder function `CharacterInteractionEnterBody(visitor Visitor)`
- Takes a `Visitor` which encodes as `[slot byte][avatar bytes][name string]` for BaseVisitorType
- atlas-channel needs to fetch character avatar data to build the visitor; check existing patterns for character data retrieval in the consumer

### 3. Sorted Set Migration for Slot Stability
- **Chosen:** Migrate Redis visitor storage from set to sorted set with timestamp scores
- **Why:** Redis `SMEMBERS` has non-deterministic order; slot assigned at enter time could differ from slot computed at exit time
- **How:** `ZADD` with `time.Now().UnixNano()` score on enter, `ZRANGEBYSCORE 0 +inf` for ordered retrieval
- Must verify atlas-redis Index API supports sorted sets or use raw Redis commands

### 4. Slot Computation via Event Enrichment
- **Chosen:** Add `Slot byte` to `StatusEventVisitorBody` in both atlas-merchant and atlas-channel message definitions
- **Why:** Visitor is already removed from Redis by the time the event arrives at atlas-channel
- atlas-merchant computes slot in `ExitShop()` by calling `GetVisitors()` BEFORE `RemoveVisitor()`
- With sorted set (Phase 2), slot order is deterministic

### 5. Ejection Events in Close/Maintenance
- **Chosen:** Callers of `EjectAllVisitors()` emit individual VISITOR_EJECTED events
- **Why:** Modifying `EjectAllVisitors()` would change the interface and require passing message buffer
- **Verified:** `mb` is in scope at all three call sites:
  - CloseShop (line 440): `mb` is parameter of `CloseShop(mb *message.Buffer)`, used at lines 448 and 452
  - PurchaseBundle (line 721): need to verify `mb` scope
  - EnterMaintenance (line 324): need to verify `mb` scope
- Pattern: GetVisitors() → EjectAllVisitors() → emit events for each with computed slot

### 6. Shop Close Map Broadcast
- **Chosen:** Fetch shop via REST in SHOP_CLOSED handler to get field data, then broadcast
- **Why:** SHOP_CLOSED event only carries shopId + closeReason, no field data
- REST pattern already established by `handleListingPurchasedEvent` (line 299)
- Shop records persist in DB after close

### 7. Mini-Room Visitor Count
- **Chosen:** Add `VisitorCount byte` field to `MiniRoomBase`, use in Spawn encoding
- Count-only approach; no visitor identity data needed for map sprite

### 8. Maintenance Exit Behavior
- **Hired merchants:** Owner receives LEAVE(slot=0, status=0) — closes management UI
- **Personal shops:** Owner receives full ENTER_RESULT room refresh — stays in shop with updated listings
- Requires REST fetch to determine shop type from MAINTENANCE_EXITED event

## Dependencies Between Changes

```
atlas-packet LEAVE + ENTER types (Phase 1)
   └── Required by all LEAVE/ENTER packet sends in Phases 3, 4, 5, 7

Sorted set migration (Phase 2)
   └── Required for deterministic slot assignment in Phase 3+

StatusEventVisitorBody Slot field (Phase 3.1-3.6)
   └── Required by ejection event emission (Phase 4)

EjectAllVisitors event emission (Phase 4)
   └── Required for shop close visitor notification

Shop REST fetch pattern (Phase 5)
   └── Same pattern reused in Phase 7 (maintenance exit)
```

## Existing Patterns to Follow

### Broadcasting to map sessions
```go
// From handleShopOpenedEvent (line 113)
_map.NewProcessor(l, ctx).ForSessionsInMap(f,
    session.Announce(l)(ctx)(wp)(interactionpkt.MiniRoomWriter)(mr.Spawn(e.CharacterId)))
```

### Broadcasting to shop viewers
```go
// From handleVisitorEvent (line 168-176)
broadcastToShopViewers(l, ctx, sc, wp, e.Body.ShopId, func(characterIds []uint32) {
    announce := session.Announce(l)(ctx)(wp)(interactionpkt.CharacterInteractionWriter)
    for _, cid := range characterIds {
        _ = sp.IfPresentByCharacterId(sc.Channel())(cid, announce(bodyFunc))
    }
})
```

### Sending to single character
```go
// From handleVisitorEvent (line 164)
_ = sp.IfPresentByCharacterId(sc.Channel())(characterId,
    session.Announce(l)(ctx)(wp)(interactionpkt.CharacterInteractionWriter)(bodyFunc))
```

### Fetching shop from REST
```go
// From handleListingPurchasedEvent (line 299)
mp := merchant.NewProcessor(l, ctx)
shop, err := mp.GetShop(shopId)
```

### Building field model from shop data
```go
// From handleShopOpenedEvent (line 92-96)
f := field.NewModel(worldId, channelId, mapId, instanceId)
```

## Critical Gotcha: Event Timing

When atlas-channel receives a VISITOR_EXITED or VISITOR_EJECTED event, the visitor has **already been removed** from Redis in atlas-merchant. This means:
- `GetShop(shopId).Visitors()` will NOT include the exited/ejected visitor
- Slot must be computed BEFORE removal and sent in the event
- `broadcastToShopViewers()` will correctly exclude the exited visitor (they're gone from the list)

## Verified: mb Scope at CloseShop

```go
// processor.go:383-453
func (p *ProcessorImpl) CloseShop(mb *message.Buffer) func(...) error {  // mb captured here
    return func(shopId uuid.UUID, characterId uint32, reason CloseReason) error {
        // ...
        p.EjectAllVisitors(shopId)  // line 440 — mb is in scope
        // ...
        for _, ls := range listings {
            acceptItemToBuffer(mb, characterId, ls)  // line 448 — mb used
        }
        return mb.Put(...)  // line 452 — mb used
    }
}
```
