# Merchant ↔ Channel Integration — Implementation Plan

Last Updated: 2026-03-17

## Executive Summary

Wire the fully-implemented `atlas-merchant` service into `atlas-channel` so that player shop and hired merchant interactions flow end-to-end: client socket → channel → Kafka command → merchant → Kafka event → channel → socket broadcast → client. This work also fixes a data-model bug (missing worldId/channelId/instanceId on shops), adds a missing `MESSAGE_SENT` event, creates a new `InteractionChat` writer packet, and implements field-entry shop spawning.

The integration touches three codebases:
- **atlas-merchant** — data model fix + new event
- **atlas-packet** — new `InteractionChat` writer
- **atlas-channel** — command producers, event consumers, handler wiring, field entry spawning

---

## Current State Analysis

### What Exists

| Component | State | Notes |
|-----------|-------|-------|
| atlas-merchant service | **Complete** | 13 command handlers, status/listing event producers, 3 background reapers |
| atlas-merchant REST API | **Partial** | `GET /merchants?mapId=` exists but lacks world/channel/instance scoping |
| atlas-merchant shop model | **Bug** | Missing `worldId`, `channelId`, `instanceId` fields on model + entity |
| atlas-merchant SEND_MESSAGE | **Partial** | Persists to DB but emits no Kafka event |
| atlas-packet readers | **Complete** | 6 merchant operation readers (PutItem, Buy, RemoveItem, NameChange, AddBlackList, RemoveBlackList) |
| atlas-packet writers | **Complete** | HiredMerchantOperation (7 structs), Interaction (5 structs), MiniRoom models |
| atlas-packet InteractionChat | **Missing** | No server→client chat broadcast packet for mini-rooms |
| atlas-channel interaction handler | **Partial** | Parses all 12 merchant modes but only logs them |
| atlas-channel hired merchant handler | **Stub** | TODO comment only |
| atlas-channel merchant Kafka producer | **Missing** | No command dispatch to atlas-merchant |
| atlas-channel merchant event consumer | **Missing** | No consumption of merchant status/listing events |
| atlas-channel field entry spawning | **Missing** | `enterMap()` spawns NPCs, monsters, drops, reactors, chalkboards, chairs — no merchants |

### Gap Analysis

1. **No bidirectional Kafka bridge** — channel cannot send commands to or receive events from merchant
2. **Shop model unscoped** — shops only store `mapId`, not `worldId`/`channelId`/`instanceId`; shops from different worlds/channels/instances collide
3. **No chat broadcast** — `SEND_MESSAGE` is write-only; no event emission, no client packet
4. **No field spawning** — players entering a map never see existing merchant shops

---

## Proposed Future State

### Data Flow

```
Client → atlas-channel (socket handler)
         ↓ Kafka command
       atlas-merchant (processes, persists, validates)
         ↓ Kafka event
       atlas-channel (event consumer)
         ↓ socket broadcast
       Client(s) (mini-room state updates)
```

### Field Entry Flow

```
Character enters map
  → atlas-channel enterMap()
    → REST GET /worlds/{w}/channels/{c}/maps/{m}/instances/{i}/merchants
      → atlas-merchant returns open shops
    → spawn MiniRoom for each shop → send to character
```

---

## Implementation Phases

### Phase 1: Fix atlas-merchant Data Model

**Goal:** Add `worldId`, `channelId`, `instanceId` to the shop model so shops are scoped to a specific field instance.

| # | Task | Effort | Depends On |
|---|------|--------|------------|
| 1.1 | Add `worldId` (byte), `channelId` (byte), `instanceId` (uuid.UUID) to `shop.Model` with accessors | S | — |
| 1.2 | Add `WorldId`, `ChannelId`, `InstanceId` columns to `shop.Entity` with GORM tags and migration | S | 1.1 |
| 1.3 | Update `NewModel` / builder to accept and store worldId, channelId, instanceId | S | 1.1 |
| 1.4 | Update `Make(entity)` to populate worldId, channelId, instanceId from entity | S | 1.2 |
| 1.5 | Update `PLACE_SHOP` command handler to persist worldId, channelId, instanceId from command envelope | S | 1.3 |
| 1.6 | Update `CommandPlaceShopBody` to include `instanceId` field (worldId/channelId already on envelope) | S | — |
| 1.7 | Replace `getByMapId` with `getByField(worldId, channelId, mapId, instanceId)` in provider.go | S | 1.2 |
| 1.8 | Update REST routes: add `/worlds/{worldId}/channels/{channelId}/maps/{mapId}/instances/{instanceId}/merchants` endpoint | M | 1.7 |
| 1.9 | Update `SHOP_OPENED` event body to include `worldId`, `channelId`, `instanceId` | S | 1.1 |
| 1.10 | Update Redis registry keys to include world/channel/instance scoping | M | 1.1 |
| 1.11 | Update existing `GET /merchants?mapId=` endpoint to require worldId/channelId/instanceId params (or deprecate) | S | 1.7 |
| 1.12 | Build + test atlas-merchant | S | 1.1–1.11 |

