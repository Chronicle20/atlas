## Backend Guidelines Audit

- **Service Path:** services/atlas-effective-stats/atlas.com/effective-stats
- **Branch under review:** task-053-equip-stat-requirements vs main
- **Guidelines source:** `.claude/skills/backend-dev-guidelines/` + `libs/atlas-rest/CLAUDE.md` + `libs/atlas-constants/README.md`
- **Date:** 2026-05-04
- **Build:** PASS (`go build ./...` clean)
- **Vet:** PASS (`go vet ./...` clean)
- **Tests:** PASS — all packages under `atlas-effective-stats/...` green (cached); test packages exercised: `character`, `external/data/equipment`, `external/inventory`, `kafka/consumer/asset`, `kafka/consumer/character`, `stat`
- **Overall:** NEEDS-WORK — build/tests pass, but two EXT-* checks fail in a way that is almost certainly a runtime bug in production (not just a style issue), plus several lower-severity issues.

### Scope notes

- The service is Redis-backed with no GORM entities, no domain `builder.go`/`entity.go`/`administrator.go` files, and a single read-only `GET /worlds/.../characters/.../stats` endpoint. DOM-01..DOM-05, DOM-08..DOM-09, DOM-15..DOM-19 are pre-existing architectural shape and not perturbed by this task; they are reported `n/a` rather than scored against this branch.
- Phase 2 packages relevant to the change: `character` (domain-ish; has `model.go`, `processor.go`, `resource.go`, `registry.go`), `external/data/equipment` (new external HTTP client), `kafka/consumer/asset` (modified — passes templateId through), `kafka/consumer/character` (modified — adds wearer-profile re-gate path).

### External HTTP Client Checklist — `external/data/equipment`

This is a new package that calls another atlas service via `requests.GetRequest[T]` so the EXT-* checklist applies.

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| EXT-01 | JSON:API target struct implements relationship interfaces | **FAIL** | `external/data/equipment/rest.go:9-30` defines `RestModel` with `GetName()=="equipment"` and `GetID/SetID` but does NOT implement `SetToOneReferenceID` or `SetToManyReferenceIDs`. **Two compounding bugs:** (1) atlas-data's actual response uses `type: "statistics"` (not `"equipment"`) — see `services/atlas-data/atlas.com/data/equipment/rest.go:50-52`. (2) atlas-data emits a `slots` toMany relationship — `services/atlas-data/atlas.com/data/equipment/rest.go:67-110` — which `api2go.Unmarshal` requires `SetToManyReferenceIDs` for. The reference correct client lives at `services/atlas-character-factory/atlas.com/character-factory/data/item_requests.go:15,33-34`. The `libs/atlas-rest/CLAUDE.md` document calls out that this exact bug surfaces as a misleading "not found" error and cites task-037 burning the same hole twice. |
| EXT-02 | httptest-backed integration test exists | **FAIL (insufficient)** | `character/initializer_test.go:64-67` and `character/stubs_test.go:173-195` do spin up `httptest.NewServer` for the data service, BUT the stub responses use `"type": "equipment"` and contain NO `relationships` block. The stub matches the *broken* client, not the real upstream. Consequently the integration test gives a false PASS — it cannot catch EXT-01. The package-internal tests in `external/data/equipment/cache_test.go:14-27` swap `defaultFetcher` (a package-level mutable var), which bypasses the entire `requests.GetRequest[T]` → `jsonapi.Unmarshal` decode path. There is no test in this branch that proves a real atlas-data response will decode. |
| EXT-03 | Errors distinguish 404 from other failures | **FAIL** | `external/data/equipment/cache.go:117-122` collapses every fetcher error — connection refused, decode failure (the EXT-01 case), 5xx, malformed JSON, and 404 — into a single `WARN` log + `(zero, false)` return. There is no `errors.Is(err, requests.ErrNotFound)` check. A genuine deploy-time bug (atlas-data unreachable, or the EXT-01 decode failure once a real response is seen) is indistinguishable from "this template id legitimately doesn't exist", and silently downgrades every dependent equipped item to "unqualified" — which is exactly the user-visible regression the design is trying to *fix* in the opposite direction. |
| EXT-04 | Service URL not hardcoded; uses `RootUrl(domain)` | PASS | `external/data/equipment/requests.go:14-23` uses `requests.RootUrl("DATA")`, no hardcoded URL. |

