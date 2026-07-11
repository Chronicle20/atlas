# Backend Audit — atlas-channel (MTS surface, task-102)

- **Service Path:** services/atlas-channel/atlas.com/channel
- **Scope:** MTS/ITC surface only (socket/handler/itc_*, mts_entry.go, mts/**, kafka/consumer/mts, kafka/consumer/saga MTS arms)
- **Guidelines Source:** backend-dev-guidelines skill (DOM-*/SUB-*/SEC-*), CLAUDE.md DOM-25
- **Date:** 2026-07-10
- **Build:** PASS (`go build ./...` exit 0)
- **Vet:** PASS (`go vet` on mts/handler/consumer pkgs exit 0)
- **Tests:** PASS (mts, mts/holding, mts/listing, mts/transaction, mts/wish, kafka/consumer/mts, socket/handler all `ok`)
- **Overall:** PASS-with-findings (no Critical; 2 Important, several Minor)

## Verdict

The MTS surface is objectively well-built: no direct DB writes in handlers (DOM-15 PASS), all
writes go through the Kafka command processor `mtsproc.NewProcessor(...).X()` → `producer.ProviderImpl`
(DOM-14/16 PASS), no `os.Getenv` in handlers (DOM-12 PASS), dispatcher mode bytes are
config-resolved via `resolveItcOperationKey` against the tenant `operations` table, not hard-coded
(DOM-25 PASS on the dispatch axis), tenant isolation is enforced (`tenant.MustFromContext` +
`t.Is(sc.Tenant())` gate in every consumer announce), and REST read models are immutable
(private fields + getters, constructed only via `Extract`). All four recently-changed hot spots
called out in the brief verify correct.

No finding blocks the branch. The two Important items are an N+1 REST fan-out and a sub-tab
filter gap; both are functional/efficiency, not data-integrity.

## Findings (ranked)

### IMPORTANT

**I-1 — `cart.Items` issues one REST round-trip per cart entry (N+1).**
`mts/cart/cart.go:39-47` — for every wish in the cart it calls
`lp.GetBySerial(worldId, w.ListingSerial())`, and `GetBySerial`
(`mts/listing/processor.go:49-58`) is itself a full `Browse(BrowseFilter{Serial:…})` HTTP GET to
atlas-mts. A cart of N favorites therefore does N sequential HTTP calls. This path runs not only on
a My-Page→Cart browse but on **every** wish add/remove re-push (`kafka/consumer/mts/consumer.go:267`
`announceWishList` cart arm) and on **every purchase** (`removeCartWishForPurchase` → WISH_REMOVED
PURCHASED → `handleWishRemoved` → `announceWishList`). Failure scenario: a player with 20 carted
listings buys any item; the channel makes 20 serial-by-serial REST calls to render the cart re-push,
serialized, on the Kafka consumer goroutine — latency and atlas-mts load scale linearly with cart
size on unrelated operations. `BrowseFilter` has no multi-serial param, so the fix is a batched
endpoint (serials IN (…)) or resolving the cart in one world browse + in-memory match. Bounded
(carts are small), so Important not Critical.

**I-2 — The Wanted tab (section 2) silently ignores the inventory sub-tab (`categorySub`) filter.**
`socket/handler/itc_operation.go:775-779` — the `category == itcSectionWanted` case calls
`mtswanted.WorldItems(l, ctx, s.WorldId(), s.CharacterId())`, whose signature
(`mts/wanted/wanted.go:22`) takes no `categorySub` and does no inventory-type narrowing (confirmed:
no `categorySub` reference anywhere under `mts/wanted/`). The sibling `wishItems`
(`itc_operation.go:876-892`) *does* implement the equip/use/setup/etc narrowing via
`inventory.TypeFromItemId`, and the public browse applies it through `applyItcViewFilters`. So on the
Wanted tab, selecting the "Equip" sub-tab returns every want-ad (use/setup/etc included) rather than
equip-only. Failure scenario: player on Wanted → clicks the Use sub-tab → still sees armor/weapon
want-ads. Lower confidence on client intent (could not verify from IDA whether the Wanted section
exposes the sub-tab row), but the code is internally inconsistent with the other list arms.

### MINOR / OBSERVATIONS

**M-1 — Register-wish re-push navigates the poster to the cross-character Wanted tab.**
`kafka/consumer/mts/consumer.go:664-666` + `wishSectionForOrigin` (`:309-318`): a `RegisterWish`
origin maps to `(mtsSectionWanted=2, TypeWanted)` and `announceWishList` sends a section-2
`GetItcListDone`. By this file's own CANCEL_WISH finding (`:702-709`, "a section-2
GET_ITC_LIST_DONE makes the client navigate to the Wanted tab"), posting a want-ad yanks the poster
onto the Wanted tab — which excludes their own new ad (that lives under My Page→Offers). CANCEL_WISH
was deliberately fixed to use `announceOwnWantAds`; REGISTER_WISH still uses the cross-character arm.
May be intended (see the board after posting); flagged for symmetry.

**M-2 — MTS config cache never invalidates.**
`mts/configuration/registry.go:61-83` — `GetTenantConfig` caches the fetched Model per tenant for the
process lifetime with no TTL/invalidation. A tenant editing MinLevel/fees in atlas-tenants is not
picked up until the channel pod restarts. This matches the service's established "load-once" config
pattern, but unlike the config-status Kafka projection used elsewhere it has no hot-reload path.

