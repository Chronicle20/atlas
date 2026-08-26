# Review — Task 4: `data/portal` model accessors and per-map tenant-scoped cache

Range: `122fe88c1..1182985b8` (`8da96de49` implementation + `1182985b8` controller-ordered
dead-code removal on top).

Brief: `.superpowers/sdd/plan/task-4-brief.md`
Implementer report: `.superpowers/sdd/plan/reports/task-4-report.md`

## Scope reviewed

`services/atlas-channel/atlas.com/channel/data/portal/{model.go,requests.go,processor.go,processor_test.go}`,
diffed against the pattern in `services/atlas-channel/atlas.com/channel/data/map/processor.go:34-73`
and the callers/mock in `data/portal/mock/processor.go` and `portal/processor.go:40`.

`git diff --stat 122fe88c1..1182985b8`:

```
model.go         |  28 ++
processor.go      |  71 ++++-
processor_test.go | 136 +++++
requests.go       |   7 +-
```

## Findings

### 1. Model accessors (Step 2) — PASS

`model.go:20-46` adds `Name()`, `Target()`, `Type() uint8`, `X() int16`, `Y() int16`,
`TargetMapId() _map.Id`, `ScriptName()` as value-receiver getters over the existing
private fields (`name`, `target`, `portalType`, `x`, `y`, `targetMapId`, `scriptName`).
Types match `Extract` in `rest.go:36-47` exactly (`RestModel.Type uint8`, `X/Y int16`,
`TargetMapId _map.Id`). No setters added — consistent with the repo's immutable-model
convention (construction only through `Extract`). `Id()` was pre-existing.

`TestModelAccessors` (`processor_test.go:115-136`) exercises all seven against the
brief's exact fixture and passes (`go test -run TestModelAccessors -v`, confirmed).

### 2. `requestInMap` whole-list request (Step 3) — PASS

`requests.go:19-25` adds `requestInMap(ctx, mapId) requests.Request[[]RestModel]`
built from `portalsInMap` (`data/maps/%d/portals`), same shape as the removed
`requestInMapByName`.

### 3. Cache shape vs `data/map/processor.go:34-73` — PASS

`processor.go:42-79` mirrors the pattern precisely:
- `cacheKey{tenantId uuid.UUID; mapId _map.Id}` — identical shape to `data/map`'s key.
- `portalCache sync.Map` / `portalLoadMu sync.Map` — same two-map double-checked-lock
  structure as `mapCache`/`mapLoadMu`.
- `getPortalsInMap`: load → miss → `LoadOrStore` a per-key `*sync.Mutex` → lock →
  re-check under lock → fetch → store → unlock (via `defer`). Same sequence as
  `data/map/processor.go:52-75`, generalized from a single `Model` to `[]Model`.
- Comment at `processor.go:49-50` ("static WZ data, cached for the process lifetime —
  pod restart is the invalidation contract") mirrors `data/map`'s framing
  (`data/map/processor.go:45-46`).
- Tenant scoping: `tenant.MustFromContext(p.ctx)` keys every lookup by `t.Id()`, so two
  different tenant contexts against the same `mapId` never share a cache entry — verified
  by `TestGetInMapByName_TenantScoped` (2 fetches, 2 tenants, same `mapId`).

No cross-tenant bleed path found: the only cache write site is `processor.go:77`, gated
by `key` which always includes `t.Id()`.

### 4. `GetInMapByName` filter + not-found path — PASS

`processor.go:95-106`: loads the cached list via `getPortalsInMap`, linear-scans for
`m.Name() == name`, returns the first match or
`fmt.Errorf("no portal named [%s] in map [%d]", name, mapId)`. `InMapByNameModelProvider`
(`processor.go:81-93`) filters the same cached list into a slice and wraps it in
`model.FixedProvider`, or `model.ErrorProvider` on a load failure (report flags this as
an intentional, behaviorally-equivalent deviation from letting `requests.SliceProvider`
own the error path — reasonable, since the cache lookup now happens outside the provider
closure).

`TestGetInMapByName_NotFound` (`processor_test.go:89-113`) asserts a non-nil error and
that `calls` stays at 1 — i.e., a filter miss on a populated cache does not trigger a
second REST fetch. This is the correct not-found semantics per the brief.

### 5. Interface/mock/caller compatibility — PASS

`Processor` interface (`processor.go:17-20`) is unchanged (`InMapByNameModelProvider`,
`GetInMapByName` — same signatures). Confirmed no other caller or the mock needed edits:

```
data/portal/mock/processor.go:11,12,17-26 — unchanged, still compiles against the interface
portal/processor.go:40 — pm, err := p.pd.GetInMapByName(f.MapId(), portalName) — untouched
```

### 6. Dead-code removal (`1182985b8`) — controller ruling, verified clean

`requestInMapByName` and the `portalsByName` const are gone from `requests.go`.
Repo-wide grep for both names returns zero matches — no stray caller left behind:

```
grep -rn "requestInMapByName\|portalsByName" . → no matches
```

`requestInMap` / `portalsInMap` (the still-used pair) are untouched by this commit. This
matches the task instructions: not reporting this as an unrequested change, only
confirming it left no callers and didn't break the build — which it didn't
(`go build ./...` exit 0, `go test ./data/portal/...` all PASS after the removal, per both
my own run and the report's).

### 7. Test honesty — PASS

- `TestGetInMapByName_CachesWholeList` — two different-name lookups, same
  tenant+mapId, asserts `calls == 1`. Without the cache (old REST-per-call code) this
  would be 2 — genuinely pins the caching behavior, not just the happy path.
- `TestGetInMapByName_TenantScoped` — two `testCtx()` calls each mint a fresh
  `uuid.New()` tenant (`processor_test.go:17-23`), so this also self-isolates against the
  package-level cache leaking across test runs, per the brief's guidance. Asserts
  `calls == 2`.
- `TestGetInMapByName_NotFound` — asserts error non-nil and `calls` still 1 after the
  miss, ruling out a "just refetch and still not find it" bug.
- Ran the full package suite fresh: `go test -run . -count=1 ./data/portal/... -v` — all
  4 tests PASS.

The REST leg is stubbed via a package-level `requestInMapFn` seam
(`processor.go:37-40`), following the `monsterByIdFn` precedent named in the brief —
not an `httptest.Server`, which the report explains is because no such fake-`atlas-data`
helper exists elsewhere in this module (checked, not asserted).

### 8. Build/lint hygiene — PASS

```
gofmt -l data/portal/*.go   → clean
go vet ./data/portal/...    → clean
go build ./...              → exit 0 (module root)
go test ./data/portal/... -v → 4/4 PASS
```

## Not evaluable

None — the whole surface (model.go, requests.go, processor.go, processor_test.go, the
mock, and the one production caller in `portal/processor.go:40`) was inspected directly.

## Verdict rationale

All five brief steps are implemented as specified, the cache pattern is a faithful,
correctly tenant-scoped copy of `data/map/processor.go:34-73`, the not-found path is
correct and cheap (no second fetch), the interface/mock/caller surface is untouched as
promised, and the tests genuinely pin cache-hit/miss/tenant-isolation behavior rather
than just exercising the happy path. The controller-ordered dead-code removal left no
stray callers and both commits build and test clean. No blocking or non-blocking
findings.
