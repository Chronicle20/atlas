# Task-071 Followups — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-05-22
Parent: [`task-071-gamedata-minio-consolidation`](../task-071-gamedata-minio-consolidation/) — see `followups.md` for the rollout-day inventory this PRD is derived from.

---

## 1. Overview

Task-071 ("gamedata minio consolidation") shipped to atlas-main on 2026-05-22. During PR-544 runtime verification and same-day production rollout, 20 distinct followup items were captured in `task-071/followups.md`: real bugs that bit during rollout, operational debt that surprised operators, code-hygiene findings from the backend audit, coverage gaps in the test suite, and items deliberately deferred from task-071's scope. None of them blocked task-071's merge — `atlas-pr-bootstrap` already falls back gracefully when the canonical baseline tar is missing, and operator-side mitigations (manual rollout restart, manual ConfigMap edits, conn-pool retries) papered over the rest on rollout day. But every item is real work that needs to land before the next major data-pipeline change, and several (F1 publish 500, F3 negative scope cache, F5 partial restore, F7 unpinned atlas-renders tag) will cause repeat production incidents if left in place.

This task closes the followups inventory in one coordinated effort. The workstream is organized into the same two waves the followups doc recommended: **Wave 1** (production-affecting hot-path bugs and operability) lands first as it has the highest user/operator impact; **Wave 2** (code hygiene, coverage gaps, deferred carve-outs) lands second. Two items with unknown root cause — F1 (publish 500) and F20 (Henesys portal duplication) — are scoped as **diagnose-then-fix**: design.md must document the confirmed root cause before any fix lands, and acceptance requires both the diagnosis and the fix.

The work spans `atlas-data`, `atlas-renders`, `libs/atlas-wz`, `libs/atlas-constants`, `deploy/k8s/`, `deploy/shared/`, `deploy/compose/`, and tests under each of those. No new services, no schema migrations, no new shared libs are introduced.

## 2. Goals

Primary goals:
- Eliminate the five production-affecting bug classes uncovered during rollout (F1, F2, F3, F5, F6) so they cannot bite on a re-run of the bootstrap/restore flow.
- Restore deployment determinism for `atlas-renders` (F7) by pinning its image tag in the `main` overlay so renovate keeps it in sync going forward.
- Dedupe the two routes-config sources (F8) so a fix in one cannot silently miss the other.
- Close the WZ-parser concurrency gap (F4) the same way CONCURRENCY-01 was closed in `b44f11b48` — by extending mutex coverage to the remaining un-guarded seek/read paths.
- Close every backend-audit LINT item (F12–F16) so the next audit starts clean.
- Add the two regression-test gaps (F17, F18) the followups doc flagged as "would have caught this pre-merge."
- Clean up the two carved-out items (F19 atlas-renders in compose, F20 Henesys portal duplication) so task-071's deferral list goes to zero.
- Document the three one-shot operational items (F9 Recreate cutover, F10 stale layer-PNG cleanup, F11 359 no-bounds maps) and execute them on atlas-main + every long-lived PR env.

Non-goals:
- No changes to the task-071 baseline format, restore mechanism, or storage scope semantics beyond the bug fixes called out below. Behavior changes are limited to bug fixes; the public APIs and storage layout stay as task-071 shipped them.
- No new features, no new endpoints, no new Kafka topics, no new shared libs.
- No expansion to followups not already in `task-071/followups.md`. New issues discovered during this task are documented but spun out as separate tasks unless they directly block a listed item.
- No retrofit of task-071's design or PRD documents (those are historical).
- F20 root-cause work is bounded: if diagnosis exposes a structural bug requiring more than a localized fix (e.g., portal data model needs reshape), the fix is spun out and only the diagnosis lands here.

## 3. User Stories

