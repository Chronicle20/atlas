---
name: Bootstrap Data Flow — Implementation Plan
description: Phased plan to close the upload → extract → ingest loop on atlas-wz-extractor, atlas-data, and atlas-ui so operators can bootstrap a tenant end-to-end from /setup.
type: plan
task: task-003-bootstrap-data-flow
---

# Bootstrap Data Flow — Implementation Plan

Last Updated: 2026-04-17
Companion docs: `prd.md`, `api-contracts.md`, `ux-flow.md`, `tasks.md`, `context.md`

## Executive Summary

Atlas bootstraps a tenant in three stages — upload the raw `.wz` archive, extract WZ to XML, ingest XMLs into `atlas-data`'s `documents` table. Stages two and three are already wired; stage one is missing. The `/setup` page has an orphan "Upload Game Data" button that PATCHes `/api/data` and receives a 404. atlas-wz-extractor reads from a flat, non-tenant-scoped input directory, which blocks multi-tenant bootstrap.

This task closes the loop. atlas-wz-extractor gains `PATCH /api/wz/input` (tenant-scoped multipart upload) plus `GET /api/wz/input` and `GET /api/wz/extractions` for observable filesystem state. atlas-data gains `GET /api/data/status` backed by the existing `documents` table with a new `UpdatedAt` column. The `/setup` page rewires the upload, adds explicit Run Extraction and Process Data buttons, polls all three statuses on a 5 s interval, and gates each button on its prerequisite. Concurrent upload-over-extract or upload-over-upload for the same tenant returns `409 Conflict` via a per-tenant in-process mutex.

The only breaking change is the extractor's on-disk input-path convention (flat `$INPUT_WZ_DIR/*.wz` → `$INPUT_WZ_DIR/<tenantId>/<region>/<major>.<minor>/*.wz`). Per PRD §2 non-goals this is a hard cutover — no fallback to the flat dir.

## Current State Analysis

**atlas-wz-extractor** (`services/atlas-wz-extractor/atlas.com/wz-extractor/extraction/`)
- `resource.go` registers handlers for `POST /api/wz/extractions` (trigger) only. No `PATCH`, no `GET`.
- `processor.go:runExtraction` globs `$INPUT_WZ_DIR/*.wz` — flat, non-tenant-scoped.
- No upload endpoint, no mutex registry, no status endpoints.
- `processor_test.go` and `resource_test.go` exist and must stay green.

**atlas-data** (`services/atlas-data/atlas.com/data/`)
- `data/resource.go`, `data/processor.go`, `data/producer.go`, `data/kafka.go` wire the existing `POST /api/data/process` trigger.
- `document/entity.go` is the GORM entity that backs the `documents` table. No `UpdatedAt` today.
- `document/db_storage.go` handles persistence; `document.DeleteAll` truncates per-tenant on ingest.

**atlas-ui** (`services/atlas-ui/`)
- `services/api/seed.service.ts` exposes `uploadGameData` pointing at `PATCH /api/data` (the orphan — 404 in prod).
- `lib/hooks/api/useSeed.ts` wraps the seven existing seed mutations + `useUploadGameData`.
- `app/setup/page.tsx` renders the "Bootstrap" section with the orphan upload button and seven seed buttons.

**deploy/shared/routes.conf** — `/api/wz/*` and `/api/data/*` are already routed. No proxy changes.

## Proposed Future State

### HTTP surface (see `api-contracts.md`)

| Method | Path | Service | Status |
|---|---|---|---|
| `PATCH` | `/api/wz/input` | atlas-wz-extractor | new |
| `GET` | `/api/wz/input` | atlas-wz-extractor | new |
| `GET` | `/api/wz/extractions` | atlas-wz-extractor | new (verb on existing path) |
| `POST` | `/api/wz/extractions` | atlas-wz-extractor | unchanged (input-path cutover only) |
| `GET` | `/api/data/status` | atlas-data | new |
| `POST` | `/api/data/process` | atlas-data | unchanged |

### Filesystem layout (hard cutover)

```
$INPUT_WZ_DIR/<tenantId>/<region>/<major>.<minor>/*.wz
$OUTPUT_XML_DIR/<tenantId>/<region>/<major>.<minor>/**/*.xml
```

`<region>` comes from `tenant.Model.Region()` (string), `<major>.<minor>` from the tenant's version fields. `<tenantId>` is the tenant UUID.

### Concurrency

One `sync.Mutex` per `<tenantId>:<region>:<major>.<minor>` key, stored in a package-level registry in atlas-wz-extractor. Both the upload handler and the extract goroutine `TryLock` at entry and `Unlock` on return. A `TryLock` miss returns `409 Conflict` for upload and is impossible for extract (serialized by the registry).

