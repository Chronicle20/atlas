# Task-076 — Implementation Context

> Companion to [plan.md](./plan.md). Captures the file layout, design decisions, and dependencies an implementer needs to land the 20 followups without re-reading task-071's design history.

## Source documents
- PRD: [prd.md](./prd.md) — 20 followup IDs (F1–F20), severities, acceptance per ID.
- Design: [design.md](./design.md) — OQ-1..OQ-8 decisions, per-followup implementation sketches, risk register.
- Parent inventory: `.worktrees/task-071-gamedata-minio-consolidation/docs/tasks/task-071-gamedata-minio-consolidation/followups.md` — full rollout-day capture.

## Worktree
- Path: `.worktrees/task-076-task071-followups`
- Branch: `task-076-task071-followups`
- Every command in plan.md MUST run with that as cwd.

## Wave plan (recap)
- **Wave 1 (production hot path):** F1, F7, F3, F2, F5 — must merge before any Wave 2 work is reviewed.
- **Wave 2 (debt):** F8, F18, F6, F4, F15, F16, F17, F14, F13, F12.
- **Operational one-shots:** F9, F10, F11.
- **Conditional:** F19 (compose), F20 (Henesys portals diagnose-then-fix).

## Files by followup

### F1 — publish 500
- `services/atlas-data/atlas.com/data/baseline/publish.go` — Publisher struct + `Publish` (32–75) + `dumpTable` + `runCopyOut`.
- `services/atlas-data/atlas.com/data/baseline/handler.go:28–55` — REST wrapping; `Publish` error currently surfaced via `http.Error(w, err.Error(), 500)`.
- `services/atlas-data/atlas.com/data/baseline/publish_test.go` — NEW test exercising the failure path.

### F2 — Commodity worker
- `services/atlas-data/atlas.com/data/commodity/processor.go` — `RegisterCommodity` wraps the whole register in one `database.ExecuteTransaction` (line 40).
- `services/atlas-data/atlas.com/data/commodity/processor_test.go` — NEW test exercising chunk commits.

### F3 — Scope resolver negative cache
- `services/atlas-renders/atlas.com/renders/storage/scope.go:18–34` — gate the `Caches.Scope.Add` on `has == true`.
- `services/atlas-renders/atlas.com/renders/storage/smap.go:61–76` — `ResolveSmapScope` falls back to `shared` and unconditionally caches it; same negative-cache-skip treatment.
- `services/atlas-renders/atlas.com/renders/storage/scope_test.go` — NEW.

### F4 — wz seek-path annotations
- `services/atlas-data/atlas.com/data/data/workers/runtime.go:108–134` — `fetchArchive`.
- `libs/atlas-wz/wz/file.go:222–293` — `tryParseWithVersion`.
- `libs/atlas-wz/wz/image.go:107–137,140–300,303–372,375–405` — `parsePropertyList`, `parsePropertyValue`, `parseExtendedProperty`, `parseCanvasProperty`, `parseSoundProperty` — all reached through `Image.parse()` under `parseMu`.

### F5 — Restore atomicity
- `services/atlas-data/atlas.com/data/baseline/restore.go:44–128` — `Restore` (per-table TX) + tail UPSERT into `tenant_baselines`.
- `services/atlas-data/atlas.com/data/baseline/restore_test.go` — NEW (or extend if exists).

### F6 — Properties() signature change
- `libs/atlas-wz/wz/image.go:43–73` — `func (i *Image) Properties() []property.Property` → `(...) ([]property.Property, error)`.
- Call sites that must be updated atomically:
  - `libs/atlas-wz/charparts/smap.go:40`
  - `libs/atlas-wz/charparts/extract.go:240`
  - `libs/atlas-wz/mapimage/minimap.go:18,52`
  - `libs/atlas-wz/mapimage/layers.go:49,135`
  - `libs/atlas-wz/mapimage/decoder.go:80,116,137`
  - `libs/atlas-wz/icons/extract.go:51,118,166,194`
  - `libs/atlas-wz/wz/parse_race_test.go:81`
  - `services/atlas-data/atlas.com/data/data/workers/ui.go:44`
  - `services/atlas-data/atlas.com/data/data/workers/item.go:77`
  - `services/atlas-data/atlas.com/data/data/workers/skill.go:64`
  - `services/atlas-data/atlas.com/data/data/wztoxml/adapter.go:76`
