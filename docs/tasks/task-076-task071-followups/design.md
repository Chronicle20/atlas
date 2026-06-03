# Task-076 Task-071 Followups — Design

Status: Draft (post-PRD)
Created: 2026-05-22
PRD: [`prd.md`](./prd.md)
Parent task: [`task-071-gamedata-minio-consolidation/followups.md`](../../../../task-071-gamedata-minio-consolidation/docs/tasks/task-071-gamedata-minio-consolidation/followups.md) (consulted via the task-071 worktree)

---

## 1. Scope and Shape of the Branch

This task closes the entire `followups.md` inventory (F1–F20). It is **not** one feature; it is twenty disjoint changes in shared services and libs, grouped into the two waves the PRD called out. The branch is laid out as one PR composed of many small commits — one commit per followup, named `fix(<service>): <followup-id> <short>` — so any single followup can be cherry-picked or reverted without touching the others.

The wave ordering is preserved because Wave 1 items (F1, F2, F3, F5, F7) cure live production bugs and must merge first; Wave 2 items (lint, hygiene, coverage, deferred carve-outs) are debt. The plan-task phase will translate that order into a concrete commit sequence. This design doc does not re-list every FR — it focuses on the decisions the PRD's Open Questions section (OQ-1 through OQ-8) defers to design.

Three followups touch shared libs (`libs/atlas-wz`) and therefore require the multi-service rebuild rule from CLAUDE.md: F4, F6, F16 each force a rebuild of every Go service that imports `libs/atlas-wz`. The remaining items are localized to a single service or to deploy config.

No new services. No schema migrations. One new lib dependency (`libs/atlas-constants` → `libs/atlas-wz`, see F16). No new Kafka topics or REST endpoints.

## 2. Decisions for Each Open Question

### OQ-1 — F1 publish 500: root cause

The PRD lists three hypotheses (`io.Pipe` race, minio chunked encoding, pgx CopyTo on a `json` column). The design picks one **diagnostic order** rather than a guessed root cause. The fix lands only after diagnosis is captured in the implementation plan's repro log.

**Diagnostic order (cheapest → most expensive):**

1. **Instrument the existing `publish.go` flow with structured logging at every step boundary.** Lines `32-69` already have a goroutine pattern with an error channel; adding `L.Debug` lines at "tar write start", "dumpTable <name> start/end", "writer goroutine exit", "MC.Put start/end", and "errc receive" — *and writing the final response with the status code* — costs zero risk and surfaces which step actually returns the 500. The PRD already mandates this in FR-F1.3's NFR ("Handler entry, intermediate steps, and completion are all logged") so the diagnostic instrumentation is also the production-quality fix. The PR-544 evidence ("Handling [POST]" fires, no completion or error log") strongly suggests the handler is returning an error path that bypasses `MarshalResponse`. The current handler (`handler.go:39-43`) calls `http.Error(w, err.Error(), http.StatusInternalServerError)` if `Publisher.Publish` returns an error; an empty body with 500 means either the `err.Error()` is the empty string or the response has already been committed. The probable root cause: the writer goroutine's `tar.Close()` (`publish.go:38`) fires after `pw.Close()` (`publish.go:37`), so any tar-finalization error (footer write to closed pipe) is silently dropped while `MC.Put` already finished and the `errc <- nil` path executes, yet between gates a downstream finalization step exits with a nil error wrapped in an unhelpful empty `errors.New("")` — the empty body is consistent with this. The diagnostic instrumentation will confirm or refute this in one bootstrap cycle.

2. **If the logs show MC.Put failing**, capture the minio-go error string and check whether the canonical bucket's policy permits multipart uploads. Hypothesis 2 in the PRD. Mitigation if confirmed: pass an explicit `int64` size by buffering the tar to a temp file (already the pattern used in `restore.go:58-68`); this also lets us hash the tar before upload instead of in parallel with it.

3. **If the logs show `dumpTable` failing**, narrow to the COPY-binary call on the `documents` table. Hypothesis 3. Mitigation if confirmed: the `documents` table's `content` column is `JSON`/`JSONB`; `COPY ... TO STDOUT (FORMAT binary)` should handle that natively, but if the failure reproduces with a smaller table list we exclude `documents` from the canonical dump tables and document the gap. Highly unlikely given task-071 perf testing — included only for completeness.

**Implementation shape:** the fix lands as a single change to `publish.go` that (a) adds the logging line at every step, (b) buffers the tar to a temp file (matching `restore.go` symmetry), and (c) returns a typed error wrap (`fmt.Errorf("publish: <step>: %w", err)`) so the error body is never empty. A unit test in `baseline/publish_test.go` exercises a failing-DB-handle case and asserts the response body contains `publish: dump-table:` as a stable substring, satisfying FR-F1.4.

