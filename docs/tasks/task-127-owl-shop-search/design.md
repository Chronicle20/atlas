# Owl of Minerva (Shop Scanner) — Design

Task: task-127-owl-shop-search
Status: Approved PRD → this document records the architecture, the IDA-verified protocol, faithful-behavior rulings on the PRD's open questions, alternatives considered, and the decisions taken.

Evidence sources used throughout: live IDA decompiles of GMS v83 (`MapleStory_dump.exe.i64`, port 13342) and GMS v95 (`GMS_v95.0_U_DEVM.exe.i64`, port 13341), the opcode CSVs/registry under `docs/packets/`, and the local Cosmic (HeavenMS-lineage) checkout at `~/source/Cosmic` used strictly as corroboration — every wire fact below is anchored to a client decompile address.

---

## 1. Protocol — verified packet inventory (answers PRD §9 Q1, Q4)

The feature spans **six** packet surfaces, one more than the PRD listed: the clientbound **SHOP_LINK_RESULT** (`CWvsContext::OnShopLinkResult`) carries the warp/enter outcome and was missing from the PRD. It already has registry rows (`docs/packets/registry/gms_v83.yaml:329`, `gms_v95.yaml:354`) and CSV row 95.

### 1.1 Opcode matrix

| Op | Dir | v83 | v84 | v87 | v92 | v95 | jms185 | Source |
|---|---|---|---|---|---|---|---|---|
| OWL_ACTION | sb | 0x42 † | 0x42 | 0x45 | 0x49 | 0x48 † | 0x3A | † = IDA (v83 `CUIShopScanner::OnCreate` 0x8a0e9a; v95 0x848b90); rest CSV/registry |
| OWL_WARP | sb | 0x43 † | 0x43 | 0x46 | 0x4A | 0x49 † | 0x3B | † = IDA (v83 `sub_8A4423`; v95 `CUIShopScanResult::OnButtonClicked` 0x848e80) |
| USE_SHOP_SCANNER_ITEM | sb | **0x53 †** | 0x53 (assumed) | unknown | unknown | 0x5A † | unknown | † = IDA (v83 0xa0a25e; v95 0x9e10e0). See §1.6 registry conflict |
| SHOP_SCANNER_RESULT | cb | 0x46 † | 0x48 | 0x48 | 0x4A | 0x49 † | 0x40 | † = IDA (dispatcher case 70 @0xa07b9d v83; case 73 @0x9e59df v95). v84 from task-100 reshift note (`gms_v84.yaml:392`) |
| SHOP_LINK_RESULT | cb | 0x47 † | 0x49 | 0x49 | 0x4B | 0x4A † | 0x41 | † = IDA (dispatcher case 71 v83; case 74 v95). v84 from `gms_v84.yaml:400` ("0x49 ShopLink") |
| USE_CASH_ITEM (523 arm) | sb | 0x4F | (existing) | (existing) | (existing) | (existing) | (existing) | already routed; only the 29-arm tail is new |

### 1.2 The use flow is two-phase (this shapes everything)

IDA shows the client does **not** send the use-packet when the owl is used. Both use routes build the packet and *stash* it:

- Cash route: `CWvsContext::SendConsumeCashItemUseRequest` case 29 (v83 jumptable case at 0xa0cd0b) builds `[opcode 0x4F][short pos][int itemId]` then calls `RunShopScanner` — **no extra fields encoded in the arm**.
- Dedicated route: `CWvsContext::SendShopScannerItemUseRequest` (v83 0xa0a25e, v95 0x9e10e0) — gated on `itemId/10000 == 231` (`is_shopscanner_item`, v95 0x4ff5c0) — builds `[opcode][short nPOS][int nItemID]` then calls `RunShopScanner`. This is the **USE-inventory owl (231xxxx family, e.g. 2310000)** double-clicked from the inventory (`CDraggableItem::OnDoubleClicked`), NOT the cash owl.

`RunShopScanner` (v83 0xa0a2dc, v95 0x9deb50) validates the current map is a Free Market map (`910000000 ≤ mapId ≤ 910000022` in both versions), opens `CUIShopScanner`, and stores the packet via `SetOutPacket`. Opening the UI fires **OWL_ACTION** (`OnCreate`: `[byte 5]`). When the player finally picks an item and confirms, `CUIShopScanner::SendScanPacket` (v95 0x83f6b0; v83 `sub_8A2407`) **appends** `[int nItemID][byte bDescendingOrder][int updateTime]` to the stored packet and sends it.

