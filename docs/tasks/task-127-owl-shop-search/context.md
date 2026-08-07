# task-127-owl-shop-search — Context

Companion to `plan.md`. Key files, locked decisions, dependencies, and the deviations/risks an implementer or reviewer needs without re-reading the whole design.

## What this task builds

Owl of Minerva (shop scanner): FM-scoped world-wide listing search over player shops / hired merchants, persisted most-searched top-10, same-channel warp-to-shop with auto-enter as visitor, across six packet surfaces (OWL_ACTION, OWL_WARP, USE_SHOP_SCANNER_ITEM, USE_CASH_ITEM 523 arm, SHOP_SCANNER_RESULT, SHOP_LINK_RESULT), byte-fixture verified on gms_v83 + gms_v95.

## Key files (existing, load-bearing)

### atlas-channel (`services/atlas-channel/atlas.com/channel/`)
- `socket/handler/character_cash_item_use.go` — cash-use dispatch; 523→CashSlotItemType(29) mapping at ~310; arms are early-return blocks; handler factory currently discards `wp` (`_ writer.Producer`) — Task 11 changes that.
- `socket/handler/character_interaction.go:117-129` — the mini-room Visit arm: `GetByCharacterId(owner)` → `shops[0].Id()` → `EnterShop` — the exact analog for owl auto-enter.
- `socket/writer/world_message.go` — the config-resolved-mode writer template (`atlas_packet.ResolveCode(l, options, "operations", key)`).
- `session/processor.go` — `Announce(l)(ctx)(wp)(writerName)(encode)(s)`; `IfPresentByCharacterId(ch)(charId, op)`; `Destroy` at :330 (no generic per-character registry fan-out — cleanup is explicit).
- `socket/init.go:46` — `socket.SetDestroyer(sp.DestroyByIdWithSpan)`; Task 12 wraps it (this file may import both session and shopscanner; session must NOT import shopscanner).
- `kafka/consumer/character/consumer.go:223-272` — `handleStatusEventMapChanged`/`warpCharacter`: SetField → SetFieldWriter announce → SpawnForSelf; Task 12 appends the pending-entry check here.
- `kafka/consumer/merchant/consumer.go` — `handleCapacityFullEvent` :283 (currently announces mini-room full error); `handleVisitorEvent` VisitorEntered case :189.
- `socket/handler/map_change.go:47-66` — HP==0 gate + `portal.NewProcessor(l, ctx).Warp(s.Field(), s.CharacterId(), targetMapId)` — the warp mechanism OWL_WARP reuses (emits WARP on `COMMAND_TOPIC_PORTAL`; the MAP_CHANGED event closes the loop).
- `merchant/` — REST client (`requestByCharacterId` = `characters/%d/merchants`), `EnterShop(characterId, shopId uuid.UUID)` → `ENTER_SHOP` on `COMMAND_TOPIC_MERCHANT`. No search-listings client exists pre-task.
- `consumable/processor.go:28` — `RequestItemConsume(f field.Model, characterId character.Id, itemId item.Id, source slot.Position, updateTime uint32)` (quantity fixed at 1; updateTime log-only).
- `character/processor.go` — `GetById()(id)` (REST, has `Name()`, `Hp()`); `GetItemInSlot(charId, invType, slot)` returns `model.Provider[asset.Model]` (call with `()`).
- `account/registry.go` — the singleton registry pattern the shopscanner registry copies.
- `main.go` — `produceHandlers()` ~:867 (handler-name → func), `produceWriters()` ~:608 (writer-name list), aliases `merchantsb`/`merchantcb` already imported (:99-100).

### atlas-merchant (`services/atlas-merchant/atlas.com/merchant/`)
- `shop/provider.go:87-125` — `searchListingsByItemId` + `listingSearchRow`. **Uses `db.Table()` which bypasses the automatic tenant callback** (`hasTenantColumn` needs `Statement.Schema`) — the pre-task query has NO tenant filter; Task 4 adds explicit `tenant_id` predicates along with world/order/limit.
- `shop/processor.go` — `ListingSearchResult` :90, `EnterShop` + `StatusEventCapacityFullProvider` :763-801, `MaxVisitors=3`.
- `shop/resource.go` — `handleSearchListings` :154; `wr` = `/worlds/{worldId}` subrouter :37.
- `shop/state.go` — `ShopType{CharacterShop=1, HiredMerchant=2}`, `State{Draft=1, Open=2, Maintenance=3, Closed=4}`.
- `listing/entity.go` — `item_id` already indexed; `ItemSnapshot asset.AssetData` (jsonb, `kafka/message/asset/kafka.go:10-42`).
- `kafka/consumer/merchant/consumer.go` — `InitHandlers` :29 (add RECORD_ITEM_SEARCH handler line; `InitConsumers` already subscribes `COMMAND_TOPIC_MERCHANT`).
- `main.go:65` — migration list (add `searchcount.Migration`).
- `shop/mock/processor.go` — function-field mock; `SearchListingsByItemIdFunc` must follow the interface change.

