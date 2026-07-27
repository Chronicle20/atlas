# Backend Audit — task-172-legacy-wz-ingest (Go changes)

- **Scope:** `libs/atlas-wz` (module `github.com/Chronicle20/atlas/libs/atlas-wz`) and `services/atlas-data/atlas.com/data` (module `atlas-data`), merge-base `c9490b724` → `718644c41`.
- **Guidelines Source:** backend-dev-guidelines skill (DOM-*, FILE-*, SUB-*, EXT-*, SEC-*)
- **Date:** 2026-07-17
- **Build:** PASS (both modules, `go build ./...`)
- **Vet:** PASS (both modules, `go vet ./...`)
- **Tests:** PASS — `go test ./... -race -count=1` clean in `libs/atlas-wz` (11 packages) and in the changed `atlas-data` packages (`data`, `data/workers`, `data/wztoxml`, `item`)
- **goroutine-guard:** PASS — `tools/goroutine-guard.sh` exit 0 from repo root
- **Overall:** PASS

## Domain Discovery (Phase 2)

No package touched by this branch has a `model.go` (no DOM package) or a `resource.go` without `model.go` (no SUB package). Every touched package is a **Support package**:

| Package | Role |
|---|---|
| `libs/atlas-wz/wz` | Binary WZ archive parser (File/Image/detection) — library internals, not a REST/domain package |
| `libs/atlas-wz/wztest` | Exported test-fixture-only PKG1 builder |
| `libs/atlas-wz/crypto` | Encryption-type/key utility |
| `libs/atlas-wz/charparts`, `icons`, `mapimage` | Canvas-decoding call sites (1-line migrations each) |
| `services/atlas-data/.../data` | Worker fan-out dispatcher (`RunWorkers`) |
| `services/atlas-data/.../data/workers` | Per-archive ingest workers (`OpenArchive`, `String`, `Character`, `UI`, …) |
| `services/atlas-data/.../item` | Item string-search registry (only a new test added; `string_registry.go` itself untouched) |

None of these packages contain `Processor`/`RestModel`/cross-service `requests.*` symbols — this is a batch WZ-ingest pipeline (k8s Job, `MODE=ingest`), not a JSON:API domain service. FILE-01..06 and EXT-01..04 therefore have no applicable symbols to misplace; SCAFFOLD-* does not trigger (no new service, no new atlas-channel writer/handler). SEC-* does not apply (not an auth service).

## File Responsibilities Checklist (run against every touched package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..05 | Processor/RestModel/requests/entity/Builder placement | N/A | Grep for `type Processor interface`, `type RestModel`, `requests.RootUrl(` etc. across all touched files returns zero hits outside the pre-existing, untouched `data/processor.go` (not in this diff's file list) — confirmed via `grep -rln` over `services/atlas-data/atlas.com/data/data/`, `item/`, and `libs/atlas-wz/` |
| FILE-06 | No package-named catch-all file | PASS | `libs/atlas-wz/wz/file.go` and `services/atlas-data/atlas.com/data/data/workers/runtime.go` each bundle multiple *related* WZ-archive-resolution responsibilities (detection, sub-file veneer, key-range table / `OpenArchive`, monolith memo, archive caching) but none of these are the FILE-01..05 REST-domain responsibilities (Processor, RestModel, requests, entity, Builder) — the file-responsibilities table's catch-all rule targets REST domain packages; this is a parser/worker-infrastructure package with its own established (pre-diff) file layout (`file.go`, `image.go`, `directory.go`, `reader.go` / `runtime.go`, `stringw.go`, `character.go`, …), unchanged in kind by this branch |

## Focus Area 1 — Concurrency