So the server receives, per route:

- `USE_CASH_ITEM` (existing op, itemType 523): `[short pos][int itemId=5230000]` + `[int searchItemId][byte descending][int updateTime]` (v83-style prefix; the existing `cashsb.ItemUse` codec's `updateTimeFirst` gate for GMS≥95 applies to the prefix as today).
- `USE_SHOP_SCANNER_ITEM`: `[short pos][int itemId=231xxxx][int searchItemId][byte descending][int updateTime]` — **no** leading updateTime even on v95 (verified 0x9e10e0).

### 1.3 OWL_ACTION (serverbound)

`[byte mode]`. A full scan of every `COutPacket(0x42)` (v83) / `COutPacket(0x48)` (v95) construction site found **exactly one sender**: `CUIShopScanner::OnCreate` with mode `5`. There are no other request modes (answers PRD Q4 — it is purely "scanner UI opened → give me the most-searched list"; re-searching from the list flows through the stored use-packet, §1.2). Handler: validate mode==5 (log-warn otherwise), respond with SHOP_SCANNER_RESULT hot-list mode.

### 1.4 SHOP_SCANNER_RESULT (clientbound)

Single fname, internally mode-switched on the first byte (v83 0xa28c29, v95 0xa076c0 — identical structure; v95's typed `ITEMDATA` struct names every field):

**Mode 6 — search result:**
```
[byte 6]
[int  nNpcShopPrice]   // >0 → client inserts a synthetic first row (NPC 2084001, "sold in regular stores"); we always send 0
[int  nItemID]         // the searched item
[int  nCount]          // record count
nCount × {
  [str  sCharacterName]  // shop owner name
  [int  dwFieldID]       // shop map id (client shows mapId % 100 as "Room %d")
  [str  sTitle]          // shop title
  [int  nNumber]         // bundles available   (Cosmic: item.getBundles())
  [int  nSet]            // quantity per bundle (Cosmic: item quantity)
  [int  nPrice]          // price per bundle
  [int  dwMiniRoomSN]    // opaque echo field → we send owner characterId (Cosmic parity, §4.4)
  [byte nChannelID]      // 0-based channel index (Cosmic writes channel-1; client compares to CWvsContext::m_nChannelID)
  [byte nTI]             // inventory type of the listed item; nTI==1 (equip) → full GW_ItemSlotBase follows
  [if nTI==1: GW_ItemSlotBase]
}
```
`nCount==0 && nNpcShopPrice==0` → client shows "Unable to find the item you have entered" (SP_3637) — the faithful no-results UX costs us nothing extra.

**Mode 7 — most-searched (hot) list:**
```
[byte 7][byte count][count × int itemId]
```

The client caps and labels the result list at **200** entries with an ascending/descending toggle (SP_3630/3631 "Show 200 results in ascending/descending order", `CUIShopScannerSearchResult::OnButtonClicked` v83 0x89d82f). Cosmic's writer (`PacketCreator.owlOfMinerva`, `~/source/Cosmic/.../PacketCreator.java:5077`) matches this byte layout field-for-field.

`nNumber`/`nSet` column semantics are corroborated by Cosmic and the v83 column tooltips ("item(s) per bundle" / "price per bundle" / "total"); the fixture phase pins them byte-exactly anyway.

### 1.5 OWL_WARP (serverbound) and SHOP_LINK_RESULT (clientbound)

OWL_WARP: `[int dwMiniRoomSN][int dwFieldID]` — the client echoes the two ints from the clicked record verbatim (v83 `sub_8A4423`; v95 0x848e80). Rows with `+44 != 0` (the synthetic NPC row) never send.

SHOP_LINK_RESULT: `[byte code]`, identical code set in v83 (0x8a4e7a) and v95 (0x847d60):

| code | client string | our trigger |
|---|---|---|
| 0 | (success — closes result window) | never sent (§4.5) |
| 1 | "The room is already closed" | shop not found / wrong map / closed / cross-channel defensive case |
| 2 | "You can't enter the room due to full capacity" | visitor cap reached at arrival |
| 3 | "Other requests are being fulfilled this minute" | shop exists but the searched listing is gone/sold out (Cosmic parity) |
| 4 | "You can't do it while you're dead" | character HP 0 |
| 7 | "You are not allowed to trade other items at this point" | unused |
| 17 (0x11) | "You may not enter this store" | visiting your own shop; future ban/deny cases |
| 18 (0x12) | "The owner of the store is currently undergoing store maintenance…" | shop in Maintenance state |
| 23 (0x17) | "This can only be used inside the Free Market" | FM-scope violation |
| other | "This character is unable to do it" | fallback |

### 1.6 Registry conflict: v83 serverbound 0x53 (stop-and-flag, resolved by IDA)

The gms_v83/gms_v84 registries assign serverbound `0x53` to `USE_SKILL_RESET_BOOK` (csv-import provenance). A full construction-site scan of the v83 binary shows the **only** `COutPacket(0x53)` sender is `CWvsContext::SendShopScannerItemUseRequest`, and no skill-reset-book sender exists in v83 at all (`func_query` for `SkillReset|resetbook` → zero hits). The CSV also (wrongly) records USE_SHOP_SCANNER_ITEM as `0x000` for v83.

Decision: this task corrects the registry — v83 `USE_SHOP_SCANNER_ITEM = 0x53` (with the IDA note), v83/v84 `USE_SKILL_RESET_BOOK` marked not-present (no sender in binary). **Coordinate with in-flight task-125 (skill-mastery-books)** before landing the row removal; the correction is IDA-proven either way. v84 gets 0x53 by the established v84-serverbound≡v83 rule, flagged unverified (no v84 IDB loaded). v87/v92/jms opcodes for this op are unknown (CSV 0x000, no IDBs) — the op stays unrouted there; the cash 523 arm still covers cash-owl usability on those versions.

---

## 2. Faithful-behavior rulings (answers PRD §9 Q2, Q3, Q5)

**Q2 — Cross-channel warp: NOT supported. Same-channel only.** The client only renders the row's warp link when `record.nChannelID == CWvsContext::m_nChannelID` (v83 `sub_8A4C6C`; v95 `LoadCurPageItemList` 0x847ac0). Other-channel rows are display-only (the player reads the channel column and travels manually). Cosmic corroborates. Consequence: **no channel-change flow is needed at all** — PRD FR-16's fallback branch is the verified behavior. This removes the riskiest piece of the PRD.

**Q3 — Consumption: 1 owl consumed per search that returns ≥1 listing; nothing consumed on empty search.** The client warns "…the Owl of Minerva will disappear" (SP_3629) before sending, but Cosmic — the only server-side evidence available — consumes only when results are non-empty (`UseCashItemHandler` itemType 523: `if (!hmsAvailable.isEmpty()) remove(...)`). This matches the PRD's working assumption; adopt it. The empty-search path sends the no-results packet and consumes nothing. (Client-side proof of the server rule is impossible by nature; the in-game acceptance pass on v83 validates it live.)

**FR-1 override — Free-Market-only IS the faithful rule.** `RunShopScanner` hard-blocks the scanner outside maps 910000000–910000022 in both v83 and v95 ("This can only be used inside the Free Market", SP_3641). The owner's "usable anywhere" preference is unreachable client-side — the client never opens the UI or sends the packet elsewhere. Per FR-1's own escape hatch, the faithful rule wins: the server also validates FM scope on every owl op (defense against packet injection) and uses SHOP_LINK code 23 where applicable. Warp targets must be FM maps too.

**Q5a — Maintenance-state shops stay in results.** Kept (current search behavior), now with state included per row so nothing is hidden; warping to one yields the faithful code 18 ("undergoing store maintenance"). This is coherent with the client's dedicated maintenance message existing at all.

**Q5b — Top-10 scope: per-tenant + per-world.** Matches search scoping and Cosmic (`c.getWorldServer().addOwlItemSearch/getOwlSearchedItems`). Counts increment on **every executed search** (Cosmic increments before searching, result-independent) — PRD FR-10 default confirmed. When fewer than 10 items have ever been searched, send what exists (short list; count byte). No hardcoded filler (Cosmic's <5 fallback list is invented data — skipped).

**Result ordering.** Server returns price-ascending by default and honors the request's `bDescendingOrder` flag (faithful: the client offers both, SP_3630/3631). Cap 200 server-side, truncating the far end of the chosen ordering (FR-8 refined: "truncate most expensive" applies to ascending; descending truncates cheapest).

---

## 3. Architecture overview

```
                     ┌──────────────────────────── atlas-channel ────────────────────────────┐
USE_CASH_ITEM 523 ──►│ cash-item handler arm ┐                                               │
USE_SHOP_SCANNER ───►│ scanner-use handler   ├─► shopscanner processor:                      │
                     │                       │    1. FM-map + item-in-slot validation        │
                     │                       │    2. GET merchant search (world+item+order)  │──REST──► atlas-merchant
                     │                       │    3. resolve owner names (character REST)    │──REST──► atlas-character
                     │                       │    4. emit RECORD_ITEM_SEARCH (fire&forget)   │──Kafka─► atlas-merchant
                     │                       │    5. write SHOP_SCANNER_RESULT (mode 6)      │
                     │                       │    6. ≥1 result → RequestItemConsume          │──Kafka─► atlas-consumables
                     │                       │    7. registry: lastOwlSearch[charId]         │
OWL_ACTION ─────────►│ owl-action handler ───┼─► GET top-10 ──► write mode 7                 │──REST──► atlas-merchant
OWL_WARP ───────────►│ owl-warp handler ─────┼─► validate shop via merchant REST             │──REST──► atlas-merchant
                     │                       │    ok → pendingShopEntry[charId]=shopId       │
                     │                       │         + portal.Warp (same channel)          │──Kafka─► atlas-portals
                     │                       │    err → write SHOP_LINK_RESULT code          │
map-changed event ──►│ character consumer: pending entry matches map → EnterShop             │──Kafka─► atlas-merchant
CapacityFull event ─►│ merchant consumer: pending owl entry → SHOP_LINK_RESULT code 2        │
VisitorEntered ─────►│ (existing mini-room open flow, unchanged)                             │
                     └───────────────────────────────────────────────────────────────────────┘
```

No new services. atlas-merchant stays free of atlas-character coupling. All new state in atlas-channel is per-pod in-memory (correct, because both packets of each two-step flow arrive on the same socket/pod).

---

## 4. Component design

### 4.1 atlas-channel — handlers and processor

New package `services/atlas-channel/atlas.com/channel/shopscanner/` (processor + registry), plus:

- `socket/handler/character_cash_item_use.go`: name the constant (`CashSlotItemTypeStoreSearch = CashSlotItemType(29)`) and add the arm. Decode tail via new `cashsb.ItemUseStoreSearch` (`[int searchItemId][byte descending][int updateTime]`), then call the shared processor. Mirrors the field-effect arm's shape.
- `socket/handler/shop_scanner_item_use.go` (`ShopScannerItemUseHandle`): decodes the dedicated-route body (`[short pos][int itemId][int searchItemId][byte descending][int updateTime]`), validates the item is a 231-family USE-inventory item present at that slot (character processor `GetItemInSlot`, `inventory.TypeValueUse`), same shared processor. Consumption uses the USE-inventory source slot.
- `socket/handler/owl_action.go` (`OwlActionHandle`): `[byte mode]`, expect 5; fetch top-10, write hot-list.
- `socket/handler/owl_warp.go` (`OwlWarpHandle`): `[int ownerId][int mapId]`; validation ladder in §4.5.

**Shared search flow** (`shopscanner.Processor.Search`):
1. Validate session's map is FM (`910000000–910000022`; add the FM range/`IsFreeMarketRoom` helper to `libs/atlas-constants/map` if absent — DOM-21 check first) and the owl is in the claimed slot. Violations log-warn and drop (client can't reach them honestly).
2. `GET /merchants/search/listings?itemId=&worldId=&order=` (extended endpoint, §4.3) — up to 200 rows, shop state/type/channel/owner id + listing fields + item snapshot for equips.
3. Resolve distinct owner ids → names via the existing `character.Processor.GetById` (atlas-channel already has this client; dedupe per request). Bounded by the 200-row cap; realistically tens of shops per world. If a name lookup fails, fall back to empty string for that row rather than failing the search.
4. Emit `RECORD_ITEM_SEARCH` command (new, `COMMAND_TOPIC_MERCHANT`) — always, result-independent; failures log-only (the increment must never block or fail the search — PRD §5).
5. Write SHOP_SCANNER_RESULT mode 6 (`nNpcShopPrice=0`, records from the search rows; channel encoded as `channel.Id - 1` matching `server_list_entry.go:76`).
6. If ≥1 row: `consumable.RequestItemConsume(field, characterId, itemId, source, updateTime)` — the same mechanism the pet-consumable arm uses; works for both cash-slot and USE-slot sources. A single destroy needs no multi-step saga (nothing to order after it); the failure path simply never emits it.
7. Record `lastOwlSearch[characterId] = {itemId, ts}` in the scanner registry.

**Registries** (`shopscanner/registry.go`, singleton + `sync.RWMutex` + tenant-scoped keys, the established pattern): `lastOwlSearch` (charId → itemId) and `pendingShopEntry` (charId → shopId + expected mapId). Entries overwrite on reuse and are cleared on session destroy (hook the existing session-destroyed path) and on consumed/mismatched arrival.

### 4.2 atlas-channel — warp + auto-enter (FR-15..18)

OWL_WARP handler ladder (each failure writes SHOP_LINK_RESULT with the table-driven code from §1.5):
1. `lastOwlSearch` present, else fallback code (stale/no search).
2. `ownerId == characterId` → code 17 (can't visit own shop).
3. Character alive (HP>0 via character processor) → else code 4.
4. Shop lookup by owner: existing merchant REST `characters/{ownerId}/merchants`. Missing → code 1.
5. Shop world == session world, shop map == echoed mapId, mapId is FM → else code 1 (echo tampering) / 23 (non-FM).
6. Shop channel == session channel → else code 1 (client never sends this honestly).
7. State: Open → continue; Maintenance → 18; anything else → 1.
8. Listing for `lastOwlSearch.itemId` still present with bundles remaining → else code 3.
9. Success: set `pendingShopEntry`, then `portal.NewProcessor(l,ctx).Warp(s.Field(), characterId, targetMapId)` — the exact `map_change.go:63` mechanism (FR-18). No success packet (Cosmic parity; the client tears the scanner windows down on field change via `ClearFieldUI` — verified xref).

Auto-enter (FR-17): in the existing `handleStatusEventMapChanged` consumer (`kafka/consumer/character/consumer.go:223`), after the warp write, check `pendingShopEntry`; on map match, `merchant.NewProcessor(l,ctx).EnterShop(characterId, shopId)` (existing command) and consume the entry; on mismatch, drop the entry. Full-at-arrival: merchant's `EnterShop` already emits `StatusEventCapacityFull` (`shop/processor.go:785`), and atlas-channel already consumes it (`kafka/consumer/merchant/consumer.go:283`) — add an owl-aware branch that writes SHOP_LINK code 2 (keeping whatever the existing behavior is for non-owl entries). Visitor entry itself rides the existing `VisitorEntered` mini-room flow untouched. Capacity is not pre-checked at OWL_WARP time — warp-then-code-2 is exactly Cosmic's observable sequence.

### 4.3 atlas-merchant — search enrichment + counts

- `searchListingsByItemId` (`shop/provider.go:95`): add `worldId` filter, `order` direction, `LIMIT 200`, and select `shops.character_id, shops.shop_type, shops.state, shops.channel_id` (channel already selected; the rest are columns on `shops` — no schema change). `listings.item_id` is already indexed (`listing/entity.go:16`) — NFR satisfied.
- `ListingSearchResult` (`shop/processor.go:90`): add `ShopOwnerId uint32`, `ShopType ShopType`, `State State` (field names matching the sibling struct at `processor.go:85`). REST model exposes them plus the listing's `ItemSnapshot` (needed to encode `GW_ItemSlotBase` for equip rows).
- REST: extend `GET /merchants/search/listings` with `worldId` (required) and `order` (asc default) query params — backward compatible.
- **New**: `searchcount` sub-package — entity per PRD §6 (`listing_search_counts`: uuid surrogate PK, unique `(tenant_id, world_id, item_id)`, `count bigint`, `updated_at`; FR-12 tenant-safe-PK pattern), atomic upsert `INSERT … ON CONFLICT (tenant_id,world_id,item_id) DO UPDATE SET count = listing_search_counts.count + 1`, `GetTop(worldId, 10)` provider. Register `Migration` in `main.go:65`.
- **New Kafka command** `RECORD_ITEM_SEARCH` on `COMMAND_TOPIC_MERCHANT` (`{worldId, itemId, characterId}`) → consumer calls the upsert. Command-not-REST keeps the search GET pure and decouples increment failures from the search path (PRD §5 requirement).
- **New REST** `GET /worlds/{worldId}/shop-searches/top` → JSON:API resource `[{itemId, count}]` (top 10), on the existing `/worlds/{worldId}` subrouter (`shop/resource.go:37`).

### 4.4 The `dwMiniRoomSN` echo: owner characterId

The client treats the two OWL_WARP ints as opaque echoes of whatever the server sent. Options considered: (a) a numeric shop SN column on shops, (b) a transient SN→shopId registry, (c) owner characterId. **Chosen: (c)** — Cosmic-compatible, needs zero schema, is stable across the search→warp gap, and resolves to the shop via the existing by-owner endpoint. A character owns at most one shop at a time, and the warp handler re-validates map/listing/state anyway, so ambiguity cannot produce a wrong warp — at worst the faithful error.

### 4.5 libs/atlas-packet — codecs

New codecs in `libs/atlas-packet/merchant/` (the player-shop domain package; `cash/` is Cash Shop and wrong here):
- `serverbound/owl_action.go`, `serverbound/owl_warp.go`, `serverbound/shop_scanner_item_use.go`
- `clientbound/shop_scanner_result.go` — two body constructors (result / hot list), shared header; per-record conditional asset encode for `nTI==1` reusing the existing `model.Asset` encoding the interaction (trade) and storage writers use, built from the listing's `ItemSnapshot`.
- `clientbound/shop_link_result.go` — single code byte.
- `cash/serverbound/item_use_store_search.go` — the 523-arm tail, `NewItemUseStoreSearch(...)` mirroring the existing arm-tail codecs.

Writers in atlas-channel: `ShopScannerResult` and `ShopLinkResult`, modeled exactly on `socket/writer/world_message.go` — every mode byte and SHOP_LINK code resolved via `atlas_packet.ResolveCode(l, options, "operations", key)` from the tenant template, never hard-coded (dispatcher-config-drive-all-modes rule; the values are version-stable 6/7 and 0–23 today, config-driven regardless).

---

## 5. Version coverage & verification plan (packet-audit discipline)

Reality check (from the audits/exports): no owl fname exists in any checked-in IDA export; live IDBs exist only for v83 and v95 today.

- **v83 + v95**: full tier-1 treatment per `VERIFYING_A_PACKET.md` — harvest the five fnames per version into `docs/packets/ida-exports/{gms_v83,gms_v95}.json` by surgical splice (never overwrite), byte-fixture tests with `packet-audit:verify` markers (template: `buddy/clientbound/list_update_test.go` for the list codec), evidence pins, serverbound audit reports (add the fname cases to `candidatesFromFName` in the tool's `cmd/run.go` — new ops require it), matrix regen. Registry fix from §1.6 lands here.
- **v84**: seed-template routing with the reshift-verified opcodes (registry notes from task-100); cells remain unverified-bannered (no v84 IDB currently loaded). Serverbound owl ops = v83 values per the established v84 rule.
- **v87, v92, jms**: seed-template routing from CSV opcodes (v92 template exists; v92 has no audit column/export at all). Cells stay ❌/n-a until IDBs exist — same accepted state as the other v87/jms gaps. `USE_SHOP_SCANNER_ITEM` stays unrouted where its opcode is unknown (§1.6); the cash 523 arm keeps the cash owl functional everywhere.
- Existing-op regression: the 523 arm changes no `ItemUse` prefix bytes; no re-verification of USE_CASH_ITEM cells needed.

Acceptance criterion 6 ("every IDB-backed version") therefore concretely means **gms_v83 and gms_v95** fixtures now, with the others promoted when their IDBs return (consistent with how prior packet tasks handled v87/jms).

## 6. Seed templates & live tenants (FR-19/20)

Per version (gms_83, 84, 87, 92, 95, jms_185): three handler entries (OWL_ACTION, OWL_WARP, USE_SHOP_SCANNER_ITEM where known), **each with `LoggedInValidator`** (validator-less entries are silently dropped — known trap), and two writer entries (`ShopScannerResult` with `options.operations {RESULT:6, HOT_LIST:7}`, `ShopLinkResult` with the code table `{SUCCESS:0, CLOSED:1, FULL:2, BUSY:3, DEAD:4, NO_TRADE:7, DENIED:17, MAINTENANCE:18, FM_ONLY:23}`), following the `WorldMessage` entry shape (gms_83 template :1637).

Deployment notes must include: patch live tenant configs with the same entries + restart atlas-channel (seed templates only apply at tenant creation — known trap), and the standard bake/rollout checklist.

## 7. Alternatives considered

| Decision | Chosen | Rejected | Why |
|---|---|---|---|
| Search transport | Synchronous REST GET from atlas-channel | Kafka command + result event round-trip | Search is a read on an existing indexed query; REST matches every other channel read; an async round-trip adds correlation state for zero resilience gain on a user-blocking UI action |
| Count increment | Kafka command, fire-and-forget | (a) side-effecting the search GET (b) synchronous REST POST | GET stays pure; increment failure can't affect search latency/outcome; consumer upsert is O(1) |
| Owner names | atlas-channel resolves via existing atlas-character client | (a) denormalized owner_name on shops (b) merchant→character REST client | No new coupling or schema; bounded by the 200 cap and per-request dedup; merchant stays character-free (it has no character client today) |
| Warp→enter sequencing | Pending-entry registry consumed by the existing map-changed consumer | (a) emit EnterShop immediately after warp command (b) new saga type | (a) races the field transition — mini-room packets could precede SetField; (b) is heavyweight for a 2-step flow whose second step already has a natural completion signal; Cosmic's own sequence is warp-then-visit |
| Shop echo id | Owner characterId | numeric shop SN column / transient SN registry | §4.4 |
| Consume mechanism | `RequestItemConsume` (consumable command) | DestroyAsset saga | Single step, nothing to order after it; saga machinery buys nothing here (task-126's saga precedent was multi-service multi-step). Conditional emission already gives the "failure never consumes" guarantee |

## 8. Testing

- Byte fixtures: every codec × {gms_v83, gms_v95} round-trips against IDA-derived bytes (markers + evidence per §5), including an equip-row fixture exercising the `nTI==1` asset block and an empty-result fixture.
- atlas-merchant: provider tests for world scoping, ordering, cap, state/owner fields; upsert concurrency test (parallel increments sum correctly); top-10 ordering; tenant isolation on the counts table. Builder pattern only — no `*_testhelpers.go`.
- atlas-channel: handler tests for the validation ladder (each SHOP_LINK code reachable), consume-only-on-results, registry lifecycle (overwrite/session-destroy/arrival-mismatch), hot-list short-count.
- Verification gates: `go test -race`, `go vet`, `go build` per changed module; `docker buildx bake` for atlas-channel, atlas-merchant, atlas-configurations (+ any lib-touched service set per services.json); `tools/redis-key-guard.sh`; `packet-audit matrix --check` with no new conflicts; live v83 in-game pass per PRD acceptance (search, empty search, hot list, warp, full shop, maintenance shop, sold-out race).

## 9. Risks / notes

- **nNumber vs nSet transposition** would show swapped quantity columns in the UI; the v83 fixture phase pins it from the row-draw decompile before any tenant sees it (Cosmic + tooltips already agree).
- **231-family data**: the USE-inventory owl (2310000) must exist in v83 WZ data for the dedicated route to be player-reachable; if absent from item data, that route is still implemented + fixture-verified (packet-level), and the cash owl is the player path. Verify against local WZ during implementation (grounding rule — not from memory).
- **task-125 overlap**: the USE_SKILL_RESET_BOOK registry correction (§1.6) touches rows that task may be reading. Flag on landing.
- **Cosmic** was used only to corroborate IDA-derived layouts and to choose defaults where server policy is inherently unobservable client-side (consumption trigger, echo semantics); no byte was taken from Cosmic alone.
