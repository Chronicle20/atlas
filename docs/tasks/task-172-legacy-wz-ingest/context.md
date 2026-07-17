# task-172: Legacy GMS + JMS WZ Ingest — Context

Companion to `plan.md`. Key files, decisions, and dependencies an implementer or reviewer needs without re-deriving the design.

## What this task is

atlas-data cannot ingest pre-v83 GMS WZ data (v48 split archives, v12 monolithic `Data.wz`) or JMS v185 (mixed per-image encryption). Root causes are empirically verified in `design.md` (RC-1..RC-4). Scope is ingest only — no packet/socket bring-up, no `List.wz` parsing, no Sound.wz worker.

## Key files

### libs/atlas-wz (shared lib — consumed by atlas-data AND atlas-renders)

| File | Role in this task |
|---|---|
| `wz/file.go` | RC-1 fix: `detectVersion` split into version phase (offset validation, key-independent) + key phase (entry-name sanity scoring, unique-candidate-or-error). New: `GameVersion()`, `NewSubFile`, `CanvasEncryptionKeyFor`, per-image key-range table, `parent` delegation |
| `wz/image.go` | RC-3 fix: `parse` retries with alternate keys ONLY on the `unexpected image tag` validation failure; winning key cached per image and registered for canvas decryption |
| `wz/reader.go` | Read-only reference: string mask (`0xAA+i` XOR + key XOR), `ReadWzOffset` math the test builder must invert |
| `crypto/keygen.go` | `EncryptionType` gets a `String()` method for detection errors/logs |
| `charparts/extract.go`, `icons/extract.go`, `mapimage/minimap.go`, `mapimage/decoder.go` | The four `canvas.Decompress` call sites; `CanvasEncryptionKey()` → `CanvasEncryptionKeyFor(cp.DataOffset())` |
| `wztest/builder.go` (new) | Exported test-fixture-only PKG1 builder; no real game archives in the repo |

### services/atlas-data

| File | Role in this task |
|---|---|
| `data/runwz.go` | `runOne` switches from `FetchAndOpen` to `workers.OpenArchive`; skip-on-`ErrCategoryAbsent`; warn-once C-5 version cross-check; `defer workers.CloseMonolith()` |
| `data/wzsource.go` | DELETED (`FetchAndOpen`'s only caller was `runOne`) |
| `data/workers/runtime.go` | New exported `OpenArchive` (per-archive object → monolithic `Data.wz` sub-view fallback), memoized monolith (`sync.Once`, job-scoped like `archiveCache`), `ErrCategoryAbsent`, `CloseMonolith`; `fetchArchive` deleted; `fetchAndSerializeArchiveOnce` rebased on `OpenArchive` |
| `data/workers/character.go` | Two `fetchArchive` → `OpenArchive` call sites (`Base.wz` smap/zmap sidecars; `Base.wz` resolves to the `Data.wz` root) |
| `data/workers/stringw.go` | RC-4 fix: `resolveStringSources` — modern flat/Eqp images win; legacy single `Item.img` engages only when no modern image exists |
| `item/string_registry.go` | UNCHANGED — see "single-pass adapter" decision below |

## Decisions (including deviations from design.md)

1. **`OpenArchive` replaces TWO fetch paths, not one.** The design calls `workers.fetchArchive` "the single chokepoint", but the primary per-worker open in `RunWorkers.runOne` used `data.FetchAndOpen` (`runwz.go:51`). Both now route through the new exported `workers.OpenArchive`; `FetchAndOpen` and `fetchArchive` are deleted.
2. **`wztest` is an exported package, not a `_test.go` helper.** atlas-data's worker tests need the same binary fixtures and cannot import another module's test files. Package doc marks it test-fixture-only.
3. **Canvas keys resolve by byte-range, not by threading `*wz.Image`.** The four `canvas.Decompress` call sites don't all have the owning image in scope (icons' link-following returns only the canvas). Every canvas lies inside its image's `[dataOffset, dataOffset+dataSize)` extent, so `File.CanvasEncryptionKeyFor(offset)` resolves the fallback key from a small RWMutex-guarded range table populated during fallback parse. One-line change per call site; correct across `_inlink`/link resolution.
4. **Single-pass legacy String adapter — no initializer refactor.** The design's C-4 table maps each legacy `Item.img` child through flat/nested initializers via a "small refactor" to accept subtrees. `item.InitStringFlat`'s walker already recurses through non-numeric nodes and harvests numeric ids at any depth, so one flat pass over the legacy `Item.img.xml` yields the same rows (including nested Eqp, at whatever nesting depth the real v12/v48 data has). Pinning test: `TestInitStringFlatLegacyItemImg`. If that test ever fails, fall back to the design's per-subtree mapping.
   - **E2E CORRECTION (Task 9, live v12/v48).** The original heuristic — "legacy `Item.img` engages only when NO modern per-category image exists" — was wrong. Real GMS v12 AND v48 `String.wz` ship the legacy `Item.img` (Con/Ins/Etc/Eqp/Pet) *and* a **standalone `Pet.img`**. `Pet.img.xml` matched the modern-flat list, so the resolver treated the set as modern and skipped `Item.img` entirely — dropping every non-pet item name (item search returned only 6 pets, named "MISSINGNO"). Fix: `resolveStringSources` now inverts the priority — **`Item.img` present ⇒ legacy layout, used as the sole item-string source** (it is the complete superset; the standalone `Pet.img` is redundant). Modern layouts (v83+/JMS) have no `Item.img` and fall through unchanged. Regression test: `TestResolveStringSourcesLegacyItemImgWins`. Verified layouts: v12/v48 have `Item.img`+`Pet.img` (legacy); JMS v185 has `Consume/Cash/Etc/Ins/Pet/Eqp`, no `Item.img` (modern).
