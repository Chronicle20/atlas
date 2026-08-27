# Backend Audit — task-269-ring-pair-behavior

- **Scope:** branch diff `32d55cb21..e5f7cf0`
- **Services touched:** atlas-cashshop, atlas-channel, atlas-character, atlas-configurations, atlas-data
- **Libs touched:** libs/atlas-constants, libs/atlas-packet
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-27
- **Build:** PASS (all 7 modules: `go build ./...` clean)
- **Tests:** PASS (all 7 modules: `go test ./... -count=1` — zero failures)
- **Overall:** NEEDS-WORK (6 FAIL checks; build/tests are clean)

## Build & Test Results

```
services/atlas-cashshop/atlas.com/cashshop        go build: clean   go test: ok (all packages)
services/atlas-channel/atlas.com/channel          go build: clean   go test: ok (all packages)
services/atlas-character/atlas.com/character      go build: clean   go test: ok (all packages, incl. pending_change 227s)
services/atlas-configurations/atlas.com/configurations  go build: clean   go test: ok
services/atlas-data/atlas.com/data                go build: clean   go test: ok
libs/atlas-constants                              go build: clean   go test: ok
libs/atlas-packet                                 go build: clean   go test: ok
```

No build errors, no test failures, in any touched module.

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| FILE-01..06 | Fired | 109 changed non-test `.go` files across 5 services |
| RUNTIME (DOM-26) | Fired | Non-test Go files changed everywhere; `tools/goroutine-guard.sh` run against all 5 touched service modules, exit 0 |
| DOM-STRUCTURE (DOM-01..05,11,16) | Fired | `model.go` present in `atlas-cashshop/ring`, `atlas-channel/ring`, `atlas-channel/cashshop/purchaserecord`, `atlas-character/equipslot`, `atlas-channel/cashshop/inventory/asset`, etc. |
| SUB-01..04 | Fired | `atlas-cashshop/purchaserecord` and `atlas-data/cashpackage` have `resource.go` with no `model.go` |
| REST (DOM-06..09,12..15,17..19,32) | Fired | Every touched domain/sub-domain package has `resource.go` and/or `rest.go` |
| CACHE (DOM-29) | Fired | `atlas-channel/ring/cache.go` new; `atlas-data/cashpackage/registry.go` singleton registry |
| MESSAGING (DOM-30) | Fired | `atlas-cashshop/cashshop/processor_ring.go` (`PurchaseRingAndEmit`) uses `message.Emit`/`message.Buffer` |
| MULTITENANCY (DOM-31) | Fired | Every changed `rest.go`/processor reads or passes tenant state |
| CONSTANTS (DOM-21) | Fired | New `Type`/`State` enums in both ring packages, new `Entity` types, etc. |
| TESTING (DOM-10,20,24,33) | Fired | Diff adds/changes many `_test.go` files; `wallet.Processor` interface gains a method |
| MIGRATION (DOM-34/35) | N/A | No symbol moved/extracted between a service and `libs/atlas-*` — the only `libs/atlas-constants` change is a one-line slot-position addition (`inventory/slot/constants.go`), not a migration |
| EXT-01..04 | Fired | `atlas-channel/ring`, `atlas-channel/character/equipslot`, `atlas-channel/cashshop/purchaserecord` all call another atlas service via `requests.*Request[T]` |
| RESILIENCE (DOM-27/28) | Fired | DB-backed handlers changed in atlas-cashshop/atlas-character; `character_data.go`'s equip-slot-extension read and `ring/processor.go`'s `Populate` are both remote-fetch fallback paths |
| SEC-01..04 | N/A | No touched package in this diff handles JWT/auth tokens, revocation, or redirects |
| CHANNEL-WIRE (DOM-25) | Fired | `libs/atlas-packet/model/ring.go`, `character/clientbound/{info,spawn,appearance_update}.go`, `character/data.go` all touch the wire codec |
| SCAFFOLD-01..09 | N/A | No new `services/atlas-<svc>/` directory added; `BUY_COUPLE`/`BUY_FRIENDSHIP` operation codes and `CashShopOperationWriter` already existed pre-diff (only the handler *bodies* changed from log stubs to real dispatch) |
| Foundational (provider/functional patterns) | Fired (provider) | `atlas-channel/ring`, `atlas-cashshop/ring` define curried providers |

