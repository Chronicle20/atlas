# Backend Audit — task-071-gamedata-minio-consolidation

- **Worktree:** `.worktrees/task-071-gamedata-minio-consolidation` (relative to repo root)
- **Review range:** `19d00ed0868cdc8dfe7c2487e5b79ecc4e6943b9..2527c45415b5380195379fc85dc7580fd985cc2d` (74 commits)
- **Guidelines Source:** `.claude/skills/backend-dev-guidelines/`
- **Date:** 2026-05-22
- **Build:** PASS (libs/atlas-wz, services/atlas-data, services/atlas-renders)
- **Tests:** PASS (`go test ./...` and `go test -race ./...` clean across all three modules)
- **Overall:** NEEDS-WORK

Builds and tests pass cleanly, so this is not a hard FAIL — but multiple
DOM-21 (shared-lib reuse) violations and at least one latent concurrency
hazard worthy of attention before merging.

## Build & Test Results

```
libs/atlas-wz                                  go build PASS, go test -race PASS (11 pkgs ok)
services/atlas-data/atlas.com/data             go build PASS, go test -count=1 PASS (37 pkgs ok)
services/atlas-renders/atlas.com/renders       go build PASS, go test -race PASS (5 pkgs ok)
go vet ./... on services/atlas-data            PASS (no findings)
```

## Scope

This audit covers only the Go-source changes called out in the dispatch
prompt: `libs/atlas-wz/`, `services/atlas-data/atlas.com/data/`,
`services/atlas-renders/atlas.com/renders/`, and `tools/cideps/main.go`.
Per dispatcher instructions, mechanical go.mod/go.sum churn,
`services/atlas-assets/` + `services/atlas-wz-extractor/` deletion, and
deploy/docs artifacts are explicitly out of scope.

Because the changed surface is overwhelmingly action-event workers,
parser libraries, and operational REST endpoints — not new GORM-model
domains — the conventional DOM-01..DOM-20 checklist does not apply
cleanly. The closest analogues (`baseline`, `tenantpurge`, `wzinput`,
`runtime/rest`) all live in support packages without `model.go`, so I
ran the relevant DOM checks against each of them by hand and reserved
the DOM-21 / DOM-22 / DOM-23 / SEC checklist for the substantive code
paths.

## Findings — Blocking (must fix)

### DOM-21 — shared-lib reinvention (`libs/atlas-constants/item`)

Three independent reinventions of the v83 item-classification arithmetic
land in `services/atlas-renders/atlas.com/renders/character/composite.go`
even though `libs/atlas-constants/item/constants.go` already supplies
typed constants and a helper:

- `services/atlas-renders/atlas.com/renders/character/composite.go:61-68`
  re-implements `isTwoHandedItem` with a comment explicitly admitting the
  duplication: *"Donor uses `libs/atlas-constants/item.IsTwoHanded`; we
  replicate inline so atlas-renders does not pick up a new transitive
  dependency for one check."* The library function exists at
  `libs/atlas-constants/item/constants.go:186` and is unit-tested
  (`constants_test.go:39-64`). Avoiding a transitive dependency is not
  a reason DOM-21 accepts.
- `services/atlas-renders/atlas.com/renders/character/composite.go:90-122`
  reinvents `partClassFor(id)` on `id / 10000` — duplicates the
  Classification enum at `libs/atlas-constants/item/constants.go:13-27`
  (ClassificationHat=100, …, ClassificationShield=109,
  ClassificationCape=110, …) and the classifier
  `item.GetClassification` at `constants.go:121`.
- `services/atlas-renders/atlas.com/renders/character/composite.go:169-196`
  reinvents `slotForItemID(id)` against the same `id / 10000` ranges.

DOM-21 status: **FAIL** for all three locations. atlas-renders must take
the transitive `libs/atlas-constants/item` dependency and call
`item.GetClassification` / `item.IsTwoHanded`. Atlas-renders already
imports `libs/atlas-tenant`, so the slippery-slope objection in the
comment isn't real.

A fourth reinvention lives in the shared library itself:
`libs/atlas-wz/charparts/extract.go:97-108` (`accessoryPartClassFor`).
This is in a lib not a service so DOM-21 doesn't textually apply, but
the spirit does — flagged as **Minor** below.