**Acceptance gate:** the implementation plan's first task is "produce a repro log of the 500 against atlas-main or a fresh PR env" and that log must be committed alongside the fix. Without it the fix is speculative.

### OQ-2 — F3 scope resolver strategy

Three candidates from the PRD: (a) don't cache negative results; (b) TTL the cache; (c) invalidate on DATA_UPDATED.

**Decision: (a) — don't cache the "shared" verdict. Cache only positive tenant-scoped hits.**

Rationale:
- The race is asymmetric. A "shared" verdict becomes wrong when a tenant later publishes data. A "tenants/<id>" verdict cannot become wrong (data does not get un-published by ingest; tenant cleanup is the only delete vector and it has its own teardown path that flushes the renders pool — see `tenantpurge` in atlas-data). So caching only positives gives correctness without complexity.
- Option (b) TTL has the same fix shape (pin temporarily then re-probe) but adds knob debt (operators need to know the TTL value, monitoring needs a dashboard) and still has a worst-case window where the negative is wrong.
- Option (c) DATA_UPDATED invalidation requires adding a Kafka consumer to atlas-renders, which currently consumes no Kafka topics (verified by grep). That's net-new infrastructure for a problem that "don't cache negatives" solves in three lines.
- The cost of re-probing on every "shared" miss is one MinIO `HasAny` per render request that didn't have published tenant data. On the steady state — most maps are shared, most probes hit shared — this is a constant ~5-15ms MinIO list per non-cached path. Acceptable; renders are already cache-bound on Map/Atlas/Smap which gate behind their own LRUs.

**Implementation shape:** change `services/atlas-renders/atlas.com/renders/storage/scope.go:28-33` so the `s.Caches.Scope.Add(cacheKey, scope)` only runs when `has == true`. The cache is read-after-write only meaningful for hits; misses always probe. The symmetric helper in `smap.go:61-76` (also flagged in LINT-08) gets the same treatment.

**Test:** add `scope_test.go` with a mock `MC.HasAny` that returns `false` then `true` across two calls; the test asserts the second call returns `"tenants/<id>"` without pod restart. Satisfies FR-F3.2.

### OQ-3 — F5 restore atomicity strategy

Two candidates: (a) wrap the whole table loop in one outer transaction; (b) two-phase commit where the `tenant_baselines` marker is written only after all tables succeed.

**Decision: (b) — two-phase finalization.**

Rationale:
- Option (a) requires wrapping the entire restore in a single transaction whose duration is the dump's restore time (worst case minutes for a full canonical dump). That holds locks (potentially `AccessExclusiveLock` on the `DELETE` paths) for the entire window and serializes any concurrent baseline operations. It also can't survive a server-side `idle_in_transaction_session_timeout`, which is exactly the failure mode that bit F2.
- Option (b) keeps the existing per-table-transactional `restoreOneTable` (which is correct in isolation: each `DELETE` + `COPY` for one table is atomic and rolled back if `COPY` fails mid-table), and only changes the **finalization gate**. The `INSERT INTO tenant_baselines` UPSERT is moved to land *only* if every table loop iteration succeeded *and* every ANALYZE succeeded. Mid-restore failure leaves per-table data possibly inconsistent across tables — but with no `tenant_baselines` marker, downstream readers behave identically to "never restored" (the marker is the only signal that a baseline exists for a tenant).
- This requires also that mid-restore failure **clean up any partially-restored data** so a subsequent successful restore is not blocked by stale rows. Strategy: on any error after the first `DELETE` in `restoreOneTable` returns, the deferred recovery path re-runs `DELETE FROM <table> WHERE tenant_id = ?` for every table in `DumpTables` before returning. The DELETE is idempotent; running it on tables that were never touched is a no-op.

**Implementation shape:**
- `restore.go:Restore()` accumulates table-write errors and, on the first error, enters a `cleanupAfterFailure(ctx, db, target)` path that DELETEs every `DumpTables` row for `target` (using a fresh transaction per table to bound the lock window).
- The `INSERT INTO tenant_baselines` block (`restore.go:118-126`) is the last side effect; nothing executes after it.
- ANALYZE (`restore.go:113-117`) stays in the success path; an ANALYZE failure now triggers the cleanup path and returns the ANALYZE error.

**Test:** new `restore_test.go` (or extend existing) injects a failure at the second iteration of the table loop using a sqlmock-style `*sql.DB`, then asserts (a) no `tenant_baselines` row exists for `target`, and (b) each table is empty for `target`. Satisfies FR-F5.2/3.

