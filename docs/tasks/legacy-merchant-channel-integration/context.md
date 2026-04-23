# Merchant ↔ Channel Integration — Context

Last Updated: 2026-03-17

## Key Files

### atlas-merchant (service modifications)

| File | Purpose | Integration |
|------|---------|-------------|
| `services/atlas-merchant/atlas.com/merchant/shop/model.go` | Shop domain model — **missing worldId, channelId, instanceId** | Add fields + accessors |
| `services/atlas-merchant/atlas.com/merchant/shop/entity.go` | GORM entity — **missing WorldId, ChannelId, InstanceId columns** | Add columns + migration |
| `services/atlas-merchant/atlas.com/merchant/shop/provider.go` | DB queries — `getByMapId` is unscoped | Replace with `getByField(worldId, channelId, mapId, instanceId)` |
| `services/atlas-merchant/atlas.com/merchant/shop/resource.go` | REST routes — `GET /merchants?mapId=` lacks field scoping | Add `/worlds/{w}/channels/{c}/maps/{m}/instances/{i}/merchants` |
| `services/atlas-merchant/atlas.com/merchant/shop/processor_emit.go` | Kafka event emission (OpenShop, CloseShop, etc.) | Update SHOP_OPENED body to include world/channel/instance |
| `services/atlas-merchant/atlas.com/merchant/kafka/message/merchant/kafka.go` | Command + event message definitions | Add MESSAGE_SENT event, add instanceId to PLACE_SHOP body, update SHOP_OPENED body |
| `services/atlas-merchant/atlas.com/merchant/kafka/consumer/merchant/consumer.go` | Command handlers — SEND_MESSAGE only persists | Add event emission after message persistence |
| `services/atlas-merchant/atlas.com/merchant/message/processor.go` | Message persistence (SendMessage, GetMessages) | May need visitor slot resolution for MESSAGE_SENT event |

### atlas-packet (library modifications)

| File | Purpose | Integration |
|------|---------|-------------|
| `libs/atlas-packet/interaction/interaction_writer.go` | Server→client interaction packets | Add `InteractionChat` struct |
| `libs/atlas-packet/interaction/interaction_writer_body.go` | Body factory functions with code resolution | Add `CharacterInteractionChatBody()` |

### atlas-channel (new files)

| File | Purpose | Pattern Reference |
|------|---------|-------------------|
| `channel/kafka/message/merchant/kafka.go` (new) | Command + event message structs | `channel/kafka/message/compartment/kafka.go` |
| `channel/merchant/producer.go` (new) | Kafka command providers (11 commands) | `channel/compartment/producer.go` |
| `channel/merchant/processor.go` (new) | Command dispatch + REST query interface | `channel/compartment/processor.go` + `channel/monster/processor.go` |
| `channel/merchant/model.go` (new) | Local shop domain model | `channel/monster/model.go` |
| `channel/merchant/rest.go` (new) | REST model + Extract function | `channel/monster/rest.go` |
| `channel/merchant/requests.go` (new) | HTTP request builders | `channel/monster/requests.go` |
| `channel/kafka/consumer/merchant/consumer.go` (new) | Status + listing event handlers | `channel/kafka/consumer/character/consumer.go` |

### atlas-channel (modified files)

| File | Purpose | Change |
|------|---------|--------|
| `channel/socket/handler/character_interaction.go` | Interaction mode handler | Wire 11 merchant modes to processor (replace log stubs) |
| `channel/kafka/consumer/map/consumer.go` | Field entry entity spawning | Add `spawnMerchantShopsForSession()` goroutine in `enterMap()` |
| `channel/main.go` | Service initialization | Register merchant event consumer |

---

## Architecture Decisions

### 1. REST Query for Field Entry (not local registry)

**Decision:** Channel queries atlas-merchant via REST (`GET /worlds/{w}/channels/{c}/maps/{m}/instances/{i}/merchants`) during field entry, rather than maintaining a local in-memory registry.

**Rationale:** Follows established pattern — NPCs, monsters, drops, reactors all use REST queries during `enterMap()`. A local registry would introduce state synchronization complexity and diverge from the architecture.

**Trade-off:** Adds a REST call per field entry, but this is consistent with 6 other entity types already queried the same way.

### 2. Kafka Commands (fire-and-forget) for Mutations

**Decision:** Socket handlers dispatch Kafka commands and return immediately. Response to the player comes asynchronously via merchant events consumed by channel.

**Rationale:** Follows the compartment (inventory) pattern exactly. All inventory operations (equip, drop, move) use the same fire-and-forget command → async event → socket broadcast flow.

### 3. InteractionChat Packet Structure

**Decision:** Two mode bytes (CHAT + CHAT_THING), one slot byte, one formatted string `"{name} : {message}"`.

**Rationale:** Verified against Java reference implementation (`getPlayerShopChat`). The CHAT_THING byte is a secondary operation code required by the client.

