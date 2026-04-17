# Bootstrap Data Flow — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-04-17

---

## 1. Overview

Atlas has three conceptually distinct steps when bootstrapping game data for a tenant: uploading the raw `.wz` archive, extracting WZ into XMLs, and ingesting XMLs into the `atlas-data` document store. Steps two and three already have backend trigger endpoints and shared PVC storage wired. Step one — uploading — is missing. The UI has an orphan "Upload Game Data" button that PATCHes `/api/data` and gets a 404, with no backend to catch it. Meanwhile the extractor reads from a flat, non-tenant-scoped input directory, which makes multi-tenant bootstrap impossible.

This task closes that loop end-to-end. atlas-wz-extractor grows a tenant-scoped multipart upload endpoint and two filesystem-state GET endpoints. atlas-data grows one GET status endpoint backed by the existing `documents` table (with a new `UpdatedAt` column). The `/setup` page rewires the upload, adds explicit Run Extraction and Process Data buttons, and gates each button on the preceding step's observable state. The result is a three-click bootstrap flow where each step is independent, re-runnable, and visibly gated — a user who hasn't uploaded WZ files can see at a glance that Extract and Ingest are not yet runnable.

Concurrency hazards — re-uploading while an extraction is in flight, or extracting on top of a partial upload — are explicitly rejected with `409 Conflict` at the HTTP layer.

## 2. Goals

Primary goals:
- Bootstrap a fresh Atlas tenant end-to-end via the web UI without shell access to the PVCs.
- Each of the three stages (upload, extract, ingest) is an independent, user-initiated action, and each is observable.
- UI buttons for Extract and Ingest are disabled when their prerequisite is not met.
- Multi-tenant isolation at every layer: two tenants with different WZ versions can coexist.

Non-goals:
- Per-tenant in-progress registry / SSE progress stream. UI shows "running" via react-query mutation state; the next status poll after success flips downstream buttons on.
- Chunked, resumable, or streaming uploads. Plain multipart, sync, single request.
- Automatic cascade (one-click upload → extract → ingest). Each stage stays manual.
- Backward compatibility with the existing flat `$INPUT_WZ_DIR/*.wz` convention. Hard cutover — wz-extractor reads only from the tenant-scoped subpath after this task.
- Upload size limits at the application layer. Reverse proxy and practical limits only.
- Deletion / cleanup endpoints for the staged files. Re-upload overwrites; housekeeping is operator-driven outside the UI.
- Automatic extract/ingest re-runs on upload. Stale-extraction is signaled via an inline warning, not an automatic re-run.

## 3. User Stories

- As an operator bootstrapping a fresh tenant, I want to drop a `.wz` zip into the UI and have it land on the extractor's input volume, so I don't need shell access to the PVC.
- As an operator, I want the Extract button to be disabled until WZ files exist for my tenant, so I can't fire a no-op extraction.
- As an operator, I want the Ingest button to be disabled until extracted XMLs exist for my tenant, so I can't fire a no-op ingest.
- As an operator, I want to see "12 .wz files", "2,341 XMLs", "18,204 documents" badges next to each button, so I know at a glance what state my tenant is in.
- As an operator, I want a warning when my uploaded WZ is newer than my extracted XMLs, so I remember to re-run extraction before ingest.
- As an operator, I want concurrent uploads for the same tenant to be rejected, so I can't corrupt the staged input by racing two browser tabs.

## 4. Functional Requirements

### 4.1 Upload — `PATCH /api/wz/input` (new, atlas-wz-extractor)

- Accepts `multipart/form-data` with a single part named `zip_file`.
- Tenant is resolved from `tenant.MustFromContext` as in other Atlas endpoints.
- Destination directory: `$INPUT_WZ_DIR/<tenantId>/<region>/<major>.<minor>/`.
- Zip contents must be flat: every entry's path has no `/` or `\` separator and ends in `.wz` (case-insensitive). Any entry that (a) contains a path separator, (b) has a different extension, (c) is a directory entry, or (d) exhibits zip-slip characteristics (absolute path, `..` in path) causes a `400 Bad Request` and **no files are written**.
- On success, the destination directory is **replaced** — if it exists, it is `rm -rf`'d before the zip is extracted. Partial failures during extraction leave the directory in an indeterminate state (acceptable because the user can re-upload).
- Concurrent uploads or an upload arriving mid-extraction for the same tenant return `409 Conflict`. Serialization is enforced by a per-tenant mutex inside atlas-wz-extractor.
- On success, returns `202 Accepted` (no body — status is discoverable via the GET endpoint).

### 4.2 Input status — `GET /api/wz/input` (new, atlas-wz-extractor)

