# Backend Audit — task-102 MTS Marketplace (Go)

- **Branch:** `task-102-mts-marketplace`
- **Diff range:** `main..HEAD` (BASE `eed47d480`, HEAD `87dfa758a`)
- **Guidelines source:** backend-dev-guidelines skill (DOM-* / SUB-* / SEC-*)
- **Date:** 2026-06-18
- **Overall:** **NEEDS-WORK** — build/vet/test all green, but 1 Critical tenant-safety bug + 3 convention FAILs (DOM-02/03/05) and several Minor items.

## Phase 1 — Build / Vet / Test gate (verified, not assumed)

All commands run from each module's `atlas.com/<module>` dir on this branch:

| Module | `go build ./...` | `go vet ./...` | `go test -race ./... -count=1` |
|---|---|---|---|
| `services/atlas-mts` | PASS | PASS | PASS (all pkgs `ok`, 1.0–2.0s each — no producer hangs) |
| `services/atlas-saga-orchestrator` | PASS | PASS | PASS (`saga`, `saga/mock` ok; `mts` no test files) |
| `services/atlas-channel` | PASS | PASS | PASS (`mts`, `mts/listing`, `mts/wish`, `socket/handler` ok) |
| `services/atlas-tenants` | PASS | PASS | PASS (`configuration` ok) |
| `libs/atlas-saga` | PASS | — | PASS |
| `libs/atlas-packet` | PASS | — | PASS (`field/serverbound` ok) |

Objective gate PASSES. The fast test times confirm emit paths are stubbed (see Kafka section).

---

## CRITICAL

### C-1 — Malformed `wishId` path param triggers a tenant-wide DELETE of all wish entries

**Evidence:**
- `services/atlas-mts/atlas.com/mts/wish/resource.go:95-100` — `handleDeleteWish` only checks `wishId != ""`; it never validates that the value parses as a UUID.
- `services/atlas-mts/atlas.com/mts/wish/administrator.go:14-20` — `parseId` returns `uuid.Nil` (not an error) on a malformed id.
- `services/atlas-mts/atlas.com/mts/wish/administrator.go:67` — `DeleteWish` runs `db.Where(&entity{Id: parseId(id)}).Delete(&entity{})`.

`uuid.UUID` is `[16]byte`; `uuid.Nil` is its Go zero value. GORM **silently elides zero-value fields in struct-condition `Where`** (the exact gotcha in `anti-patterns.md:24` and `patterns-multitenancy-context.md:95`). The atlas-database tenant callback still injects `WHERE tenant_id = ?`, which (a) satisfies GORM's `ErrMissingWhereClause` global-write guard so the delete is **not** blocked, and (b) scopes the blast radius to the whole tenant.

**Exploit:** `DELETE /characters/{characterId}/mts/wishlist/not-a-uuid` →
`DELETE FROM wish_entries WHERE tenant_id = ?` → **every wish entry for the tenant is deleted.**

The author was demonstrably aware of this hazard: `listing/administrator.go:178-201` (`AdvanceAuctionBid`) uses a map-keyed `Where(map[string]interface{}{...})` *specifically* "so a struct condition would [not] elide a zero-valued prior bid" — the safe pattern was known but not applied to the delete/update functions.

**Fix:** reject an unparseable id at the handler (return 400/404 before the DB), OR use map-keyed `.Where(map[string]interface{}{"id": parseId(id)})` in `DeleteWish` so a `uuid.Nil` matches nothing.

---

## IMPORTANT

### I-1 — Same struct-condition elision in the holding/listing/bid write functions (latent, defense-in-depth)

Same anti-pattern as C-1, in:
- `services/atlas-mts/atlas.com/mts/holding/administrator.go:121` (`SoftDelete`)
- `services/atlas-mts/atlas.com/mts/holding/administrator.go:131` (`Restore`)
- `services/atlas-mts/atlas.com/mts/listing/administrator.go:151` (`UpdateState` — has a `State` predicate, so a `uuid.Nil` would mass-update only rows in `from` state)
- `services/atlas-mts/atlas.com/mts/listing/administrator.go:164` (`UpdateAuction` — **no** state predicate; a `uuid.Nil` id would rewrite the auction fields of *every* listing in the tenant)

