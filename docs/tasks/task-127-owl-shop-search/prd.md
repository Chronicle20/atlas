# Owl of Minerva (Shop Scanner) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-02
---

## 1. Overview

The Owl of Minerva (item 5230000, `Item.wz/Cash/0523.img.xml`, the only item in its category) is the shop-scanner cash item: a player uses it, picks (or types) an item, and the server searches all open player shops and hired-merchant shops for listings of that item. The client shows the results in the `CUIShopScanner` UI — owner, shop title, price, quantity, channel — and the player can click a result to warp directly to that shop and enter it as a visitor.

Atlas has zero owl support today. The classification constant exists (`ClassificationStoreSearch = Classification(523)` in `libs/atlas-constants/item/constants.go:89`) and atlas-channel's cash-item-use handler already maps 523 to `CashSlotItemType(29)` (`services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:310`) — but that arm then falls through to the warn-and-drop path, so using the owl is a silent no-op. All four packet operations are unimplemented across every version in the coverage matrix (`docs/packets/audits/STATUS.md`): serverbound `OWL_ACTION` ❌ (row 553) and `OWL_WARP` ❌ (row 554), clientbound `SHOP_SCANNER_RESULT` ❌ (row 91), and the v92+ dedicated serverbound `USE_SHOP_SCANNER_ITEM` ❌ (row 579).

The data side is largely built: atlas-merchant already exposes `GET /merchants/search/listings?itemId=` backed by `SearchListingsByItemId` (`services/atlas-merchant/atlas.com/merchant/shop/provider.go:95` — joins listings→shops, filters to Open/Maintenance states, orders by price ascending) and has an `ENTER_SHOP` Kafka command for visits. This task wires the socket surface end-to-end, enriches the search (world scoping, owner identity), adds persisted search-count tracking to power the owl's "most searched items" list, and implements the warp-and-auto-enter flow — with byte-fixture packet verification per the packet-audit discipline. It slots alongside the in-flight cash-item campaign (tasks 123–126).

## 2. Goals

Primary goals:
- A player can use an Owl of Minerva (5230000) to search current player-shop and hired-merchant listings in their world for a chosen item, and sees faithful results (owner, title, price, quantity, channel, sold-out state).
- The owl UI's "most searched items" list is populated from real, persisted per-tenant search counts (top 10 by search count).
- Clicking a search result warps the player to the shop — including across channels if that is the faithful client behavior (verify, see §9 Q2) — and automatically enters the shop as a visitor on arrival.
- The owl is consumed according to faithful behavior — believed to be 1 consumed per successful search (verify, see §9 Q3).
- Both request routes work: the generic `USE_CASH_ITEM` path with itemType 523 (v83-era) and the dedicated `USE_SHOP_SCANNER_ITEM` opcode (v92+).
- All owl packet operations are byte-fixture verified (packet-verifier discipline) on every IDB-backed version.