## Findings — Non-Blocking (should fix)

### CONCURRENCY-01 — `*wz.File.reader` is NOT goroutine-safe; atlas-renders shares one `*wz.File` across concurrent map renders

This is the most serious latent bug I found. Severity is "non-blocking"
only because the test suite never exercises concurrent renders against
a single `*wz.File` — production traffic will.

**Evidence chain:**

- `libs/atlas-wz/wz/reader.go:42-44` — `Reader.Seek` calls
  `r.f.Seek(offset, whence)` directly on the shared `*os.File`.
- `libs/atlas-wz/wz/reader.go:47-50, 53-57, 60-64, 77-84, …` — every
  positional read (`Skip`, `ReadByte`, `ReadBytes`, `ReadInt16`,
  `ReadUInt16`, `ReadInt32`, etc.) uses `r.f.Seek` / `io.ReadFull(r.f, …)`
  on the same shared file handle. The only concurrency-safe entry is
  `Reader.ReadAt` at `reader.go:66-75`, which the parser path doesn't
  use.
- `libs/atlas-wz/wz/image.go:53-81` (`Image.parse()`) — lazy parser
  calls `i.wzFile.reader.Seek(i.dataOffset, …)` then chases through
  `parsePropertyList` → `parsePropertyValue` → `parseExtendedProperty`
  → `parseCanvasProperty`, every step of which uses non-positional reads
  against the shared reader.
- `libs/atlas-wz/wz/` — no `sync.Mutex` / `sync.RWMutex` anywhere in
  the package (verified with `grep -rn`). The parser has no internal
  serialization.
- `services/atlas-renders/atlas.com/renders/storage/wzcache.go:28-100`
  intentionally shares one `*wz.File` per (scope, region, version,
  archive) across all incoming requests — see the docstring at
  lines 14-27 explaining the memory rationale.
- `services/atlas-renders/atlas.com/renders/mapr/handler.go:104-115`
  fetches the cached file, builds `mapimage.NewIndex(file)`, looks up
  the map's `*wz.Image`, then calls `CompositeFromWZ` which iterates
  layers and lazily parses Tile/Obj cross-refs (`libs/atlas-wz/mapimage/
  layers.go:182-219`).

Two simultaneous requests for two different maps that both hit the same
cached `*wz.File` will race on its `*os.File` seek pointer. The
classical failure mode is silent data corruption: a parse will read
bytes from the wrong offset, decode as one of the WZ property tag
opcodes (1, 3, 4, 5, 8, 9), and either error out (returning nil props,
which `Image.Properties()` then silently swallows at `image.go:44-50`)
or succeed-with-garbage. Either way: 500s under load or wrong-pixels in
the resulting PNG.

**Why this didn't catch in tests:** `go test -race` ran clean against
the renders module because nothing in the test set spawns concurrent
renders against a shared `*wz.File`. The race only triggers when
atlas-renders is serving real traffic.

**Suggested fix:** Either (a) make every `Reader` method that needs to
seek/read use `ReadAt` with a tracked offset (the file-handle is then
truly safe under concurrency), or (b) wrap each lazy `Properties()`
call in a per-File `sync.Mutex` so map-image parsing serializes.
Option (a) preserves the rendering parallelism the cache is designed
for; option (b) is simpler but caps render throughput at one CPU per
process per archive.

### CONCURRENCY-02 — WZCache pins permanent errors via `sync.Once`

`services/atlas-renders/atlas.com/renders/storage/wzcache.go:68-100`
uses `e.once.Do(…)` per cache key, and on failure stores `e.err` in the
entry. Because `sync.Once` only ever runs the body once, a transient
MinIO blip during the first download of `Map.wz` causes every
subsequent map render to return the cached error until the process is
restarted.

In production, this is an availability bug — partial MinIO outage at
startup poisons the WZCache forever. The mitigation (drop failed
entries from the map on error) is one-liner.

### CONCURRENCY-03 — `DownloadToScratch` races on basename collision (latent)