**Why IMPORTANT not CRITICAL:** every caller of these functions passes a UUID obtained from a DB row or a Kafka saga payload via `.String()` (e.g. `kafka/consumer/custody/consumer.go:177,213,278`, `listing/processor.go:323,698,815`) — always well-formed, so the elision cannot trigger today. But these are one refactor away from being reachable with attacker-controlled input, and the fix is mechanical. Apply the map-keyed `Where` (as `AdvanceAuctionBid` already does) to all five write functions.

### I-2 — DOM-02 `ToEntity()` missing in every domain

`grep "func.*ToEntity"` over `listing`, `holding`, `wish`, `bid` returns nothing. Entity construction is inlined in each administrator's `Create*` (e.g. `wish/administrator.go:49-55`, `holding/administrator.go` `CreateHolding`, `listing/administrator.go` `CreateListing`). Functionally correct (tests pass), but violates the model→entity convention.

### I-3 — DOM-03 `Make(Entity) (Model, error)` missing in every domain

No `func Make(` anywhere. The role is filled by an unexported `modelFromEntity` living in `provider.go` (e.g. `wish/provider.go:40`, `holding/provider.go:82`, `listing/provider.go:182`, `bid/provider.go:41`) rather than the named `Make` in `entity.go`.

### I-4 — DOM-05 `TransformSlice` missing in `listing`, `holding`, `wish`

No domain `TransformSlice`; list handlers inline `model.SliceMap(Transform)(...)`:
- `services/atlas-mts/atlas.com/mts/listing/resource.go:207`
- `services/atlas-mts/atlas.com/mts/holding/resource.go:119`
- `services/atlas-mts/atlas.com/mts/wish/resource.go:46`

---

## MINOR

### M-1 — DOM-17: `listing.List` collapses DB errors to HTTP 400
`listing/resource.go:139` maps every `List` error to 400, including a DB failure from the active-cap count (`processor.go:403`). A transport/DB error would be reported to the client as a bad request.

### M-2 — DOM-21: `SourceInventoryType`/`InventoryType` as raw `byte`
`holding/rest.go`, `listing/rest.go:90`, `listing/processor.go:50`, `kafka/message/mts/kafka.go:142,161`. `libs/atlas-constants/inventory` defines the canonical compartment enum (`inventory.Type`, `int8`). **However**, the shared `libs/atlas-saga/payloads.go` itself uses raw `byte` for these fields (lines 104, 502, 514, 568), so atlas-mts is *matching the established cross-service saga wire contract*, not reinventing a type. This is a repo-wide convention gap, not a task-102 regression — downgraded to informational.

### M-3 — Service-local REST wrapper
`services/atlas-mts/atlas.com/mts/rest/handler.go` reimplements `RegisterHandler`/`RegisterInputHandler`/`ParseInput` instead of calling `server.*` directly (ai-guidance §"Manual HTTP Handler Registration"). It does delegate to `server.RetrieveSpan`/`server.ParseTenant`, so tenant/tracing propagation is preserved; this is extra boilerplate, not a correctness defect.

---

## Confirmed CORRECT (notable PASSes, verified by reading source)

### Tenant safety (key risk — the prior "slug-only PK collides" bug is absent)
- Composite `(tenant_id, id)` unique indexes on all four entities: `wish/entity.go:26-27`, `listing/entity.go:42-43`, `holding/entity.go:42-43`, `bid/entity.go:27-28` (priority 1 = tenant_id, priority 2 = id — NOT tenant_id alone).
- Per-world serial uniqueness is tenant-scoped: `uniqueIndex idx_listings_world_serial (tenant_id, world_id, serial)` `listing/entity.go:43-45`; same for holding `entity.go:43-45`.
- **Read** providers consistently use explicit name-keyed-map `Where` for world-0 / zero-eligible filters, each with an explanatory comment: `listing/provider.go:33,76,128`, `holding/provider.go:32,53,72`, `wish/provider.go:30`, `bid/provider.go:31`.
- `database.Connect` in `main.go`; `RegisterTenantCallbacks` only in `test/database.go:28` (not in main.go) — matches `anti-patterns.md:23`.

