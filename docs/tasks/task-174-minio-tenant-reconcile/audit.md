# Plan Audit — task-174-minio-tenant-reconcile

**Plan Path:** docs/tasks/task-174-minio-tenant-reconcile/plan.md
**Audit Date:** 2026-07-17
**Branch:** task-174-minio-tenant-reconcile
**Base Branch:** main (base commit 5c40994e2, HEAD 7c8e51ebf)

## Executive Summary

All 7 plan tasks were faithfully implemented; the branch is functionally complete. `go build`/`go test ./minioreconcile/...` pass, `go vet` is clean, `tools/redis-key-guard.sh`/`tools/goroutine-guard.sh`/`tools/service-registration-guard.sh` all exit 0, all 9 bats tests in the two affected suites pass, and both `kubectl kustomize` renders (main, pr, pr-cleanup) succeed. The implementer went beyond the plan's pseudocode in two good ways: it fixed a partial-report-on-error bug (commit `2cd82ee4b`) and discovered + fixed the real predelete-purge root cause (missing tenant headers → 400, commit `fcbf28a5a`/`7c8e51ebf`) mid-execution, updating plan.md and design.md accordingly. Two non-blocking gaps remain: (1) plan.md's 42 checkboxes were never marked `[x]` despite the work being done, and (2) Task 7 Step 6 ("code review + commit audit.md") had not been executed prior to this audit — this document fulfills it. One Important operational concern was flagged by the implementer and remains open: the `atlas-minio-reconcile` CronJob's `ghcr-pull` imagePullSecret does not currently exist in the `atlas-main` namespace, so the CronJob will `ImagePullBackOff` until a cluster-infra change (outside this repo) replicates it there.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Reconcile core (Store/Request/Report/Reconcile, empty-keep-list refusal, 48h age guard, canonical-sentinel skip, dry-run) + tests | DONE | `services/atlas-data/atlas.com/data/minioreconcile/reconcile.go:1-114`; `reconcile_test.go` (7 tests, all pass). Improved beyond plan: `Reconcile` returns the partial `Report` on a mid-sweep `RemovePrefix` error instead of an empty one (reconcile.go:80,85,97 `return rep, err`), pinned by `TestReconcile_PartialReportOnRemoveError` (reconcile_test.go:132-153, commit `2cd82ee4b`). Age-guard boundary test also pins the `>=` semantics explicitly with an `old(48)` "exactly" case (reconcile_test.go:104-108), a superset of the plan's 47h/49h example. |
| 2 | minio Store adapter (ListTenantPrefixes delimiter listing, PrefixInfo aggregation, parseTenantID) + test | DONE | `storage/minio/client.go:115-130` (`ListTenantPrefixes`, delimiter listing `Recursive:false`); `minioreconcile/store.go:1-70` (`minioStore`, `NewStore`, `parseTenantID`); `store_test.go:1-18` (`TestParseTenantID`, 5 cases, passes). |
| 3 | `POST /api/data/minio/reconcile` endpoint (operator gate 403, empty-list 422, nil-store 503, minAgeHours default 48) wired in main.go | DONE (test-coverage gap noted) | `minioreconcile/handler.go:1-86`, `rest.go:1-40`; wired at `main.go:19,176` (`AddRouteInitializer(minioreconcile.InitResource(mc)(GetServer()))`, immediately after `tenantpurge.InitResource` per plan). 403 and 422 gates are both implemented (handler.go:47-48, 662-666-equiv) and covered by `handler_test.go` (`TestHandler_RequiresOperator`, `TestHandler_EmptyKeepListIs422`, both pass). The nil-store→503 path (`mcStoreOrNil`, handler.go:33-38, 44-46) and the `minAgeHours<=0 → defaultMinAgeHours` path (handler.go:52-55) are implemented correctly but have **no corresponding test** — the plan's own File Structure note ("handler_test.go — httptest for 403/422/nil-mc gates") and this audit's brief both called for a nil-mc test that was not written. Minor test-coverage gap, not a functional gap. |
| 4 | `reconcile-minio.sh` orchestrator (cross-ns union, fail-closed, empty-union refusal, synthetic tenant headers on POST) + bats | DONE | `services/atlas-pr-bootstrap/scripts/reconcile-minio.sh:1-95`; namespace enumeration + fail-closed abort (lines 24-30, 40-49 `record_error ... return 1` on any per-namespace enumeration failure); empty-union refusal (lines 55-58); synthetic tenant headers on the POST (lines 71-76, 80-83, matches the plan's Global Constraints addendum from `fcbf28a5a`). `test/reconcile_minio_test.bats` (3 tests, all pass) verifies union-and-post, fail-closed-on-unreachable-namespace, and refuse-empty-union, including asserting `TENANT_ID`/`MAJOR_VERSION` appear in the posted args. Implementation deviates cosmetically from the plan's pseudocode (uses `curl -sf` exit-code branching instead of `-w '%{http_code}'` status parsing) — functionally equivalent, correctly commented as a deliberate choice (script lines 78-83). |
| 5 | `atlas-minio-reconcile` CronJob + RBAC (list namespaces only) + dry-run ConfigMap in base; excluded from pr/pr-cleanup overlays | DONE, with one flagged operational concern | `deploy/k8s/base/atlas-minio-reconcile.yaml` (ServiceAccount, ClusterRole `namespaces:list` only, ClusterRoleBinding, ConfigMap `RECONCILE_DRY_RUN:"true"`/`RECONCILE_MIN_AGE_HOURS:"48"`, CronJob `schedule:"0 3 * * *"`); registered in `deploy/k8s/base/kustomization.yaml:37`. `deploy/k8s/overlays/pr/kustomization.yaml:116-152` deletes all 5 resources via `$patch: delete` (broader than the plan's "delete or suspend the CronJob" suggestion — correctly extended to the cluster-scoped RBAC objects too, per `.superpowers/sdd/task-5-report.md` reasoning about cross-PR name collisions). `pr-cleanup` overlay needs no exclusion (does not include base). Verified live via `kubectl kustomize deploy/k8s/overlays/{main,pr,pr-cleanup}` — all three render cleanly; `main` render confirmed to contain all 5 resources correctly wired (namespace `atlas-main`, image tag resolved via the overlay's `images:` override, script path `/atlas/reconcile-minio.sh` matching `Dockerfile:64,67`); `pr`/`pr-cleanup` renders contain zero `atlas-minio-reconcile` occurrences. **Flagged concern (unresolved, correctly not silently shipped):** the CronJob's pod spec carries `imagePullSecrets: [ghcr-pull]` (added beyond the plan's literal template, matching sibling `atlas-pr-bootstrap` manifests), but `ghcr-pull` does not currently exist in `atlas-main` — it's Reflector-replicated only into `atlas-pr-.*` namespaces by a separate cluster-infra repo. As shipped, the CronJob will `ImagePullBackOff` in `atlas-main` until that external secret-replication scope is extended. This is a genuine external/cluster-infra blocker outside this branch's scope, and was explicitly flagged in `task-5-report.md` rather than silently shipped — appropriate per CLAUDE.md's "no silent deferral" rule, but it means the feature is not yet operative post-merge without that follow-up. |
| 6 | predelete-purge.sh: tenant headers on the DELETE (bugfix) + bounded retry + bats | DONE | `services/atlas-pr-bootstrap/scripts/predelete-purge.sh`: synthetic tenant headers added to `delete_tenant_once`'s curl call (lines 34-40, `PURGE_TENANT_ID`/`REGION`/`MAJOR_VERSION`/`MINOR_VERSION`, defaults matching the live-verified synthetic tenant); bounded retry via `lib.sh`'s `retry` helper wrapping `delete_tenant_once` (line 89, `retry "$PURGE_DELETE_RETRIES" "$PURGE_DELETE_RETRY_SLEEP" delete_tenant_once "$id"`), `PURGE_DELETE_RETRIES=3`/`PURGE_DELETE_RETRY_SLEEP=2` defaults (lines 28-29), final-failure `log error` (line 90-91), non-zero exit preserved (`rc=1`, line 92). `test/predelete_test.bats` adds both required cases: `"predelete: DELETE sends synthetic tenant headers"` (asserts all 4 headers present in captured curl args) and `"predelete: DELETE retries a transient failure then succeeds"` (503-then-202 via a counter-file shim) — both pass, along with the 4 pre-existing cases (9/9 total). |
| 7 | Full verification (go test -race/vet/build, guards, bats 66/66, kustomize renders) + code review | PARTIAL | Go-level gates independently re-verified in this audit: `go build ./...` clean, `go test ./minioreconcile/...` 9/9 pass (including via `-race` implicitly exercised — not explicitly re-run with `-race` flag in this audit pass, but no goroutines are spawned in the new package per the goroutine-guard result, so race exposure is minimal), `go vet ./...` clean. `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, `tools/service-registration-guard.sh` all exit 0 (re-run in this audit). `bats test/reconcile_minio_test.bats test/predelete_test.bats` → 9/9 pass (re-run in this audit; the plan's claimed "66/66" figure refers to the full atlas-pr-bootstrap bats suite, not independently re-run here — the task brief states this gate was already run by the controller). `kubectl kustomize` renders for main/pr/pr-cleanup all succeed (re-run in this audit). **Gap:** Step 6 ("Invoke superpowers:requesting-code-review... commit the audit doc") was never executed before this audit — no `docs/tasks/task-174-minio-tenant-reconcile/audit.md` existed on the branch, and no `docs(task-174): code-review audit` commit exists in the log. This audit document is being produced now to close that gap. |

**Completion Rate:** 7/7 tasks functionally implemented (100%)
**Skipped without approval:** 0
**Partial implementations:** 1 (Task 7 — code-review/audit-commit step not executed prior to this audit; now being remedied)

## Skipped / Deferred Tasks

None of Tasks 1–6 were skipped or deferred — all have direct file:line evidence of implementation and passing tests. Task 7 is marked PARTIAL only because its final sub-step (dispatch code review, commit `audit.md`) had not run yet; the code itself and all automated gates it was meant to precede are complete and passing. Two smaller items are worth tracking even though they do not block merge:

1. **Plan checkboxes never marked.** All 42 `- [ ]` checkboxes in `plan.md` remain unchecked despite every corresponding step's deliverable being present and tested in the git history. Purely a tracking/documentation nicety — the audit's own evidence trail (commit-by-commit diff review, live test runs) confirms the work was done regardless of checkbox state.
2. **Nil-store 503 / minAgeHours-default test coverage.** `handler.go`'s `store == nil → 503` branch and the `minAgeHours <= 0 → 48` default are both implemented correctly (confirmed by code inspection) but have no dedicated test in `handler_test.go`, unlike the 403/422 branches. Low risk (trivial logic, exercised implicitly by every other passing test using a non-nil store and an explicit `MinAgeHours`), but a gap versus the plan's own file-structure note calling out "nil-mc gates" as an expected test target.

## Build & Test Results

| Service | Build | Tests | Notes |
|---------|-------|-------|-------|
| atlas-data (`services/atlas-data/atlas.com/data`) | PASS | PASS | `go build ./...` clean; `go test ./minioreconcile/... -v` 9/9 pass; `go vet ./...` clean. |
| atlas-pr-bootstrap (bats) | N/A (shell) | PASS | `bats test/reconcile_minio_test.bats test/predelete_test.bats` → 9/9 pass (`ok 1`–`ok 9`). |
| Repo-root guards | N/A | PASS | `tools/redis-key-guard.sh` exit 0; `tools/goroutine-guard.sh` exit 0; `tools/service-registration-guard.sh` → `service-registration-guard: clean`. |
| deploy/k8s kustomize | N/A | PASS | `kubectl kustomize deploy/k8s/overlays/{main,pr,pr-cleanup}` all render without error. |

`docker buildx bake atlas-data` was not re-run in this audit (not required to re-verify per the task brief, which states T7's gates were already run by the controller); the plan's own commit history and the successful `go build`/`go vet` give high confidence the image build would succeed, but this is unverified in this session.

## Overall Assessment

- **Plan Adherence:** FULL (all 7 tasks' engineering deliverables are present, correct, and tested; only the process-level "commit the audit doc" sub-step of Task 7 was outstanding, and is resolved by this document)
- **Recommendation:** NEEDS_FIXES (not for code correctness — for the flagged `ghcr-pull` secret gap in Task 5, which will cause the shipped CronJob to `ImagePullBackOff` in `atlas-main` until a cluster-infra-side secret-replication change lands; and to run the `superpowers:requesting-code-review` dispatch this audit substitutes for)

## Action Items

1. **Before/at merge:** File or coordinate the cluster-infra follow-up to make `ghcr-pull` (or an equivalent dockerconfigjson secret) available in the `atlas-main` namespace — either by extending the Reflector `reflection-auto-namespaces` regex to include `atlas-main`, or by making the `atlas-pr-bootstrap` GHCR package public like the other `atlas-*` packages. Until this lands, `atlas-minio-reconcile`'s CronJob pods will `ImagePullBackOff` and the reconciliation backstop will not actually run in `atlas-main`.
2. **Optional, low priority:** Add a `TestHandler_NilStoreIs503` (and optionally a `TestHandler_MinAgeHoursDefault`) case to `minioreconcile/handler_test.go` to close the test-coverage gap noted in Task 3/7.
3. **Optional, cosmetic:** Mark the 42 plan.md checkboxes `[x]` to reflect actual completion state, or note in the plan why they were left unchecked.
4. **Process:** Run `superpowers:requesting-code-review` (plan-adherence-reviewer + backend-guidelines-reviewer, since only Go + shell + k8s changed, no TS) if a second independent review pass is desired beyond this audit before opening the PR.

---

# Backend Guidelines Review — task-174-minio-tenant-reconcile

- **Scope:** `services/atlas-data/atlas.com/data/minioreconcile/` (reconcile.go, store.go, rest.go, handler.go + tests), `storage/minio/client.go` (`ListTenantPrefixes`), `main.go` wiring.
- **Package classification (Phase 2):** Support package — no `model.go`, no `resource.go` (uses `handler.go` + free-function business logic, matching the sibling `tenantpurge`/`baseline` packages in this same service). Full File Responsibilities Checklist applies regardless.
- **Build:** PASS — `go build ./...` clean in `services/atlas-data/atlas.com/data`.
- **Tests:** PASS — `go test ./minioreconcile/... -count=1 -race` → `ok atlas-data/minioreconcile 1.023s`.
- **go vet:** clean on `./minioreconcile/...` and `./storage/minio/...`.
- **Goroutine guard:** no bare `go` statements in the diff (`grep -rnE '^\s*go (func|[A-Za-z_])'` — zero hits).
- **os.Getenv in handler:** zero matches in `minioreconcile/`.

## Findings

### Important — FILE-02: Model→RestModel transform lives in `handler.go`, not `rest.go`

`services/atlas-data/atlas.com/data/minioreconcile/handler.go:77-86` defines `toOutput(rep Report) ReconcileOutputModel`, which performs exactly the domain-Model→RestModel serialization responsibility that `file-responsibilities.md` assigns to `rest.go` ("Implement `Transform(Model) (RestModel, error)` to convert domain models to REST representations"). `rest.go` (`services/atlas-data/atlas.com/data/minioreconcile/rest.go:1-41`) contains only the `RestModel` structs and JSON:API interface methods — no `Transform`/`Extract` function at all. The reverse direction has the same issue: the REST-input→domain-`Request` mapping is inlined at `handler.go:54-58` (`Request{KeepTenantIDs: input.KeepTenantIDs, MinAgeHours: minAge, DryRun: input.DryRun}`) rather than an `Extract`-style function in `rest.go`.

`file-responsibilities.md`'s `resource.go` entry is explicit that the handler file's job is route registration and thin dispatch — "Delegate ALL business logic to processors" — and lists `rest.go` as the sole owner of Model↔RestModel conversion. Here the conversion logic (a struct-literal construction on the way in, a loop building `[]OutputRow` with per-row timestamp formatting on the way out) is business/serialization logic embedded in the handler file. This is a smaller instance of the same class of violation the checklist is designed to catch (task-102 `wallet.go`): serialization logic that belongs in `rest.go` living in the wrong file. Graded against the table, not against the sibling `tenantpurge`/`baseline` packages (whose handlers do no non-trivial RestModel translation, so that comparison doesn't excuse this file).

**Fix:** move `toOutput` (and ideally the `Request{...}` construction) into `rest.go`, named `Transform`/`Extract` per convention, called from the handler.

### Minor — untested error paths in `handler.go`

- `handler.go:42-45` (`store == nil` → 503 "minio unavailable") has no test. `handler_test.go`'s two tests (`TestHandler_RequiresOperator`, `TestHandler_EmptyKeepListIs422`) both construct the router with a non-nil `*fakeStore`; nothing exercises the nil-store branch or `mcStoreOrNil` (`handler.go:32-37`) directly.
- `handler.go:59-66`'s generic `server.WriteErrorResponse` passthrough (a `Store` method returning a non-`ErrEmptyKeepList` error) is unit-tested at the `Reconcile` level (`reconcile_test.go:129-151`, `TestReconcile_PartialReportOnRemoveError`) but not through the HTTP handler, so the handler's error-to-status mapping for that branch is unverified end-to-end.

Per `testing-guide.md` "Common Testing Pitfalls #5 — Not Testing Error Paths": write table-driven tests with both success and failure cases. Non-blocking since the core algorithm's error path is covered at the `Reconcile` level, but the handler-layer wiring for a nil client and a generic store error is not. (This overlaps with the plan-adherence section above, which flags the same nil-store gap from a plan-completeness angle; recorded here as the guidelines-side citation.)

### Minor — gofmt violation

`minioreconcile/store_test.go:6-11` is not `gofmt`-clean — the map-literal alignment is off (`gofmt -l minioreconcile/` flags the file; `gofmt -d` shows a whitespace-only diff realigning the `"tenants/abc/"`/`"tenants/"`/`"shared/x/"`/`"tenants/a/b/"` value column). Cosmetic, but `gofmt -l` failing means this file would fail a `gofmt` CI gate if one exists.

## Checklist Results

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor in processor.go | N/A | No `Processor` interface/impl in this package; business logic is a free `Reconcile` function (`reconcile.go:64`), matching the sibling `tenantpurge.Purge`/`baseline` free-function shape used throughout this service for non-entity operator actions. Nothing to misplace. |
| FILE-02 | RestModel + Transform/Extract in rest.go | **FAIL (Important)** | See finding above — `handler.go:77-86` (`toOutput`) and `handler.go:54-58`. |
| FILE-03 | Cross-service requests.go | N/A | Package makes no calls to other atlas services (`requests.RootUrl`/`GetRequest`/`PostRequest` — zero matches). |
| FILE-04 | entity.go / Migration / TableName | N/A | No GORM entity — package talks to MinIO object storage, not Postgres. |
| FILE-05 | Builder/Model/administrator/provider/state placement | N/A | No domain `Model` requiring a `Builder`; `store.go` (`services/atlas-data/atlas.com/data/minioreconcile/store.go:1-70`) is the single adapter over `*minio.Client` covering both reads (`ListTenantIDs`, `PrefixInfo`) and the one write (`RemovePrefix`) — same shape as this service's other MinIO-adapter packages (`baseline`, `tenantpurge`), which do not split provider.go/administrator.go for non-entity object-storage access. |
| FILE-06 | No package-named catch-all file | PASS | No `minioreconcile.go`/`<pkg>.go` file exists; responsibilities are split across `reconcile.go` (algorithm + `Store` port), `store.go` (MinIO adapter), `rest.go` (RestModels), `handler.go` (route + dispatch) — though see FILE-02 for the transform-placement issue within `handler.go`. |
| DOM-06 | Processor accepts FieldLogger | N/A | No `Processor`; `Reconcile` takes `l logrus.FieldLogger` directly (`reconcile.go:64`) — correct type either way. |
| DOM-07 | Handlers pass `d.Logger()` | PASS | `handler.go:54` passes `d.Logger()` into `Reconcile`. |
| DOM-08 | POST uses RegisterInputHandler | PASS | `handler.go:25` — `rest.RegisterInputHandler[ReconcileInputModel](l)(si)("minio_reconcile", ...)` for the `POST /reconcile` route (`handler.go:26`). |
| DOM-09 | Transform errors handled | N/A | No `Transform(...)` call site exists in this package (see FILE-02). |
| DOM-12 | No os.Getenv in handlers | PASS | Zero matches in `minioreconcile/*.go`. |
| DOM-13/14/15 | No cross-domain logic / no direct provider calls / no direct DB writes in handler | PASS | `handler.go` only calls `Reconcile(...)` (`handler.go:54`) and the injected `Store` interface; no `db.Create/Save/Delete`, no provider calls. |
| DOM-17 | Error → HTTP status mapping | PASS | `ErrEmptyKeepList` → 422 (`handler.go:60-63`); store-nil precondition → 503 (`handler.go:42-45`); no-operator-header → 403 (`handler.go:46-49`); other errors → `server.WriteErrorResponse` (`handler.go:64`), which composes the transient-error classifier registered service-wide in `main.go:135-136`. |
| DOM-18 | JSON:API interface on RestModel | PASS | `rest.go:11-15` (`ReconcileInputModel`) and `rest.go:36-40` (`ReconcileOutputModel`) both implement `GetName()`/`GetID()`/`SetID()`/`SetToOneReferenceID()`/`SetToManyReferenceIDs()`. |
| DOM-19 | Flat request model | PASS | `ReconcileInputModel` (`rest.go:4-9`) is flat — no nested Data/Type/Attributes. |
| DOM-20 | Table-driven tests | PASS | `reconcile_test.go` uses case-driven `fakeStore` fixtures per scenario; `store_test.go:6-12` uses a literal `cases := map[string]string{...}` table. |
| DOM-21 | No atlas-constants duplication | PASS (N/A) | New types (`PrefixInfo`, `Store`, `Request`, `Report`, `ReportRow`, `OutputRow`) are reconcile-specific, not item/inventory/world/job/skill/monster id types covered by `libs/atlas-constants`. |
| DOM-22 | Dockerfile lib-mention count | N/A | `go.mod`/`go.sum` unchanged in this diff — no new `Chronicle20/atlas/libs/*` direct require added. |
| DOM-24 | Kafka producer stubbed in tests | N/A | Package emits no Kafka messages (`AndEmit`/`message.Emit`/`producer.Produce` — zero matches). |
| DOM-25 | Client wire-value config resolution | N/A | Not channel/packet code. |
| DOM-26 | Goroutines via routine.Go | PASS | Zero bare `go` statements in the diff. |
| DOM-27 | Transient DB errors → 503 | N/A | Package is not DB-backed (MinIO only); the service-wide classifier is registered once in `main.go:135-136` and this handler already routes non-empty-keep-list errors through `server.WriteErrorResponse` (`handler.go:64`) rather than a bare `w.WriteHeader(http.StatusInternalServerError)`. |
| SUB-04 | No manual JSON parsing | PASS | No `json.NewDecoder`/`json.Unmarshal`/`io.ReadAll` in `handler.go` — decoding is delegated to `rest.RegisterInputHandler[ReconcileInputModel]`. |
| EXT-01..04 | External HTTP client checklist | N/A | Package makes no cross-service REST calls (MinIO SDK only, not `requests.GetRequest`/`PostRequest`). |
| SEC-01..04 | Security review | N/A (not an auth service) | Informational: the operator gate (`r.Header.Get("X-Atlas-Operator") != "1"` at `handler.go:46`) is a header-equality check with no cryptographic verification — but this is byte-for-byte identical to the existing, already-shipped `tenantpurge.purgeInner` gate (`tenantpurge/handler.go:37-40`) and the task context explicitly scopes this as an accepted, by-design pattern shared with `tenantpurge`. Not a new defect introduced by this diff. |

## Summary

### Blocking (must fix)
- FILE-02: move `toOutput` (`handler.go:77-86`) and the `Request{...}` extraction (`handler.go:54-58`) into `rest.go` as `Transform`/`Extract` functions per `file-responsibilities.md`.

### Non-Blocking (should fix)
- Add handler-level test coverage for the nil-`Store`/503 branch (`handler.go:42-45`, `mcStoreOrNil` at `handler.go:32-37`) and for a generic `Store`-error → `WriteErrorResponse` passthrough (`handler.go:59-66`).
- Run `gofmt -w` on `minioreconcile/store_test.go` (currently fails `gofmt -l`).

**Overall: NEEDS-WORK** (build and tests pass; one Important structural finding, two Minor findings).

---

## Resolution (final-review fixes applied)

Commit `5e7586644` addressed the review findings:

- **Important (FILE-02):** `ToRequest()` (input→domain Extract, incl. the
  `minAgeHours <= 0 → 48` default) and `toOutput()` (domain→RestModel Transform),
  plus the `defaultMinAgeHours` const, moved from `handler.go` into `rest.go`.
  `handler.go` now delegates via `input.ToRequest()` / `toOutput(rep)`. Pure move,
  behavior identical.
- **Minor:** added `TestHandler_NilStoreIs503` (handler_test.go) and
  `TestToRequest_DefaultsMinAgeHours` (rest_test.go) covering the previously
  untested 503 and default-minAgeHours paths.
- **Minor:** `gofmt -w` applied; `gofmt -l ./minioreconcile/` now clean.

Post-fix gates: `go test -race ./minioreconcile/` 11/11 PASS, `go vet`/`go build`
clean, `gofmt -l` empty. Whole-branch gates (controller-run): go test -race /
vet / build clean in atlas-data; redis/goroutine/service-registration guards
clean; atlas-pr-bootstrap bats 66/66; kustomize renders clean for main/pr/
pr-cleanup with the CronJob present only in main.

### Remaining external prerequisite (NOT a code defect)

The `atlas-minio-reconcile` CronJob references `imagePullSecrets: [ghcr-pull]`,
which currently exists only in `atlas-pr-*` namespaces (Reflector-scoped), not in
`atlas-main`. Before/soon after merge, cluster-infra must replicate `ghcr-pull`
into `atlas-main` (or make the `atlas-pr-bootstrap` package public), else the
CronJob `ImagePullBackOff`s. Deletion stays disabled (`RECONCILE_DRY_RUN=true`)
until an operator flips the ConfigMap flag after reviewing a dry-run report.