`services/atlas-data/atlas.com/data/storage/minio/scratch.go:11-30`
writes to `filepath.Join(scratchDir, filepath.Base(key))`. Two
concurrent calls for archives with the same basename (e.g. via
`fetchArchive` in `data/workers/runtime.go:115-134`) race on
`os.Create` (which truncates) and `defer os.Remove`. The mitigation —
`archiveCache` in `runtime.go:155-181` — covers only
`fetchAndSerializeArchive`, not the slimmer `fetchArchive` path.

Today only the Character worker calls `fetchArchive` (for Base.wz),
and nothing else touches Base.wz, so the race is not reachable. But the
helper is a footgun for the next worker that needs `fetchArchive`. Either
funnel `fetchArchive` through `archiveCache` too, or unique-suffix the
scratch filename.

### CORRECTNESS-01 — Per-table-atomic restore is not whole-dump-atomic

`services/atlas-data/atlas.com/data/baseline/restore.go:130-139`
opens a fresh `db.Transaction` for each table inside the dump
iteration (`restore.go:107` → `restoreOneTable`). If COPY-FROM on
table N succeeds and table N+1 fails, the tenant is left with a mixed
baseline — old rows in tables N+1.., new rows in tables 1..N. The
final `tenant_baselines` INSERT (`restore.go:119-126`) is also outside
any transaction.

The PRD-mandated restore contract is "destructive replace" which most
consumers would read as all-or-nothing. Consider wrapping the entire
restore in a single transaction (or staging via SAVEPOINTs) and folding
the `tenant_baselines` UPSERT into the same txn.

### LINT-01 — `wzinput` status handler hand-rolls a JSON:API envelope