- As an **operator running the bootstrap flow on a fresh PR env**, I want `POST /api/data/baseline/publish` to succeed for the canonical tenant so that fast-baseline-restore optimizes subsequent PR bootstraps, instead of every PR paying the full-ingest cost (F1).
- As an **operator ingesting Etc.wz**, I want the Commodity worker to survive a transient Postgres connection blip without losing the entire import so I don't have to manually re-run shared-scope ingest on every cold deploy (F2).
- As a **renderer (atlas-renders pod) handling a tenant's first probe**, I want a negative scope-resolver result to *not* pin "shared" forever, so that tenant data published seconds later is visible without a pod restart (F3).
- As a **developer adding a new WZ-archive worker**, I want all `*wz.File` seek/read paths to be mutex-guarded by default so I cannot accidentally re-introduce the CONCURRENCY-01 race (F4).
- As an **operator running a baseline restore**, I want the restore to either fully succeed or leave no `tenant_baselines` marker, so I cannot end up with half-restored state that subsequent reads silently consume (F5).
- As a **WZ-worker author**, I want `Properties()` to surface parse errors so a silent-fail can never again produce zero-row imports without flagging (F6).
- As a **release engineer**, I want every atlas-* service pinned in the main overlay so renovate keeps the deployed SHA in sync and we don't deploy stale `:latest` (F7).
- As a **routes-config maintainer**, I want a single source of truth for ingress route declarations so I cannot edit one file and miss the other (F8, F18).
- As a **release engineer migrating a Deployment's strategy**, I want a documented playbook for the `RollingUpdate → Recreate` cutover so I don't rediscover the SSA-orphan-field workaround under deploy pressure (F9).
- As a **storage operator**, I want stale `layer-*.png` files from the pre-lazy-refactor era removed so MinIO storage cost reflects current code (F10).
- As a **data-curation owner**, I want the 359 "no-bounds" maps reported by every ingest to be triaged once so we know which (if any) are user-visible (F11).
- As a **backend auditor**, I want the LINT-* findings (F12–F16) closed so the next backend-guidelines-reviewer run starts from a clean baseline.
- As a **WZ-parser maintainer**, I want a regression test that exercises the concurrent-Properties()-call shape (F17) so a re-introduction of CONCURRENCY-01 fails CI, not production.
- As a **local-dev user (docker-compose)**, I want `atlas-renders` available in `docker-compose.core.yml` so I can exercise the render paths against MinIO without a k8s cluster (F19).
- As a **player or tester**, I want the Henesys portal list to not contain duplicate entries (F20).

## 4. Functional Requirements

Requirements are grouped by followup ID. Each requirement is testable. Severities are inherited from `task-071/followups.md`.

### 4.1 Hot-path bugs

**FR-F1 — Baseline publish endpoint returns 500 [HIGH]**

- FR-F1.1: Reproduce the 500 on atlas-main using the curl recipe in `followups.md:18-27` and confirm the failure mode (empty body, no completion log).
- FR-F1.2: Document the confirmed root cause in `design.md` before implementing a fix. Acceptable evidence includes log capture, a unit test that fails, or a deterministic local repro. (Hypotheses are listed in `followups.md:29-32` — `io.Pipe` race vs. `MC.Put` chunked encoding vs. `pgx.CopyTo` on `json` column.)
- FR-F1.3: Land the fix. Acceptance: the same curl request returns 2xx with a JSON:API envelope describing the published baseline. Handler entry, intermediate steps, and completion are all logged.
- FR-F1.4: Add a unit or integration test that fails against the pre-fix code and passes against the fixed code. Test must live in the touched package's `*_test.go` (no new test-helper files per the project Test Helper Pattern).

**FR-F2 — Commodity worker single-transaction is conn-drop-fragile [MEDIUM]**

- FR-F2.1: Refactor `services/atlas-data/atlas.com/data/commodity/processor.go:36-46` so the Etc.wz Commodity.img register no longer wraps the entire import in one `database.ExecuteTransaction`. Match the chunking pattern used by other per-img workers (chunk by sub-img or by reasonable row batch; commit each chunk).
- FR-F2.2: After a transient connection drop mid-import, retrying the worker must succeed without re-importing committed rows (or, if a clean re-import is acceptable, the chunked commits must roll back cleanly without orphaning state).
- FR-F2.3: Add a test that simulates a conn drop mid-chunk and asserts the next attempt converges. Acceptable form: a transaction-boundary unit test against the chunking helper; full conn-drop simulation is not required.

**FR-F3 — Scope resolver pins negative results forever [MEDIUM]**

- FR-F3.1: Change `services/atlas-renders/atlas.com/renders/storage/scope.go:18-34` so a "shared" verdict is not cached for the lifetime of the pod. Choose one of: (a) cache positive tenant-hits only; (b) TTL the cache (5–30 min); (c) invalidate on the `DATA_UPDATED` Kafka event. Design.md justifies the chosen approach.
- FR-F3.2: Add a test that probes a tenant before data lands (returns "shared"), simulates the data landing, and asserts the next probe returns the tenant-scoped result without requiring a pod restart.
- FR-F3.3: After the fix, manual `kubectl rollout restart deploy/atlas-renders` is no longer required as part of the bootstrap/restore flow.

