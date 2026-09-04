# Review — Task 20: `atlas-maker` recipe cache and its indexes

Range reviewed: `31253c02a..06b48f3` (single commit `06b48f362`).

```
services/atlas-maker/atlas.com/maker/recipe/mock/processor.go       |  36 +++
services/atlas-maker/atlas.com/maker/recipe/model.go                | 108 +++++++++
services/atlas-maker/atlas.com/maker/recipe/processor.go            | 204 ++++++++++++++++
services/atlas-maker/atlas.com/maker/recipe/processor_test.go       | 267 +++++++++++++++++++++
4 files changed, 615 insertions(+)
```

Scope matches the brief exactly: four new files under `recipe/`, nothing else touched. No scope mismatch.

## 1. Constraint C-6 (group-0-only leftover index)

Verified in source, not just test names.

- `services/atlas-maker/atlas.com/maker/recipe/processor.go:89-93` — the build loop only inserts into `ti.byLeftover` when `rm.Group() == crystallizationGroup` (`crystallizationGroup = uint32(0)`, line 30). A non-zero-group recipe is never written into `byLeftover`, full stop — there is no fallback path that consults `byItemId` or scans all recipes if the group-0 lookup misses.
- `GetByLeftover` (processor.go:186-196) does a single map lookup against `ti.byLeftover` and returns `ErrNoCrystalMapping` on miss — no fallthrough to any other index.
- `TestGetByLeftoverIgnoresNonZeroGroups` (processor_test.go:174-204) is the executable proof: seeds a group-0 recipe (`4260000`, material `4000000`) alongside a group-1 recipe (`1382005`) that also lists material `4000000`; asserts `GetByLeftover(4000000)` resolves to `4260000`. It then rebuilds under a **fresh tenant** with only the group-1 recipe present and asserts `GetByLeftover(4000000)` returns `ErrNoCrystalMapping` — not the group-1 recipe. Using a fresh tenant for the second half is deliberate and correct: it prevents the process-wide cache from the first half masking a broken filter.
- I hand-traced what would happen if the `rm.Group() == crystallizationGroup` guard were removed: the fixture order in the test (`groupZero` then `groupOneCollision`) means `groupOneCollision` would overwrite the `4000000` key in `byLeftover`, and the first assertion (`m.Id() == 4260000`) would fail. The test is therefore falsifiable, not tautological.

C-6: **PASS**. This is the load-bearing constraint and it holds under both static reading and test execution (`go test ./recipe/... -race` — PASS, see §6).

## 2. Contract for Tasks 21-23

All accessors and methods from the brief's Produces list exist with the exact names and signatures:

