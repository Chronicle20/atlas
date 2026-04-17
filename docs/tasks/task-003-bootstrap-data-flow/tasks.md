---
name: Bootstrap Data Flow — Task Checklist
description: Progress checklist for the upload → extract → ingest loop across atlas-wz-extractor, atlas-data, and atlas-ui.
type: tasks
task: task-003-bootstrap-data-flow
---

# Tasks — Bootstrap Data Flow

Last Updated: 2026-04-17

Legend: effort = S (≤0.5d) / M (0.5–2d) / L (2–5d) / XL (>5d). Phases are sequentially load-bearing unless noted.

## Phase 0 — Safety rails (S)

- [x] **0.1** Baseline build: `go build ./...` in `atlas-wz-extractor` and `atlas-data`. *(effort: S)*
- [x] **0.2** Baseline test: `go test ./...` in `atlas-wz-extractor` and `atlas-data`; capture pass list. *(effort: S)*
- [x] **0.3** Baseline UI build: `npm run build && npm test` in `atlas-ui`. *(effort: S)*
- [x] **0.4** Confirm `INPUT_WZ_DIR` and `OUTPUT_XML_DIR` env vars are set in the local extractor config; note current values. *(effort: S)*

**Acceptance:** All three projects build + test green at baseline on `deploy-reorg` (or its successor).

## Phase 1 — atlas-data schema + status endpoint (S/M)

- [x] **1.1** Add `UpdatedAt time.Time` to `document/entity.go` with GORM tag `autoUpdateTime`. *(effort: S)*
- [x] **1.2** Confirm `document.Migration` / `AutoMigrate` picks up the new column on a clean DB and on an existing DB (column added, no backfill). *(effort: S)*
- [x] **1.3** Add `data/status.go` (or extend `data/resource.go`) registering `GET /api/data/status`. *(effort: S)*
- [x] **1.4** Implement the count + max-mtime query: `SELECT COUNT(*), MAX(updated_at) FROM documents WHERE tenant_id = ?`. *(effort: S)*
- [x] **1.5** Render JSON:API `type: "dataStatus"`. Zero-value `MAX(updated_at)` → `null`. *(effort: S)*
- [x] **1.6** Unit test: empty tenant → `documentCount: 0, updatedAt: null`. *(effort: S)*
- [x] **1.7** Unit test: populated tenant → correct count + non-null RFC 3339 timestamp. *(effort: S)*
- [x] **1.8** Update `services/atlas-data/atlas.com/data/README.md` REST table with the new endpoint. *(effort: S)*
- [x] **1.9** `go build ./... && go test ./...` green in atlas-data. *(effort: S)*

**Acceptance:** Endpoint returns correct shapes for both states; existing ingest path auto-populates `updated_at` on write.

## Phase 2 — atlas-wz-extractor mutex + path helper (S)

- [x] **2.1** Add `extraction/tenant_path.go` with `ResolveTenantInputDir(tenant.Model) string` and `ResolveTenantOutputDir(tenant.Model) string` composing `<tenantId>/<region>/<major>.<minor>`. *(effort: S)*
- [x] **2.2** Add `extraction/mutex.go` — package-level registry `var tenantMu = struct{ sync.Mutex; m map[string]*sync.Mutex }` with `Acquire(key string) *sync.Mutex` / `TryAcquire(key string) (*sync.Mutex, bool)` / `Release(key string)`. *(effort: S)*
- [x] **2.3** Unit test `tenant_path_test.go` for happy path + unusual region strings. *(effort: S)*
- [x] **2.4** Unit test `mutex_test.go`: 128-goroutine contention on the same key serializes; distinct keys do not block. *(effort: S)*
- [x] **2.5** `go test ./extraction/... -race` green. *(effort: S)*

**Acceptance:** Path helper composes correctly; mutex registry race-clean.

## Phase 3 — atlas-wz-extractor upload endpoint (M/L)