5. **Skip-tolerance is monolithic-only, by construction.** `ErrCategoryAbsent` can only be returned when a `Data.wz` exists in scope but lacks the category subdirectory. Split-layout misses keep today's hard-failure path — the error message differs only by mentioning the absent `Data.wz`.
6. **Sanity check is strict printable ASCII** (`0x20..0x7E`, ≤100 chars). Root entry names in every known archive generation are ASCII; the design's "BMP text" allowance is intentionally not implemented until a real archive needs it (would weaken garbage rejection).
7. **Detection never guesses**: zero or multiple sane key candidates is a descriptive hard error naming the candidates tried (design §Error handling).
8. **Fallback retry triggers only on tag-validation failure** (`errBadImageTag` sentinel). I/O errors, truncation, unknown property types keep existing error semantics — no retry, no behavior change for verified archives.
9. **Monolith `*wz.File` is shared across workers.** Lazy image parse serializes on the parent's `parseMu` (same contract atlas-renders' WZCache relies on), trading fan-out parallelism for correctness on v12 ingests. `CloseMonolith` runs only after `g.Wait()` via defer ordering in `RunWorkers`.
10. **`Image.parsed` promoted to `atomic.Bool` (Task 3).** `Image.Properties()` has an unlocked fast-path read of `parsed` before taking `parseMu`; the C-2 concurrency test was the first to exercise it under `-race` and exposed a pre-existing data race against the locked write. Fix: `parsed` is now `sync/atomic.Bool`, with `properties`/`parseErr` published before the `Store(true)` on both success and error paths. Minimal and necessary — atlas-renders' `storage/wzcache.go` shares one `*wz.File` across concurrent renders, so multiple goroutines calling `Properties()` on the same `*Image` is a live production pattern, not just a test artifact.
11. **Fifth canvas call site migrated (Task 3 fix wave).** The plan enumerated four `CanvasEncryptionKey()` → `CanvasEncryptionKeyFor(cp.DataOffset())` call sites in `libs/atlas-wz`, but a fifth exists in the service: `services/atlas-data/atlas.com/data/data/workers/ui.go:68`. It was migrated in the same task so JMS mixed-encryption UI canvases resolve the per-image fallback key rather than silently decrypting under the file-level key.

## Dependencies / blast radius

- **atlas-renders** consumes `libs/atlas-wz` (`storage/wzcache.go` shares one `*wz.File` across concurrent renders). No renders code changes, but its full test/vet/build + `docker buildx bake atlas-renders` are mandatory (Task 8). It does not call `CanvasEncryptionKey` directly — the four migrated call sites are all inside the lib.
- No new services, no config surface, no k8s/services.json/docker-bake changes → `service-registration-guard.sh` not required. `wztest/` ships inside the existing `COPY libs/...` — the bake proves it.
- Existing GMS v83/v95 ingest must stay byte-identical (regression = existing lib+service suites plus the GMS round-trip fixture).

## Test strategy

- **Lib**: generated binary fixtures via `wztest` (unencrypted / GMS / KMS / mixed per-image / monolithic) — detection, fallback, sub-file veneer, canvas key resolution, plus a `-race` concurrency probe.
- **Service**: `monolithSubArchive` resolution against a fixture `Data.wz` (no MinIO needed — there is no MinIO test seam; do not invent one), `resolveStringSources` layout detection, legacy-XML `InitStringFlat` pinning test on the sqlite (`file::memory:?cache=shared`) pattern from `item/string_search_test.go`.
- **E2E (Task 9, mandatory)**: real v12/v48/JMS sample sets (out-of-repo, `/tmp/wz`) through the live upload→process flow; results recorded in `e2e-results.md`. Expected magnitudes from design: v48 8,062 images, v12 3,613, JMS ~19k, zero parse failures. Domain-reader schema drift found there is fixed iteratively on this branch.

## Known verification gates before PR

`go test -race`, `go vet`, `go build` in `libs/atlas-wz`, atlas-data, atlas-renders; `tools/redis-key-guard.sh` + `tools/goroutine-guard.sh` from repo root (no global `GOWORK=off`); `docker buildx bake atlas-data atlas-renders`; then `superpowers:requesting-code-review` (reviewer subagents pinned to Sonnet/Haiku) before opening the PR.