- `Model`: `Id() item.Id`, `Group() uint32`, `ReqLevel() uint32`, `ReqSkillLevel() uint32`, `ItemNum() uint32`, `Tuc() uint32`, `Meso() uint32`, `Catalyst() item.Id`, `ReqItem() item.Id`, `ReqEquip() item.Id`, `Materials() []Material`, `RandomRewards() []Reward`, `QuestRequirements() []QuestRequirement` — all present in `recipe/model.go:47-107`, types matching exactly (note `Catalyst`/`ReqItem`/`ReqEquip` are correctly `item.Id`, converted from the upstream's plain `uint32` at `processor.go:121-123`).
- `Processor`: `GetById(item.Id) (Model, error)`, `GetByLeftover(item.Id) (Model, error)`, `GetAll() ([]Model, error)` — `processor.go:132-142`, implemented on `ProcessorImpl` at lines 174-204.
- `Material{ItemId item.Id; Count uint32}`, `Reward{ItemId item.Id; ItemNum uint32; Prob uint32}`, `QuestRequirement{QuestId uint32; State uint32}` — `model.go:6-27`, field names and types match the brief's literal syntax.

Contract: **PASS**, no drift.

## 3. Cache correctness

- `sync.RWMutex` is used (`processor.go:47`), not `sync.Once` — matches the brief's explicit ruling.
- Tenant scoping is real: the cache key is `tenant.MustFromContext(p.ctx).Id()` (a `uuid.UUID`), a `map[uuid.UUID]*tenantIndex` (line 48). `TestIndexesAreTenantScoped` (processor_test.go:225-244) seeds two tenants with disjoint one-recipe catalogs and asserts each processor returns not-found for the other tenant's item id — genuine cross-tenant isolation, not just "different context passed."
- `Invalidate(tenantId uuid.UUID)` (processor.go:58-62) takes the write lock and deletes the tenant's map entry — a real clear, not a soft flag. `TestIndexIsBuiltOnceUntilInvalidated` (processor_test.go:246-267) counts upstream `GetAll()` calls: 1 call across three lookups (`GetById`/`GetByLeftover`/`GetAll`), then 2 after `Invalidate` and one more lookup. This is a genuine falsifying test — if `Invalidate` were a no-op, the count would stay at 1.
- Locking: `get()` takes `RLock`, `build()` takes `Lock` and double-checks the map before rebuilding (processor.go:76-78) so a losing goroutine's build is discarded rather than clobbering the winner's. The window between `RUnlock` (miss) and the write `Lock` in `build()` is not itself a data race — no shared mutable state is touched by `p.im.GetAll()` outside the lock, only the local `ms` slice. Ran `go test ./recipe/... -race -count=1`: **PASS**, no race reported.
- One inefficiency (non-blocking): the unlocked window means two concurrent goroutines racing on a cold cache both call `p.im.GetAll()` (the upstream HTTP round-trip) before `build()` de-dupes; this is a lost-update-avoided-but-duplicate-work pattern, not a correctness bug, and is outside what the brief's `sync.RWMutex`-not-`sync.Once` language required.

Cache correctness: **PASS**.

## 4. List order preservation / immutability leak

- Order preservation: `modelFromItemMake` (processor.go:99-128) builds `materials`/`rewards`/`quests` via a single ordered `for _, mat := range ...` append loop each, with no sort call anywhere in the package (confirmed by `grep -rn sort` — no hits). `TestGetByIdReturnsRecipe` (processor_test.go:123-150) asserts `Materials()[0]`, `[1]`, `[2]` element-by-element against the fixture's declared order, which would fail if anything reordered the list (e.g., a stray sort by item id would move `4021007` before `4011002`). **PASS**.
- Mutability leak (non-blocking finding): `Materials()`, `RandomRewards()`, `QuestRequirements()` (model.go:92-107) and `GetAll()` (processor.go:198-203) all return the cache's own backing slice directly, with no defensive copy. A caller in Task 21-23 that does `m.Materials()[0].Count = 0` would mutate the shared cached `Model` for every subsequent tenant lookup, since `recipe.Model` stores slice fields by reference and the same `Model` value is served from the `byItemId`/`byLeftover`/`all` maps/slice to every caller. `Material`/`Reward`/`QuestRequirement` are plain value structs (not pointers), so a caller cannot mutate the struct via the returned slice's *elements* without an assignment through the slice (e.g., `s[i].Count = 0`), but that assignment absolutely does reach back into the cache because slices are reference types over their backing array. Neither the brief nor design.md's C-6 text mandates a copy-on-read here, and no test in this task exercises mutation-through-accessor, so this is not a violation of a stated requirement — flagging as **non-blocking**, since Tasks 21-23 are read-only consumers per their own briefs' Produces sections, but the risk is real and worth a one-line note for whoever writes Task 21-23 if they ever need a mutable working copy.

## 5. Upstream consumption — full catalog, not truncated

- `services/atlas-maker/atlas.com/maker/data/itemmake/processor.go:33-39` — `itemmake.Processor.GetAll()` calls `requests.DrainProvider[RestModel, Model](p.l, p.ctx)(url, 250, Extract, model.Filters[Model]())()`, i.e., it drains every page (page size 250) via `DrainProvider`, not a single `GetRequest`. `data/itemmake/requests.go:28-36` has an explicit comment confirming `/data/item-makes` is paginated (task-117 rollout) and that a bare `GetRequest` would silently truncate.
- `recipe.ProcessorImpl.ensureIndex` (processor.go:161-172) calls `p.im.GetAll()` — the paginated, draining `Processor.GetAll()`, not `GetById` in a loop or any single-page call. There is no re-implementation of pagination inside `recipe/` that could diverge from Task 19's drain logic.
- This confirms the carried ruling: the recipe index reads the full catalog. **PASS**.

## 6. No call to `GET /items/{itemId}/drops`

`grep -rn "drops\|/items/" recipe/` — no hits (the only "drops" match is the English word in the `Invalidate` doc comment, unrelated). The package's only upstream dependency is the injected `itemmake.Processor`. **PASS**, matches OQ-3's resolution and the report's claim.

## 7. Independent build/test re-run

Ran from `services/atlas-maker/atlas.com/maker`:

```
$ go build ./...
(no output — success)

$ go test ./... -count=1
ok  	atlas-maker	0.020s
ok  	atlas-maker/character	0.026s
ok  	atlas-maker/compartment	0.030s
ok  	atlas-maker/crystalband	0.061s
ok  	atlas-maker/data/equipment	0.032s
ok  	atlas-maker/data/itemmake	0.030s
ok  	atlas-maker/quest	0.039s
ok  	atlas-maker/reagent	0.053s
ok  	atlas-maker/recipe	0.016s
ok  	atlas-maker/seed	0.024s
ok  	atlas-maker/skill	0.040s
(all others: [no test files])

$ go test ./recipe/... -count=1 -race
ok  	atlas-maker/recipe	1.027s

$ gofmt -l recipe/
(no output)

$ go vet ./recipe/...
(no output)
```

All claims in the implementer report are confirmed independently, including the `-race` run (which the report did not claim to have run but which passes clean).

## 8. Repo conventions

- `libs/atlas-constants`: only `item.Id` is used, reused from the existing library; no new domain type or numeric constant defined outside the package's own `crystallizationGroup` sentinel (a package-local constant, not a domain constant — appropriate).
- Tenancy: `tenant.MustFromContext(p.ctx)` (processor.go:162), the standard pattern used elsewhere in the service (`data/skill/cache.go` precedent cited in the report — confirmed present at `services/atlas-channel/atlas.com/channel/data/skill/cache.go:20`).
- Builder pattern / no test-only constructors: `processor_test.go` uses `tenant.Create(...)` (the library's own constructor) and ordinary `t.Helper()` functions (`mustExtract`, `sixGroupFixture`, `newProcessor`, `newCountingProcessor`) — no `*_testhelpers.go` file, no hand-rolled unexported-field struct literals bypassing the model's own constructors (`itemmake.Extract` is used to build fixtures, the real decode path).
- Mocks: `recipe/mock/processor.go` follows the same `ProcessorMock` shape (overridable `...Func` fields, `var _ Processor = (*ProcessorMock)(nil)`) as the pre-existing `data/itemmake/mock/processor.go`.
- No `libs/`-level changes, no new REST surface, no DB surface despite the fact block flagging `db_surface=true` for the overall task-285 branch — this specific commit adds no table/entity, consistent with the "compute-only domain" pattern it copies from `reward-pools/reward/`.

Conventions: **PASS**.

## Not evaluable

None — the full review surface (four new files, plus the one upstream file — `data/itemmake/processor.go` — whose contract this unit's correctness depends on) was read and checked against runnable evidence.

## Verdict

APPROVED. One non-blocking note (§4, accessor mutability) for downstream awareness; no blocking findings.