### 4. Shop Scoping to World/Channel/Map/Instance

**Decision:** All four coordinates are required. Shops are NOT cross-channel.

**Rationale:** User confirmed hired merchants are channel-specific. The existing model's lack of world/channel/instance is a bug. Each field instance is uniquely identified by the (world, channel, map, instance) tuple.

### 5. Owner vs Visitor Detection for EXIT Mode

**Decision:** TBD — need to determine how channel knows if the exiting character owns the shop (EXIT → CLOSE_SHOP) vs is visiting (EXIT → EXIT_SHOP).

**Options:**
- Track character's current room + role in channel-local mini-room state
- Query atlas-merchant REST to check ownership
- Include ownership info in the VISITOR_ENTERED/SHOP_OPENED event

---

## Dependencies

### Internal Libraries
- `github.com/Chronicle20/atlas-kafka` — producer/consumer patterns
- `github.com/Chronicle20/atlas-rest` — REST client (`requests.SliceProvider`, `requests.GetRequest`)
- `github.com/Chronicle20/atlas-packet` — packet encoding (modified)
- `github.com/Chronicle20/atlas-socket` — socket session management

### External Services
- `atlas-merchant` — command consumer + event producer + REST API
- `atlas-channel` — socket server + command producer + event consumer

### Kafka Topics
| Topic | Direction | Purpose |
|-------|-----------|---------|
| `COMMAND_TOPIC_MERCHANT` | channel → merchant | All 11 shop commands |
| `EVENT_TOPIC_MERCHANT_STATUS` | merchant → channel | Shop lifecycle + visitor + message events |
| `EVENT_TOPIC_MERCHANT_LISTING` | merchant → channel | Purchase events |

---

## Key Constraints

1. **All field coordinates required** — worldId, channelId, mapId, instanceId must be present on every shop and every query
2. **Fire-and-forget commands** — socket handlers must NOT block waiting for merchant response
3. **Event-driven UI updates** — all client-visible state changes come from consuming merchant events, not from handler return values
4. **Code resolution** — packet operation modes are resolved at runtime from options maps, not hardcoded byte values
5. **Curried function pattern** — all processors, handlers, and spawn functions follow the deeply-nested curried function pattern (`func(l) func(ctx) func(wp) func(...)`)
6. **Announce pattern** — socket writes use `session.Announce(l)(ctx)(wp)(WriterName)(BodyFunc)(session)`

---

## Command ↔ Client Action Mapping

### Shared Interaction Modes (context-dependent)

| Mode | Code | For Merchant | Command |
|------|------|-------------|---------|
| CREATE | 0x00 | + MerchantShopMiniRoomType | `PLACE_SHOP` |
| VISIT | 0x04 | + merchant room | `ENTER_SHOP` |
| CHAT | 0x06 | + merchant room | `SEND_MESSAGE` |
| EXIT | 0x0A | visitor in merchant room | `EXIT_SHOP` |
| EXIT | 0x0A | owner of merchant room | `CLOSE_SHOP` |
| OPEN | 0x0B | + merchant context | `OPEN_SHOP` |

### Merchant-Specific Modes

| Mode | Code | Command |
|------|------|---------|
| MERCHANT_PUT_ITEM | 0x21 | `ADD_LISTING` |
| MERCHANT_BUY | 0x22 | `PURCHASE_BUNDLE` |
| MERCHANT_REMOVE_ITEM | 0x26 | `REMOVE_LISTING` |
| MERCHANT_MAINTENANCE_OFF | 0x27 | `EXIT_MAINTENANCE` |
| MERCHANT_EXIT | 0x29 | `EXIT_SHOP` |

### Special Triggers

| Trigger | Command | Notes |
|---------|---------|-------|
| Owner enters own hired merchant | `ENTER_MAINTENANCE` | Via CASH_TRADE_OPEN (nProc=4, MerchantShopMiniRoomType) |

---

## Event ↔ Client Packet Mapping

| Merchant Event | Client Packet | Recipients |
|---------------|---------------|------------|
| `SHOP_OPENED` | MiniRoom spawn | All characters on map |
| `SHOP_CLOSED` | MiniRoom despawn + exit | All characters on map + room viewers |
| `VISITOR_ENTERED` | InteractionEnter | Room viewers |
| `VISITOR_EXITED` | Exit packet | Room viewers |
| `VISITOR_EJECTED` | Exit + error | Ejected player + room viewers |
| `MAINTENANCE_ENTERED` | Exit for visitors | Room viewers |
| `MAINTENANCE_EXITED` | Room unlock | Owner |
| `LISTING_PURCHASED` | Room refresh | Room viewers |
| `CAPACITY_FULL` | InteractionEnterResultError | Joining player |
| `PURCHASE_FAILED` | Error packet | Buyer |
| `FREDERICK_NOTIFICATION` | FreeFormNotice | Shop owner (if online) |
| `MESSAGE_SENT` (new) | InteractionChat | Room viewers |