**Acceptance:** Shops are persisted with full field coordinates. REST query returns only shops for the requested world/channel/map/instance tuple.

---

### Phase 2: Add MESSAGE_SENT Event to atlas-merchant

**Goal:** Emit a Kafka event when a chat message is sent in a shop, so channel can broadcast to visitors.

| # | Task | Effort | Depends On |
|---|------|--------|------------|
| 2.1 | Add `StatusEventMessageSent = "MESSAGE_SENT"` constant and `StatusEventMessageSentBody` struct (`shopId`, `characterId`, `content`, `slot`) to kafka message definitions | S | — |
| 2.2 | Update `SEND_MESSAGE` handler to resolve visitor slot and emit `MESSAGE_SENT` event after persistence | M | 2.1 |
| 2.3 | Build + test atlas-merchant | S | 2.1–2.2 |

**Acceptance:** Sending a message in a shop produces a `MESSAGE_SENT` status event on `EVENT_TOPIC_MERCHANT_STATUS` with the sender's slot, shopId, characterId, and content.

---

### Phase 3: Add InteractionChat Packet to atlas-packet

**Goal:** Create the server→client packet for broadcasting chat within a mini-room.

Reference (Java):
```java
p.writeByte(PlayerInteractionHandler.Action.CHAT.getCode());
p.writeByte(PlayerInteractionHandler.Action.CHAT_THING.getCode());
p.writeByte(slot);
p.writeString(chr.getName() + " : " + chat);
```

| # | Task | Effort | Depends On |
|---|------|--------|------------|
| 3.1 | Add `CharacterInteractionModeChat` and `CharacterInteractionModeChatThing` constants to interaction_writer_body.go | S | — |
| 3.2 | Create `InteractionChat` struct in interaction_writer.go with fields: `mode byte`, `chatThing byte`, `slot byte`, `message string` | S | 3.1 |
| 3.3 | Implement `Encode` / `Decode` on `InteractionChat` following existing pattern | S | 3.2 |
| 3.4 | Create `CharacterInteractionChatBody(slot byte, name string, content string)` factory in interaction_writer_body.go — resolves both CHAT and CHAT_THING modes, formats message as `"{name} : {content}"` | S | 3.2 |
| 3.5 | Add round-trip test for `InteractionChat` | S | 3.3 |
| 3.6 | Build + test atlas-packet | S | 3.1–3.5 |

**Acceptance:** `InteractionChat` encodes two mode bytes, a slot byte, and a formatted string. Round-trip test passes.

---

### Phase 4: atlas-channel — Merchant Command Producers

**Goal:** Create a merchant processor/producer in atlas-channel that dispatches Kafka commands to atlas-merchant, following the compartment pattern.

| # | Task | Effort | Depends On |
|---|------|--------|------------|
| 4.1 | Create `channel/kafka/message/merchant/kafka.go` — command envelope struct and constants (mirror atlas-merchant definitions) | M | — |
| 4.2 | Create `channel/merchant/producer.go` — command provider functions for all 11 commands: PlaceShop, OpenShop, CloseShop, EnterMaintenance, ExitMaintenance, AddListing, RemoveListing, PurchaseBundle, EnterShop, ExitShop, SendMessage | L | 4.1 |
| 4.3 | Create `channel/merchant/processor.go` — `Processor` interface with methods matching each command, dispatching via `producer.ProviderImpl` | M | 4.2 |
| 4.4 | Build atlas-channel (compile check) | S | 4.1–4.3 |

**Acceptance:** `merchant.NewProcessor(l, ctx).AddListing(...)` compiles and produces a Kafka message to `COMMAND_TOPIC_MERCHANT`.

---

### Phase 5: atlas-channel — Wire Socket Handlers to Merchant Processor

**Goal:** Replace log-only stubs with actual Kafka command dispatch.