### Domain Checklist — `character` package (the touched files)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` exists | n/a | Pre-existing service shape — no builder file in `character/`; `Model` exposes a `NewModel` constructor + `With*` chain (`character/model.go:127-238`). Not introduced or required by this task. |
| DOM-02 | `ToEntity()` method | n/a | No GORM persistence in this service; Redis-backed via `atlas.TenantRegistry` (`character/registry.go:20`). |
| DOM-03 | `Make(Entity)` function | n/a | Same reason as DOM-02. |
| DOM-04 | `Transform` function | PASS | `stat.Transform` exists and is consumed by `character/resource.go:37`. |
| DOM-05 | `TransformSlice` function | n/a | The single endpoint is a one-record GET, not a list. |
| DOM-06 | Processor accepts `FieldLogger` | PASS | `character/processor.go:44` — `func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor`. |
| DOM-07 | Handlers pass `d.Logger()` | PASS | `character/resource.go:30` — `NewProcessor(d.Logger(), d.Context())`. |
| DOM-08 | POST/PATCH use `RegisterInputHandler` | n/a | Service exposes only one GET handler (`character/resource.go:21`); no POST/PATCH endpoints in this branch. |
| DOM-09 | Transform errors handled | n/a | `stat.Transform` returns no error in current signature (`character/resource.go:37` ignores no error because none is returned). |
| DOM-10 | Test DB has tenant callbacks | n/a | No SQL DB. Redis tests use `miniredis.RunT(t)` (`character/processor_test.go:22`). |
| DOM-11 | Providers use lazy evaluation | n/a | No `provider.go` in changed scope; the new `equipment.Provider` is itself a closure-based provider (`external/data/equipment/cache.go:25,111-125`) and is appropriately lazy. |
| DOM-12 | No `os.Getenv()` in handlers | PASS | Grep of `character/*.go` and `external/data/equipment/*.go` finds zero `os.Getenv` calls (test files do, via `t.Setenv`, which is correct test plumbing). |
| DOM-13 | No cross-domain logic in handlers | PASS | `character/resource.go:25-45` calls only its own processor. |
| DOM-14 | Handlers don't call providers directly | PASS | Handler in `character/resource.go` calls `NewProcessor(...).GetEffectiveStats(...)` only. |
| DOM-15 | No direct entity creation in handlers | n/a | No DB writes anywhere; Redis writes only via Registry (`character/registry.go`). |
| DOM-16 | `administrator.go` exists for write operations | n/a | No SQL persistence layer. |
| DOM-17 | Domain error → HTTP status mapping | PASS (existing) | The single GET handler returns 500 on any error (`character/resource.go:32-34`); acceptable for a read-through cache where the only realistic error is "registry / upstream unavailable". Pre-existing. |
| DOM-18 | JSON:API interface on REST models | PASS | `external/character/rest.go:28-43` (modified — added `JobId` and `Level` fields), `external/data/equipment/rest.go:19-30` both implement `GetName/GetID/SetID`. (Note: equipment also fails EXT-01 — the *additional* relationship interfaces.) |
| DOM-19 | Request models use flat structure | n/a | No request models — read-only service. |
| DOM-20 | Table-driven tests | PASS (where used) | `character/qualification_test.go:16-49` — `TestWearerClassMask_StandardClasses` uses the table pattern. Other tests are scenario-style assertions, which is acceptable for state-machine / iteration logic. |
| DOM-21 | No duplication of atlas-constants types | **PASS, but with one warning** | Confirmed the new types (`WearerProfile`, `EquippedAsset`, `EquipmentRequirements`, `AppliedStats`, `Provider`, `fetcher`) are service-local concepts that don't duplicate `libs/atlas-constants/`. `WearerProfile` correctly stores `job.Id` (`character/wearer.go:9`). `RestModel.JobId` in `external/character/rest.go:17` correctly uses `job.Id`. **Warning (non-blocking):** `character/qualification.go:30,52` invents `wearerClassMask(job.Id) → uint16` representing the v83 class bitmask. This is a domain-meaningful mapping (v83 reqJob bitmask) that is currently service-local; if any other service ever needs to test reqJob, this should move to `libs/atlas-constants/job/`. Not strictly a duplication today — flagged for future. The mapping is also potentially incomplete: `wearerClassMask` returns 0 for `branch == 22` (Evan, jobId/100 == 22 = ids 2200-2218) without a Magician bit even though Evans are magic-class. The comment claims `case 2, 12, 22` covers this, but `jobId/100` for Evan stage1 (2200) = 22 → maps to mask 2 correctly. So Evan is covered. Aran (2100..2112) → 21 → mask 1 (Warrior). OK. **However:** the switch lists `case 0, 10, 20: → 0` (Beginner/Noblesse/Legend) but `Legend.Id == 2000` so `2000/100 == 20` → 0 (correct), and `LegendId` resolves into Aran branch 21 only after first job advance — so this is also OK. PASS overall. |