- Returns the filesystem state of `$INPUT_WZ_DIR/<tenantId>/<region>/<major>.<minor>/`.
- Response attributes:
  - `fileCount` — count of `*.wz` entries at the top level. `0` if the directory is missing.
  - `totalBytes` — sum of their sizes. `0` if `fileCount == 0`.
  - `updatedAt` — max `mtime` among those files, RFC 3339. `null` when `fileCount == 0`.
- Always `200`. See `api-contracts.md` for the full JSON:API shape.

### 4.3 Extraction status — `GET /api/wz/extractions` (new verb on existing path)

- Same response shape as §4.2, reading from `$OUTPUT_XML_DIR/<tenantId>/<region>/<major>.<minor>/` with `fileCount` counting `*.xml` files (recursive walk — extractor output is nested like `String.wz/Map.img.xml`).
- Coexists with the existing `POST /api/wz/extractions` trigger; the two verbs share the same path.

### 4.4 Extract — updated behavior

- `runExtraction` now globs `$INPUT_WZ_DIR/<tenantId>/<region>/<major>.<minor>/*.wz` instead of `$INPUT_WZ_DIR/*.wz`. Output paths are unchanged.
- If the tenant-scoped input dir is missing or empty, extraction fails with the existing "no WZ files found" error. **No fallback to the flat dir.**
- Extract acquires the same per-tenant mutex as §4.1 for the duration of the goroutine. Concurrent extract + upload is serialized.

### 4.5 Data status — `GET /api/data/status` (new, atlas-data)

- Response attributes:
  - `documentCount` — `SELECT COUNT(*) FROM documents WHERE tenant_id = ?`.
  - `updatedAt` — `SELECT MAX(updated_at) FROM documents WHERE tenant_id = ?`, RFC 3339. `null` when `documentCount == 0`.
- Always `200`.

### 4.6 UI changes (atlas-ui, `/setup` page)

- Rename `seedService.uploadGameData` → `uploadWzFiles`. Change the URL from `PATCH /api/data` to `PATCH /api/wz/input`. Multipart body unchanged (`zip_file`).
- Add "Run Extraction" button — calls `POST /api/wz/extractions`.
- Add "Process Data" button — calls `POST /api/data/process`.
- Three react-query status hooks polling every 5 s while the page is mounted:
  - `useWzInputStatus` → `GET /api/wz/input`.
  - `useExtractionStatus` → `GET /api/wz/extractions`.
  - `useDataStatus` → `GET /api/data/status`.
- Each button's `disabled` is derived from prerequisite + mutation state (see `ux-flow.md` for the full matrix).
- Each button shows a small badge: "N .wz files", "N XMLs extracted", "N documents loaded". Badge reads from the corresponding status; while the status query is pending, show a neutral "—".
- After a successful mutation, invalidate the relevant status queries so the downstream button re-gates without waiting for the next 5 s tick.
- When `wzInputStatus.updatedAt > extractionStatus.updatedAt`, render an inline warning next to the Ingest button: *"Uploaded WZ files are newer than the last extraction. Re-run extraction before ingest to avoid stale data."*

## 5. API Surface

Full specs in `api-contracts.md`. Summary:

| Method | Path | Service | Status |
|---|---|---|---|
| `PATCH` | `/api/wz/input` | atlas-wz-extractor | new |
| `GET` | `/api/wz/input` | atlas-wz-extractor | new |
| `GET` | `/api/wz/extractions` | atlas-wz-extractor | new (path exists for POST) |
| `POST` | `/api/wz/extractions` | atlas-wz-extractor | unchanged |
| `GET` | `/api/data/status` | atlas-data | new |
| `POST` | `/api/data/process` | atlas-data | unchanged |

No URL renames, no breaking changes to existing HTTP callers. The only breaking change is the extractor's on-disk input-path convention (flat → tenant-scoped), which has no existing production dependency per user confirmation.

## 6. Data Model

One schema change: the `documents` table on atlas-data gains `UpdatedAt` (GORM auto-managed `time.Time`). Migration is additive via the existing `document.Migration` / `AutoMigrate` hook. No backfill — existing rows get the zero-value on schema migration; the next ingest run updates them as content is re-written. The status endpoint treats zero-value `UpdatedAt` as `null` (UI renders "—").

No new tables, no relational changes. TenantId scoping on the count-query uses the existing tenant-filter middleware.

atlas-wz-extractor introduces no DB state. The per-tenant upload/extract mutex is an in-process `sync.Mutex` keyed by `<tenantId>:<region>:<major>.<minor>`, held only while an HTTP request or extract goroutine is active. Lost across restarts, which is fine — a restart means neither request is still running.

## 7. Service Impact

