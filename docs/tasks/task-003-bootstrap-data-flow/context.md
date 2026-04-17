---
name: Bootstrap Data Flow — Context
description: Key files, decisions, dependencies, and gotchas for the upload → extract → ingest loop across atlas-wz-extractor, atlas-data, and atlas-ui.
type: context
task: task-003-bootstrap-data-flow
---

# Context — Bootstrap Data Flow

Last Updated: 2026-04-17

## Key Files (current repo, to be touched)

### `atlas-wz-extractor`

Service root: `services/atlas-wz-extractor/atlas.com/wz-extractor/`

- `extraction/resource.go` — registers the existing `POST /api/wz/extractions` trigger. **Adds** `PATCH /api/wz/input`, `GET /api/wz/input`, `GET /api/wz/extractions`.
- `extraction/processor.go` — `runExtraction` currently globs `$INPUT_WZ_DIR/*.wz`. **Cuts over** to `<input>/<tenantId>/<region>/<major>.<minor>/*.wz`. Acquires the tenant mutex for the duration of the goroutine.
- `extraction/processor_test.go` — existing tests; **update** to the tenant-path layout.
- `extraction/resource_test.go` — existing tests; **update** to the tenant-path layout.
- `extraction/upload.go` — **new**. Zip validation (zip-slip / nested / non-`.wz` rejection), streaming multipart parse, `rm -rf`+recreate semantics, per-tenant mutex acquisition.
- `extraction/tenant_path.go` — **new**. Helpers `ResolveTenantInputDir(tenant.Model)` / `ResolveTenantOutputDir(tenant.Model)`.
- `extraction/mutex.go` — **new**. Package-level per-tenant mutex registry.
- `README.md` — **update**: add three new endpoints to the REST table; remove the "copy `.wz` files to `$INPUT_WZ_DIR`" instruction.

### `atlas-data`

Service root: `services/atlas-data/atlas.com/data/`

- `data/resource.go` — existing `POST /api/data/process` handler. **Adds** `GET /api/data/status` (or in a new `data/status.go`).
- `data/processor.go`, `data/producer.go`, `data/kafka.go` — no change.
- `document/entity.go` — **adds** `UpdatedAt time.Time` with GORM tag `autoUpdateTime`.
- `document/db_storage.go` — no behavior change; GORM auto-populates `UpdatedAt` on save.
- `document/registry.go`, `document/reg_storage.go`, `document/storage.go` — no change.
- `README.md` — **update**: add `/api/data/status` to the REST table.

### `atlas-ui`

Service root: `services/atlas-ui/`

- `services/api/seed.service.ts` — **rename** `uploadGameData` → `uploadWzFiles`; **change** URL from `PATCH /api/data` to `PATCH /api/wz/input`. **Add** `runWzExtraction`, `runDataProcessing`, `getWzInputStatus`, `getExtractionStatus`, `getDataStatus`.
- `lib/hooks/api/useSeed.ts` — **add** `useUploadWzFiles`, `useRunWzExtraction`, `useRunDataProcessing`, `useWzInputStatus`, `useExtractionStatus`, `useDataStatus`. **Remove** `useUploadGameData`.
- `app/setup/page.tsx` — **replace** orphan upload button with the "Game Data" card (three rows, badges, stale-warning).

### `deploy/shared/routes.conf`

- No changes. `/api/wz/*` and `/api/data/*` already route correctly.

## Key Decisions

- **Hard cutover** on the extractor input path (flat → tenant-scoped). No fallback. User-confirmed no prod dependency.
- **Per-tenant mutex** is in-process (`sync.Mutex`). Lost on restart, which is fine — a restart means in-flight requests are dead. No Redis / etcd.
- **Upload is synchronous, plain multipart.** No chunking, no resumability. Single request, 202 on success.
- **Status endpoints always return 200.** Missing tenant dir is zeros + null, not 404. UI renders identically.
- **Upload does not auto-cascade.** Each stage stays manual per PRD §2 non-goals.
- **Stale warning is informational.** It does NOT disable the Ingest button — operators can still ingest older XMLs intentionally.
- **No application-level upload size cap.** Reverse proxy and practical PVC limits only.
- **Zero-value `UpdatedAt` renders as `null`.** Pre-migration rows won't trip the stale-warning on first load.
- **JSON:API shapes** per `api-contracts.md` — single-resource objects with typed `attributes`.

## Dependencies

- `tenant.MustFromContext(ctx)` — already used elsewhere in both services. `Region()` returns `string` (per memory, NOT `world.Id`). Version `major/minor` fields live on the tenant model.
- `jsonapi` transport helpers on atlas-data and atlas-wz-extractor (both already JSON:API-compliant).
- atlas-tenants running, with two tenant records whose `(region, major, minor)` tuples differ.
- Shared PVC mounted at `$INPUT_WZ_DIR` and `$OUTPUT_XML_DIR` on both the extractor and (for status reads) itself.
- atlas-ui dev proxy pointed at atlas-wz-extractor and atlas-data.

## Gotchas

- **`ParseMultipartForm` buffers 32 MB in memory by default.** Real WZ bundles are hundreds of megabytes. Use `r.MultipartReader()` and spool to a tempfile. This is easy to miss and will OOM the pod in prod.
- **Zip-slip.** Validate entry names for `..`, absolute paths, and OS-specific separators (`/` and `\`). Do it in a first pass before any filesystem writes; a 400 must leave the destination untouched.
- **Directory replacement semantics.** `os.RemoveAll(dst)` is used, not selective overwrite. A partial failure during extraction leaves an indeterminate state; operator re-uploads.
- **GORM `autoUpdateTime` vs `autoCreateTime`.** We want update semantics so every ingest re-write bumps the mtime, not just the initial insert.
- **`MAX(updated_at)` on empty table** returns NULL from PostgreSQL, not zero-time. Handle both in Go (zero value and scan into `sql.NullTime`).
- **Tenant mutex vs long-running extract goroutine.** If the current extractor does async extract, the mutex must be held inside the goroutine, not just the HTTP handler — otherwise an upload during the async phase won't observe the lock.
- **Path separators.** `filepath.Base(entry.Name)` handles `/` but some zip implementations use `\` on Windows. Reject entries containing either rather than relying on `filepath.Base`.
- **Recursive XML walk** for `/api/wz/extractions` can be slow on large tenants (thousands of files). Acceptable at sub-10 ms for a few thousand files per PRD §8; don't add caching until observed hot.
- **`tenant.Model.Region()` returns `string`.** Do not confuse with `world.Id`. Per memory.
- **Test files reference internal functions** — renaming handlers may break `processor_test.go` / `resource_test.go`. Per memory gotcha.
- **`_, _ = ...()` error-discard pattern** — don't add one. Audited against in recent tasks.
- **Read files before editing them** — tool requirement. Per memory.

## Reference Files

- `docs/tasks/task-003-bootstrap-data-flow/prd.md` — requirements.
- `docs/tasks/task-003-bootstrap-data-flow/api-contracts.md` — endpoint shapes.
- `docs/tasks/task-003-bootstrap-data-flow/ux-flow.md` — UI state machine and gating matrix.
- `docs/tasks/task-002-character-creation-error-cascade/plan.md` — prior task's phase structure as a template.

## Out of Scope (per PRD §2)

- Per-tenant in-progress registry / SSE progress stream.
- Chunked / resumable / streaming uploads.
- One-click cascade (upload → extract → ingest).
- Backward compatibility with the flat `$INPUT_WZ_DIR/*.wz` layout.
- Application-layer upload size limits.
- Delete / cleanup endpoints for staged files.
- Automatic extract re-run on upload (stale-warning only).