## Checklist Results

### atlas-cashshop/ring (domain — builder, entity, model, provider, administrator, processor, resource, rest)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` with `NewBuilder()`/`Build()` | PASS | `ring/builder.go:27-33` `NewBuilder()`, `:131-149` validating `Build()` |
| DOM-02 | `Model.ToEntity()` in `entity.go` | PASS | `ring/entity.go:58` `func (m Model) ToEntity(tenantId uuid.UUID) Entity` |
| DOM-03 | `Make(Entity)(Model,error)` in `entity.go` | PASS | `ring/entity.go:39` |
| DOM-04 | `Transform(Model)(RestModel,error)` in `rest.go` | PASS | `ring/rest.go:52` |
| DOM-05 | `TransformSlice` defined and used by list handlers | FAIL (Important) | `ring/resource.go:67` list handler (`handleGetRings`) calls `model.SliceMap(Transform)(...)` directly; `ring/rest.go` defines no `TransformSlice`. (Matches the pre-existing `coupon/redemption/resource.go:88` idiom in the same service, but per audit mindset a new file continuing a documented anti-pattern is still a finding, not an exemption.) |
| DOM-11 | Providers lazy, `database.EntityProvider[T]` | PASS | `ring/provider.go:13-44` return `func(db *gorm.DB) model.Provider[...]{ return func() (...) {...} }` — lazy, no premature `FixedProvider` wrap; shape matches sibling `coupon/redemption/provider.go` |
| DOM-16 | Writes in `administrator.go` | PASS | `ring/administrator.go:25` `CreatePair`, called from `cashshop/processor_ring.go:191` |
| DOM-06 | Processor ctor takes `logrus.FieldLogger` | PASS | `ring/processor.go:39` |
| DOM-07 | Handlers pass `d.Logger()` | PASS | `ring/resource.go:59-60,85-86` |
| DOM-08 | POST/PATCH via `RegisterInputHandler[T]` | N/A | Package registers GET-only routes (`ring/resource.go:29-30`) — no write route exists (write path lives in `cashshop.PurchaseRingAndEmit`) |
| DOM-09 | Every `Transform(` call site checks error | PASS | `ring/resource.go:67-72,104-109` |
| DOM-12 | No `os.Getenv` in `resource.go` | PASS | none found |
| DOM-13 | No cross-domain orchestration in handlers | PASS | handlers call only `ring.NewProcessor(...)` |
| DOM-14 | Handlers call processor, not provider | PASS | `ring/resource.go:60,86` call `NewProcessor(...)`, never `byCharacterIdProvider` etc. directly |
| DOM-15 | No `db.Create`/`Save`/`Delete` in `resource.go` | PASS | none found |
| DOM-17 | Domain errors → correct HTTP status | PASS | `ring/resource.go:95-98` maps `gorm.ErrRecordNotFound` → 404 |
| DOM-18 | `RestModel` implements JSON:API iface | PASS | `ring/rest.go:31-50` |
| DOM-19 | Request models flat | N/A | No write request model exists in this package |
| DOM-21 | No redeclared `libs/atlas-constants` type | PASS (checked, not shared) | `ring/model.go:8-9` doc comment records the check: `ClassificationRing` in `libs/atlas-constants/item` is an item classification, not a pairing type — no shared equivalent exists |
| DOM-27 | 503-vs-500 via `WriteErrorResponse` | PASS | `ring/resource.go:63,70,100,107` all use `restserver.WriteErrorResponse`; no raw `http.StatusInternalServerError` |
| DOM-30 | Write+emit atomic via `AndEmit`/`message.Buffer` | PASS | `cashshop/processor_ring.go:83-84,207` `database.ExecuteTransaction` + `message.Emit(outbox.EmitProvider(...))` |
| DOM-31 | No tenant field on RestModel/request | PASS | `ring/rest.go` — no `TenantId` field |
| DOM-33 | Mocks updated for interface changes | N/A | `ring.Processor` is new in this diff — no prior mock to go stale |
| EXT-01..04 | N/A (this package is a REST *server*, not a client of another service for ring data; its only outbound call — `p.chaP.GetById()` — is on the pre-existing `character.Processor` interface, unchanged here) | | |
| FILE-01..06 | Each responsibility in its own file, no catch-all | PASS | `builder.go`/`entity.go`/`model.go`/`administrator.go`/`provider.go`/`processor.go`/`resource.go`/`rest.go` all single-purpose |

