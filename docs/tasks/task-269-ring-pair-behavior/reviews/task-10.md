# Review: Task 10 — atlas-channel/ring cache and GetRingSet

Range reviewed: `41cb9b35c..08c042550` (single commit
`feat(atlas-channel): per-tenant ring cache and GetRingSet`).

Scope confirmed: exactly the four files named in the brief —
`ring/cache.go`, `ring/cache_test.go`, `ring/processor.go`,
`ring/processor_test.go` (716 insertions, 0 deletions, per
`git diff --stat 41cb9b35c..08c042550`). No other file touched. Read-only
dependencies (`ring/requests.go`, `ring/rest.go`, `ring/model.go`,
`listener/evict.go`, `equipment/*`) predate this commit (Task 9) and were read
only to verify the call chain, not re-reviewed.

## 1. Unused-symbol enforcement — PASS

Traced the call chain by hand:

- `processor.go:78-84` `upstreamFn` calls `requestByCharacterId(ctx, characterId)`
  (`requests.go:23`) and pipes the result into
  `requests.DrainProvider[RestModel, Model](l, ctx)(url, 250, Extract, ...)`.
- `requestByCharacterId` (`requests.go:23-29`) calls `getBaseRequest(ctx)`
  (`requests.go:14-16`).
- `Populate` (`processor.go:86-95`) calls `upstreamFn`, and `Populate` is a
  `Processor` interface method backed by real tests.

Verified with `go build ./ring/...` and `go vet ./ring/...` — both clean, no
unused-symbol diagnostics. `TestPopulateDecodesRealJSONAPIDocument` and
`TestPopulateFailsSoftOnCashshopOutage` (`processor_test.go:322,358`) drive
this through a real `httptest.Server`, so the fetch is a genuine REST call
through `requests.DrainProvider`, not a token reference kept alive only for
the compiler. This resolves the prior gate failure for real.

## 2. FR-15 selection rule — PASS

Implementation (`processor.go:117-135`, `selectPair`): iterates
`eq.Slots()`, filters to `ringSlotPositions` (`-12,-13,-15,-16`) with a
non-nil `CashEquipable`, matches against ACTIVE halves of the requested
`Type` by cash id, and picks the entry with the numerically **highest**
(least negative) position, i.e. `ring1 (-12)` before `ring2 (-13)` before
`ring3 (-15)` before `ring4 (-16)`; ties broken by lower `CashId()`.

- `s.Position > bestPosition` (line 131) is exactly "ring1-first" — I
  fault-injected the operator to `<` and reran `go test ./ring/... -run
  TestGetRingSet`; only `two_couple_halves,_lowest_slot_wins_(FR-15)` failed
  (`Couple.OwnSN=1111`, want `7777`), confirming the test genuinely pins this
  ordering rather than tolerating either direction. Reverted; tree confirmed
  clean afterward (`git status --porcelain`).
- Tie-break: `slot_tie_broken_by_cashId` subtest
  (`processor_test.go:244-266`) exercises two distinct slot keys pinned to
  the same position (`-12`), asserting the lower cash id wins.