### libs
- `libs/atlas-packet/merchant/` — `operation_body.go` (`WithResolvedCode` factories) + one `operation.go` per direction; new codecs go in new files in the same dirs.
- `libs/atlas-packet/cash/serverbound/item_use.go` — prefix codec with the GMS≥95 leading-updateTime gate; `item_use_pet_consumable.go` — arm-tail shape.
- `libs/atlas-packet/model/asset.go` — `NewAsset(zeroPosition, slot, templateId, expiration)`; `zeroPosition=true` skips the slot prefix (`encodeSlot` :330) = the slotless GW_ItemSlotBase the scanner rows need; `Decode` (:372) self-dispatches on the type byte and never reads a slot, so it is symmetric with zeroPosition encodes. `SetEquipmentStats` takes 15 uint16 (:119); `SetEquipmentMeta(slots uint16, levelType, level byte, experience, hammersApplied uint32, flag uint16)` (:138).
- `libs/atlas-packet/test/` — `Variants` (v28/83/87/95/jms185/84/86), `CreateContext`, `RoundTrip(t, ctx, encode, decode, options)`. **RoundTrip has no fixture-bytes parameter** — byte pinning is done with explicit wire-shape tests (`in.Encode(l, ctx)(nil)` + byte assertions), per `fame/clientbound/response_test.go`.
- `libs/atlas-constants/map` — package `_map`; NO FM helper pre-task (Task 1 adds `IsFreeMarketRoom`).
- `libs/atlas-database/databasetest/testdb.go` — `NewInMemoryTenantDB(t, migrations...)`, `TenantContext(uuid)` (carries a full GMS/83 tenant → `tenant.MustFromContext` works).

### Config / tooling / docs
- Seed templates: `services/atlas-configurations/seed-data/templates/template_{gms_83,gms_84,gms_87,gms_92,gms_95,jms_185}_1.json` (handlers: `{opCode, validator, handler[, options]}`; writers: `{opCode, writer, options.operations}` — `WorldMessage` at gms_83 :1635 is the model).
- `tools/packet-audit/cmd/run.go` — `candidatesFromFName` switch (:278); `candidate{name, pkg, dir}`.
- Registries: `docs/packets/registry/gms_v83.yaml` (owl sb rows exist :2122-2130; SHOP_SCANNER/SHOP_LINK cb rows :324-333; USE_SKILL_RESET_BOOK :2217 to delete; **no USE_SHOP_SCANNER_ITEM row exists in v83/v84 — Task 14 adds it at 0x53**). v95 already has it at 0x5A (:2570).
- Evidence: `docs/packets/evidence/<version>/<packet dots>.yaml`, category `TIER1-FIXTURE`; playbook `docs/packets/audits/VERIFYING_A_PACKET.md` is authoritative for Task 15.
- IDA exports: `docs/packets/ida-exports/gms_v83.json` / `gms_v95.json` — splice-only, never overwrite.

## Locked decisions (from design.md; do not relitigate)