**atlas-wz-extractor**
- New handlers on `extraction/resource.go` for `PATCH /wz/input`, `GET /wz/input`, `GET /wz/extractions`.
- New `extraction/upload.go` — zip validation, tenant-scoped extraction of the archive to the input dir, per-tenant mutex registry.
- Update `extraction/processor.go` `runExtraction` to glob the tenant-scoped path and acquire the mutex.
- `README.md` — add upload + status endpoints to the REST table; remove the "copy `.wz` files to `$INPUT_WZ_DIR`" instruction.

**atlas-data**
- New `data/status.go` (or extend `data/resource.go`) with the `GET /data/status` handler.
- `document/entity.go` — add `UpdatedAt time.Time`.
- `document/db_storage.go` — no behavior change; GORM auto-populates `UpdatedAt` on save.
- `README.md` — add `/data/status` to the REST table.

**atlas-ui**
- `services/api/seed.service.ts` — rename `uploadGameData` → `uploadWzFiles`; change URL; add trigger wrappers for extract + process; add three status getters.
- `lib/hooks/api/useSeed.ts` — add `useUploadWzFiles`, `useRunWzExtraction`, `useRunDataProcessing`, `useWzInputStatus`, `useExtractionStatus`, `useDataStatus`.
- `app/setup/page.tsx` — replace existing upload button; add Extract + Ingest buttons; wire status polls, badges, `disabled` state, stale-extraction warning.

**deploy/shared/routes.conf**
- No changes. `/api/wz/*` and `/api/data/*` already route correctly.

## 8. Non-Functional Requirements

- **Performance.** Status endpoints are filesystem `stat`/walk calls; sub-10 ms for a few thousand files. UI poll every 5 s per status → 3 req/tenant/5 s total.
- **Upload size.** No application-level cap. atlas-wz-extractor streams the multipart reader to a temp file (do not buffer in memory; default `ParseMultipartForm` buffer is 32 MB which is too small for real WZ bundles). Zip extraction reads from the temp file.
- **Security.** Zip-slip mitigation is mandatory (§4.1). Only `.wz` entries accepted at top level. Upload endpoint is tenant-scoped via existing middleware; same auth as other Atlas admin endpoints — no new auth surface.
- **Observability.** Upload log line includes tenant, zip byte size, file count, duration. Extraction and ingest triggers already log.
- **Multi-tenancy.** Enforced in path computation for all new endpoints. Status endpoints return empty payloads (not errors) when the tenant's directory is missing.
- **Testing.** Unit tests for zip-slip rejection, non-`.wz` rejection, directory-replacement semantics, per-tenant mutex serialization. Integration test for `PATCH /api/wz/input` that verifies the on-disk layout. `go test ./... -count=1` green.
- **Idempotency / re-runs.** Upload is idempotent on replacement. Extract and ingest are already idempotent (ingest truncates documents via `document.DeleteAll`; extract overwrites XMLs).

## 9. Open Questions

None remaining after scope confirmation. Deferred items are captured under §2 non-goals.

## 10. Acceptance Criteria

- `PATCH /api/wz/input` with a valid zip of flat `.wz` files returns 202; files appear under `$INPUT_WZ_DIR/<tenantId>/<region>/<major>.<minor>/`.
- Concurrent `PATCH /api/wz/input` for the same tenant returns 409; the first one wins.
- `PATCH /api/wz/input` during a running extraction for the same tenant returns 409.
- A zip with a non-`.wz` entry (or a nested path, or a zip-slip path) returns 400; no partial writes on disk.
- Re-uploading to a populated tenant dir removes the prior contents first.
- `GET /api/wz/input` reflects the on-disk state within one status poll cycle (5 s) of an upload completing.
- `GET /api/wz/extractions` reflects the on-disk state of the extractor's XML output.
- `POST /api/wz/extractions` reads only from `<tenant>/<region>/<major>.<minor>/` — it emits "no WZ files found" when the tenant-scoped dir is empty, even if `$INPUT_WZ_DIR/*.wz` at the flat level has files.
- `GET /api/data/status` returns `documentCount` = `SELECT COUNT(*)` and `updatedAt` = `MAX(updated_at)` for the tenant.
- On the `/setup` page: Extract button disabled until Upload completes; Ingest button disabled until Extract completes; each button shows a badge reflecting status; the stale-extraction warning appears when `wzInput.updatedAt > extraction.updatedAt`.
- On mutation success (upload, extract, or ingest), the relevant status query is invalidated and the UI re-gates within one paint.
- `go build ./... && go test ./...` green for atlas-wz-extractor and atlas-data.
- atlas-ui lint + build green; manually verified against a real tenant.