**Caveat documented in design.md:** the cleanup is best-effort. If the cleanup itself fails (e.g., the database is unreachable mid-cleanup), the tenant's per-table state is partially restored and no marker is set. The next restore attempt's `DELETE FROM <table> WHERE tenant_id = ?` will sweep the residue. Operators are not expected to take manual action. This is documented inline near the cleanup helper.

### OQ-4 — F6 Properties() API shape

Two candidates: (a) add `Err()` accessor on `*Image`; (b) change `Properties()` signature to `([]Property, error)`.

**Decision: (b) — `Properties() ([]Property, error)`.**

Rationale:
- The audit log lists 18 call sites across `libs/atlas-wz/`, `services/atlas-data/`, and one test. That's a manageable atomic update across the monorepo's go.work workspace.
- Option (a)'s `Err()` accessor preserves the current call shape but creates a footgun: callers that don't check `Err()` keep their silent-fail behavior; the original LINT-03 finding stays open as "callers may still ignore." The new pattern is opt-in for safety, which is the opposite of what we want.
- Option (b) is a compile break — every existing caller must be updated to either propagate the error or pin it down with an explicit `_ = err` and a comment. The compile-break enforces the discipline that LINT-03 wanted: silent-fail is impossible because the type system requires a decision at each call site.
- The audit listed three call patterns where ignoring the error is acceptable: (1) icons/extract.go's linked.Properties() (already wrapped in a "best effort" sweep over linked images), (2) test-only NewParsedImage paths whose Properties() always succeeds, (3) charparts/smap.go's smapImg.Properties() where the worker emits a sentinel and continues. Each of these gets an inline `// best-effort: <reason>` comment instead of a discarded value.
- This is the same shape used elsewhere in the codebase (e.g., `wz.Open` already returns `(*File, error)`); the change brings `Properties()` in line with the lib's surrounding conventions.

**Implementation shape:**
- `libs/atlas-wz/wz/image.go:Properties()` changes signature.
- All 18 call sites in `libs/atlas-wz/`, `services/atlas-data/`, and the one in `parse_race_test.go` are updated. Each caller's existing error handling style is preserved; new callers that want to propagate use `fmt.Errorf` wrapping.
- The `Image.parse()` private method is unchanged; the public surface is the only break.

**Build impact:** every Go service that imports `libs/atlas-wz` will need a rebuild. Per `.github/config/services.json`, that includes at minimum `atlas-data`, `atlas-renders`, and `atlas-character-factory` (the three known wz consumers). `docker buildx bake` for each — per CLAUDE.md rule.

### OQ-5 — F8 routes-config source of truth

Two candidates: (i) `deploy/shared/routes.conf` canonical; the k8s file generated; (ii) the k8s file canonical; the shared file deleted; or (iii) merge by templating.

**Decision: (i) — `deploy/shared/routes.conf` is canonical; the k8s ConfigMap entry is generated from it.**

Rationale:
- The shared file is a self-contained nginx config that already works as-is in compose. The k8s file is a *templated* variant (uses `${POD_NAMESPACE}.svc.cluster.local`) that needs nginx env-templating to render at pod startup. The two differ in **hostname expansion**, not routes. So "generate" means "rewrite each `set $u "X:8080"` → `set $u "X.${POD_NAMESPACE}.svc.cluster.local:8080"`".
- Kustomize's `configMapGenerator` can read a file from disk and produce a ConfigMap. Combined with a `transformers:` patch (or a pre-build sed step in a Makefile), this gives a deterministic single-source flow.
- Option (ii) loses the compose-local-dev shape. We'd have to extract the bare-hostname form anyway because compose doesn't have `POD_NAMESPACE`. So (ii) doesn't actually eliminate the duplication; it just moves it.

**Implementation shape:**
- Add a generator step in `deploy/k8s/base/`'s Makefile or a `tools/gen-routes.sh` script that reads `deploy/shared/routes.conf`, applies the FQDN rewrite, and writes the resulting block into a kustomize-managed ConfigMap.
- Replace the inline `routes.conf.template` block in `deploy/k8s/base/atlas-ingress.yaml:46-80+` with a `configMapGenerator` entry in `deploy/k8s/base/kustomization.yaml` that consumes the generated file.
- Add a `make` target (or document the script) and run it as a prerequisite step in CI (and locally before kustomize-build). FR-F18 then validates the generated output matches what's deployed.
- **Atomic update with PR-544 routes:** the routes added in commits 6da0bc363, e9244ba14, 2527c4541 must already be present in `deploy/shared/routes.conf` (verified pre-task). The dedupe step regenerates the k8s side from the shared file and confirms byte-equivalence of the resulting nginx config.