### Data model delta

- `document/entity.go` gains `UpdatedAt time.Time` (GORM auto-managed).
- Migration is additive via the existing `AutoMigrate` hook. No backfill. Zero-value `UpdatedAt` is rendered as `null` by the status endpoint.

### Key invariants

- Upload 400 writes nothing to disk. Validation is a two-pass: inspect `*zip.Reader` entries, then extract. If any entry is invalid, reject before touching the destination.
- Upload 202 leaves the destination directory as a clean replacement. Prior contents are `rm -rf`'d before extraction.
- Status endpoints always return `200` — a missing tenant dir is rendered as `fileCount: 0, totalBytes: 0, updatedAt: null`, not a 404.
- Extractor `runExtraction` no longer globs the flat dir under any circumstance.
- UI mutation success invalidates the downstream status query so the next button re-gates within one paint.

## Implementation Phases

Phases are sequentially load-bearing unless noted. Each phase leaves the tree in a build-passing state.

### Phase 0 — Safety rails (S)

Baseline the affected services so regressions are attributable. No code changes.

- Confirm `go build ./...` green in `atlas-wz-extractor` and `atlas-data`.
- Confirm `npm run build` green in `atlas-ui`.
- Note the current extractor env: `INPUT_WZ_DIR`, `OUTPUT_XML_DIR`.

**Acceptance:** Baseline build + test green for both Go services and the UI.

### Phase 1 — atlas-data schema + status endpoint (S/M)

Additive schema change; read-only endpoint. No other layers depend on this yet.

1. Add `UpdatedAt time.Time` to `document/entity.go` with GORM tag `autoUpdateTime`. Confirm `AutoMigrate` picks up the new column.
2. Add `data/status.go` (or extend `data/resource.go`) registering `GET /api/data/status`. Query `SELECT COUNT(*), MAX(updated_at) FROM documents WHERE tenant_id = ?`. Render JSON:API with `type: "dataStatus"`.
3. Handle zero-value `MAX(updated_at)` → `null`.
4. Unit test: empty tenant returns `documentCount: 0, updatedAt: null`; populated tenant returns real counts.

**Acceptance:** `go build ./... && go test ./...` green. `curl` against a local atlas-data returns the expected shape for both states. Existing ingest writes `updated_at` automatically.

### Phase 2 — atlas-wz-extractor mutex + path helper (S)

Foundation for the upload and extract changes. No HTTP change yet.

1. Add `extraction/tenant_path.go` with a `ResolveTenantInputDir(tenant.Model) string` / `ResolveTenantOutputDir(tenant.Model) string` helper that composes `<tenantId>/<region>/<major>.<minor>`.
2. Add `extraction/mutex.go` with a package-level `var tenantMu = struct{ sync.Mutex; m map[string]*sync.Mutex }{...}` registry and `Acquire(key string) (*sync.Mutex, bool)` / `Release`. The outer `Mutex` guards map access; the inner per-tenant `Mutex` is what callers lock/try-lock.
3. Unit test: 128-goroutine concurrent `Acquire` for the same key serializes; distinct keys do not block.

**Acceptance:** Helpers compile, unit tests pass under `-race`.

### Phase 3 — atlas-wz-extractor upload endpoint (M/L)

Net-new handler with the bulk of validation risk.