**FR-F4 — Close remaining wz.Reader seek/read gaps [LOW–MEDIUM]**

- FR-F4.1: Audit all callers of `*wz.File` Seek+Read paths outside the `parseMu`-guarded `Image.parse()`. Specific entry points: `fetchArchive` in `services/atlas-data/atlas.com/data/data/workers/runtime.go`, `tryParseWithVersion` in `libs/atlas-wz/wz/file.go`, and `extractZmap`'s deeper parser internals.
- FR-F4.2: For each unguarded path, either extend mutex coverage or document why it is safe (e.g., runs only during single-threaded `Open()`).
- FR-F4.3: Annotate the documented-safe paths inline so future archive additions don't reintroduce the footgun.

**FR-F5 — Restore is per-table atomic, not whole-dump atomic [MEDIUM]**

- FR-F5.1: Change `services/atlas-data/atlas.com/data/baseline/restore.go:107,119-126` so a failed restore cannot leave a `tenant_baselines` row pointing at incomplete data. Acceptable options: (a) one outer transaction across all tables; (b) two-phase commit where the `tenant_baselines` marker is the last write after all tables succeed.
- FR-F5.2: Add a test that injects a failure mid-restore and asserts no `tenant_baselines` row exists for that (tenant, region, version) afterward.
- FR-F5.3: Subsequent reads against the failed (tenant, region, version) must behave identically to "never restored" (no half-restored data visible).

**FR-F6 — `Properties()` silently masks parse errors [MEDIUM]**

- FR-F6.1: Change `libs/atlas-wz/wz/image.go:43-71` so callers can distinguish "parsed and empty" from "parse failed". Two acceptable shapes:
  - (a) Add an `Err()` accessor on `*Image` that returns the parse error if any, while keeping `Properties()` returning `[]Property`.
  - (b) Change `Properties()` signature to `([]Property, error)`. This is a larger API break — all callers must be updated.
  Design.md picks one and justifies.
- FR-F6.2: Every existing call site to `Properties()` must propagate or log the error. A `_` discard is unacceptable; an explicit "best-effort, ignored because <reason>" comment is acceptable where applicable.
- FR-F6.3: Add a test fixture where parsing fails and assert the caller observes the failure (no zero-row import).

### 4.2 Operational debt

**FR-F7 — Pin `atlas-renders` in main overlay [HIGH FOR DETERMINISM]**

- FR-F7.1: Add an entry to `deploy/k8s/overlays/main/kustomization.yaml` `images:` list pinning `ghcr.io/chronicle20/atlas-renders/atlas-renders` to the current good SHA (`main-<sha>` form, matching siblings).
- FR-F7.2: Confirm the next renovate run picks up the entry and produces a `bot/main-image-bump-*` PR for it on the next image push.
- FR-F7.3: Update any "service onboarding" docs that list the required overlay-pinning step, if such docs exist.

**FR-F8 — Dedupe `deploy/shared/routes.conf` and k8s `routes.conf.template` [MEDIUM]**

- FR-F8.1: Choose a single source of truth between `deploy/shared/routes.conf` and the embedded ConfigMap in `deploy/k8s/base/atlas-ingress.yaml`'s `routes.conf.template`. Design.md picks one with rationale.
- FR-F8.2: If the source-of-truth is the shared file, generate the k8s ConfigMap from it at deploy time (kustomize `configMapGenerator` or equivalent). If the source-of-truth is the k8s file, remove the duplicate.
- FR-F8.3: Verify both PR-544-fix commits (route additions in `6da0bc363` / `e9244ba14` / `2527c4541`) survive the dedupe — the resulting deployed routes must match the post-fix state.
- FR-F8.4: Update `deploy/shared/test/routes_nginxt.sh` if needed so the test still runs against the live config path (also see FR-F18).

**FR-F9 — Document `RollingUpdate → Recreate` cutover playbook [LOW, ONE-SHOT]**

- FR-F9.1: Document the SSA-orphan-field workaround used on 2026-05-22 (the `kubectl patch --type=json` recipe) in `docs/` (location chosen during design; likely under `docs/deploy/` or alongside the affected overlay).
- FR-F9.2: Note that the issue is resolved on atlas-main and only resurfaces on similar strategy migrations elsewhere.
- FR-F9.3: Optional: add a kustomize patch in the relevant overlay that explicitly nulls `/spec/strategy/rollingUpdate` for first-deploy safety, then remove it.