| # | Task | Effort | Depends On |
|---|------|--------|------------|
| 5.1 | Wire `CREATE` (0x00) + MerchantShopMiniRoomType → `PLACE_SHOP` (title, mapId, x, y, permitItemId from OperationCreate) | M | Phase 4 |
| 5.2 | Wire `VISIT` (0x04) + merchant context → `ENTER_SHOP` | S | Phase 4 |
| 5.3 | Wire `OPEN` (0x0B) + merchant context → `OPEN_SHOP` | S | Phase 4 |
| 5.4 | Wire `EXIT` (0x0A) → `EXIT_SHOP` (visitor) or `CLOSE_SHOP` (owner) — requires knowing if character is the shop owner | M | Phase 4 |
| 5.5 | Wire `CHAT` (0x06) + merchant context → `SEND_MESSAGE` | S | Phase 4 |
| 5.6 | Wire `MERCHANT_PUT_ITEM` (0x21) → `ADD_LISTING` (map inventoryType, slot, quantity, set→bundleSize, price) | M | Phase 4 |
| 5.7 | Wire `MERCHANT_BUY` (0x22) → `PURCHASE_BUNDLE` (index→listingIndex, quantity→bundleCount) | S | Phase 4 |
| 5.8 | Wire `MERCHANT_REMOVE_ITEM` (0x26) → `REMOVE_LISTING` (index→listingIndex) | S | Phase 4 |
| 5.9 | Wire `MERCHANT_MAINTENANCE_OFF` (0x27) → `EXIT_MAINTENANCE` | S | Phase 4 |
| 5.10 | Wire `MERCHANT_EXIT` (0x29) → `EXIT_SHOP` | S | Phase 4 |
| 5.11 | Handle `ENTER_MAINTENANCE` — triggered when owner enters their hired merchant shop (determine client trigger mechanism) | M | Phase 4 |
| 5.12 | Build + test atlas-channel | S | 5.1–5.11 |

**Key design question for 5.4:** Channel needs to know if the exiting character is the shop owner. This likely requires either:
- Tracking the character's current shop + role in channel-local state, or
- Including this info from the mini-room model

**Key design question for 5.11:** `ENTER_MAINTENANCE` occurs when the owner enters their own hired merchant. The client flow appears to be: `CASH_TRADE_OPEN` (nProc=4, MerchantShopMiniRoomType) sends the owner into the shop in maintenance mode. This needs to both `ENTER_MAINTENANCE` and `ENTER_SHOP`.

**Acceptance:** All implemented interaction modes dispatch the correct Kafka command to atlas-merchant.

---

### Phase 6: atlas-channel — Merchant Event Consumers

**Goal:** Consume merchant status and listing events and broadcast to connected players via socket.

| # | Task | Effort | Depends On |
|---|------|--------|------------|
| 6.1 | Create `channel/kafka/message/merchant/events.go` — status event and listing event structs (mirror atlas-merchant definitions) | M | — |
| 6.2 | Create `channel/kafka/consumer/merchant/consumer.go` — `InitHandlers` + `InitConsumers` following character consumer pattern | M | 6.1 |
| 6.3 | Handle `SHOP_OPENED` — spawn MerchantShopMiniRoom on map for all characters in the field | L | 6.2 |
| 6.4 | Handle `SHOP_CLOSED` — despawn MiniRoom from map, send exit to all visitors in room | L | 6.2 |
| 6.5 | Handle `VISITOR_ENTERED` — send InteractionEnter packet to all room viewers | M | 6.2 |
| 6.6 | Handle `VISITOR_EXITED` — send exit packet to room viewers | M | 6.2 |
| 6.7 | Handle `VISITOR_EJECTED` — force-exit + error to ejected player | M | 6.2 |
| 6.8 | Handle `MAINTENANCE_ENTERED` — eject visitors, lock room | M | 6.2 |
| 6.9 | Handle `MAINTENANCE_EXITED` — unlock room, allow visitors | M | 6.2 |
| 6.10 | Handle `LISTING_PURCHASED` — refresh listing display for all viewers | M | 6.2 |
| 6.11 | Handle `CAPACITY_FULL` — send InteractionEnterResultError to joining player | S | 6.2 |
| 6.12 | Handle `PURCHASE_FAILED` — send error packet to buyer | S | 6.2 |
| 6.13 | Handle `FREDERICK_NOTIFICATION` — send FreeFormNotice to character | S | 6.2 |
| 6.14 | Handle `MESSAGE_SENT` — broadcast InteractionChat to all room viewers | M | Phase 3, 6.2 |
| 6.15 | Register merchant consumer in atlas-channel main.go | S | 6.2 |
| 6.16 | Build + test atlas-channel | S | 6.1–6.15 |