| Check | Status | Evidence |
|-------|--------|----------|
| `image.go` `atomic.Bool` parsed-flag correctness | PASS | `libs/atlas-wz/wz/image.go:80-101` — fast path `Properties()` uses `i.parsed.Load()`; slow path acquires `i.wzFile.LockParse()`, re-checks under lock (double-checked locking), writes `i.properties`/`i.parseErr` before `i.parsed.Store(true)`. Go's `sync/atomic` `Load`/`Store` establish the required happens-before edge, so a concurrent reader observing `parsed==true` is guaranteed to see the fully-populated `properties` slice. Exercised by a genuine multi-goroutine `-race` test (`libs/atlas-wz/wz/image_fallback_test.go:104-124`, `TestPerImageFallbackConcurrent`), which passed under `go test -race` |
| `parseMu` / `keyRangesMu` interplay | PASS | `libs/atlas-wz/wz/file.go:39-49` — `parseMu` (Seek-based parsing) and `keyRangesMu` (RWMutex over the key-range table) are separate locks by design (canvas decompression stays outside `parseMu`, `file.go:25-27`). Writes to `keyRanges` happen in `registerImageKey` (`file.go:58-66`) under `keyRangesMu.Lock()`, called from `Image.parse()` (`image.go:119-121`) which itself runs under the caller's `parseMu` hold — no lock-ordering inversion since `keyRangesMu` is never held while acquiring `parseMu`. Reads in `CanvasEncryptionKeyFor` (`file.go:73-85`) use `keyRangesMu.RLock()` only, matching the RWMutex's read/write contract |
| Sub-file (`parent`) delegation | PASS | `file.go:58-66,73-85,92-98` — `registerImageKey`, `CanvasEncryptionKeyFor`, and `LockParse` all delegate to `wz.parent` when non-nil, so every sub-view (`NewSubFile`, `file.go:114-128`) synchronizes through the single owning `*File`'s mutexes rather than maintaining independent (and therefore racy) state. `libs/atlas-wz/wz/subfile_test.go` verifies a sub-file's `Close()` is a no-op and the parent stays readable afterward (`subfile_test.go:88-97`) |
| `monolith` package-level `sync.Once` memo | PASS | `services/atlas-data/atlas.com/data/data/workers/runtime.go:117-125,127-153` — `monolithState.once` gates concurrent `monolithFile` calls from the parallel worker fan-out (`runwz.go:100-109`, `g.Go(...)` via `errgroup`); `sync.Once.Do` provides the standard happens-before guarantee for all readers of `monolith.file/localPath/found/err` after `Do` returns. Comment at `runtime.go:113-116` explicitly documents the job-scoped-singleton assumption (mirrors the pre-existing `archiveCache sync.Map`, `runtime.go:254`), and this holds because `RunWorkers` is invoked exactly once per `MODE=ingest` k8s Job process (`services/atlas-data/atlas.com/data/runtime/ingest/run.go:48`, package doc `runtime.go:1-9`) |
| `CloseMonolith` ordering vs. in-flight workers | PASS | `services/atlas-data/atlas.com/data/data/runwz.go:47` — `defer workers.CloseMonolith()` is registered at the top of `RunWorkers`'s returned closure; the function's last statement is `return g.Wait()` (`runwz.go:110`), so the defer fires only after every fanned-out worker goroutine (both the sequential prerequisite phase and the `errgroup` phase) has completed. `CloseMonolith` (`runtime.go:158-164`) closes the shared handle and resets the memo exactly once, matching the doc comment's invariant ("Must only run after all workers have finished") |
| atlas-renders shared-`*wz.File` compatibility | PASS | No behavior change to the concurrency contract atlas-renders' `WZCache` relies on: `parseMu` remains the single serialization point for all Seek-based parsing (`file.go:17-27`), and the new per-image key-range table / sub-file veneer both delegate to that same mutex rather than introducing a second one. Confirmed no `libs/atlas-renders` files are in this diff's changed-file list, and the `File` struct's exported surface (`Root()`, `ReadCanvasData`, `CanvasEncryptionKey`) is unchanged; only the new `CanvasEncryptionKeyFor` and `GameVersion` are additive |

No data race found; `-race` passed on both modules' full relevant test set (see Build & Test Results below).

## Focus Area 2 — Error Handling