**FR-F10 — Clean stale `layer-*.png` files [LOW]**

- FR-F10.1: Script a one-time `mc rm --recursive --force` against `atlas-assets/tenants/<id>/regions/<r>/versions/<v>/map/*/layers/` for each (tenant, region, version) in atlas-main and every long-lived PR env.
- FR-F10.2: Confirm post-cleanup MinIO listing shows no `layers/` prefixes remaining under the affected paths.
- FR-F10.3: Document the cleanup runbook so it can be re-run if a new env appears.

**FR-F11 — Triage 359 "no-bounds" maps [INFO]**

- FR-F11.1: Cross-reference the map IDs reported by Map worker (`extractLayoutErrs=359` on PR-544 and atlas-main) against the in-game-accessible map list (sourced from MapleStory game data — see Verification Over Memory rule in CLAUDE.md).
- FR-F11.2: For any map ID that is user-visible, file a follow-up task (do not fix here — fix is data-curation, not code).
- FR-F11.3: For map IDs confirmed unreachable, document them in design.md or a triage notes file so subsequent ingests don't re-prompt the triage.

### 4.3 Code hygiene / lint

**FR-F12 — Comment the `wzinput` PATCH bypass [INFO]**

- FR-F12.1: Add an inline comment in `services/atlas-data/atlas.com/data/wzinput/` PATCH handler explaining why it reads the multipart body manually instead of using `rest.RegisterInputHandler[T]`.

**FR-F13 — Remove dead orphan handler in `data/resource.go` [LOW]**