**Key design concern:** Events like `SHOP_OPENED` include a mapId but the channel consumer needs to know which sessions are on that map to broadcast the spawn. This requires the existing map/session tracking infrastructure in atlas-channel.

**Acceptance:** Each merchant event triggers the correct socket packet to the correct set of connected players.

---

### Phase 7: atlas-channel — Field Entry Shop Spawning

**Goal:** When a character enters a map, spawn all existing merchant shops for them.

| # | Task | Effort | Depends On |
|---|------|--------|------------|
| 7.1 | Create `channel/merchant/rest.go` — `RestModel` + `Extract` function for JSON:API | M | Phase 1.8 |
| 7.2 | Create `channel/merchant/requests.go` — `requestInField(f field.Model)` building URL `/worlds/{w}/channels/{c}/maps/{m}/instances/{i}/merchants` with env base URL `MERCHANTS` | S | 7.1 |
| 7.3 | Create `channel/merchant/model.go` — local shop model (id, shopType, title, ownerName, mapId, x, y, items, meso) | M | — |
| 7.4 | Add `ForEachInField(f field.Model, o model.Operator[Model])` to merchant processor using `requests.SliceProvider` | S | 7.1, 7.2 |
| 7.5 | Create `spawnMerchantShopsForSession()` in `kafka/consumer/map/consumer.go` following the NPC/monster pattern — encodes MiniRoom spawn packet for each shop | M | 7.3, 7.4 |
| 7.6 | Add goroutine in `enterMap()` calling `merchant.NewProcessor(l, ctx).ForEachInField(f, spawnMerchantShopsForSession(l)(ctx)(wp)(s))` | S | 7.5 |
| 7.7 | Build + test atlas-channel | S | 7.1–7.6 |
| 7.8 | Docker build verification for atlas-merchant + atlas-channel | M | All phases |

**Acceptance:** A character entering a Free Market map sees all active merchant shops spawned as MiniRoom objects.

---

## Deferred Work

| Item | Reason |
|------|--------|
| `MERCHANT_ORGANIZE` (0x28) | No backend command exists |
| `MERCHANT_WITHDRAW_MESO` (0x2B) | No backend command exists |
| `HiredMerchantOperationHandleFunc` | Cash shop item trigger — needs IDA research |
| Name change (0x2D) | No backend command exists |
| Blacklist operations (0x30, 0x31) | No backend blacklist model |
| Visit list viewing (0x2E) | No backend query for this |
| Black list viewing (0x2F) | No backend blacklist model |
| `UPDATE_LISTING` command | No client-side trigger identified |
| `RETRIEVE_FREDERICK` command | Deferred with HiredMerchantOperation handler |

---

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| Shop model migration breaks existing data | High | Add columns with defaults, backfill existing rows |
| Owner vs visitor detection for EXIT mode | Medium | Track character's current room + role in channel-local state |
| Event ordering — shop opened before channel processes it | Medium | Idempotent handlers; query REST on miss |
| Mini-room state drift between merchant and channel | Medium | Events are source of truth; channel is a projection |
| High-traffic maps with many shops | Low | REST query is per-field-entry, not per-tick |

---

## Success Metrics

1. Player can place a hired merchant shop and other players see it spawn
2. Visitors can enter a shop, browse listings, purchase items, and see real-time updates
3. Chat messages in a shop are visible to all room occupants
4. Shop owner can add/remove listings, enter/exit maintenance, and close shop
5. Characters entering a map see all existing shops
6. Shops are correctly scoped to world/channel/map/instance — no cross-instance leakage

---

## Effort Estimates

| Phase | Effort | Description |
|-------|--------|-------------|
| Phase 1 | L | Data model fix + REST endpoint + migration |
| Phase 2 | S | Add MESSAGE_SENT event |
| Phase 3 | S | InteractionChat packet |
| Phase 4 | L | Command producers |
| Phase 5 | XL | Wire all socket handlers |
| Phase 6 | XL | Event consumers + socket broadcasts |
| Phase 7 | M | Field entry spawning |
| **Total** | **XL** | ~60 tasks across 7 phases |