### Sub-Domain / Domain miscellaneous (immutability + concurrency, per the user's flagged concerns)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| MOD-IMMUT | `Model` immutability via `shallowCopy` + `With*` | PASS | `character/model.go:240-254` defines `shallowCopy()`; every `With*` method calls it. `withQualifiedSnapshot` (`model.go:233-238`) is appropriately package-private. |
| MOD-COPY-EQ | Defensive copy of `equipped` map on mutation | PASS | `character/model.go:256-262` `copyEquipped` is invoked in `WithEquippedAsset` (line 220) and `WithoutEquippedAsset` (line 228). |
| MOD-COPY-EXP | `Equipped()` returns a map copy | PASS | `character/model.go:87-93` builds a fresh map. **Caveat:** the inner `EquippedAsset` value contains a slice of bonuses that is NOT copied here; callers that mutate `equipped[k].bonuses` could corrupt model state. However `EquippedAsset.bonuses` is unexported and `EquippedAsset.Bonuses()` (`equipped_asset.go:30-34`) does return a defensive copy — so the leak surface is contained. PASS. |
| MOD-COPY-QS | `qualifiedSnapshot` is treated read-only | WARN | The cached `qualifiedSnapshot` map (`character/model.go:50`) is shared by reference across `shallowCopy` (line 249) and read by `Bonuses()` (line 106) without copy. This is *documented* as read-only ("Always treated as read-only after construction" — line 51), but `withQualifiedSnapshot` (line 234) accepts the caller's map directly without defensive copy. The two callers (`Recompute` line 355, `RecomputeWith` line 363) own freshly-allocated maps and don't retain references, so the contract holds today. Risky if anyone ever calls `withQualifiedSnapshot` from outside the package — it's package-private, so this is acceptable. |
| MOD-JSON-RT | JSON round-trip for new map fields with string keys | PASS | `character/model.go:377-415` (Marshal) and `:417-463` (Unmarshal) convert `map[uint32]X` ↔ `map[string]X` via `strconv.FormatUint`/`ParseUint`. Test `TestModel_JSONRoundTrip_PreservesWearerAndEquipped` (`character/model_test.go:450-487`) covers wearer + equipped + qualifiedSnapshot. |
| CACHE-LOCK | Equipment cache singleton uses RWMutex + sync.Once | PASS | `external/data/equipment/cache.go:45-60` — `cache` has `sync.RWMutex`; `sync.Once` initializer for the singleton; `get` uses `RLock`; `put`/`reset` use `Lock`. |
| CACHE-TENANT | Cache is tenant-scoped | PASS | Nested `map[uuid.UUID]map[uint32]EquipmentRequirements` keyed by `tenant.MustFromContext(ctx).Id()` (`cache.go:47, 62-82, 113`). Test `TestProvider_TenantIsolation` (`cache_test.go:76-90`) confirms two tenants do not collide. |
| TEST-SEAMS | `SeedForTest` / `ResetCacheForTest` surface size | PASS | Two functions, one populates one entry for the current tenant, one resets all state — minimum viable surface. Both are clearly named with the `ForTest` suffix and have `// Intended for tests` comments (`cache.go:92-105`). The package-level `defaultFetcher` is a mutable `var` swapped via `swapFetcher` in `cache_test.go:14-27`; this is acceptable as the only alternative would be a constructor injection pattern that the rest of the codebase doesn't use. |

### Per-package summary of changes