- The doc comment on `Processor.GetRingSet` (`processor.go:43-52`) states the
  rule explicitly, including the prose-vs-example discrepancy resolution
  (matches the report's self-disclosed correction), so it will not be
  re-litigated by a later reader.

## 3. `Populate` split — PASS, coherent

`GetRingSet` (`processor.go:102-113`) touches only `getRingCache().lookup`
and `selectPair` — grepped for `requests\.|http\.|upstreamFn` inside
`processor.go` and the only hits are in `Populate`/`upstreamFn` themselves
(lines 73-95), confirming `GetRingSet` is genuinely REST-free and cache-only.
`Populate` is the sole method that performs I/O. The split matches the
brief's REST-free-`GetRingSet` requirement and slots cleanly into Task 12's
expected "population entry point." Not literally named in the brief's
"Produces" list, but the implementer flagged this explicitly in the report
(§"Concerns for the controller") — a transparent, justified addition, not a
silent scope expansion.

## 4. `init()`-registered `EvictTenant` — judgment: acceptable, with a caveat worth recording

- `cache.go:91-95` registers `EvictTenant` via `listener.RegisterEvictor` in
  `init()`, matching the brief's explicit instruction and the precedent
  blessed at `listener/evict.go:22-23` ("Safe to call from `init()` of any
  package that holds tenant-scoped state").
- **No double-registration risk**: Go guarantees a package's `init()` runs
  exactly once regardless of how many other packages import it.
- **No ordering hazard relative to `main.go`'s central closure**
  (`main.go:299-310`): every package's `init()` completes before `main()`
  begins executing, and the central closure is registered *inside* `main()`
  via a `listener.RegisterEvictor` call in the function body. So whenever
  `ring` ends up imported into the binary, its evictor will always occupy an
  earlier slot in `evictors` than the central closure, deterministically —
  not the reverse, and not race-dependent. `fireEvictorsForTenant`
  (`listener/evict.go:30-38`) snapshots and calls each in registration order,
  and none of the central closure's per-cache evictions depend on `ring`'s
  cache or vice versa, so relative order carries no functional risk here.
- **Caveat, worth recording rather than treating as resolved**: as of this
  commit, nothing in the `atlas-channel` module imports `atlas-channel/ring`
  outside of the package's own files (`grep -rln '"atlas-channel/ring"'`
  across all non-test `.go` files returned nothing, and a repo-wide grep for
  `atlas-channel/ring` including tests also returned nothing). That means
  `ring`'s `init()` is not yet linked into the `atlas-channel` binary at all
  — `EvictTenant` is not actually registered or reachable in production
  today. This is expected and correct given the task boundary (Tasks 11/12
  will add the import), not a defect in this commit, but it means point 4's
  question "does the cache actually get evicted on tenant drain" has the
  honest answer "not yet, and this task alone cannot make it so." Recommend
  the controller confirm, once Task 11 or 12 lands, that a full-binary check
  (e.g. `main.go`'s eviction block or a startup grep) still shows `ring`'s
  evictor firing — the divergent `init()` registration path means it will
  never show up in `main.go`'s single audit point, so it is easy to miss in
  a future refactor of that block. Non-blocking for this task.

## 5. FR-5 fail-soft — PASS, fault-injected

Fault-injected `Populate` to `return err` directly instead of warn-and-swallow
(`processor.go:86-95`), reran `go test ./ring/... -run
'TestGetRingSet/upstream_error|TestPopulateFailsSoftOnCashshopOutage'`: both
failed as expected —
`Populate() = cashshop unreachable, want nil` and
`Populate() = after 3 attempts, ..., want nil`. Reverted the file
(`cp` from a pre-edit backup) and confirmed `git status --porcelain` clean
before continuing. The tests genuinely pin fail-soft behavior, not merely
exercise it.

The warn-log assertion (`processor_test.go:282-295`) also checks the message
contains the character id, matching the brief's requirement.

## 6. Cache correctness

- **Tenant isolation**: `cache.go:45-65`, `perTenant map[uuid.UUID]map[uint32]cacheEntry`
  keyed correctly; `TestRingCacheTenantIsolation/tenant_isolation` puts
  distinct entries under two tenant UUIDs at the same character id and
  asserts both are independently retrievable. Genuine, not tautological —
  it exercises the real nested-map keying, and a bug that collapsed the two
  tenant maps would fail it.
- **RWMutex discipline**: `lookup` takes `RLock`, `put`/`invalidate`/`EvictTenant`
  take `Lock`; all four use `defer` unlock immediately after acquiring,
  standard and correct for this shape. No lock is held across a REST call
  (`Populate` computes `halves` outside any lock, then calls `put` which
  takes its own lock internally) — no risk of blocking readers on I/O.
  `go test -race ./ring/...` was not run (guard exclusion for this session:
  `--quick`-equivalent scope; `-race` is part of the full `verify.sh` gate,
  not this review), but the lock discipline read is unambiguous by
  inspection.
- **Tautology check**: none of the cache tests assert a value the production
  code trivially satisfies regardless of correctness (e.g. asserting `ok ==
  ok`). Each subtest fault-injectable region (verified above for FR-15 and
  FR-5) demonstrably fails when the underlying logic is wrong. I did not
  separately fault-inject the four `TestRingCacheTenantIsolation` subtests,
  but they call the same `lookup`/`put`/`invalidate`/`EvictTenant` primitives
  directly (white-box `package ring` test) and assert on returned values, so
  a broken nested-map implementation would fail them by construction.

## Non-blocking findings

1. `Processor.Invalidate` (`processor.go:97-100`) has no test at the
   `Processor` level — only the underlying `ringCache.invalidate` is tested
   directly (`cache_test.go:75-92`). `Invalidate` is a two-line wrapper
   (`tenant.MustFromContext` + `cache.invalidate`), so risk is low, but a
   regression that mis-threaded the tenant id here (e.g. always used a fixed
   tenant, or called `EvictTenant` instead of `invalidate`) would not be
   caught by any current test in this package. Recommend one thin
   `Processor.Invalidate` test in whichever task first calls it (or now, at
   low cost).
2. The `init()`-registration/central-closure divergence (finding 4 above) —
   recorded as a note for whoever wires Task 11/12's import and for anyone
   later auditing `main.go`'s eviction block, since it will not appear there.

## Not evaluable

- `-race` behavior was not directly verified in this review (see §6); the
  full `tools/verify.sh` gate covers that and was intentionally not run here
  per the task's constraints (a gate run concurrently against the working
  tree).
- Whether `ring`'s `init()` actually gets linked in and fires correctly once
  Task 11/12 add the import is not evaluable from this commit alone — flagged
  as a forward-looking caveat in finding 4, not a defect of this unit.

## Verdict rationale

All six focus areas resolve to PASS or an accepted judgment call, with two
non-blocking notes recorded for forward tasks. No blocking defect found in
this unit's scope.