- [x] **3.1** Register `PATCH /api/wz/input` handler in `extraction/resource.go`. *(effort: S)*
- [x] **3.2** `extraction/upload.go`: `streamToTempFile(r *http.Request) (*os.File, error)` using `r.MultipartReader()`; ensure no in-memory buffering of the archive. *(effort: M)*
- [x] **3.3** `extraction/upload.go`: `validateZip(f *os.File) error` — reject entries with `/` or `\` in name, `..`, absolute paths, directory entries, non-regular files, non-`.wz` extensions. Case-insensitive extension check. *(effort: M)*
- [x] **3.4** `extraction/upload.go`: `extractFlat(f *os.File, dst string) error` — `os.RemoveAll(dst)`; `os.MkdirAll(dst, 0o755)`; extract each entry to `filepath.Join(dst, filepath.Base(entry.Name))`. *(effort: S)*
- [x] **3.5** Handler flow: resolve tenant → `TryAcquire` → 409 if busy → spool → validate → extract → 202. Error paths write structured JSON `{"error": "..."}` bodies. *(effort: M)*
- [x] **3.6** Log line at success: tenant id, zip byte size, entry count, duration. *(effort: S)*
- [x] **3.7** `defer os.Remove(tempfile.Name())` for the spooled archive. *(effort: S)*
- [x] **3.8** Unit test: flat valid zip → 202, files under expected path with correct sizes. *(effort: S)*
- [x] **3.9** Unit test: zip with nested path entry → 400, destination untouched (pre-existing contents unchanged). *(effort: S)*
- [x] **3.10** Unit test: zip with `..` entry → 400. *(effort: S)*
- [x] **3.11** Unit test: zip with a non-`.wz` entry → 400. *(effort: S)*
- [x] **3.12** Unit test: zip with an absolute path entry → 400. *(effort: S)*
- [x] **3.13** Unit test: second upload for same tenant while first holds the mutex → 409. *(effort: M)*
- [x] **3.14** Unit test: re-upload replaces prior contents (old file gone, new file present). *(effort: S)*
- [x] **3.15** Manual test: `curl -X PATCH -F zip_file=@fixture.zip` against a local extractor pod; verify tenant-scoped output directory. *(effort: S)*

**Acceptance:** All unit tests pass under `-race`. 400 paths never write to disk. 409 path is deterministic under contention.

## Phase 4 — atlas-wz-extractor status endpoints + extract cutover (M)

- [x] **4.1** Register `GET /api/wz/input` handler; coexist with `PATCH /api/wz/input`. *(effort: S)*
- [x] **4.2** Implement `wzInputStatus`: count top-level `*.wz`, sum sizes, max mtime under `<input>/<tenant path>/`. Missing dir → zeros + null. *(effort: S)*
- [x] **4.3** Register `GET /api/wz/extractions` handler; coexist with the existing `POST /api/wz/extractions`. *(effort: S)*
- [x] **4.4** Implement `wzExtractionStatus`: recursive `.xml` walk under `<output>/<tenant path>/`, sum sizes, max mtime. *(effort: S)*
- [x] **4.5** Cut over `processor.go:runExtraction` to `<input>/<tenant path>/*.wz`. **No fallback to flat dir.** *(effort: M)*
- [x] **4.6** Extract output path cut over to `<output>/<tenant path>/...`. Preserve existing subtree structure. *(effort: S)*
- [x] **4.7** Extract acquires the same tenant mutex as upload. If extract runs async in a goroutine, take the lock inside the goroutine body and release on return. *(effort: M)*
- [x] **4.8** Update `processor_test.go` for the new path layout. *(effort: S)*
- [x] **4.9** Update `resource_test.go` for the new path layout. *(effort: S)*
- [x] **4.10** New test: extract with empty tenant dir returns "no WZ files found" even when flat `$INPUT_WZ_DIR/*.wz` has files. *(effort: S)*
- [x] **4.11** New test: `GET /api/wz/input` matches on-disk state after an upload. *(effort: S)*
- [x] **4.12** New test: `GET /api/wz/extractions` matches on-disk state after an extract. *(effort: S)*
- [x] **4.13** Update `services/atlas-wz-extractor/atlas.com/wz-extractor/README.md` REST table; remove the "copy `.wz` files to `$INPUT_WZ_DIR`" line. *(effort: S)*
- [x] **4.14** Grep the repo for any remaining references to the flat `$INPUT_WZ_DIR/*.wz` glob outside of tests; confirm zero. *(effort: S)*
- [x] **4.15** `go build ./... && go test ./... -race` green in atlas-wz-extractor. *(effort: S)*

**Acceptance:** Extractor reads only from the tenant path; both status endpoints return the correct shape for empty and populated tenants; all tests green.

## Phase 5 — atlas-ui service + hook layer (S/M)

- [x] **5.1** In `services/api/seed.service.ts`: rename `uploadGameData` → `uploadWzFiles`; change URL from `PATCH /api/data` to `PATCH /api/wz/input`. *(effort: S)*
- [x] **5.2** Add `runWzExtraction()` to seed.service (or reuse if an existing trigger wraps `POST /api/wz/extractions`). *(effort: S)*
- [x] **5.3** Add `runDataProcessing()` to seed.service (or reuse existing). *(effort: S)*
- [x] **5.4** Add typed getters: `getWzInputStatus()`, `getExtractionStatus()`, `getDataStatus()` mapping to the shapes in `api-contracts.md`. *(effort: S)*
- [x] **5.5** In `lib/hooks/api/useSeed.ts`: add `useUploadWzFiles` mutation; `onSuccess` invalidates `wzInputStatus` + `extractionStatus`. *(effort: S)*
- [x] **5.6** Add `useRunWzExtraction` mutation; `onSuccess` invalidates `extractionStatus` + `dataStatus`. *(effort: S)*
- [x] **5.7** Add `useRunDataProcessing` mutation; `onSuccess` invalidates `dataStatus`. *(effort: S)*
- [x] **5.8** Add `useWzInputStatus` / `useExtractionStatus` / `useDataStatus` query hooks with `staleTime: 0, refetchInterval: 5000`. *(effort: S)*
- [x] **5.9** Remove or soft-deprecate `useUploadGameData`. *(effort: S)*
- [x] **5.10** `npm run build && npm test` green. *(effort: S)*

**Acceptance:** All six new hooks type-check and wire to the expected URLs; existing tests remain green.

## Phase 6 — atlas-ui /setup page rewire (M)

- [x] **6.1** In `app/setup/page.tsx`, add a "Game Data" card above the seed-action grid with three rows (Upload / Extract / Ingest) per `ux-flow.md`. *(effort: M)*
- [x] **6.2** Upload row: `<input type="file" accept=".zip">` with visible label; on change fires `useUploadWzFiles`. *(effort: S)*
- [x] **6.3** Extract row: "Run Extraction" button fires `useRunWzExtraction`; `disabled = wzInputStatus.fileCount === 0 || anyMutationPending`. Tooltip on disabled reason. *(effort: S)*
- [x] **6.4** Ingest row: "Process Data" button fires `useRunDataProcessing`; `disabled = extractionStatus.fileCount === 0 || extractIngestMutationPending`. *(effort: S)*
- [x] **6.5** Badge renderers: "N .wz files, N MB" / "N XMLs extracted" / "N documents loaded" with `Intl.NumberFormat` formatting. Pending state → "—". *(effort: S)*
- [x] **6.6** Stale-extraction warning: yellow, `role="status"`, rendered before the Ingest button when `wzInputStatus.updatedAt && extractionStatus.updatedAt && wzInputStatus.updatedAt > extractionStatus.updatedAt`. *(effort: S)*
- [x] **6.7** Toast copy wired per `ux-flow.md` (upload success/400/409, extract success/failure, ingest success/failure). *(effort: S)*
- [x] **6.8** `aria-live="polite"` on badges; keyboard tab order Upload → Extract → Ingest. *(effort: S)*
- [x] **6.9** Remove the orphan "Upload Game Data" button that PATCHes `/api/data`. *(effort: S)*
- [x] **6.10** Manual e2e: upload a fixture zip → observe Extract enable within 5 s → click → observe Ingest enable → click → observe document count climb. *(effort: S)*
- [x] **6.11** Manual e2e: after ingest, re-upload and confirm stale-warning renders before the Ingest button. *(effort: S)*
- [x] **6.12** Manual e2e: two tenants with different versions coexist without cross-leakage on disk. *(effort: S)*

**Acceptance:** The three-click bootstrap works end-to-end against a real tenant; gating, badges, and stale warning all behave per `ux-flow.md`.

## Phase 7 — Cross-phase checklist + sweep (S)

- [x] **7.1** Walk PRD §10 acceptance criteria bullet-by-bullet; tick each against a reproducing test or manual check. *(effort: S)*
- [x] **7.2** `go build ./... && go test ./... -race` green in both Go services. *(effort: S)*
- [x] **7.3** `npm run build && npm test` green in atlas-ui. *(effort: S)*
- [x] **7.4** Grep check: zero remaining references to the flat `$INPUT_WZ_DIR/*.wz` glob outside of tests. *(effort: S)*
- [x] **7.5** Grep check: zero remaining references to `PATCH /api/data` in atlas-ui. *(effort: S)*
- [x] **7.6** Update `docs/TODO.md` if it tracks any of the three services. *(effort: S)*
- [x] **7.7** Commit + push; open PR referencing this plan. *(effort: S)*

**Acceptance:** All PRD acceptance criteria ticked; build + test green everywhere; cleanup greps return zero; PR open.