- `character/model.go` — adds `wearer`, `equipped`, `qualifiedSnapshot` fields + `MarshalJSON`/`UnmarshalJSON`. Pattern adherence: PASS.
- `character/equipped_asset.go` — new immutable value type with defensive copies. PASS.
- `character/wearer.go` — small immutable record. PASS.
- `character/qualification.go` — fixed-point iteration + class mask. Logic looks correct against the design. PASS modulo the DOM-21 future-portability warning above.
- `character/registry.go` — adds `PutEquippedAsset`, `RemoveEquippedAsset`, `SetWearerProfile`. They intentionally do NOT recompute — this is documented (`registry.go:184-186, 212-213`). The Processor wraps them with `RecomputeEquipmentBonuses` (`processor.go:152-174, 177-197, 144-148`).
- `character/processor.go` — adds `SetWearerProfile`, `RecomputeEquipmentBonuses`, plumbing for templateId, wires `equipment.GetProvider(p.l)` into recompute. PASS.
- `character/initializer.go` — orders fetch → snapshots → buffs → passives → `RecomputeWith` → mark initialized. PASS.
- `external/data/equipment/cache.go,requests.go,rest.go` — new client. **Two FAILs** (EXT-01, EXT-03) plus one EXT-02 weakness (test stub doesn't reflect the real upstream shape).
- `kafka/consumer/asset/consumer.go` — passes `templateId` into `AddEquipmentBonuses`. PASS.
- `kafka/consumer/character/consumer.go` — adds wearer-profile re-gate when `LEVEL` / `JOB` updates appear. PASS.
- `external/character/rest.go` — adds `Level` and `JobId` fields. PASS.

### Summary

#### Blocking (must fix)

- **EXT-01 (`external/data/equipment/rest.go:19`)** — `RestModel.GetName()` returns `"equipment"`; atlas-data emits `type: "statistics"` (`services/atlas-data/atlas.com/data/equipment/rest.go:50-52`). On its own, an api2go type-mismatch will produce a decode error; combined with the missing relationship interfaces it is guaranteed to fail against the real upstream. Fix: rename to `"statistics"` AND add the no-op stubs:
  ```go
  func (r *RestModel) SetToOneReferenceID(_, _ string) error             { return nil }
  func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error  { return nil }
  ```
  (Reference implementation: `services/atlas-character-factory/atlas.com/character-factory/data/item_requests.go:15,33-34`.)
- **EXT-03 (`external/data/equipment/cache.go:117-122`)** — `defaultFetcher` errors are not classified. Wrap the error and route on `errors.Is(err, requests.ErrNotFound)` so a missing template id can be cached as "intentionally absent" while transport / decode / 5xx errors are not silently treated as `not qualified`. Without this, the EXT-01 bug — and any future deploy bug — silently degrades every player's equipment to "unqualified" instead of surfacing as an error.

#### Non-blocking (should fix)

- **EXT-02 (`character/initializer_test.go:28-62`, `character/stubs_test.go:173-195`)** — the data stub responds with `"type": "equipment"` and no `relationships` block, mirroring the broken client. To actually exercise the production decode path, change the data stub to mirror the real atlas-data shape (`type: "statistics"`, with a `relationships.slots` block). Once EXT-01 is fixed and the stub is realistic, the integration test will be a real regression net for the next person who edits the client.
- **DOM-21 (`character/qualification.go:30-48`)** — `wearerClassMask` is a domain-meaningful (v83 reqJob bitmask) mapping. It is currently service-local and not duplicated, so this is not a hard violation. If any other service grows a need for the same mask, promote it to `libs/atlas-constants/job/`.

#### Confirmed PASS (against the user's flagged concerns)

- Immutability of `Model` via `shallowCopy` + `With*` methods (`character/model.go:240-254`).
- Defensive copy on `Equipped()` map exposure (`character/model.go:87-93`) and on `EquippedAsset.Bonuses()` (`character/equipped_asset.go:30-34`).
- `RWMutex` + `sync.Once` discipline on the equipment cache singleton (`external/data/equipment/cache.go:45-60`).
- JSON round-trip correctness for `equipped` and `qualifiedSnapshot` via string-keyed map encoding (`character/model.go:377-463`, test `:450-487`).
- Tenant scoping of the equipment cache via nested `map[uuid.UUID]map[uint32]...` keyed by `tenant.MustFromContext(ctx).Id()` (`cache.go:47, 62-82, 113`; test `:76-90`).
- `SeedForTest` / `ResetCacheForTest` surface is minimal and clearly named (`cache.go:92-105`).
- `wearerClassMask` reqJob mapping (`character/qualification.go:30-48`) is correct for all in-use jobIds (Explorer 100/200/300/400/500, Cygnus 1100..1500, Aran 2100..2112, Evan 2200..2218, Beginner/Noblesse/Legend → 0).