**Test:** FR-F18 satisfied by extending `deploy/shared/test/routes_nginxt.sh`: after running `nginx -t` on the shared file, the script re-runs `nginx -t` on the generated k8s file (with `POD_NAMESPACE=test` substituted), and asserts both succeed and the route declarations are equivalent (sorted-diff on `location ~ ^...` lines must be empty).

### OQ-6 — F20 service location

The PRD's likely candidates are `services/atlas-data/atlas.com/data/portal/...` (ingest-side extraction) and `services/atlas-portals/...` (read-side).

**Triage approach: read-side first, then ingest-side.**

Rationale:
- The bug report ("portal list has duplicates") is a *symptom* — whoever calls "the portal list" sees duplicates. The cheapest first probe is to dump what atlas-portals returns for Henesys map IDs (104000000, 100000000, 200000001, etc.) and see if the duplicates are present at the read endpoint.
- If yes, the duplicates exist in storage (atlas-data extracted them with duplicates and atlas-portals just forwards them). Move investigation to atlas-data's portal worker.
- If no, the duplicate is layered on at read time by atlas-portals (e.g., a join, a re-emission of the static block).

**Likely-root-cause hypothesis to verify against repo source (not committing to it):**
- `extractPortals` in `libs/atlas-wz/mapimage/layers.go:295-317` iterates portal subtrees with **no dedup** by portal name or by (target, x, y) tuple. WZ data for some maps is known to contain shadow portal entries (e.g., `0`, `00`, `out_n`) that share coordinates with player-visible portals.
- Diagnostic: dump the portal subtree for Henesys map 100000000 from a freshly-parsed Map.wz and inspect for either (a) entries with the same `pn` repeated, or (b) entries with `pn = ""` co-located with named portals.
- **Likely fix:** filter or dedup at extraction time. If WZ data is the source, the fix is in `extractPortals`. If atlas-portals layers its own copies, the fix is there.

**Implementation gate (per PRD):** if diagnosis reveals the fix is structural (e.g., portal schema needs a "kind" field added to disambiguate visible-vs-internal portals), spin out the fix as a separate task and only commit the diagnosis+repro to this branch. Otherwise land the fix here.

**Test:** regardless of fix location, add a regression test against a hand-crafted Henesys-style portal fixture asserting that the deduplicated list contains the expected named portals exactly once.

### OQ-7 — F11 map-id list source

The PRD asks where the in-game-accessible map list comes from. Per CLAUDE.md's "Verification Over Memory" rule, we cannot cite map IDs from memory.

**Decision: derive the accessibility set from the same data the ingest just parsed.**

Approach:
- The Map worker in `atlas-data` emits one document per parsed Map.img, keyed by mapID. The 359 "no-bounds" maps are the subset whose `extractLayoutErrs` is non-zero. So the input list is already known to the ingest run — it's logged at end-of-worker.
- A map is "user-visible" if at least one portal (in some other map) targets its ID. Cross-reference: take the 359 candidate map IDs and ask "is any of you the `tm` (target map) of a portal in some other map's extracted layout?"
- Implementation: a one-shot script in `tools/triage-no-bounds.sh` (or a Go binary under `services/atlas-data/cmd/triage-no-bounds/`) that opens the ingest's emitted MinIO/DB state, runs the cross-reference, and writes the answer as a JSON file committed alongside the design under `docs/tasks/task-076-task071-followups/no-bounds-triage.json`.
- For any map ID that *is* targeted by a portal, file a follow-up task as `task-NNN-no-bounds-<map-id>` (one task per cluster, not per ID).
- For unreachable IDs, the JSON triage file is the documentation. `extractLayoutErrs=359` becomes the expected baseline going forward; CI doesn't need to gate on it.

### OQ-8 — F9 documentation location

The PRD asks where operational runbooks live.

**Decision: create `docs/deploy/runbooks/` and seed it with this playbook.**