### atlas-cashshop/purchaserecord (sub-domain-shaped: has `resource.go`, no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SUB-01 | Business logic not in handler | PASS | `purchaserecord/resource.go:33` calls `NewProcessor(...).Get`, no inline logic |
| SUB-02 | Writes via administrator, no `db.Create` in `resource.go` | PASS | `purchaserecord/administrator.go` owns writes; `resource.go` is GET-only |
| SUB-03 | POST via `RegisterInputHandler[T]` | N/A | No POST route registered here |
| SUB-04 | No manual JSON parsing in `resource.go` | PASS | none found |
| DOM-04 | `Transform(Model)(RestModel,error)` | N/A | Package has no `Model` type at all — `RestModel` is built directly from a primitive `uint32` count returned by `Get` (`resource.go:40-44`); there is no domain model to transform from |
| DOM-31 | No tenant field on RestModel | PASS | `purchaserecord/rest.go` — no tenant field |

### atlas-channel/ring (domain-shaped: has `model.go`, no `entity.go`/`administrator.go` — read-only REST-client cache)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` with `NewBuilder()`/`Build()` | **FAIL (Important)** | Package has `model.go` (`ring/model.go:31-42` `Model` struct) but defines no `builder.go`; the `Model` is constructed by a raw struct literal inside `Extract` (`ring/rest.go:68-79`), bypassing any validating `Build()`. The same diff's sibling read-only client package `atlas-channel/cashshop/inventory/asset` (also model.go-only, also REST-sourced) *does* define `builder.go` with `NewModelBuilder`/`Build`/`MustBuild` (`asset/builder.go:26,76,93`) — proving this is not an accepted shape for this class of package in this service, just an omission here. |
| DOM-04 | `Transform(Model)(RestModel,error)` in `rest.go` | **FAIL (Important)** | `ring/rest.go` defines only `Extract(RestModel)(Model,error)` (`rest.go:51`); no `Transform` function exists. Sibling `cashshop/inventory/asset/rest.go` defines both `Transform` (`:81`) and `Extract` (`:97`) in the same diff. |
| DOM-05 | `TransformSlice` | N/A | Package has no `resource.go` / list handler to evaluate against (it is a REST client only, never serves its own endpoint) |
| DOM-06 | Processor ctor takes `logrus.FieldLogger` | PASS | `ring/processor.go:82` |
| DOM-11/FILE-05 | N/A (no `provider.go`/DB reads — REST client only) | | |
| DOM-16 | N/A (no writes — read-only cache) | | |
| DOM-21 | No redeclared `libs/atlas-constants` type | PASS (checked, not shared) | `ring/model.go:5-9` doc comment: no ring-pair `Type` exists in `libs/atlas-constants`; re-declared service-local to mirror `atlas-cashshop/ring/model.go` |
| DOM-28 | Fallible remote-fetch enrichment degrades loudly (`degrade.Observe`) | **FAIL (Important)** | `ring/processor.go:107-111` `Populate`'s upstream-fetch failure branch only does `p.l.WithError(err).Warnf(...)` and returns `nil` — it never calls `degrade.Observe(l, "<svc>.<domain>.<enrichment>", characterId, err)`. The same diff's own `character_data.go:131` (`degrade.Observe(l, "channel.character_data.equip_slot_ext", c.Id(), err)`) is the correct shape for an equivalent fail-open remote enrichment. As written, a cashshop outage silently empties every character's ring cache with no `atlas_enrichment_degraded_total` signal for an operator. |
| DOM-29 | Cache is an application-scoped singleton via `GetCache()`-shaped accessor | PASS | `ring/cache.go:26-38` package-level `ringCachePtr`, `sync.Once`, `sync.RWMutex` (`:22`), `getRingCache()` accessor; DOM-29 grades scope only, an unexported accessor name still passes per `patterns-cache.md:330-331` |
| DOM-31 | Tenant only via context, never on RestModel | PASS | `ring/rest.go` — no tenant field; `ring/processor.go:102,117,122,135` all derive `tenant.MustFromContext(p.ctx)` |
| EXT-01 | Target `RestModel` implements `SetToOneReferenceID`/`SetToManyReferenceIDs` | PASS | `ring/rest.go:42,45` |
| EXT-02 | `httptest`-backed integration test, populated fixture | PASS | `ring/processor_test.go` (509 lines; exercises `Populate`/`GetRingSet`/`GetRingRecords` against an httptest server per the file's existence and package test pass) |
| EXT-03 | Only genuine 404 → not-found; transport/5xx bubble | PASS | `ring/requests.go`/`processor.go` propagate `upstreamFn`'s error verbatim on failure (`processor.go:107-111`), no error-class conversion |
| EXT-04 | URL via `requests.RootUrlFor` | PASS | `ring/requests.go:15` `requests.RootUrlFor(ctx, "CASHSHOP")` |
| Multi-tenancy on cache/session-destroy paths | PASS | `EvictTenant` wired in `main.go:307`; `session/processor.go:430,498-502` `clearRingsOnDestroy` calls `ring.NewProcessor(l, ctx).Invalidate(characterId)`, tenant derived from `ctx` inside the processor |
| FILE-01..06 | No catch-all file | PASS | `cache.go`/`model.go`/`processor.go`/`requests.go`/`rest.go` each single-purpose |

### atlas-channel/cashshop/purchaserecord (domain-shaped: has `model.go`, REST-client only)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` | **FAIL (Important)** | `purchaserecord/model.go:3-19` defines `Model`; no `builder.go` exists in the package (`ls`: `model.go processor.go processor_test.go requests.go rest.go`). Same defect class as `atlas-channel/ring` above; same sibling counter-example (`cashshop/inventory/asset/builder.go`) in the same diff. |
| DOM-04 | `Transform(Model)(RestModel,error)` | **FAIL (Important)** | `purchaserecord/rest.go` defines only `Extract` (`:36`), no `Transform`. |
| EXT-01 | `SetToOneReferenceID`/`SetToManyReferenceIDs` | PASS | `purchaserecord/rest.go:31,34` |
| EXT-04 | URL via `requests.RootUrlFor` | PASS | `purchaserecord/requests.go:15` |
| DOM-31 | No tenant on RestModel | PASS | `purchaserecord/rest.go` — none |
| FILE-01..06 | No catch-all | PASS | one responsibility per file |

### atlas-channel/character/equipslot (support: no `model.go`, REST-client-only, `RestModel` doubles as domain type)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-04 | `Transform` present | PASS (identity transform) | `equipslot/rest.go:40` `func Transform(r RestModel) (RestModel, error)` |
| DOM-06 | ctor `logrus.FieldLogger` | PASS | `equipslot/processor.go:24` |
| EXT-01 | `SetToOneReferenceID`/`SetToManyReferenceIDs` | PASS | `equipslot/rest.go:32,35` |
| EXT-02 | httptest integration test | PASS | `equipslot/processor_test.go:24-55` `TestGetActive_DecodesActiveExtensions` |
| EXT-03 | Only 404 → not-found | PASS | `GetActive` propagates the transport error verbatim, no conversion |
| EXT-04 | `requests.RootUrlFor` | PASS | `equipslot/requests.go:15` |
| FILE-01..06 | No catch-all | PASS | `processor.go`/`requests.go`/`rest.go` |

### atlas-character/equipslot (domain: builder, entity, model, provider, administrator, processor, resource, rest)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` | PASS | `equipslot/builder.go:18-20` `NewBuilder()`, `:47-61` validating `Build()` |
| DOM-02 | `Model.ToEntity()` | PASS | `equipslot/entity.go:55` |
| DOM-03 | `Make(Entity)` | PASS | `equipslot/entity.go:35` |
| DOM-04 | `Transform(Model)` | PASS | `equipslot/rest.go:32` |
| DOM-05 | `TransformSlice` used by list handler | PASS | `equipslot/rest.go:43` `TransformSlice`; `equipslot/resource.go:40` `handleGetEquipSlotExtensions` calls it |
| DOM-08 | POST via `RegisterInputHandler[T]` | PASS | `equipslot/resource.go:26` |
| DOM-11 | Lazy `database.SliceQuery` provider | PASS | `equipslot/provider.go:18` |
| DOM-16 | Writes in `administrator.go` | PASS | `equipslot/administrator.go:24` `Extend`, called from `processor.go:38` |
| DOM-19 | Flat request model | PASS | `equipslot/rest.go:64-69` `ExtendInputRestModel` — flat, no nested Data/Attributes |
| DOM-31 | No tenant on RestModel | PASS | `equipslot/rest.go` — none |
| FILE-01..06 | No catch-all | PASS | one responsibility per file |

### atlas-data/cashpackage (sub-domain-shaped: `resource.go`, no `model.go`, static-XML-backed via shared `document.Storage`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| SUB-01 | Business logic in processor, not handler | PASS | `cashpackage/resource.go:32,60` delegate to `NewProcessor(...)` |
| SUB-02 | No `db.Create`/`Save` in `resource.go` | PASS | none — writes go through `document.Storage.Add` in `processor.go:57` |
| SUB-04 | No manual JSON parsing in `resource.go` | PASS | none found |
| DOM-29 | Registry as application-scoped singleton | PASS | `cashpackage/registry.go:13-18` package-level, `sync.Once`, `GetModelRegistry()` accessor (backed by shared `document.Registry`, matching the established pattern used by every other atlas-data XML-backed domain) |
| FILE-01..06 | No catch-all | PASS | `processor.go`/`reader.go`/`registry.go`/`resource.go`/`rest.go` |

### libs/atlas-packet/model/ring.go (wire codec)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-25 | Client-interpreted bytes resolved from tenant table, not literal | N/A | `RingSet.EncodeField`/`RingRecords.EncodeRecords` write structural presence-flag bytes (`bCouple`/`bFriendship`/`bMarriage`, entry counts) gated on `t.Region()`/`t.MajorVersion()` inline, the same shape every other structural flag byte in this codec family uses — not a dispatcher mode byte, sub-op code, message type, or notice/fail-reason code (the category DOM-25 actually targets per `anti-patterns.md:130-165`) |
| Version-gating documented and IDA-derived | PASS (documented) | `ring.go:14-22,33-39,60-68` cite the specific v83/v87/v95/v48/gms_jms_185 IDA addresses backing every field-order and gating decision |

### atlas-channel/kafka/consumer/cashshop (consumer.go — ring purchase status events)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-31 | Tenant checked before acting on event | PASS | `consumer.go:514-517` `handleStatusEventRingPurchased` checks `t.Is(sc.Tenant())` before invalidating any cache or announcing |
| DOM-24 | Producer stub for tests reaching an emit path | N/A | `consumer.go`'s ring handlers never call `AndEmit`/`message.Emit`/`producer.Produce` — they only announce to the local session socket; no direct or transitive emit call site found in this file |
| Ring cache invalidation cross-tenant scoping | PASS | Both buyer (`consumer.go:519`) and resolved partner (`:522`) invalidations go through `ring.NewProcessor(l, ctx).Invalidate(...)`, tenant derived inside the processor from the same `ctx` the message handler received |

### atlas-cashshop/cashshop/processor_ring.go (PurchaseRingAndEmit)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-30 | Write+emit atomic via outbox `AndEmit`/`message.Buffer` | PASS | `processor_ring.go:83-84,207` |
| DOM-31 | Tenant not exposed, derived from ctx | PASS | `processor_ring.go:201` uses `p.t.Id()` (`t` set from `tenant.MustFromContext(ctx)` in `processor.go:145`, unchanged in this diff) |
| Idempotency claimed before any write | PASS | `processor_ring.go:96-102` `ledger.Claim` is step 1, before any read/write |
| DOM-33 | Mocks updated for `wallet.Processor` interface addition | N/A | No mock of `wallet.Processor` exists anywhere in the service (`find`/`grep` for `Mock struct` under `wallet/` returns nothing) — nothing to go stale |

### Goroutine safety (DOM-26, all touched services)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-26 | Every goroutine via `routine.Go`, bare `go` justified | PASS | `tools/goroutine-guard.sh` run against all 5 touched service modules (`atlas-channel`, `atlas-cashshop`, `atlas-character`, `atlas-data`, `atlas-configurations`), exit code 0, zero findings |

## Security Review

SEC-* family did not fire: no touched package in this diff handles JWT parsing, token revocation, logout-claim resolution, or redirect targets. `atlas-cashshop`/`atlas-channel`/`atlas-character`/`atlas-data` changes in scope are all REST/Kafka domain logic (ring pairs, equip-slot extensions, purchase records, cash packages) with no auth surface touched.

## Not evaluable from the diff

- **DOM-20 (table-driven tests) exhaustive check across all ~30 new/changed `_test.go` files** — spot-verified `ring/cache_test.go` (table-driven `t.Run` subtests) and `equipslot/processor_test.go`; did not read every one of the ~30 changed test files line-by-line to confirm every single one uses the `tests := []struct{}` shape rather than a sequence of ad hoc `t.Run` calls (which is itself an accepted DOM-20 shape per some sibling files but not exhaustively re-verified here).
- **DOM-25 version-fixture byte-pinning for `character/clientbound/{info,spawn,appearance_update}.go` and `character/data.go`** — confirmed the ring codec's *shape* (flag bytes, JMS entry-count wrapping) is documented and IDA-cited, but did not re-derive or independently verify every cited IDA address (`@0x94d600`, `@0x8f05d0`, etc.) against a binary; taking the derivation-doc citations (`ring-field-derivation.md`) as evidence of the process having been followed, not as independently re-verified byte-for-byte.
- **`services/atlas-cashshop/atlas.com/cashshop/kafka/producer/cashshop/producer.go` and `kafka/message/cashshop/kafka.go` (`RingPurchasedBody`, `ErrorEventBody`) full field-by-field review** — read only the call sites in `processor_ring.go` and `consumer.go`; did not read every producer/message file in full for the wallet and package purchase paths that got touched incidentally in the same diff (206-line `kafka/message/cashshop/kafka.go` diff).
- **`services/atlas-cashshop/atlas.com/cashshop/main.go`, `services/atlas-character/atlas.com/character/main.go`, `services/atlas-data/atlas.com/data/main.go` diffs** — not read; would need to confirm the new `equipslot`/`ring`/`cashpackage` `InitResource`/`Migration` wiring is actually registered (build passing is corroborating but not conclusive per DOM-33's own caveat about cross-module compilation).
- **`services/atlas-configurations/atlas.com/configurations/socket/corpus_test.go` (4-line diff)** — not read; small enough to be low-risk but not inspected.
- **`services/atlas-data/atlas.com/data/data/processor.go` and `data/workers/commodity.go` (4-5 line diffs)** — not read.
- **Full JMS/GMS version-matrix coverage in `libs/atlas-packet/model/ring_test.go` (428 lines)** — package tests pass, but did not enumerate every version/region combination pinned to confirm the full matrix (v29..v60 legacy shape, v61-v82, v83+, JMS) is actually exercised rather than a subset.

## Summary

### Blocking (must fix)
- DOM-01: `services/atlas-channel/atlas.com/channel/ring/` has `model.go` but no `builder.go` — `Model` built by raw struct literal in `rest.go:68-79`'s `Extract`, bypassing a validating `Build()`.
- DOM-01: `services/atlas-channel/atlas.com/channel/cashshop/purchaserecord/` has `model.go` but no `builder.go` — same defect.
- DOM-04: `services/atlas-channel/atlas.com/channel/ring/rest.go` defines no `Transform(Model)(RestModel,error)`.
- DOM-04: `services/atlas-channel/atlas.com/channel/cashshop/purchaserecord/rest.go` defines no `Transform(Model)(RestModel,error)`.
- DOM-05: `services/atlas-cashshop/atlas.com/cashshop/ring/resource.go:67` list handler inlines `model.SliceMap(Transform)` instead of calling a `TransformSlice` defined in `rest.go` (none exists).
- DOM-28: `services/atlas-channel/atlas.com/channel/ring/processor.go:107-111` `Populate`'s upstream-fetch failure path logs a bare `Warnf` and never calls `degrade.Observe(...)` — no `atlas_enrichment_degraded_total` signal when a cashshop outage silently empties the ring cache.

### Non-Blocking (should fix)
- None identified beyond the blocking items above; all other checked rules PASS or are legitimately N/A per their own trigger.

### Not evaluable
- 6 items listed above (test-file exhaustiveness, IDA byte-pin re-verification, producer/message body detail, three `main.go` wiring diffs, one small configurations test diff, and full version-matrix enumeration in `ring_test.go`).