1. Add `extraction/upload.go`:
   - `streamToTempFile(r *http.Request) (*os.File, error)` — spools the `zip_file` multipart part to a tempfile. Use `r.MultipartReader()` (not `ParseMultipartForm`) so large archives don't buffer in memory.
   - `validateZip(f *os.File) error` — open with `zip.NewReader`, iterate `zr.File`, reject any entry that (a) has `/` or `\` in Name, (b) has a `..` segment or absolute path (zip-slip), (c) is a directory, (d) is not a regular file, (e) does not end in `.wz` (case-insensitive).
   - `extractFlat(f *os.File, dst string) error` — removes and recreates `dst`, then copies each entry to `filepath.Join(dst, filepath.Base(entry.Name))`.
2. Register `PATCH /api/wz/input` in `extraction/resource.go`. Handler flow:
   - Resolve tenant via `tenant.MustFromContext`.
   - `TryLock` the tenant mutex. On failure, write 409 JSON body, return.
   - Spool multipart to tempfile. `defer os.Remove`.
   - Validate zip. On failure, write 400 with the reason, return.
   - `os.RemoveAll(dst); os.MkdirAll(dst, 0o755)`, then extract.
   - Write 202 (empty body).
   - Log: tenant id, byte size, entry count, duration.
3. Unit tests (use `zip.Writer` to build fixtures in-memory):
   - Flat zip of `.wz` files → 202, files on disk.
   - Zip with a nested path → 400, destination untouched.
   - Zip with `..` in path → 400.
   - Zip with a non-`.wz` entry → 400.
   - Second concurrent upload for same tenant → 409 (use a held mutex in the test).
   - Re-upload to populated dir replaces contents (old file gone, new file present).

**Acceptance:** `go test ./extraction/... -race` green. Manual `curl -X PATCH -F zip_file=@fixture.zip` against a local extractor produces the expected tenant-scoped directory.

### Phase 4 — atlas-wz-extractor status endpoints + extract cutover (M)

Three related changes shipped together so the extract cutover does not leave status endpoints pointing at the old layout.

1. Add `GET /api/wz/input` handler — reads `<input>/<tenant path>/`, counts top-level `*.wz`, sums sizes, maxes mtime. Render JSON:API `wzInputStatus`.
2. Add `GET /api/wz/extractions` handler — walks `<output>/<tenant path>/` recursively, counts `*.xml`, sums sizes, maxes mtime. Render `wzExtractionStatus`. Must coexist with the existing `POST /api/wz/extractions` registration.
3. Update `processor.go:runExtraction`:
   - Resolve tenant from the request context (same pattern as the new upload handler).
   - Glob `<input>/<tenant path>/*.wz`. No fallback.
   - Acquire the same tenant mutex for the duration of the goroutine. If the extract is kicked off after the HTTP handler returns, wrap the goroutine body in the lock; otherwise take the lock in the handler before spawning.
   - Write outputs under `<output>/<tenant path>/...` (preserve existing subtree structure).
4. Update `processor_test.go` and `resource_test.go` for the new path layout. Add a test that verifies "no WZ files found" fires when the tenant dir is empty even if the flat dir has files.

**Acceptance:** All existing extractor tests pass on the new layout; new status tests pass; `curl` against both GETs returns the expected shapes for empty and populated tenants.

### Phase 5 — atlas-ui service + hook layer (S/M)

Wire the TypeScript surface before touching the page.

1. `services/api/seed.service.ts`:
   - Rename `uploadGameData` → `uploadWzFiles`. Change URL to `PATCH /api/wz/input`. Multipart body unchanged.
   - Add `runWzExtraction()` — `POST /api/wz/extractions` (trigger, no body).
   - Add `runDataProcessing()` — `POST /api/data/process` (trigger, no body). Audit for an existing method; reuse if present.
   - Add `getWzInputStatus()`, `getExtractionStatus()`, `getDataStatus()` — typed GETs returning JSON:API single-resource shapes per `api-contracts.md`.
2. `lib/hooks/api/useSeed.ts`:
   - Add mutation hooks `useUploadWzFiles`, `useRunWzExtraction`, `useRunDataProcessing`. Each hook's `onSuccess` calls `queryClient.invalidateQueries` per the invalidation map in `ux-flow.md`.
   - Add query hooks `useWzInputStatus`, `useExtractionStatus`, `useDataStatus` with `staleTime: 0, refetchInterval: 5000`.
   - Remove (or soft-deprecate) `useUploadGameData`.
3. Type the response attributes with narrow TS types that mirror `api-contracts.md`.

**Acceptance:** `npm run build` + `npm test` green. Hook signatures compile against the existing tests.

### Phase 6 — atlas-ui /setup page rewire (M)

Visible UX changes gated on all prior phases.

1. In `app/setup/page.tsx`, add a "Game Data" card above the existing seed-action grid with three rows (Upload / Extract / Ingest) per the layout table in `ux-flow.md`.
2. Wire each row:
   - Upload row renders a `<input type="file" accept=".zip">` with a visible label; on change, fires `useUploadWzFiles`.
   - Extract row's "Run Extraction" button fires `useRunWzExtraction`; `disabled = wzInputStatus.fileCount === 0 || anyMutationPending`.
   - Ingest row's "Process Data" button fires `useRunDataProcessing`; `disabled = extractionStatus.fileCount === 0 || extractIngestMutationPending`.
3. Each row's badge renders from the corresponding status: "N .wz files, N MB" / "N XMLs extracted" / "N documents loaded". Use `Intl.NumberFormat` for bytes/counts per `ux-flow.md`. While the query is pending, render "—".
4. Render the stale-extraction warning (yellow, `role="status"`) before the Ingest button when `wzInputStatus.updatedAt && extractionStatus.updatedAt && wzInputStatus.updatedAt > extractionStatus.updatedAt`.
5. Toast copy per `ux-flow.md`. 409 on upload produces the specific busy-tenant toast.
6. Accessibility: `aria-live="polite"` on badges, keyboard tab order Upload → Extract → Ingest.
7. Remove the orphan "Upload Game Data" button that PATCHes `/api/data`.

**Acceptance:** Full `/setup` walkthrough with a real tenant: upload a fixture zip, wait up to 5 s for the Extract button to enable, click Run Extraction, wait for the Ingest button to enable, click Process Data, observe the document count badge climb. Stale-warning test: after ingest, re-upload and confirm the warning renders.

### Phase 7 — Docs, READMEs, cross-phase checklist (S)

1. `services/atlas-wz-extractor/atlas.com/wz-extractor/README.md` — add the three new endpoints to the REST table; remove the "copy `.wz` files to `$INPUT_WZ_DIR`" line.
2. `services/atlas-data/atlas.com/data/README.md` — add `GET /api/data/status` to the REST table.
3. This `plan.md` marked complete; `tasks.md` checkboxes all ticked; PRD acceptance criteria walked one-by-one.
4. Final sweep: `go build ./... && go test ./...` on both services; `npm run build && npm test` on atlas-ui.

**Acceptance:** All phases' acceptance criteria ticked, all PRD §10 bullets verified, build+test green across all three affected projects.

## Risk Assessment and Mitigation

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Zip-slip / path-escape on upload writes outside `INPUT_WZ_DIR`. | Low (with validation) | High (arbitrary file write) | Two-pass validation; unit test the explicit attack vectors; refuse absolute paths, `..`, and any separators. |
| Partial-write on 500 leaves the tenant dir in an indeterminate state. | Low | Medium | PRD accepts this; operator re-uploads. Logged explicitly. |
| Per-tenant mutex lost across restarts if an upload is in flight. | Very low | Low | A restart means the in-flight request is dead; fresh upload can proceed. Documented in PRD §6. |
| Hard cutover of the extract input path breaks an environment still using the flat dir. | Low (user-confirmed no prod dependency) | Medium | Announce the cutover in the wz-extractor README; test that the "no WZ files found" error fires cleanly on empty tenant dirs. |
| `MultipartReader` not used → large uploads OOM the extractor pod. | Medium (easy to miss) | High | Explicit code review checkpoint in Phase 3. Test with a > 500 MB fixture in integration. |
| Stale-extraction warning produces false positives on clock skew between the upload and the extractor pods. | Low | Low | Both timestamps are mtimes on the same shared PVC; single-host file system, no skew. |
| `documents.UpdatedAt` zero-value for existing rows makes the stale-warning always fire after migration. | Medium | Low | Render zero-value as `null`; UI treats `null` as "no data" and skips the comparison. First ingest populates real values. |
| UI polling at 5 s per status × 3 statuses × N tenants adds noise to extractor logs. | Low | Low | Access logs already ignore GETs below WARN; document in observability section of PRD. |
| 409 toast confuses users with multiple browser tabs open. | Medium | Low | Toast copy is explicit about "another upload or extraction is in progress"; the next status poll reconciles. |

## Success Metrics

- A fresh tenant can be bootstrapped via `/setup` in three clicks without shell access.
- `go test ./...` green for atlas-wz-extractor and atlas-data; `-race` clean.
- `npm run build && npm test` green for atlas-ui.
- Manual e2e: upload → extract → ingest against two tenants with different WZ versions; no cross-tenant leakage observed on disk or in the DB.
- Zero instances of the flat `$INPUT_WZ_DIR/*.wz` glob in the extractor after the cutover (grep check).
- Upload-during-extract returns 409 (manually reproduced).

## Required Resources and Dependencies

- Local atlas-wz-extractor + atlas-data + atlas-tenants running against a shared PVC (or its local equivalent) for manual verification.
- Two tenant records in atlas-tenants with distinct `(region, version.major, version.minor)` combinations for isolation testing.
- A fixture `.wz` zip (can be small — five synthetic `.wz` files) and a negative fixture (same zip with a nested path).
- atlas-ui connected to the running backends via the dev proxy.

No new libraries, no new infra.

## Timeline Estimates

| Phase | Effort |
|---|---|
| 0 — Safety rails | S |
| 1 — atlas-data schema + status | S/M |
| 2 — extractor mutex + path helper | S |
| 3 — extractor upload endpoint | M/L |
| 4 — extractor status + cutover | M |
| 5 — UI service + hooks | S/M |
| 6 — /setup page rewire | M |
| 7 — Docs + sweep | S |

Total: ~3–5 dev days in single-owner flow. Phases 1 and 2 can go in parallel; Phase 5 can start once Phase 1 lands and the extractor contracts are frozen.

## Open Questions

None remaining — see PRD §9.