1. **Two-phase client flow**: the owl use-packet is stashed and sent only after item pick — both routes end with `[int searchItemId][byte descending][int updateTime]`; the dedicated route has NO leading updateTime even on v95.
2. **Same-channel warp only** (client renders warp link only for same-channel rows); no channel-change flow anywhere in this task.
3. **Consume 1 owl only on ≥1-result search**; increment the search counter on EVERY executed search (before/independent of results).
4. **FM-only is the faithful rule** (overrides PRD FR-1's "usable anywhere" via its own escape hatch); every owl op validates `_map.IsFreeMarketRoom` server-side; violations of unreachable-honestly paths log-warn and drop, reachable ones use SHOP_LINK code 23.
5. **`dwMiniRoomSN` = owner characterId**; warp handler resolves the shop via `characters/{ownerId}/merchants` and re-validates everything (a stale/ambiguous echo can only produce the faithful error, never a wrong warp).
6. **Maintenance shops stay in search results**; warping to one yields code 18.
7. **Top-10 scope = tenant+world**; short lists send actual count; no filler (Cosmic's <5 filler is invented data — skipped).
8. **Warp→enter sequencing** via pending-entry registry consumed by the map-changed consumer (immediate EnterShop would race SetField; a saga is overkill). Capacity is NOT pre-checked at warp time — warp-then-code-2 is Cosmic's observable sequence.
9. **Count increment via Kafka command** (`RECORD_ITEM_SEARCH`), fire-and-forget from the channel — the search GET stays pure and increment failure can't affect search latency.
10. **Owner names resolved in atlas-channel** via the existing character client (dedup per request, ≤200 rows); merchant stays character-free.
11. **Consume via `RequestItemConsume`**, not a DestroyAsset saga (single step, nothing to order after it).
12. **SHOP_LINK code mapping** (design §1.5): stale/no-search & generic validation failures → CLOSED(1); own shop → DENIED(17); dead → DEAD(4); listing gone/sold out → BUSY(3); maintenance → 18; FM violations → 23; FULL(2) only from the capacity-full event. SUCCESS(0) is never sent (client tears down scanner UI on field change).
13. **Registry corrections**: v83/v84 serverbound 0x53 = USE_SHOP_SCANNER_ITEM (IDA-proven, sole COutPacket(0x53) sender); USE_SKILL_RESET_BOOK rows deleted (no sender exists in v83). **Coordinate with in-flight task-125 before landing.**
14. **v84/v87/v92/jms stay seed-routed-but-unverified** (no IDBs); `USE_SHOP_SCANNER_ITEM` unrouted where its opcode is unknown (v87/v92/jms) — the cash 523 arm keeps the cash owl functional there. "Every IDB-backed version" = gms_v83 + gms_v95 today.

## Plan-level decisions (made during planning, consistent with design)

- **REST `worldId`/`order` params are optional** on `GET /merchants/search/listings` (absent ⇒ pre-task tenant-wide asc behavior) — reconciles design §4.3's "required" with its "backward compatible"; the owl path always passes both. Processor signature uses `ListingSearchCriteria{ItemId, WorldId *world.Id, Descending}`.
- **Task 4 fixes a latent tenant leak**: the search join never had a tenant filter (`.Table()` bypasses the callback). Explicit `listings.tenant_id`/`shops.tenant_id` predicates added and tested.
- **Pending-entry lifecycle**: kept after EnterShop emission; removed on VisitorEntered (success), CapacityFull (announce code 2), arrival-map mismatch, warp-command failure, session destroy.
- **Session-destroy cleanup** lives in a `socket/init.go` destroyer wrapper (shopscanner imports session ⇒ session cannot import shopscanner).
- **OwlAction's expected mode byte** is resolved from the handler entry's `options.operations {OPEN:5}` via `readerOptions` (the `isCharacterInteraction` precedent) — config-driven even though the client value is fixed.
- **Warp ladder is a pure function** (`shopscanner.EvaluateWarp(WarpCheck)`) so every SHOP_LINK code is table-tested without HTTP mocking; handlers gather, the function decides.
- **Listing-still-present check** at warp time re-uses the world-scoped search endpoint (match on shopId + BundlesRemaining>0) instead of relying on the by-owner response's listings relationship.
- **Byte-fixture style**: `pt.RoundTrip` (full-consumption + field assertions) + explicit wire-shape tests pinning bytes, since the test harness has no fixture-bytes parameter. Markers carry the design's IDA addresses; Task 15 re-derives from live IDA and pins evidence hashes.

## Dependencies / external requirements

- **Live IDA instances for Task 15 only**: v83 dump (expected port 13342), v95 U_DEVM (13341). Ports rotate — `list_instances` and match binary NAME first. Unavailable ⇒ Task 15 BLOCKED (report; never substitute evidence).
- **task-125 (skill-mastery-books)**: overlaps the USE_SKILL_RESET_BOOK registry rows Task 14 deletes. Flag in the PR; check `.worktrees/` for its worktree before landing.
- **No ingress change**: new/extended merchant endpoints are service-internal (channel → `MERCHANT` base URL), like the existing field-merchants world route.
- **No Dockerfile/go.work change**: no new lib; atlas-constants/atlas-packet already have COPY lines and workspace entries.
- **Live tenants need a config patch + atlas-channel restart** after deploy (seed templates apply at tenant creation only) — deployment.md (Task 16) is the checklist.

## Known deviations from design §8 testing (conscious, with rationale)

- **"consume-only-on-results" has no unit test**: the branch is `if len(listings) > 0` inside `Processor.Search`, whose collaborators (merchant REST, character REST, consumable Kafka) have no mocking seam in atlas-channel today. Covered by the ladder/conversion/codec tests around it plus the mandatory in-game acceptance pass (search-with-results consumes exactly 1; empty search consumes 0). If a reviewer wants it unit-tested, extract the conditional into a named pure helper first — do not bolt an HTTP-mock framework onto the service for one branch.
- **Handler-level socket tests** (decode→dispatch) are likewise not present anywhere in atlas-channel; the plan follows the existing convention (codec tests in the lib, pure-logic tests in the service).

## Verification gates (Task 16, all must pass before PR)

`go test -race` / `go vet` / `go build` in `libs/atlas-constants`, `libs/atlas-packet`, `atlas-merchant`, `atlas-channel`, `tools/packet-audit`; `tools/redis-key-guard.sh` (repo root, no GOWORK=off prefix); `docker buildx bake atlas-channel atlas-merchant atlas-configurations`; `packet-audit matrix --check` clean; `superpowers:requesting-code-review` (plan-adherence + backend-guidelines) before PR; in-game v83 acceptance pass per PRD §10.