- FR-F13.1: Locate the orphan handler in `services/atlas-data/atlas.com/data/runtime/rest/resource.go` (grep for un-referenced exported functions, narrow to those that look like REST handlers).
- FR-F13.2: Confirm via grep that no route registration references it.
- FR-F13.3: Delete it (no backwards-compatibility re-exports per the project's anti-pattern guidance).

**FR-F14 — Use `server.MarshalResponse[T]` in `wzinput/status.go` [LOW]**

- FR-F14.1: Replace the manual JSON:API envelope construction with a call to `server.MarshalResponse[T]`.
- FR-F14.2: Confirm response shape matches the pre-change wire format (byte-for-byte where possible).

**FR-F15 — Extract shared helper for `ExtractLayout`/`ExtractLayers` [LOW]**

- FR-F15.1: Factor the ~80% shared body (layer-discovery loop, foothold/portal/NPC extraction, bounds resolution) of `libs/atlas-wz/mapimage/layers.go` `ExtractLayout` and `ExtractLayers` into a private helper.
- FR-F15.2: Confirm both public functions still return identical results to pre-refactor (regression test).

**FR-F16 — Delegate `accessoryPartClassFor` to `libs/atlas-constants` [LOW]**

- FR-F16.1: Refactor `libs/atlas-wz`'s `accessoryPartClassFor` to call the existing classification helper in `libs/atlas-constants/item` (the same fix shape applied in atlas-renders for DOM-21).
- FR-F16.2: Add `libs/atlas-constants` as a `go.mod` dependency of `libs/atlas-wz` if not already present. Confirm both root `Dockerfile` `COPY` lines are present (CLAUDE.md build rule).

### 4.4 Coverage gaps

**FR-F17 — Add concurrent-Properties() regression test [MEDIUM]**

- FR-F17.1: Add a hand-crafted small (~MB) WZ fixture under `libs/atlas-wz/wz/testdata/` that contains multiple Image children with parseable property trees.
- FR-F17.2: Add a test that calls `wz.Open(fixture)`, spawns ≥16 goroutines calling `Properties()` on different `*Image` instances backed by the same `*wz.File`, and asserts no race under `go test -race`.
- FR-F17.3: Test must fail when `parseMu` is removed (negative-control validation during design).

**FR-F18 — Validate k8s-embedded routes config in CI [MEDIUM]**

- FR-F18.1: Extend `deploy/shared/test/routes_nginxt.sh` (or add a sibling script) to also validate the route declarations inside `deploy/k8s/base/atlas-ingress.yaml`'s embedded `routes.conf.template` ConfigMap.
- FR-F18.2: A divergence between the two files (post-FR-F8 dedupe, "divergence" means deviation from the chosen source-of-truth) must fail the script.
- FR-F18.3: If FR-F8 chose to generate the k8s file from the shared file, FR-F18 may collapse into validating the generation step itself.

### 4.5 Deferred carve-outs

**FR-F19 — Add `atlas-renders` to `docker-compose.core.yml` [LOW]**

- FR-F19.1: Add a service block for `atlas-renders` to `deploy/compose/docker-compose.core.yml` mirroring the k8s manifest at `deploy/k8s/base/atlas-renders.yaml` (image, env, dependencies on MinIO).
- FR-F19.2: Verify locally with `docker-compose up atlas-renders` that the service starts and is reachable on its expected port.
- FR-F19.3: Document the local-dev workflow for exercising the render path against compose's MinIO instance.

**FR-F20 — Diagnose-then-fix Henesys portal duplication [UNKNOWN]**

- FR-F20.1: Reproduce the duplication: confirm one or more Henesys maps return a portal list containing duplicate entries. (Reproduction recipe to be filled in during design.)
- FR-F20.2: Document the confirmed root cause in `design.md`. Likely entry points to investigate: `services/atlas-data/atlas.com/data/portal/...` extraction, or `services/atlas-portals/...` if portals are a separate service.
- FR-F20.3: If the fix is small (single-file, no schema change), land it. If the fix is structural (e.g., portal data model needs reshape), document the diagnosis here and spin the fix out as a separate task.
- FR-F20.4: Add a regression test against the diagnosed root cause regardless of where the fix lands.

## 5. API Surface

No new endpoints. No modifications to existing endpoint shapes. The only "API surface" change is the request-handling behavior of `POST /api/data/baseline/publish` going from 500 to 2xx (FR-F1.3) — the JSON:API request body and response envelope shape stay as task-071 designed them.

If FR-F6 selects option (b) (`Properties()` signature change), the change is an internal-library API break, not a transport API break. All callers within the monorepo are updated atomically as part of the task.

## 6. Data Model

No schema changes. No migrations.

The MinIO storage layout stays as task-071 shipped it. FR-F10's cleanup removes objects under existing prefixes but does not change the prefix scheme.

FR-F5 changes transaction boundaries on the `tenant_baselines` table and per-tenant data tables but does not alter the table schemas themselves.

## 7. Service Impact

| Service / Library | Followups affecting it | Change shape |
|---|---|---|
| `services/atlas-data/` | F1, F2, F5, F11, F12, F13, F14 | Handler/processor bug fixes; transaction-boundary refactor; lint cleanups |
| `services/atlas-renders/` | F3, F7 (deploy) | Cache-invalidation fix; deploy pin |
| `services/atlas-portals/` (or atlas-data portals) | F20 | Diagnose-then-fix (location depends on diagnosis) |
| `libs/atlas-wz/` | F4, F6, F15, F16, F17 | Concurrency sweep; API change for `Properties()`; dedupe; classification delegation; regression test fixture |
| `libs/atlas-constants/` | F16 (consumer) | None directly; new consumer added |
| `deploy/k8s/overlays/main/` | F7 | Add image pin |
| `deploy/shared/`, `deploy/k8s/base/atlas-ingress.yaml` | F8 | Source-of-truth dedupe |
| `deploy/shared/test/` | F18 | Test extension |
| `deploy/compose/` | F19 | Add `atlas-renders` service |
| Operational (no code) | F9, F10, F11 (partial) | Runbook docs; one-shot MinIO cleanups |

CLAUDE.md build rule: any `go.mod` touched (potentially `libs/atlas-wz` for F16, `services/atlas-data` for F1/F2/F5, `services/atlas-renders` for F3) requires `docker buildx bake atlas-<svc>` from the worktree root before the branch is claimed done. If F16 adds `libs/atlas-constants` to `libs/atlas-wz/go.mod`, confirm the root `Dockerfile` has both `COPY libs/atlas-constants ...` lines and `go.work` has `./libs/atlas-constants`. If not present, add them (one mod-only `COPY`, one source `COPY`, one `go.work` line — per CLAUDE.md).

## 8. Non-Functional Requirements

**Multi-tenancy.** Every changed handler/processor must continue to use `tenant.MustFromContext(ctx)` for tenant scoping. F1's publish handler, F2's Commodity processor, F3's scope resolver, F5's restore — all are tenant-scoped today and must remain so.

**Observability.** F1's fix must add complete request lifecycle logging (entry, each step, completion or error). F3's fix must log cache invalidation events at info level so operators can correlate with `DATA_UPDATED` Kafka events. F5's fix must log the start and successful finalization of each restore so partial-restore-rolled-back states are visible in Loki.

**Performance.** No NFR regressions vs. task-071's baseline. Specifically: F2's chunked transactions must not slow the Etc.wz import by more than 20% (measured on the same fixture used in task-071's perf check); F5's whole-dump atomic restore must not slow the baseline-restore by more than 10%; F17's regression test must run in under 5 s on CI hardware.

**Security.** F19's compose addition must not expose `atlas-renders` outside compose's internal network (no `ports:` mapping unless explicitly needed).

**Resilience.** F2 and F5 are explicitly resilience fixes (conn-drop tolerance, partial-failure atomicity). Acceptance for both includes a failure-injection test, not just a happy-path test.

## 9. Open Questions

- **OQ-1 (F1 root cause):** Which of the three hypotheses in `followups.md:29-32` is the actual cause? Resolved during design.
- **OQ-2 (F3 strategy):** Of the three acceptable strategies (no-cache-on-negative / TTL / event-invalidation), which is chosen? Resolved during design.
- **OQ-3 (F5 strategy):** Outer-transaction vs. two-phase-finalization for restore atomicity? Resolved during design.
- **OQ-4 (F6 API shape):** `Err()` accessor (smaller break) vs. `Properties() ([]Property, error)` (cleaner long-term)? Resolved during design.
- **OQ-5 (F8 source-of-truth):** Is the shared file or the k8s file canonical? Resolved during design.
- **OQ-6 (F20 service location):** Does portal duplication live in atlas-data extraction or atlas-portals (or somewhere else)? Resolved during diagnosis.
- **OQ-7 (F11 map-id list):** Where does the in-game-accessible map list come from for cross-referencing? Resolved during F11 triage.
- **OQ-8 (F9 doc location):** Where do operational runbooks live in this repo? Resolved during design.

## 10. Acceptance Criteria

### Wave 1 (must land first)

- [ ] **F1:** `POST /api/data/baseline/publish` returns 2xx for the curl recipe in `followups.md:18-27`. Pre-fix-failing test added.
- [ ] **F7:** `deploy/k8s/overlays/main/kustomization.yaml` lists `atlas-renders` under `images:` with a `main-<sha>` tag. Next renovate run produces an image-bump PR for it.
- [ ] **F2:** Commodity worker chunks the Etc.wz import. Conn-drop simulation test passes.
- [ ] **F3:** Scope resolver no longer pins negative results. Probe-before-data-then-probe-after-data test passes.
- [ ] **F5:** Restore is whole-dump atomic. Mid-restore-failure test asserts no `tenant_baselines` row.

### Wave 2 (after Wave 1)

- [ ] **F8:** Single source of truth chosen and implemented for routes.conf. Both PR-544-fix routes preserved.
- [ ] **F4:** All `*wz.File` Seek+Read paths either mutex-guarded or documented-safe inline.
- [ ] **F6:** Parse errors surfaced via chosen API shape. All call sites updated.
- [ ] **F12–F16:** All five lint findings closed. `backend-guidelines-reviewer` run shows none of these specific LINT items.
- [ ] **F17:** WZ fixture committed under `libs/atlas-wz/wz/testdata/`. Concurrent-Properties() race test passes with mutex, fails without.
- [ ] **F18:** Routes test validates the k8s-embedded config too. A simulated divergence fails CI.
- [ ] **F19:** `docker-compose.core.yml` includes `atlas-renders`. `docker-compose up atlas-renders` starts cleanly.
- [ ] **F20:** Henesys portal duplication root cause documented. Fix landed here OR a follow-up task opened. Regression test added.

### Operational one-shots

- [ ] **F9:** Recreate-cutover playbook documented.
- [ ] **F10:** Stale `layer-*.png` files removed from atlas-main and every long-lived PR env. Runbook documented.
- [ ] **F11:** 359 "no-bounds" maps triaged. User-visible map IDs (if any) filed as a separate task; unreachable IDs documented.

### Branch-level acceptance

- [ ] `go test -race ./...` clean in every changed module.
- [ ] `go vet ./...` clean in every changed module.
- [ ] `go build ./...` clean in every changed service.
- [ ] `docker buildx bake atlas-<svc>` clean from the worktree root for every service whose `go.mod` was touched (CLAUDE.md hard requirement).
- [ ] Code review run via `superpowers:requesting-code-review` before opening the PR (CLAUDE.md hard requirement; do not skip).
- [ ] PR description links back to `task-071-gamedata-minio-consolidation/followups.md` and notes which followups (F1–F20) are addressed and which (if any) are explicitly carved out.