Non-goals:
- MTS / auction marketplace (task-102).
- The shop visit/purchase mechanics themselves (already exist in atlas-merchant); this task only triggers entry.
- Cash Shop purchase flow for the owl (commodity data is the cash shop feature's concern).
- atlas-ui changes.
- Post-Big-Bang owl variants or generalized item search beyond the shop scanner.

## 3. User Stories

- As a player, I want to use an Owl of Minerva and search for an item so that I can find which shops sell it and at what price.
- As a player, I want to see the most-searched items when I open the owl so that I can gauge market demand.
- As a player, I want to click a result and be taken straight to that shop (even on another channel, if supported) so that I don't have to hunt through Free Market rooms.
- As a player, I want the shop to open automatically when I arrive so that I can buy immediately.
- As a player, I want a clear "no results" message (not a silent no-op) when nothing sells the item.
- As a player, I want stale results handled gracefully — if the shop closed or sold out between search and warp, I get the correct client error instead of a broken warp.

## 4. Functional Requirements

### 4.1 Item semantics

| Item | Name | Source | Kind | Consumption |
|---|---|---|---|---|
| 5230000 | Owl of Minerva | `Item.wz/Cash/0523.img.xml` (sole item) | cash item, classification 523 (`ClassificationStoreSearch`), cash-slot type 29 | believed 1 per **successful** search — MUST verify (§9 Q3) |

- FR-1: Using the owl MUST be permitted anywhere (no Free-Market-only restriction), per owner decision — unless design-phase verification (IDA/WZ field limits) surfaces a faithful restriction, in which case the faithful rule wins and is documented.
- FR-2: Consumption timing MUST be verified during design (client/IDA; may require live-server observation — §9 Q3). The working assumption is: consume 1 on a search that returns ≥1 result; a failed/empty search consumes nothing. Whatever is verified MUST be enforced via the standard consume flow (saga-ordered so a failure path never consumes).

### 4.2 Search request routes (serverbound)

- FR-3: The `USE_CASH_ITEM` handler's 523 arm (`character_cash_item_use.go`) MUST parse the owl-use payload (searched item id; exact byte layout per version IDA-derived during design) and trigger the search flow, replacing the current warn-and-drop fallthrough.
- FR-4: The dedicated `USE_SHOP_SCANNER_ITEM` opcode (`CWvsContext::SendShopScannerItemUseRequest`, v92+; 0x05A on gms_95 per STATUS.md) MUST be handled identically for versions that send it. Design MUST determine per-version which route each supported tenant version actually uses, and both routes MUST be covered.
- FR-5: `OWL_ACTION` (`CUIShopScanner::OnCreate`) MUST be handled. Its exact semantics (open-UI request for the top-10 list vs. re-search from the list vs. both, and its payload) MUST be IDA-derived during design; the handler MUST implement whatever modes the client actually sends.

### 4.3 Search behavior

- FR-6: Search MUST be scoped to the requesting character's tenant and world. The existing `searchListingsByItemId` query (currently tenant-wide only) MUST gain a world filter.
- FR-7: Results MUST include, per listing: shop id, shop type (player shop vs hired merchant), owner character id and name, shop title, channel id, map id, item id, quantity/bundle info, price per bundle, and sold-out/state information — whatever the `SHOP_SCANNER_RESULT` wire format requires (IDA-derived). Owner character id MUST be added to `ListingSearchResult`; owner name resolution goes through atlas-character (read-only).
- FR-8: Results MUST remain ordered by price ascending. Any client- or protocol-imposed result cap (IDA-derived) MUST be applied server-side; if the server truncates, it truncates the most expensive results.
- FR-9: Shops in Open and Maintenance states are searchable (current behavior); design MUST verify whether Maintenance-state shops should be included faithfully and adjust if not.

### 4.4 Most-searched-items tracking

- FR-10: atlas-merchant MUST persist per-tenant search counts keyed by searched item id. Every executed owl search increments the count for the searched item (exact increment trigger — every search vs. successful search — is a design decision, default: every executed search).
- FR-11: The top-10 items by count (per tenant) MUST be retrievable to populate the owl UI's opening list. Whether the scope is per-tenant or per-tenant+world MUST match how the client presents it (design decision; default per-tenant+world to match search scoping).
- FR-12: The counts table MUST follow the tenant-safe PK pattern (surrogate uuid PK + unique index on `(tenant_id, ...)` — never a bare business-key PK).

### 4.5 Result delivery (clientbound)

- FR-13: A `SHOP_SCANNER_RESULT` (`CWvsContext::OnShopScannerResult`) writer MUST be implemented with all modes the client dispatches on (result list, no-results message, top-10 searched list, and any error modes — exact mode set and per-mode bodies IDA-derived). If the packet is mode-dispatched, every supported mode gets a discrete body implementation and fixture — mode-byte enumeration alone is NOT verification.
- FR-14: Any mode byte or sub-code MUST be config-resolved from the tenant template (operations table pattern), never hard-coded, and the tables MUST be populated for every supported version.

### 4.6 Warp to shop

- FR-15: `OWL_WARP` (`CUIShopScanResult::OnButtonClicked`) MUST be handled: the payload identifies the chosen result (exact layout IDA-derived). The server MUST re-validate against current shop state (shop still open, listing still present/not sold out) before warping; stale selections yield the correct client error response, not a broken warp.
- FR-16: Warp MUST place the character in the shop's map. When the shop is on a different channel, the believed-faithful behavior is a channel change + warp — design MUST verify this against the client (IDA/Cosmic) and implement the verified flow (§9 Q2). If cross-channel warp is verified as unsupported, same-channel-only with the faithful error is the fallback.
- FR-17: On arrival, the server MUST automatically enter the character into the shop as a visitor (existing `ENTER_SHOP` command flow, respecting `MaxVisitors=3` — a full shop yields the faithful "shop full" outcome rather than a bare warp with no feedback, exact outcome IDA-verified).
- FR-18: The warp MUST use the existing character warp path (same mechanism portal/mystic-door flows use) so map membership, spawn packets, and transition invariants are preserved.

### 4.7 Version and configuration coverage

- FR-19: Handler entries for all serverbound ops (each with `LoggedInValidator` — a validator-less entry is silently dropped) and the writer entry for `SHOP_SCANNER_RESULT` MUST be added to the seed templates for **all** supported tenant versions: gms_83, gms_84, gms_87, gms_92, gms_95, jms — with per-version opcodes from STATUS.md/IDA (OWL_ACTION 0x042/0x042/0x045/0x048/0x03A; OWL_WARP 0x043/0x043/0x046/0x049/0x03B; SHOP_SCANNER_RESULT 0x046/0x048/0x048/0x049/0x040 across gms_v83/v84/v87/v95/jms; USE_SHOP_SCANNER_ITEM only where the version defines it).
- FR-20: Existing live tenants do not pick up seed-template changes — deployment notes MUST include patching live tenant configs and restarting atlas-channel.

## 5. API Surface

### atlas-merchant (REST, JSON:API)

- `GET /merchants/search/listings?itemId={id}&worldId={id}` — extend the existing endpoint with a required-for-owl `worldId` filter; response gains owner character id (name resolution is the caller's concern or embedded — design decision) and shop type/state fields needed by the wire format.
- `GET /worlds/{worldId}/shop-searches/top` (shape/path per design) — top-10 searched items `[{itemId, count}]` for the tenant(+world). New JSON:API resource.

### Kafka

- New or extended command on `COMMAND_TOPIC_MERCHANT` (or a direct processor call from the search REST path) to record a search-count increment — design decides REST-side synchronous write vs. command; either way the increment must not add latency-coupling failure modes to the search response.
- Warp + auto-enter reuses existing flows: character field transition (existing warp mechanism) and `CommandEnterShop` on `COMMAND_TOPIC_MERCHANT`. Cross-channel case reuses the existing channel-change flow (design-verified).

### Socket (atlas-channel)

- Serverbound handlers: `USE_CASH_ITEM` 523 arm, `USE_SHOP_SCANNER_ITEM` (v92+), `OWL_ACTION`, `OWL_WARP`.
- Clientbound writer: `SHOP_SCANNER_RESULT` (all modes).

## 6. Data Model

New table in atlas-merchant (name per design, e.g. `listing_search_counts`):

| column | type | notes |
|---|---|---|
| id | uuid | surrogate PK |
| tenant_id | uuid | required |
| world_id | smallint | if per-world scoping is chosen (default) |
| item_id | int (uint32) | searched item id |
| count | bigint | monotonically increasing |
| updated_at | timestamp | |

Constraints: unique index on `(tenant_id, world_id, item_id)`; increment is an atomic upsert. No changes to `shops`/`listings` schemas beyond what owner-identity enrichment needs (owner character id already exists on the shop — `CharacterId()`).

## 7. Service Impact

| Service | Change |
|---|---|
| atlas-channel | `USE_CASH_ITEM` 523 arm; `USE_SHOP_SCANNER_ITEM`, `OWL_ACTION`, `OWL_WARP` handlers; `SHOP_SCANNER_RESULT` writer; warp + auto-enter orchestration |
| atlas-merchant | world-scoped search + owner/state enrichment; search-count table, upsert, top-10 provider + REST resource |
| libs/atlas-packet | owl packet codecs (serverbound + clientbound) + byte fixtures per version |
| seed templates / tenant config | opcodes, handler entries (with validators), writer entry, operations/mode tables for all supported versions |
| atlas-character | none (read-only owner-name lookups) |

## 8. Non-Functional Requirements

- Multi-tenancy: every query and Kafka message tenant-scoped via context; search additionally world-scoped.
- Performance: search is an indexed query (`listings.item_id` join `shops`); verify an index exists on `listings.item_id` and add one if missing. Count upsert is O(1) and must not block the search response path.
- Observability: log search requests (item id, result count) and warp outcomes at debug/info; failed warps (stale shop, full shop) at info with reason.
- Packet discipline: fixtures with `packet-audit:verify` markers, evidence records, matrix regeneration — the three artifacts committed together per cell.

## 9. Open Questions

1. **Exact packet layouts** — per-version byte layouts for all four ops (and the `SHOP_SCANNER_RESULT` mode set) are IDA-derived during design. Any op whose fname does not resolve in the IDA export is a stop-and-ask.
2. **Cross-channel warp mechanics** — owner believes warp follows the shop to another channel; verify against client/Cosmic during design (does the client drive a channel change from `OWL_WARP` result, or does the server initiate the transfer?). Implement the verified flow.
3. **Consumption trigger** — believed consume-1-on-successful-search; verify via IDA/live observation. If genuinely unverifiable offline, this needs an online verification pass before the task is called done.
4. **`OWL_ACTION` semantics** — precise request modes (open list vs. search-from-list) from the client decompile.
5. **Maintenance-state shops in results** (FR-9) and **top-10 scope** (FR-11) — small faithful-behavior checks during design.

## 10. Acceptance Criteria

- [ ] Using an owl performs a world-scoped search and the client displays the result list (owner, title, price, quantity, channel) — verified in-game on a v83 tenant.
- [ ] Empty search shows the faithful no-results response; consumption follows the verified rule (no consumption on the failure path).
- [ ] Owl UI opens with a top-10 most-searched list backed by persisted counts; counts survive service restart.
- [ ] Clicking a result warps to the shop (cross-channel per verified behavior) and auto-opens it as a visitor; stale/closed/full shops yield the correct client responses.
- [ ] Both request routes implemented: `USE_CASH_ITEM` 523 arm and `USE_SHOP_SCANNER_ITEM` where the version defines it.
- [ ] All owl ops byte-fixture verified on every IDB-backed version; coverage matrix cells promoted; `packet-audit matrix --check` clean.
- [ ] Seed templates updated for gms_83, gms_84, gms_87, gms_92, gms_95, jms with validators on every handler entry; live-tenant patch documented.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in changed modules; `tools/redis-key-guard.sh` clean; `docker buildx bake` clean for every touched service.
- [ ] Code review (plan-adherence + backend-guidelines) run before PR.
