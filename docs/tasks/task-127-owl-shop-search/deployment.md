# task-127 deployment notes — Owl of Minerva

## Live-tenant config patch (REQUIRED)

Seed templates apply only at tenant creation. Existing tenants MUST be
patched or the owl ops are silently dropped (unhandled op / missing writer):

1. For each live tenant, PATCH its socket configuration with the same
   entries Task 13 added to its version's template:
   - handlers: OwlActionHandle (with options.operations {OPEN:5}),
     OwlWarpHandle, ShopScannerItemUseHandle (gms_83/84/95 only) — each with
     LoggedInValidator. Opcodes per version: see plan.md Global Constraints
     matrix.
   - writers: ShopScannerResult (operations {RESULT:6, HOT_LIST:7}),
     ShopLinkResult (operations {SUCCESS:0, CLOSED:1, FULL:2, BUSY:3, DEAD:4,
     NO_TRADE:7, DENIED:17, MAINTENANCE:18, FM_ONLY:23}).
2. Restart atlas-channel after patching — the handler/writer projection does
   not hot-reload.

## Rollout order

1. atlas-merchant (schema migration `listing_search_counts` is additive;
   new command type is ignored by old channel pods).
2. atlas-channel.
3. Tenant config patch + atlas-channel restart (step above).

## In-game acceptance pass (v83 tenant, per PRD §10)

- Search with results (owner/title/price/quantity/channel correct; owl
  consumed exactly 1).
- Empty search ("Unable to find..." message; owl NOT consumed).
- Hot list on scanner open (top-10 by count; survives service restart).
- Warp to open shop -> auto-enter as visitor.
- Full shop -> "full capacity" (SHOP_LINK code 2).
- Maintenance shop -> code 18. Sold-out race -> code 3. Own shop -> code 17.
- Cross-channel row shows channel number, no warp link (client-side).

## 231-family owl WZ verification (design §9 risk)

Checked the local v83 WZ extract (Cosmic — GMS v83 server emulator's bundled
client data, `wz/Item.wz/Consume/0231.img.xml`) for item 2310000, the
USE-inventory owl consumed by the dedicated `USE_SHOP_SCANNER_ITEM` route.

Result: **present**. `Item.wz/Consume/0231.img.xml` contains
`<imgdir name="02310000">` with `info` fields (`timeLimited=1`, `slotMax=1`,
`only=1`, `price=1`, `tradeBlock=1`, `notSale=1`) — a normal untradeable
event/reward consumable, consistent with a scanner item. The cash owl
(5230000, `Item.wz/Cash/0523.img.xml`, `<imgdir name="05230000">`) is also
present in the same extract.

Conclusion: both routes are player-reachable on v83. The dedicated
`USE_SHOP_SCANNER_ITEM` route is not a dead/unreachable code path on this
tenant version — item 2310000 exists and can be granted/dropped like any
other consumable — in addition to the always-reachable cash owl
(`USE_CASH_ITEM` itemType 523) path.

## Legacy GMS versions (v61 / v72 / v79) — added post-ship

The owl feature was extended to the pre-v83 GMS clients added to main by
PR #971. Every opcode/format below is IDA-verified in that version's IDB
(`GMS_v61.1`, `GMS_v72.1`, `GMS_v79_1`).

Per-version routing (seed templates already carry these; **live tenants must be
patched the same way** — see the live-tenant section above):

| Op | Dir | v61 | v72 | v79 | Notes |
|---|---|---|---|---|---|
| OWL_WARP | sb | 0x3F | 0x42 | 0x41 | body `[int ownerId][int mapId]` (wire-stable) |
| USE_SHOP_SCANNER_ITEM | sb | — (absent) | 0x66 | 0x65 | legacy body `[str serial][short pos][int itemId]` |
| SHOP_SCANNER_RESULT | cb | 0x43 | 0x43 | 0x43 | v61 header omits `nNpcShopPrice` (2-int) |
| SHOP_LINK_RESULT | cb | 0x44 | 0x44 | 0x44 | code enum differs (below) |

**OWL_ACTION is intentionally NOT routed on any legacy client** — there is no
`CUIShopScanner` input dialog before v83, so the client never sends the
hot-list request. The HOT_LIST writer mode is still configured (the client can
decode it) but is never triggered.

**SHOP_LINK_RESULT operations values differ per version** (client-interpreted;
DOM-25 — must come from each tenant's writer table):
- v79: same as v83 — `DENIED:17, MAINTENANCE:18, FM_ONLY:23`.
- v61 & v72: `DENIED:15, MAINTENANCE:16, FM_ONLY:21` (SUCCESS..NO_TRADE = 0-4,7 unchanged).

**v48 is NOT supported.** Its `OnShopScannerResult` (0x38) is a primitive
count-only chat notice with no result list, and it has no shop-link, warp, or
scanner-item-use packet — the owl feature does not exist in that client. No owl
entries were added to `template_gms_48_1.json`.

**Legacy search-trigger caveat (needs live verification):** the pre-v83
`USE_SHOP_SCANNER_ITEM` frame carries no `searchItemId`/`descending`/`updateTime`
(the item is used immediately from `CDraggableItem::OnDoubleClicked`, not from a
search-input dialog), so the decoded request has no server-side search target on
that path — `searchItemId` stays 0 and the search returns nothing. The clientbound
result rendering + warp are fully faithful; how a legacy client actually seeds a
search term should be confirmed against a live v72/v79 tenant.