- Build impact services (per `go.work` workspace consumers of `libs/atlas-wz`): `atlas-data`, `atlas-renders`, `atlas-character-factory`. Per CLAUDE.md run `docker buildx bake atlas-<svc>` for each from the worktree root.
- Note: `services/atlas-party-quests/.../instance/processor.go` and `.../stage|definition/rest.go` call `.Properties()` on local domain types (Stage/StageBonus), NOT on `wz.Image` — they are NOT affected.

### F7 — Pin atlas-renders
- `deploy/k8s/overlays/main/kustomization.yaml:179–291` — `images:` list (alphabetical, currently goes atlas-reactors → atlas-saga-orchestrator).

### F8 — Routes-config dedupe
- `deploy/shared/routes.conf` — canonical source (no FQDN).
- `deploy/k8s/base/atlas-ingress.yaml:6–497` — `routes.conf.template` inline block to be replaced with `configMapGenerator`.
- `deploy/k8s/base/kustomization.yaml` — add `configMapGenerator` entry.
- NEW: `tools/gen-routes.sh` — rewrites bare hostnames to `<svc>.${POD_NAMESPACE}.svc.cluster.local`.

### F9 — Recreate cutover runbook
- NEW: `docs/deploy/runbooks/recreate-strategy-cutover.md` (folder doesn't exist yet).

### F10 — Stale layer-png cleanup
- NEW: `docs/deploy/runbooks/clean-stale-layer-pngs.md`.

### F11 — No-bounds maps triage
- NEW: `tools/triage-no-bounds.sh` (or `tools/triage-no-bounds/main.go`).
- NEW: `docs/tasks/task-076-task071-followups/no-bounds-triage.json`.

### F12 — wzinput PATCH comment
- `services/atlas-data/atlas.com/data/wzinput/resource.go:15–24` — `InitResource` — add docstring above the PATCH line.
- `services/atlas-data/atlas.com/data/wzinput/upload.go` (the multipart handler — verify path during impl).

### F13 — Dead `processData`
- `services/atlas-data/atlas.com/data/data/resource.go:31–59` — `processData` is the orphan. The comment block at 22–25 already labels it.

### F14 — wzinput status.go MarshalResponse
- `services/atlas-data/atlas.com/data/wzinput/status.go:21–50` — replace `json.NewEncoder(w).Encode(map[...])` with `server.MarshalResponse[Status]`.

### F15 — Extract ExtractLayout/ExtractLayers helper
- `libs/atlas-wz/mapimage/layers.go` — bodies of `ExtractLayout` (45–94) and `ExtractLayers` (131–197) share ~80%.
- Helper signature (proposed): `extractLayoutCommon(img *wz.Image) (maplayout.Layout, []layerSubInfo, error)` where `layerSubInfo` is a private struct `{ID int, Name string, props []property.Property, objs []objEntry, tiles []tileEntry}`.
- `libs/atlas-wz/mapimage/layers_test.go` — existing regression coverage (if any) drives no-output-drift verification.

### F16 — accessoryPartClassFor → atlas-constants
- `libs/atlas-wz/charparts/extract.go:93–108` — function body to refactor.
- `libs/atlas-wz/go.mod` — add `require github.com/Chronicle20/atlas/libs/atlas-constants v...`.
- `libs/atlas-constants/item/constants.go` — already has `ClassificationFaceAccessory=101`, `ClassificationEyeAccessory=102`, `ClassificationEarring=103`. No new symbols to add.
- Note: `go.work` already contains `./libs/atlas-constants`. Root `Dockerfile` already `COPY`s it for every service. No Dockerfile changes required.

### F17 — Concurrent Properties() regression test
- `libs/atlas-wz/wz/parse_race_test.go` — extend with a real concurrent-Properties-on-different-Images test. The existing `TestLockParseIsExclusive` only tests the mutex primitive.
- NEW: `libs/atlas-wz/wz/testdata/concurrent.wz` — small WZ fixture with ≥4 Image children. **Strategy:** use the smallest legal WZ slice from an existing test fixture (or generate programmatically if a writer exists). Confirm during step 1 of this task.

### F18 — Routes k8s validation
- `deploy/shared/test/routes_nginxt.sh` — extend (or split into a sibling) to validate the kustomize-generated k8s ConfigMap.

### F19 — atlas-renders in compose
- `deploy/compose/docker-compose.core.yml:177` — mirror atlas-data block shape.

### F20 — Henesys portal duplication
- `libs/atlas-wz/mapimage/layers.go:294–317` — `extractPortals` (likely root-cause site).
- `services/atlas-portals/` — read-side service to probe first per OQ-6.
- NEW: `libs/atlas-wz/mapimage/layers_portal_test.go` with a Henesys-shaped portal fixture.

## Build & verify gates (CLAUDE.md)
Run from worktree root before claiming branch done. For each Go module touched:
1. `go test -race ./...`
2. `go vet ./...`
3. `go build ./...`
4. `docker buildx bake atlas-<svc>` for every service whose `go.mod` was touched. Specifically:
   - F16 touches `libs/atlas-wz/go.mod` → bake `atlas-data`, `atlas-renders`, `atlas-character-factory`.
   - F6 changes a public symbol in `libs/atlas-wz` (no `go.mod` change) but the symbol is consumed by services; bake the same three to confirm.
5. Code review via `superpowers:requesting-code-review` BEFORE opening PR (CLAUDE.md hard rule).

## Decisions locked in design.md (recap)
- **OQ-1 (F1 RC):** instrument → repro → fix in a single change to `publish.go` (buffer tar to temp file mirroring `restore.go`, wrap errors with step context, log every step).
- **OQ-2 (F3):** option (a) — don't cache "shared" verdicts. Cache only positive tenant-scoped hits. Same shape in both `scope.go` and `smap.go`.
- **OQ-3 (F5):** option (b) — two-phase finalization. `tenant_baselines` UPSERT lands last; mid-restore failure triggers DELETE sweep across `DumpTables`.
- **OQ-4 (F6):** option (b) — `Properties() ([]property.Property, error)`. All 18 call sites updated atomically.
- **OQ-5 (F8):** `deploy/shared/routes.conf` canonical; k8s ConfigMap generated via `tools/gen-routes.sh` + `configMapGenerator`.
- **OQ-6 (F20):** read-side first (atlas-portals dump for Henesys 100000000 etc.) then walk to atlas-data extraction if needed.
- **OQ-7 (F11):** cross-reference 359 candidate map IDs against any other map's portal `tm` field via a one-shot tool reading post-ingest state.
- **OQ-8 (F9):** create `docs/deploy/runbooks/` as the home for ops runbooks; seed with `recreate-strategy-cutover.md`.

## Risk register
- **R1 — F1 diagnosis time:** if 1st repro cycle doesn't pin the cause, split F1 into instrumentation-only + a follow-up task.
- **R2 — F6 compile break:** atomic update across 18 call sites; rebase quickly to dodge merge conflicts.
- **R3 — F8 generator complexity:** kustomize `configMapGenerator` can fight with templating; fallback is "FR-F18-only validation, no dedupe."
- **R4 — F20 structural fix:** if root cause needs a portal schema change, land diagnosis-only and spin out the fix.
- **R5 — F17 fixture license:** if a writer doesn't exist, slice an existing WZ fixture; confirm attribution before commit.
- **R6 — F2 perf:** chunking adds commit overhead. PRD caps regression at 20%; re-tune chunk size if exceeded.

## What's explicitly NOT in this task
- No new Kafka topics, REST endpoints, or shared libs.
- No retrofit of task-071's PRD/design/plan.
- No expansion of the followups inventory.
- F11 user-visible map fixes (data curation) — those become separate tasks.
- F20 structural reshape — separate task if diagnosis warrants it.