### Settle-move race-loss acks ERROR (the headline correctness fix, commit `5cb1bb17a`)
- `kafka/consumer/custody/consumer.go:294-306` — when the conditional `active→sold` transition affects 0 rows AND no prior buyer holding exists (concurrent cancel/expire won), the handler returns `errMoveLostRace` and creates no holding.
- `consumer.go:345-350` — on any handler error it emits `ErrorStatusEventProvider` and returns; it does **not** emit the MOVED ack. So the saga compensates the buyer's prepaid debit (no currency desync).

### Dupe-safety suite genuinely asserts invariants (not just "runs")
- `kafka/consumer/custody/dupe_safety_test.go:172-189` (`CancelWins`) and `:230-242` (`SettleWins`) assert `countHoldingsForOwner == 1`/`0` (single-custody) **and** that the losing move acked `custody.StatusEventTypeError` (currency invariant). Take-home replay (`:260+`) asserts the second delivery affects 0 rows.

### Saga timeouts explicitly set and step-scaled (heeds `bug_preset_creation_saga_flat_timeout`)
- `listing/processor.go:467-473` (list), `:577-580` (buy, N=3), `:594-600` + `:725,746` (bid escrow), `:755-762` + `:841` (auction settle); `holding/processor.go:153-158` (take-home). All `base + numSteps*perStep`, none flat-default.

### Kafka wire-shape consistency (channel ↔ atlas-mts ↔ orchestrator)
- Command body JSON tags identical between `services/atlas-channel/.../kafka/message/mts/kafka.go` and `services/atlas-mts/.../kafka/message/mts/kafka.go` (verified field-by-field for RegisterSale, Buy, PlaceBid, RegisterWish, MoveLtoS, etc.).
- Command `Type` strings and `COMMAND_TOPIC_MTS` env key match (channel produces, mts consumes).
- Status-event `Type` strings and `EVENT_TOPIC_MTS_STATUS` env key match; channel consumes a subset (BID_PLACED / ITEM_MOVED_TO_HOLDING / LISTING_EXPIRED / OUTBID are internal/unwired-to-packet, not a mismatch).
- All MTS saga actions have decode cases in `libs/atlas-saga/unmarshal.go:366-403`, covered by `unmarshal_test.go` (35 Mts references).

### No hardcoded packet mode bytes (heeds the dispatcher-family bug-memory)
- `services/atlas-channel/.../socket/handler/itc_operation.go:92-115` (`resolveItcOperationKey`) reverse-resolves the inbound dispatcher mode byte against the tenant `options["operations"]` config table; mode bytes appear only in doc comments. Mirrors `isMessengerShopOperation`.

### Producer stubbing in tests (no 42s emit hangs)
- Emit-path consumer tests inject a `recordingProducer` per handler (`kafka/consumer/custody/dupe_safety_test.go` passes `rp.provider()` into the handler) — Pattern B injection that also captures events for assertion.
- Processor flow tests inject stubs via the Option pattern: `listing/processor.go:197 WithSagaEmitter`, `:209 WithBalanceReader`; the live `saga.NewProcessor`/`wallet.NewProcessor` defaults are overridden in tests, so no real Kafka writer is exercised.

### Other
- Immutable models (private fields + getters, no setters) across `listing`/`holding`/`wish`/`bid`; Builders with `Build()` validation (`builder.go` in each, e.g. `listing/builder.go:280` tenantId check).
- Config registry is a `sync.Once` singleton (`configuration/registry.go:26,31`) — matches the cache-as-singleton rule.
- No `*_testhelpers.go` files introduced. No `// TODO`/stub/501 introduced by this branch (the one TODO at `atlas-channel/main.go:269` predates the branch — commit `b3de17d5ed`, 2026-05-27 — and is out of scope).

---

## Summary

| Severity | Count | IDs |
|---|---|---|
| Critical | 1 | C-1 |
| Important | 4 | I-1, I-2, I-3, I-4 |
| Minor | 3 | M-1, M-2 (informational), M-3 |

**Recommended before merge:** fix C-1 (and ideally I-1 in the same pass — same one-line map-keyed-`Where` change applied to all five write functions). DOM-02/03/05 (I-2/3/4) are convention debt that does not affect correctness; land or follow-up per reviewer discretion.