`services/atlas-data/atlas.com/data/wzinput/status.go:40-48` builds the
JSON:API response inline with `json.NewEncoder(...).Encode(map[string]any{
"data": map[string]any{"type": ..., "id": ..., "attributes": ...}})`.
The backend guidelines (`ai-guidance.md` "Manual JSON:API Envelope
Handling") and `patterns-rest-jsonapi.md` both flag this as an
anti-pattern: use `server.MarshalResponse[T]` (as the `baseline`
publish handler does at `baseline/handler.go:52`).

### LINT-02 — Dead orphan handler in `data/resource.go`

`services/atlas-data/atlas.com/data/data/resource.go:31-59` defines
`processData` and the file's comment at line 22-25 explicitly calls it
"now-orphaned." The function is unreachable. Per
`ai-guidance.md` "Clean Up Dead Code After Extraction" it should be
removed in this PR rather than left to a follow-up.

### LINT-03 — `Image.Properties()` silently masks parse errors

`libs/atlas-wz/wz/image.go:44-50`: when lazy parse fails, the error is
logged at Warn and `i.parsed = true` is set, leaving
`i.properties = nil`. Subsequent `Properties()` calls return the empty
nil slice with no signal that parsing failed. Every downstream call
that depends on the result (icon extract, atlas pack, mapimage
extract) then treats the image as "exists but has no content". That
silently produces empty output assets in ingest workers and empty
composites in atlas-renders. Recommend returning `(props, error)` or
exposing a `LastParseError()` accessor.

### LINT-04 — `UI` worker discards the tenanted ctx

`services/atlas-data/atlas.com/data/data/workers/ui.go:33`: the worker
calls `withTenant(ctx, p)` but assigns all three returns to
`_, _, err`, dropping the tenanted ctx that subsequent code might need
to forward (e.g. into `mc.Put`). Today the dispatcher pre-injects
tenant via `data.RunWorkers` (`runwz.go:29`) so the omission is safe,
but the worker's own comment in `runtime.go:67-73` warned this exact
pattern is fragile. Reassign to keep the worker self-contained.

### LINT-05 — Mostly-duplicated `ExtractLayout` / `ExtractLayers`

`libs/atlas-wz/mapimage/layers.go:38-86` (ExtractLayout) and
`layers.go:101-167` (ExtractLayers) duplicate ~80% of their body —
both walk `info` for bounds, build the Layout struct, then iterate the
8 numbered layer subtrees. ExtractLayout is a metadata-only subset of
ExtractLayers. The shared work should be lifted into a private helper
so the two entry points are 5-line wrappers. Functional duplication
is a future drift hazard: one consumer fix lands in ExtractLayers, the
other forgets ExtractLayout.

### LINT-06 — `accessoryPartClassFor` reinvents item classification inside `libs/atlas-wz`

`libs/atlas-wz/charparts/extract.go:97-108` does the same
`id / 10000` reinvention flagged above as DOM-21. Inside the shared lib
the violation is softer (`libs/atlas-wz` does not currently depend on
`libs/atlas-constants`), but the choice should be deliberate — either
add the dependency or hoist this helper into `libs/atlas-constants`
where it belongs.

### LINT-07 — `wzinput` PATCH route does not use `RegisterInputHandler[T]`

`services/atlas-data/atlas.com/data/wzinput/resource.go:20` registers
PATCH /data/wz via `rest.RegisterHandler(...)` rather than
`RegisterInputHandler[T]`. Strict DOM-08 would FAIL this. **However**,
the upload body is multipart/form-data carrying the ~1.6 GB atlas.zip,
not JSON:API — `RegisterInputHandler[T]` would block-buffer the body
into a typed struct it can't usefully type. The implementation reads
the request via `r.MultipartReader()` in `handler.go:30-66` which is
the right call.

Recommend documenting the exception inline (the comment exists in
`main.go:147-153` but the resource.go file itself has nothing
explaining why PATCH bypasses the typed input handler). Without that
context the deviation looks like a mistake.

### LINT-08 — Known `scope.go` negative-cache bug (user-acknowledged, deferred)

`services/atlas-renders/atlas.com/renders/storage/scope.go:18-34` and
`storage/smap.go:61-76` cache the "shared" verdict permanently when
the tenant-scoped HEAD probe returns false. A subsequent tenant ingest
landing data won't be visible to atlas-renders until the process
restarts. Per the dispatch prompt this is a user-acknowledged
follow-up — flagged here so it's documented in the audit chain.

## Round-2-graduated workers re-audit

The dispatch prompt specifically asked me to re-audit the 10 workers
that R1 marked STUBBED and R2 graduated to DONE based on LOC + presence.
Findings against the DOM-style checklist:

| Worker | File | DOM-style verdict | Notes |
|--------|------|-------------------|-------|
| MAP | `data/workers/mapw.go` | OK | Calls `withTenant`, uses pre-injected tenant via `t`, splits `ExtractLayout` (lazy path) from `ExtractMinimap`, logs counters. |
| MOB | `data/workers/mob.go` | OK | Tenanted ctx, registry-clear defer pattern correct, icon emit best-effort. |
| NPC | `data/workers/npc.go` | OK | Same shape as MOB; leaves NPC registry populated for Map worker (intentional, comment at line 37). |
| REACTOR | `data/workers/reactor.go` | OK | Uses `_, _` discard at line 23 to ignore tenant — see LINT-04, but safe given pre-injection. |
| SKILL | `data/workers/skill.go` | OK | MobSkill folded in. Iterates `skill.<id>` children correctly. |
| QUEST | `data/workers/quest.go` | OK | Smallest worker; cleanly delegates to `quest.RegisterQuest`. |
| STRING | `data/workers/stringw.go` | OK | Stats existence before init to avoid spurious warnings. |
| CHARACTER | `data/workers/character.go` | OK | Largest worker. Atlas emission, smap sidecar, equipment icons all bounded by per-template skip-on-error. |
| UI | `data/workers/ui.go` | NEEDS-WORK (LINT-04) | Discards tenanted ctx return; safe today but pattern is what `runtime.go:67-73` explicitly warned against. |
| ITEM | `data/workers/item.go` | OK | Dedupes single-item .img + multi-item SubProperty walks; this is the fix R2 graduated. |
| COMMODITY | `data/workers/commodity.go` | OK | Etc.wz worker; correctly uses fetchArchive-equivalent (file is its own archive). |

Net new findings from the re-audit beyond R2: **LINT-04 for UI worker**
plus **CONCURRENCY-03** about `fetchArchive`'s basename collision — a
property of `runtime.go` rather than any individual worker, but visible
only when reading the workers as a set.

## DOM-23 — Kafka topic naming (out of scope)

No Kafka topic constants are introduced or modified in the changed
files. The existing `consumer/data` wiring at `main.go:140-143` is
unchanged. DOM-23 N/A for this audit.

## DOM-22 — Dockerfile lib mention count

Per the worktree-local `CLAUDE.md` the Dockerfile model changed under
this branch (shared `Dockerfile` at repo root parameterized by
`ARG SERVICE`, libs declared in repo root + `go.work`). DOM-22 in its
original form does not apply. The dispatch prompt asks me to skip
deploy concerns; not graded.

## SEC — Security review

This task is not auth/JWT-related, but the changed surface includes a
destructive baseline restore + tenant purge + WZ upload, so I ran the
SEC checklist defensively:

- **SEC-01 (operator gating)**: `baseline/handler.go:35, 64`,
  `tenantpurge/handler.go:35`, `wzinput/scope.go:29-32`,
  `runtime/rest/resource.go:44-47` all check `X-Atlas-Operator: 1`
  before destructive / shared-scope ops. PASS.
- **SEC-02 (sha-before-mutation)**: `baseline/restore.go:51-72`
  downloads the dump, computes its sha256, and compares against the
  sidecar BEFORE the first DELETE. PASS.
- **SEC-03 (canonical-tenant refusal)**: `tenantpurge/purge.go:32-34`
  refuses to purge the canonical sentinel tenant. PASS.
- **SEC-04 (zip-slip / symlink / extension whitelist)**:
  `wzinput/validate.go:10-22` rejects path traversal, symlinks, and
  non-.wz entries before writing to MinIO. PASS.
- **SEC-05 (cross-tenant cache poisoning)**:
  `services/atlas-renders/atlas.com/renders/character/handler.go:65-71`
  validates that URL path tenant/region/version match the
  request-context tenant before any cache lookup. PASS.
- **SEC-06 (hardcoded secrets)**: None found in changed code.

## Summary

### Blocking (must fix)

- **DOM-21**: `services/atlas-renders/atlas.com/renders/character/composite.go:61-68, 90-122, 169-196` — three independent reinventions of `libs/atlas-constants/item` (IsTwoHanded, GetClassification, Classification* enum, classification arithmetic). The comment at composite.go:59-60 explicitly admits to the violation.

### Non-Blocking (should fix)

- **CONCURRENCY-01**: `libs/atlas-wz/wz/{reader,image}.go` is not goroutine-safe; atlas-renders intentionally shares one `*wz.File` across concurrent map renders via `storage/wzcache.go:28-100`. Latent — `go test -race` does not exercise it.
- **CONCURRENCY-02**: `services/atlas-renders/atlas.com/renders/storage/wzcache.go:78-99` pins WZ download/parse errors via `sync.Once` for the process lifetime.
- **CONCURRENCY-03**: `services/atlas-data/atlas.com/data/storage/minio/scratch.go:15` uses `filepath.Base(key)` so concurrent `fetchArchive` calls on same-named archives race on `os.Create`. Reachable today only via Character→Base.wz (no peer).
- **CORRECTNESS-01**: `services/atlas-data/atlas.com/data/baseline/restore.go:107, 119-126` is per-table-atomic, not whole-dump-atomic; partial restore on table-N failure plus orphaned `tenant_baselines` row possible.
- **LINT-01**: `services/atlas-data/atlas.com/data/wzinput/status.go:40-48` hand-rolls a JSON:API envelope.
- **LINT-02**: `services/atlas-data/atlas.com/data/data/resource.go:31-59` ships dead `processData` orphan.
- **LINT-03**: `libs/atlas-wz/wz/image.go:44-50` silently masks lazy-parse errors.
- **LINT-04**: `services/atlas-data/atlas.com/data/data/workers/ui.go:33` discards tenanted ctx return (safe today, fragile pattern).
- **LINT-05**: `libs/atlas-wz/mapimage/layers.go:38-86` vs `:101-167` ~80% duplicated.
- **LINT-06**: `libs/atlas-wz/charparts/extract.go:97-108` reinvents item classification inside the shared library.
- **LINT-07**: `services/atlas-data/atlas.com/data/wzinput/resource.go:20` uses `RegisterHandler` for PATCH (correct given multipart body, but undocumented at the call site).
- **LINT-08**: User-acknowledged — `storage/scope.go:18-34` and `storage/smap.go:61-76` pin negative scope verdicts. Carried forward for documentation.