- The repo currently has `docs/superpowers-integration.md`, `docs/TODO.md`, and per-task `docs/tasks/...` but no general operations folder.
- The Recreate-cutover playbook is the first in what will likely be a small but growing set (similarly: how to flush atlas-maps spawn cache after atlas-data redeploy, per the user's memory note). Putting it under `docs/deploy/runbooks/` gives a single home.
- The file: `docs/deploy/runbooks/recreate-strategy-cutover.md`. Includes the exact `kubectl patch --type=json` recipe used on 2026-05-22, an explanation of the SSA orphan-field workaround, and a "when to use" trigger (any `RollingUpdate → Recreate` migration on an existing Deployment).
- FR-F9.3's optional kustomize patch is **rejected for atlas-main** (the incident is resolved on that env); kept as a recipe inside the playbook for future cutovers elsewhere.

## 3. Per-Followup Implementation Sketches (Wave 1)

Sketches below assume the OQ decisions above and are descriptive, not exhaustive. The plan-task phase derives concrete task-list entries from each sketch.

### F1 — publish 500

- Touch `services/atlas-data/atlas.com/data/baseline/publish.go` to: (a) buffer the tar to a temp file before upload (mirror `restore.go` shape), (b) log at every step boundary, (c) wrap all errors with `fmt.Errorf("publish: <step>: %w", err)`.
- Touch `services/atlas-data/atlas.com/data/baseline/handler.go:39-43` to surface the error message in the body (`http.Error(w, fmt.Sprintf("publish failed: %s", err), 500)`) when the publish returns non-nil.
- Add `publish_test.go` with a test using a stub `*gorm.DB` and a stub `MC` that simulates a failure in the dump-table phase; assert the response body is non-empty and contains the expected substring.
- Repro the bug on a fresh PR env or atlas-main before declaring done. Capture the log output in the implementation plan.

### F2 — Commodity chunked transactions

- Touch `services/atlas-data/atlas.com/data/commodity/processor.go:36-46` to remove the outer `database.ExecuteTransaction` wrap and instead let `Register` commit each `s.Add(ctx)(m)` operation in its own micro-transaction.
- `document.Storage.Add` already calls `tx.Save(...).Error` internally; if it currently inherits a wrapping transaction, change to commit per `Add` (verify by reading `document/storage.go`).
- If per-row commits are too slow, batch by 100 rows per transaction — chunk size is a tunable.
- Test: a unit test with a `sqlmock`-style DB that fails on the second batch's `tx.Commit()` and asserts (a) the first batch's rows are still persisted, (b) the error surfaces to the caller. Satisfies FR-F2.3.
- **Perf gate:** measure full Etc.wz import time against the same task-071 fixture; PRD NFR caps regression at 20%. If exceeded, increase chunk size and re-measure.

### F3 — Scope cache (negative-only-skip)

- Touch `services/atlas-renders/atlas.com/renders/storage/scope.go:28-33` and `services/atlas-renders/atlas.com/renders/storage/smap.go:61-76` so `s.Caches.Scope.Add(cacheKey, ...)` is gated on `has == true` (scope.go) and on the "tenants/<id>" branch only (smap.go).
- Add `scope_test.go` with a mock `MinioHasAny`-like interface that returns false then true; assert the second call returns the tenant-scoped result. Same for smap_test.go.
- **No observability change** needed: the existing render-path logs already include the resolved scope. Operators see the recovery in the next render attempt's log line. (FR-F3 NFR for cache-invalidation logging is satisfied by virtue of "never cached negative" — there's nothing to log on invalidation because there's no invalidation event.)

### F5 — Whole-dump-atomic restore (two-phase finalization)

- Touch `services/atlas-data/atlas.com/data/baseline/restore.go` to: (a) accumulate all-tables-loop errors, (b) on first error, run `cleanupAfterFailure(ctx, db, target)` which DELETEs each `DumpTables` row for `target` in its own transaction, (c) move the `INSERT INTO tenant_baselines` UPSERT to land only on full success after the ANALYZE pass.
- Add structured logs at restore start, per-table completion, ANALYZE completion, and finalization (or cleanup) so operators see the state machine in Loki.
- Add `restore_test.go` (extend if exists) with a fail-injection on the second table; assert no tenant_baselines row and all touched tables are empty for `target`.

### F7 — Pin atlas-renders in main overlay

- One-line(ish) edit to `deploy/k8s/overlays/main/kustomization.yaml`: insert under `images:`, lexically sorted between `atlas-reactors` and `atlas-saga-orchestrator`:
  ```yaml
  - name: ghcr.io/chronicle20/atlas-renders/atlas-renders
    newTag: main-<current-good-sha>
  ```
- Pick the SHA: the most recent successful `bot/main-image-bump-*` PR's atlas-data SHA is a safe choice (renovate runs across the org and bumps siblings together).
- After merge, observe the next renovate run; if a bump PR opens for atlas-renders within 24h, FR-F7.2 is satisfied.

## 4. Per-Followup Implementation Sketches (Wave 2)

### F4 — wz seek-path concurrency sweep

- Audit targets: `fetchArchive` (`services/atlas-data/atlas.com/data/data/workers/runtime.go:115-134`), `tryParseWithVersion` (`libs/atlas-wz/wz/file.go:222-293`), and deeper parser internals beneath `extractZmap`.
- `fetchArchive` is called only during `Open(path)` from a single goroutine per worker; it Seeks during `wz.Open`'s header/version/root parsing before the file is published to any concurrent consumer. Safe by construction; annotate inline: `// fetchArchive runs single-threaded during Open(); no parseMu needed.`
- `tryParseWithVersion` is in the same single-threaded `Open()` path. Same annotation.
- The deeper parser internals (`parsePropertyList`, `parsePropertyValue`, `parseExtendedProperty`, `parseCanvasProperty`) are all reached through `Image.parse()` which holds `parseMu` for the full sub-tree parse. Safe today. Annotate the top-level entries (`parsePropertyList`) with `// invariant: caller holds wzFile.parseMu (entered via Image.parse()).` so future contributors can't accidentally call these from outside Properties().
- No mutex coverage added (none needed); only inline annotations.

### F6 — Properties() signature change

- See OQ-4 decision. 18 call sites updated atomically across `libs/atlas-wz/` and `services/atlas-data/`.
- Build-impact services: `atlas-data`, `atlas-renders`, `atlas-character-factory` (and any other go.work member that imports `libs/atlas-wz`). Run `docker buildx bake` per CLAUDE.md.

### F8 — Routes-config dedupe

- See OQ-5 decision. Implementation requires: (a) a generation script, (b) a kustomize generator entry, (c) removing the inline ConfigMap block from `atlas-ingress.yaml`, (d) updating `routes_nginxt.sh` to validate both.
- The PR-544 fix routes (commits 6da0bc363/e9244ba14/2527c4541) are verified present in the shared file before the dedupe lands; the generated k8s file post-dedupe matches the deployed-state byte-for-byte (modulo FQDN rewrite).

### F12 — wzinput PATCH comment

- Add a 2-3 line block comment above the PATCH handler in `services/atlas-data/atlas.com/data/wzinput/resource.go:20` (per audit LINT-07) explaining why it doesn't use `rest.RegisterInputHandler[T]` (the multipart body is byte-streamed, not JSON-decoded).
- No code change.

### F13 — Dead orphan: processData

- Per audit LINT-02 (confirmed): the orphan is `processData` in `services/atlas-data/atlas.com/data/data/resource.go:31-59`. The comment block at `:22-25` already labels it "now-orphaned."
- Delete the function. Grep confirms no test or route references it. The accompanying handler-level comment can stay or be tightened.

### F14 — wzinput/status.go manual envelope

- Replace `json.NewEncoder(w).Encode(map[string]any{...})` in `services/atlas-data/atlas.com/data/wzinput/status.go:40-48` with a call to `server.MarshalResponse[Status](...)` matching the pattern used in `baseline/handler.go:52`.
- `Status` already has the correct JSON shape; add a `GetName()` and `GetID()` if `MarshalResponse` requires them on the model type.

### F15 — Extract shared helper for ExtractLayout / ExtractLayers

- Pull out a private `func extractLayoutCommon(img *wz.Image) (maplayout.Layout, []layerSubInfo, error)` in `libs/atlas-wz/mapimage/layers.go` that returns the bounds + layout + a slice of `{layerIndex, layerSub, objs, tiles}` entries for layers that pass the "has tiles or objs" filter.
- `ExtractLayout` calls the helper and emits the `Layers` slice as it does today.
- `ExtractLayers` calls the helper and additionally composites each layer.
- Regression-test by running the existing `mapimage` tests pre- and post-refactor and asserting byte-identical outputs.

### F16 — accessoryPartClassFor → libs/atlas-constants

- Add `require github.com/Chronicle20/atlas/libs/atlas-constants v...` to `libs/atlas-wz/go.mod`. Workspace already includes both; just declare the dep.
- Rewrite `accessoryPartClassFor` in `libs/atlas-wz/charparts/extract.go:97-108` to call `item.ClassificationFor(id)` (or whatever the existing helper is) and map the returned `item.Classification` (101/102/103) to the strings "FaceAccessory"/"EyeAccessory"/"Earrings".
- **CLAUDE.md rule check:** `libs/atlas-constants` is already in `go.work` (verified). The root `Dockerfile` already `COPY`s it for every service. No Dockerfile change needed.
- Test: `extract_test.go:91-94` already cases this function; outputs must be identical.

### F17 — Concurrent Properties() race regression test

- Add `libs/atlas-wz/wz/testdata/concurrent.wz` — a hand-crafted small WZ archive with ≥4 Image children whose parsed property trees differ. Generating this fixture from scratch is non-trivial; either (a) check in a slice of an existing WZ file (~MB-scale subset) with an in-repo CC-by-* attribution, or (b) generate it programmatically via the lib's own writer (if one exists; verify).
- **Decision: programmatic generation.** Check whether `libs/atlas-wz/` exposes a writer; if yes, the fixture is generated by a `TestMain` setup step (so it doesn't bloat git). If no, fall back to the smallest legal WZ slice committed verbatim. The plan-task phase resolves this.
- The test itself (`libs/atlas-wz/wz/parse_race_test.go`) opens the fixture, spawns 16 goroutines, each calling `Properties()` on a different `*Image`. Run under `-race`. Negative-control validation: temporarily remove `parseMu` from `LockParse()` and rerun; the test must fail.

### F18 — Routes k8s validation

- Extend `deploy/shared/test/routes_nginxt.sh` to additionally validate the kustomize-generated k8s ConfigMap content (per OQ-5). The script runs `kustomize build deploy/k8s/base/` (or just `tools/gen-routes.sh`), captures the generated routes-config, expands `${POD_NAMESPACE}=test`, and feeds it through the same `nginx -t` wrapper.
- A "divergence" between shared and k8s is impossible post-dedupe — the k8s side is generated from the shared file. The test instead asserts that if the shared file changes, the generated file is regenerated (CI guard: any PR touching `deploy/shared/routes.conf` must also touch the generated artifact or the script fails).

### F19 — atlas-renders in docker-compose.core.yml

- Insert an `atlas-renders` block between existing alphabetical neighbors in `deploy/compose/docker-compose.core.yml` matching the atlas-data block's shape:
  - `build: { context: ../.., dockerfile: Dockerfile, args: { SERVICE: atlas-renders } }`
  - `image: atlas-renders:${ATLAS_IMAGE_TAG:-local}`
  - `environment: { LOG_LEVEL, REST_PORT=8080, MINIO_ENDPOINT, MINIO_ACCESS_KEY, MINIO_SECRET_KEY }` — values match the compose-local MinIO instance.
  - `volumes: - ../../tmp/wz-scratch:/scratch/wz` (mirrors the k8s emptyDir, lets local dev keep parsed WZ files across restarts).
  - No `ports:` mapping (per PRD NFR §8 Security — internal only).
- Verify locally: `docker-compose up atlas-renders` starts cleanly, `/healthz` returns 200 inside the compose network.

### F20 — Henesys portal duplication

- Per OQ-6: first probe is read-side (atlas-portals' Henesys 100000000 dump), then walk back to extraction if duplicates exist in storage.
- If the diagnosis is "extractPortals doesn't dedup and Map.wz has shadow entries," the fix is a dedup step in `extractPortals` keyed on portal name (`pn`) — drop the empty-name entries that overlap with named portals. This matches the cosmic-source treatment of internal portals.
- If the diagnosis is "atlas-portals double-emits," the fix is in atlas-portals' handler.
- Either way, add a regression test against a Henesys portal fixture asserting the deduplicated list size.

## 5. Operational One-Shots

These are not code; they are commands run once against atlas-main and every long-lived PR env, then documented.

- **F9** — write `docs/deploy/runbooks/recreate-strategy-cutover.md`. Document the SSA orphan-field workaround and the `kubectl patch --type=json` recipe used on 2026-05-22. One commit.
- **F10** — run `mc rm --recursive --force adm/atlas-assets/tenants/<id>/regions/<r>/versions/<v>/map/*/layers/` for each (tenant, region, version) on atlas-main and on every long-lived PR env. Commit the runbook to `docs/deploy/runbooks/clean-stale-layer-pngs.md` with the iteration loop. The actual `mc rm` execution is operator-side (out of the plan-task code work); the runbook is the deliverable.
- **F11** — generate `docs/tasks/task-076-task071-followups/no-bounds-triage.json` per OQ-7. Commit. File follow-up tasks for any user-visible IDs.

## 6. Service Build Impact

| Service / Lib | Followups touching it | Bake required? |
|---|---|---|
| `services/atlas-data` | F1, F2, F5, F11, F12, F13, F14 | Yes (go.mod touched indirectly via F6 propagation) |
| `services/atlas-renders` | F3 | Yes (F3 + F6 propagation) |
| `services/atlas-character-factory` | (transitive via F6) | Yes |
| `services/atlas-portals` | F20 (conditional) | Yes if touched |
| `libs/atlas-wz` | F4, F6, F15, F16, F17 | N/A directly; forces dependent service rebuilds |
| `libs/atlas-constants` | F16 (new consumer) | N/A |
| `deploy/*` | F7, F8, F18, F19 | N/A (config only) |
| `docs/deploy/runbooks/` | F9, F10 | N/A |

**CLAUDE.md hard gate:** any `go.mod` touched (specifically `libs/atlas-wz/go.mod` for F16) requires `docker buildx bake atlas-data`, `docker buildx bake atlas-renders`, and `docker buildx bake atlas-character-factory` from the worktree root before claiming done. Per CLAUDE.md: this is mandatory.

## 7. Risk Register

- **R1 (high) — F1 diagnosis time.** If logging instrumentation doesn't surface the root cause in the first repro cycle, the plan must allow at least two more diagnosis cycles before declaring the fix speculative. Budget: 1 design+plan cycle for instrumentation, 2 follow-on cycles for narrowing. If still unresolved, F1 is split out as `task-NNN-publish-500-deep-dive` and only the instrumentation lands here.
- **R2 (medium) — F6 compile break.** Atomic update across 18 call sites; if any caller is missed the workspace won't compile and CI catches it. Low risk of leaking past CI; medium risk of merge conflicts with concurrent unrelated branches. Mitigation: land F6 first within Wave 2 and rebase quickly.
- **R3 (medium) — F8 generator complexity.** A script + kustomize generator + CI integration is more moving parts than the simpler "validate both" alternative. If the generator approach gets bogged down in the implementation phase (e.g., kustomize's configMapGenerator semantics don't cooperate), fall back to "FR-F18-only" — the routes test gates divergence, no dedupe required. The plan-task phase commits to a single path and includes the fallback as a branchpoint.
- **R4 (medium) — F20 diagnosis exposes structural bug.** Per PRD §2 non-goal, structural fixes spin out. The plan-task phase includes a checkpoint after diagnosis: if the fix is bounded, continue; if it's structural, the diagnosis-only commit lands and a follow-up task is opened.
- **R5 (low) — F17 fixture generation.** If `libs/atlas-wz/` has no writer (likely), the fixture is a slice of an existing WZ file. License/attribution for that slice needs sorting; the plan-task phase confirms this before committing the file.
- **R6 (low) — F2 perf regression.** Chunking adds commit overhead. PRD NFR caps it at 20%. Mitigation: re-tune chunk size before declaring done.

## 8. Out of Scope (Confirmed)

- No retrofit of task-071's PRD, design, or plan docs.
- No new Kafka topics, REST endpoints, or shared libs.
- No changes to the canonical baseline format (DumpTables list, tar shape, sha256 sidecar key).
- No expansion of the followups inventory itself; new bugs discovered during this task become follow-up tasks.
- The F11 user-visible-map-ID fixes (if any) are explicitly out of scope; the triage commits the inventory and files follow-ups.
- The F20 fix is conditional (see OQ-6, R4); structural reshape is out of scope here.

## 9. Test & Verification Plan Summary

Per PRD §10 acceptance, each followup carries its own test (where testable in code) or its own verification step (where operational):

- **Unit tests added:** F1, F2, F3 (scope.go + smap.go), F5, F15, F17.
- **CI guards added:** F18 (routes-config validation), F4 (annotations + existing parse_race_test stays green).
- **Manual verification:** F1 (curl recipe), F7 (renovate observation), F19 (compose up), F11 (triage JSON committed).
- **Documentation:** F9 (runbook), F10 (runbook), F12 (inline comment), F13 (deletion).
- **Operator-side execution:** F10's MinIO cleanup.

Branch-level acceptance gates from PRD §10 stand without amendment:
- `go test -race ./...` clean per changed module.
- `go vet ./...` clean per changed module.
- `go build ./...` clean per changed service.
- `docker buildx bake atlas-<svc>` from worktree root for every service whose `go.mod` was touched.
- Code review run via `superpowers:requesting-code-review` **before** opening the PR (CLAUDE.md hard rule).
- PR description references back to `task-071/followups.md`.

## 10. Sequencing for Plan-Task

The plan-task phase produces a per-followup task list. Suggested commit order:

**Wave 1 (production hot-path):**
1. F1 — publish 500 (instrumentation + fix)
2. F7 — pin atlas-renders in main overlay
3. F3 — scope cache negative-skip (both scope.go + smap.go)
4. F2 — Commodity chunked transactions
5. F5 — restore atomicity (two-phase finalization)

**Wave 2 (debt):**
6. F8 — routes-config dedupe + generator
7. F18 — routes test extension (validates F8)
8. F6 — Properties() signature change (atomic monorepo update)
9. F4 — wz seek-path concurrency annotations
10. F15 — ExtractLayout/ExtractLayers helper
11. F16 — accessoryPartClassFor → atlas-constants
12. F17 — concurrent Properties() race test
13. F14 — wzinput status.go MarshalResponse
14. F13 — delete processData orphan
15. F12 — wzinput PATCH comment

**Operational one-shots (any time after Wave 1 lands):**
16. F9 — runbook for Recreate cutover
17. F10 — runbook + execute layer-png cleanup
18. F11 — triage 359 no-bounds maps + JSON commit + follow-ups

**Conditional:**
19. F19 — atlas-renders in docker-compose.core.yml
20. F20 — diagnose Henesys portal duplication (then conditional fix)

Each commit follows the form `fix(<service>): F<id> <short>`. The PR description lists all 20 IDs with checkboxes per PRD §10.