**M-3 — DOM-25 gray area on the section/category tab numerals.**
`socket/handler/itc_operation.go:292-295` (`itcSectionWanted=2`, `itcSectionCart=4`), `:301`
(`mtsSearchOptionSellerName=0`), and the literal `Category:"1"`/`"3"` filters
(`mts_entry.go:143`, `consumer.go:200`) are client-interpreted wire values used as literals rather
than resolved through a tenant writer table. The **dispatcher mode bytes** are correctly
config-resolved (`resolveItcOperationKey`), and these tab numerals are largely client-echoed and
documented as IDA-verified/version-uniform, so this is almost certainly acceptable — but CLAUDE.md's
DOM-25 note ("version-stable never exempts") makes it worth recording.

**M-4 — `emitRemoveCartWishByListingSerial` header doc contradicts its implementation.**
`socket/handler/itc_operation.go:931-938` (header) says "resolve it to the listing's item, find the
character's CART entry for that item", but the body (`:942-961`) and its inline comment match the
cart wish **directly by `ListingSerial`** with no listing lookup. The implementation is the correct
one (works even after the listing sells); only the stale header is misleading.

**M-5 — Auction-settle contract date is `time.Now()`, not the listing's actual end.**
`kafka/consumer/mts/consumer.go:466` — `SuccessBidInfoResult` stamps `packetmodel.MsTimeBytes(time.Now())`
though the listing carries `EndsAt`. Cosmetic (the "contract date" shown to both parties is a few ms
off the true settle instant); the settle event does not carry the end time so this is a pragmatic choice.

## Checklist notes (why the DDD DOM items are N/A here)

- **DOM-01/02/03 (builder.go / ToEntity / Make):** N/A. `mts/{wish,listing,holding,transaction}`
  are channel-side REST **read-projection** packages (RestModel + `Extract` → immutable Model),
  the established atlas-channel gateway convention — there is no GORM entity or write side in this
  service. Models are correctly immutable (private fields, getters only; no setters), e.g.
  `mts/wish/model.go:24-45`, `mts/listing/model.go:14-99`.
- **DOM-11/EXT-01 (lazy providers / JSON:API stubs):** PASS. Providers use
  `requests.SliceProvider[...]` lazily (`mts/wish/processor.go:47-61`,
  `mts/listing/processor.go:37-43`); both RestModels implement `SetToOneReferenceID` /
  `SetToManyReferenceIDs` no-op stubs (`mts/wish/rest.go:33-34`, `mts/listing/rest.go:74-75`).
- **DOM-14/15/16 (no handler-side writes):** PASS. Zero `db.Create/Save/Delete` in the MTS
  handlers; all mutations are Kafka commands via `mts/processor.go` → `producer.ProviderImpl`.
- **DOM-24 (Kafka producer stubbed in emitting tests):** N/A. The MTS test files
  (`itc_operation_test.go`, `kafka/consumer/mts/consumer_test.go`, etc.) exercise only pure functions
  (build-args, filter mapping, reason routing); none drive an emit path (direct or transitive), so no
  `producertest` stub is required and none is present.
- **DOM-25 (config-resolved wire values):** PASS on the dispatch axis (mode bytes reverse-resolved
  from the tenant `operations` table; failure reasons via `noticeFailReasons`); see M-3 for the tab
  numerals gray area.
- **SEC-*:** N/A (atlas-channel MTS is not an auth/token surface).

## Blocking (must fix)
- None.

## Non-Blocking (should fix)
- I-1: batch the cart listing resolution (N+1 REST fan-out) — `mts/cart/cart.go:39-47`.
- I-2: thread `categorySub` through the Wanted tab or confirm the client does not expose the sub-tab
  there — `socket/handler/itc_operation.go:775-779`, `mts/wanted/wanted.go:22`.
- M-1..M-5: symmetry / doc / staleness nits as described.

## Final resolution (post-audit fixes)

- **I-1 (Important, cart N+1) — FIXED.** `mts/cart/cart.go` now resolves all favorited listings in ONE browse (`BrowseFilter.Serials` -> atlas-mts `serial IN (?)`) instead of a `GetBySerial` per entry, indexing results by `ItcSn()`. Added `Serials []uint32` to both the channel and atlas-mts `BrowseFilter` (mirrors the existing `itemIds`/`TemplateIds` pattern) + the `serials` query param. Removes the per-re-push fan-out.
- **I-2 (Important, Wanted sub-tab not filtered) — DEFERRED (unverified client intent).** `mtswanted.WorldItems` ignores `categorySub`. The auditor could not confirm from IDA that the Wanted section even exposes the equip/use/etc sub-tab row, so filtering it could be wrong; left as-is pending IDA verification rather than guessed (verify-don't-invent). Pre-existing, not introduced by this work.
- **Minors — DEFERRED/noted.** M-1 register-wish re-push navigates to the Wanted tab (asymmetric with the CANCEL_WISH->Offers fix, but posting vs cancelling may want different destinations — flagged for an owner decision, not guessed); M-2 config cache no hot-invalidation (restart, known pattern); M-3 literal section/category numerals (DOM-25 gray area, client-fixed tab ids); M-4 stale `emitRemoveCartWishByListingSerial` header doc; M-5 auction-settle contract `time.Now()`.

## Update (owner-confirmed)

- **I-2 (Wanted sub-tabs) — FIXED.** Owner confirmed the Wanted tab DOES expose
  item-type sub-tabs. `mtswanted.WorldItems` now takes `categorySub` and filters
  each want-ad by its item's inventory type (`inventory.TypeFromItemId`), mirroring
  `wishItems`/the public browse. The synchronous Wanted browse passes the client's
  sub-tab; the section-level re-push passes 0 (all).
- **M-1 (register-wish navigation) — FIXED.** Owner confirmed posting a want-ad
  should land on My Page -> Offers. `handleWishAdded` now re-pushes
  `announceOwnWantAds` for the RegisterWish origin (requestSent=1 clears the latch),
  symmetric with the CANCEL_WISH -> Offers fix.