| Check | Status | Evidence |
|-------|--------|----------|
| `ErrCategoryAbsent` sentinel discrimination | PASS | Defined `runtime.go:111`; wrapped with `%w` at the only production call site, `monolithSubArchive` (`runtime.go:180`, `fmt.Errorf("%s: %w", archive, ErrCategoryAbsent)`); discriminated via `errors.Is(err, workers.ErrCategoryAbsent)` in `runwz.go:63` — only that specific sentinel triggers the skip-and-continue path (`runwz.go:64-65`); any other error from `OpenArchive` still hard-fails the worker (`runwz.go:67`). Unit-tested directly: `monolith_test.go:88-93` (`TestMonolithSubArchiveAbsentCategory`) |
| Split-layout miss still fails loudly (design §C-3.4) | PASS | `OpenArchive` (`runtime.go:218-221`) returns a **plain** `fmt.Errorf` (not wrapping `ErrCategoryAbsent`) when neither the per-archive object nor a monolithic `Data.wz` exists for the scope — `errors.Is` in `runwz.go:63` is false for this path, so `runOne` returns the wrapped error and aborts the run (`runwz.go:67`). `ErrCategoryAbsent` is reachable only through `monolithSubArchive`, which by construction requires `found==true` (a `Data.wz` present, `runtime.go:219`) — matches context.md decision 5 ("Skip-tolerance is monolithic-only, by construction") |
| Fallback retry gated strictly on `errBadImageTag` | PASS | `image.go:16` defines the sentinel; `image.go:108-127` — `parse()` calls `parseWithKey(fileKey)` first, and only enters the fallback loop when `err != nil && errors.Is(err, errBadImageTag)` (the inverse check at `image.go:111`: `if err == nil \|\| !errors.Is(err, errBadImageTag) { return err }`). Any I/O error, truncation, or unknown-property-type error (all of which produce non-`errBadImageTag` errors from `parseWithKey`'s `r.Seek`/`r.ReadWzStringBlock`/`wz.parsePropertyList` calls, `image.go:137-153`) returns immediately without retrying, matching context.md decision 8 |
| No silent guess in two-phase detection | PASS | `file.go:323-339` — the key-detection switch on `len(sane)` has exactly three arms: `1` (accept), `0` (hard error naming every encryption type tried, `file.go:332`), and `default`/ambiguous (hard error naming every sane candidate, `file.go:338`) — there is no fallback branch that silently picks a key. Verified by `libs/atlas-wz/wz/detect_test.go:94-112` (`TestDetectNoSaneCandidateErrors`) |
| No swallowed errors in the new/changed control flow | PASS | Every new production error path (`OpenArchive`, `monolithFile`, `monolithSubArchive`, `CloseMonolith`'s use in `runwz.go`, `image.parse`) either returns the error wrapped with `%w` or explicitly logs-and-continues at a call site pattern that predates this diff (e.g. `stringw.go:74-93`'s log-and-continue for `InitStringFlat`/`InitStringNested` failures mirrors the identical pre-existing pattern at the pre-diff revision — `git show c9490b724:.../stringw.go:36-49` — the C-4 addition reuses, not introduces, that convention) |
| C-5 version cross-check is warn-only | PASS | `runwz.go:70-74` — a `GameVersion()` mismatch only calls `l.Warnf(...)`; no error is returned, and `versionWarnOnce sync.Once` (`runwz.go:53`) ensures at most one warning per job even though every worker's `runOne` call reaches this check |

## Focus Area 3 — goroutine-guard

| Check | Status | Evidence |
|-------|--------|----------|
| No unjustified bare `go` statements in production code | PASS | `tools/goroutine-guard.sh` exits 0 from repo root (includes `libs/atlas-wz` and `services/atlas-data`) |
| Test-local goroutine has a justification marker | PASS | `libs/atlas-wz/wz/image_fallback_test.go:117` — `go func() { //goroutine-guard:allow — test-local concurrency probe, joined by wg.Wait` — properly justified and joined via `wg.Wait()` (`image_fallback_test.go:122`) |
| `errgroup.Group.Go` fan-out is not a bare `go` statement | PASS (not a finding) | `runwz.go:102` uses `g.Go(func() error {...})` (the `golang.org/x/sync/errgroup` idiom, pre-existing in this dispatcher before the diff), not a raw `go` statement — outside DOM-26's regex (`^\s*go (func\|[A-Za-z_])`) and outside the guard's scan target by construction |

## Focus Area 4 — DOM-21 (atlas-constants reuse)

| Check | Status | Evidence |
|-------|--------|----------|
| New types/constants checked against `libs/atlas-constants` | PASS | This branch adds `EncryptionType.String()` (`crypto/keygen.go:44-53`), `File.GameVersion()`/`gameVersion int` (`file.go:36,201-203`), `keyRange`/`keyRanges` (`file.go:51-56`), `wztest.Kind`/`Prop`/`Image`/`Dir`/`Builder` (`wztest/builder.go`). None of these are item/inventory/weapon/world/channel/map/character/job/skill/monster classifications — they are WZ-binary-format internals (encryption variant, detected client version, byte-range key cache, test-fixture DSL) with no equivalent in `libs/atlas-constants` (confirmed via `grep -in "wz\|encryption"` over `libs/atlas-constants/README.md` — zero hits, and `grep -ril` for `EncryptionType`/`GameVersion`/`WzKey` across `libs/atlas-constants/` — zero hits) |

## Focus Area 5 — Resource Handling

| Check | Status | Evidence |
|-------|--------|----------|
| `OpenArchive` per-archive cleanup | PASS | `runtime.go:213` — returns `func() { f.Close(); _ = os.Remove(localPath) }`; on the immediately-preceding `wz.Open` failure it also removes the partially-downloaded file before returning (`runtime.go:210`) |
| `OpenArchive` monolithic-view cleanup | PASS | `runtime.go:226` — returns the shared `noop` (`runtime.go:194`) for sub-archive views, matching the doc comment (`runtime.go:186-188`) that the parent handle is closed exactly once by `CloseMonolith` at job end, not per-worker |
| `monolithFile` cleanup on open failure | PASS | `runtime.go:143-147` — if `wz.Open` fails after a successful download, the downloaded temp file is removed (`os.Remove(localPath)`) before the error is recorded in `monolith.err` |
| `CloseMonolith` releases the shared handle + scratch file | PASS | `runtime.go:158-164` — closes `monolith.file` and removes `monolith.localPath`, then resets the whole `monolithState` to its zero value, preventing a stale handle from leaking into a hypothetical next `sync.Once` firing (moot under the one-job-per-process invariant, but correct regardless) |
| No double-`Close()` risk on sub-file views | PASS | `Close()` on a sub-file (`file.go:165-172`) is a no-op whenever `wz.parent != nil`, so worker code that calls the per-archive `cleanup()` on a monolithic sub-view (which is always `noop`, see above) never touches the shared parent's `*os.File` |

## Build & Test Results

```
$ cd libs/atlas-wz && go build ./...          # PASS
$ cd libs/atlas-wz && go vet ./...            # PASS
$ cd libs/atlas-wz && go test ./... -race -count=1
ok  	.../libs/atlas-wz/atlas
ok  	.../libs/atlas-wz/atlas/pngenc
ok  	.../libs/atlas-wz/canvas
ok  	.../libs/atlas-wz/charparts
ok  	.../libs/atlas-wz/crypto
ok  	.../libs/atlas-wz/icons
ok  	.../libs/atlas-wz/manifest
ok  	.../libs/atlas-wz/mapimage
ok  	.../libs/atlas-wz/maplayout
ok  	.../libs/atlas-wz/wz
ok  	.../libs/atlas-wz/wz/property

$ cd services/atlas-data/atlas.com/data && go build ./...   # PASS
$ cd services/atlas-data/atlas.com/data && go vet ./...     # PASS
$ cd services/atlas-data/atlas.com/data && go test ./data/... ./item/... -race -count=1
ok  	atlas-data/data
ok  	atlas-data/data/workers
ok  	atlas-data/data/wztoxml
ok  	atlas-data/item

$ tools/goroutine-guard.sh     # exit 0
```

(Full-repo `go test ./... -count=1` in every module and `docker buildx bake` were reported already run green by the task controller; this audit re-ran a focused `-race` pass over every changed package rather than repeating the full-repo sweep.)

## Additional Verification

- **Dead-code cleanup:** `FetchAndOpen` (deleted, `data/wzsource.go` removed) and `fetchArchive` (deleted, superseded by `OpenArchive`) have zero remaining references anywhere in `services/atlas-data` (`grep -rn "FetchAndOpen\|fetchArchive\b"` → no hits), satisfying the ai-guidance.md "Clean Up Dead Code After Extraction" rule.
- **`wztest` test-fixture boundary:** package doc explicitly marks it `TEST FIXTURES ONLY` (`wztest/builder.go:1-5`); `grep -rl "atlas-wz/wztest" --include="*.go" .` outside `_test.go` files returns zero hits — no production code imports the test-fixture builder.
- **Four canvas call sites migrated to `CanvasEncryptionKeyFor`:** `charparts/extract.go:578`, `icons/extract.go:432`, `mapimage/decoder.go:47`, `mapimage/minimap.go:34`, plus the atlas-data-side `ui.go:68` (all five confirmed via direct grep, not diff excerpt).
- **No real game archives committed:** the E2E harness described in `e2e-results.md` as "diagnostic only — deleted after the run, never committed" was verified absent from the tree (no stray harness file in the diff's changed-file list).

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- None. Every focus area (concurrency, error-handling/sentinel discrimination, goroutine-guard, DOM-21, resource cleanup) has direct file:line evidence of correct behavior, exercised by passing `-race` tests where concurrency is the concern.
