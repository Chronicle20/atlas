# Plan Audit — task-071-gamedata-minio-consolidation

**Plan Path:** `docs/tasks/task-071-gamedata-minio-consolidation/plan.md`
**Audit Date:** 2026-05-20
**Branch:** `task-071-gamedata-minio-consolidation`
**Base Branch:** `main`
**Scope:** Tasks 1-16. Task 17 (compose, pr-bootstrap, cutover smoke, delete dead services) is intentionally deferred and is NOT audited.

## Executive Summary

Tasks 1-7, 9, 11, 14, 16 are fully implemented. Tasks 8, 10, 12, 13, 15 landed as deliberate scope-reduced scaffolding with explicit follow-up markers in source: worker bodies log TODOs (Task 8); COPY in/out are stubbed (Task 10); watchdog sweep and processStatus are no-ops and the Job template falls back to a hardcoded spec (Task 12); atlas-renders character + map handlers return 501 (Task 13); and the ingress regression harness ships only as a `nginx -t` syntax check (Task 15). Build + test verification is fully green across `libs/atlas-wz`, `services/atlas-data`, `services/atlas-renders`, and `services/atlas-ui`; both `docker build` invocations succeed. Without Task 17 the system is not runtime-cutover-ready, but each step that the plan claimed to perform was performed at the documented fidelity.

## Task Completion

| # | Task | Status | Evidence / Notes |
|---|------|--------|------------------|
| 1 | Scaffold `libs/atlas-wz` | DONE | `libs/atlas-wz/{go.mod,README.md,doc.go}`; `go.work:21` adds `./libs/atlas-wz`. Commit `6a8607e41`. |
| 2 | Port wz/, crypto/, canvas/ | DONE | `libs/atlas-wz/{wz,crypto,canvas}` populated; tests pass (commit `489fc50ee`). |
| 3 | Vendor frozen go1.21 PNG encoder | DONE | `libs/atlas-wz/atlas/pngenc/` with byte-identity test passing (commit `b7e00f01f`). |
| 4 | manifest/ + maplayout/ pure-type subpackages | DONE | `libs/atlas-wz/manifest/{types,encode,encode_test}.go`, `libs/atlas-wz/maplayout/{types,encode,encode_test}.go` (commit `3b4d5c10a`). |
| 5 | MaxRects-BSSF atlas packer | DONE | `libs/atlas-wz/atlas/{pack,pack_internal,pack_test}.go` + `README.md`; determinism test passes (commit `676a719a0`). |
| 6 | Port icons + mapimage extractors | DONE | `libs/atlas-wz/icons/extract.go`, `libs/atlas-wz/mapimage/{layers,minimap,zmap,decoder,entries,property,…}.go`; tests pass (commit `c8bbb9f2b`). |
| 7 | atlas-data: MinIO SDK + MODE switch + scratch helper | DONE | `services/atlas-data/atlas.com/data/runtime/{rest,ingest,all}/run.go`; `storage/minio/{client,config,scratch}.go`; Dockerfile four-location pattern for `libs/atlas-wz` verified; `docker build` clean (commit `c61761601`). |
| 8 | atlas-data: rewrite domain workers | STUBBED | Worker contract + registry + fan-out + WZ source helper exist; all 10 worker `Run` bodies log `TODO Task 8: implement <NAME> worker` and return nil. Commit `7d21fc507`. |
| 9 | PATCH/GET /api/data/wz | DONE | `services/atlas-data/atlas.com/data/wzinput/{handler,scope,validate,status,resource,…}.go`; tests pass; mounted via `wzinput.InitResource` in `main.go:139`. Commit `2685ddd55`. |
| 10 | baseline publish/restore + rewriter | STUBBED | Migration + dump shape + rewriter + restoreOneTable + handler scaffolding present; `runCopyOut` and `copyInBinary` return `not yet implemented`. Rewriter has passing round-trip test. Commit `0e57b3e0a`. |
| 11 | DELETE /api/data/tenants/<id> | DONE | `services/atlas-data/atlas.com/data/tenantpurge/{handler,purge}.go` + tests; mounted via `tenantpurge.InitResource` in `main.go:142`. Commit `53026101b`. |
| 12 | MODE=rest Job machinery + watchdog + recovery | STUBBED | `runtime/rest/{jobs,watchdog,recovery,lock,run,resource}.go` exist and compile. `Watchdog.sweep` body is `_ = ctx` with TODO (`watchdog.go:34-38`); `processStatus` returns `{"jobs": []}` (`resource.go:74-83`); `JobCreator.Template` falls back to a minimal hardcoded spec (`jobs.go:25-26`). MODE=ingest entrypoint partial (no DB connect path). Commit `7f4648617`. |
| 13 | atlas-renders service | STUBBED | Service tree, Dockerfile (four-location pattern), `go.mod`, storage layer with LRU + scope resolver, import-lint test pass. Both `character.Handler` (`character/handler.go:19-21`) and `mapr.Handler` (`mapr/handler.go:20-22`) return `501 not yet implemented`. Compositing logic NOT ported. Commit `39e3bc43b` (+ `b064cd5b2` go.work.sum sync). |
| 14 | k8s manifests | DONE | `deploy/k8s/base/{atlas-renders,atlas-minio-init,atlas-data-ingest-job-template}.yaml` exist and are registered in `deploy/k8s/base/kustomization.yaml`. Commit `dfea3f965`. |
| 15 | atlas-ingress routes.conf | PARTIAL | `deploy/shared/routes.conf` rewrite complete (4 new blocks; old `/api/wz` block removed). K8s ingress template mirrored in `deploy/k8s/base/atlas-ingress.yaml` (commit `89bb18ab2`). Test harness ships only as `deploy/shared/test/routes_nginxt.sh` (syntax-only `nginx -t`) per explicit deferral documented in `deploy/shared/test/README.md`. The full upstream-stub harness from plan §15.2-§15.5 is deferred. Commits `b1e187c86` + `89bb18ab2`. |
| 16 | atlas-ui SetupPage rewrite | DONE | `seed.service.ts` augmented (DataStatus adds `baselineRestoredAt`/`baselineSha256`); `baseline.service.ts` + `useBaseline.ts` created; `ScopeToggle.tsx` created; `SetupPage.tsx` wires scope, Restore row (`SetupPage.tsx:217`), Publish CTA; service worker cache renamed `atlas-character-images-v2-task071`; no `Extraction`/`runWzExtraction` references remain. 73 vitest files / 695 tests pass. Commit `c5a7abf91`. |

**Completion Rate:** 11/16 DONE, 4 STUBBED, 1 PARTIAL (Task 15). 100% of plan steps were attempted; STUBBED tasks deliberately left specific sub-steps with explicit follow-up markers in source.

**Skipped without approval:** 0
**Partial implementations:** 5 (Tasks 8, 10, 12, 13, 15)

## Stubs that violate the user's no-TODOs rule

Stubs that must be resolved before claiming feature completeness:

1. **Task 8 worker bodies (10 files)** — each logs `TODO Task 8: implement <NAME> worker (...)` and returns nil:
   - `services/atlas-data/atlas.com/data/data/workers/item.go:19`
   - `services/atlas-data/atlas.com/data/data/workers/mob.go:19`
   - `services/atlas-data/atlas.com/data/data/workers/npc.go:19`
   - `services/atlas-data/atlas.com/data/data/workers/reactor.go:19`
   - `services/atlas-data/atlas.com/data/data/workers/skill.go:19`
   - `services/atlas-data/atlas.com/data/data/workers/quest.go:19`
   - `services/atlas-data/atlas.com/data/data/workers/stringw.go:19`
   - `services/atlas-data/atlas.com/data/data/workers/mapw.go:19`
   - `services/atlas-data/atlas.com/data/data/workers/character.go:19`
   - `services/atlas-data/atlas.com/data/data/workers/ui.go:19`
2. **Task 10 COPY binary I/O**:
   - `services/atlas-data/atlas.com/data/baseline/publish.go:93-95` — `runCopyOut` returns `"runCopyOut: not yet implemented; see task-071 follow-up (requires pgx CopyTo against the gorm postgres connection)"`.
   - `services/atlas-data/atlas.com/data/baseline/restore.go:159-161` — `copyInBinary` returns the analogous pgx CopyFrom stub error.
3. **Task 12 control plane**:
   - `services/atlas-data/atlas.com/data/runtime/rest/watchdog.go:34-38` — `Watchdog.sweep` is a no-op (`_ = ctx`) with TODO comment.
   - `services/atlas-data/atlas.com/data/runtime/rest/resource.go:74-83` — `processStatus` returns `{"jobs": []}` placeholder.
   - `services/atlas-data/atlas.com/data/runtime/rest/jobs.go:25-26, 58` — `JobCreator.Template` falls back to a "minimal hardcoded template if no ConfigMap is wired in"; Task 14's ConfigMap is not actually read.
4. **Task 13 render handlers**:
   - `services/atlas-renders/atlas.com/renders/character/handler.go:19-21` — logs `not yet implemented` warn + returns `http.StatusNotImplemented`.
   - `services/atlas-renders/atlas.com/renders/mapr/handler.go:20-22` — same shape, returns 501.
5. **Task 15 regression harness**:
   - `deploy/shared/test/routes_nginxt.sh` is the only test artifact. The full `upstream-stub.go`, `expectations.txt`, and `routes_test.sh` from the plan are not present; `deploy/shared/test/README.md` documents the deferral explicitly.

## Cross-cutting checks

- **atlas-renders import policy:** `services/atlas-renders/atlas.com/renders/import_lint_test.go` exists (forbidden list at lines 19-25: `wz`, `wz/property`, `crypto`, `canvas`, `atlas`, `atlas/pngenc`); `go test ./...` includes it and passes. The lint exempts `mapimage` and `icons`, which is broader than plan §1.2's import-policy table — when Task 13 map render logic actually lands the lint will need a re-check.
- **Four-location Dockerfile pattern, atlas-data:** verified for `libs/atlas-wz` (Dockerfile contains all four `atlas-wz` references) and `libs/atlas-redis` (Dockerfile contains all four `atlas-redis` references, even though no `atlas-redis` Go import currently exists in atlas-data — added for Task 12's Redis lock helper which currently uses `go-redis/v9` directly, not via the shared lib).
- **Four-location Dockerfile pattern, atlas-renders:** new Dockerfile contains all four locations for `atlas-wz`, `atlas-tenant`, `atlas-rest`, `atlas-tracing`. `docker build` passes.
- **Legacy XML path untouched:** `git diff main..HEAD --stat services/atlas-data/atlas.com/data/data/processor.go` returns empty. Confirms plan §8 requirement.
- **Commit ↔ task mapping:** 22 commits on branch. Each plan task maps cleanly to one or more `feat:` commits (Task 1=`6a8607e41`, 2=`489fc50ee`, 3=`b7e00f01f`, 4=`3b4d5c10a`, 5=`676a719a0`, 6=`c8bbb9f2b`, 7=`c61761601`, 8=`7d21fc507`, 9=`2685ddd55`, 10=`0e57b3e0a`, 11=`53026101b`, 12=`7f4648617`, 13=`39e3bc43b`+`b064cd5b2`, 14=`dfea3f965`, 15=`b1e187c86`+`89bb18ab2`, 16=`c5a7abf91`). One housekeeping commit `223d61d4f` syncs `go.work.sum`.

## Build & Test Results

| Module / Service | Build | Vet | Tests (race) | Notes |
|---|---|---|---|---|
| `libs/atlas-wz` | PASS | PASS | PASS | 10 packages tested, all OK |
| `services/atlas-data` | PASS | PASS | PASS | All packages OK; workers package has tests (`workers_test.go`); 26 packages built |
| `services/atlas-renders` | PASS | PASS | PASS | `import_lint_test.go` runs and passes |
| `services/atlas-ui` (vitest) | n/a | n/a | PASS | 73 test files, 695 tests, 9.31s duration |
| `docker build -f services/atlas-data/Dockerfile .` | PASS | — | — | exit 0 |
| `docker build -f services/atlas-renders/Dockerfile .` | PASS | — | — | exit 0 |

## Overall Assessment

- **Plan Adherence:** MOSTLY_COMPLETE — every plan task delivered the scaffolding the plan called for, but five tasks (8, 10, 12, 13, 15) ship with explicit, documented stubs for the load-bearing sub-steps. The branch does NOT yet implement the full MinIO-fed ingest path, baseline publish/restore COPY transfer, k8s Job watchdog, atlas-renders compositing, or upstream-stub ingress regression harness.
- **Recommendation:** NEEDS_REVIEW — appropriate for a scope-reduced "scaffolding lands first" PR if the follow-ups are tracked. NOT ready to cut over Task 17 (the new ingest path can't actually populate Postgres or MinIO; atlas-renders can't serve renders).

## Action Items

1. Implement the 10 worker bodies in `services/atlas-data/atlas.com/data/data/workers/*.go` per plan §8.3-§8.6.
2. Implement `runCopyOut` / `copyInBinary` using pgx CopyTo/CopyFrom (plan §10.3, §10.6).
3. Wire `Watchdog.sweep` to list k8s jobs by label selector and emit metrics (plan §12.4).
4. Wire `processStatus` to query active jobs and surface their state (plan §12.5).
5. Load `JobCreator.Template` from the `atlas-data-ingest-job-template` ConfigMap (plan §12.3 final paragraph).
6. Port `mapimage/renderer.go`, `blit.go`, `sort.go`, `bounds.go`, `background.go` from the extractor into atlas-renders, then replace both 501 handlers (plan §13.6-§13.7). Re-run `import_lint_test.go` and extend its forbidden list to match plan §1.2 if the composite path turns out to need only `manifest` and `maplayout`.
7. Build out `deploy/shared/test/{upstream-stub.go,expectations.txt,routes_test.sh}` per plan §15.2-§15.4 and wire into CI (plan §15.5).
8. Execute Task 17 (compose, atlas-pr-bootstrap, smoke, delete dead services) once 1-7 are done.

---

## Backend audit

- **Reviewer:** backend-guidelines-reviewer
- **Date:** 2026-05-20
- **Scope:** Go changes only (libs/atlas-wz + atlas-data + atlas-renders) on branch `task-071-gamedata-minio-consolidation`
- **Build (atlas-data, atlas-renders, libs/atlas-wz):** PASS
- **Tests (atlas-data, atlas-renders, libs/atlas-wz):** all green under `go test ./... -count=1`
- **Overall:** **FAIL** — 18 acknowledged stubs landed as functional code; 2 critical SEC failures; multiple DOM/SUB violations; new service missing CI scaffolding.

### Scope summary

No package in the diff has a `model.go`; all are sub-domain or support packages. Audit applies SUB-* + SCAFFOLD-* + SEC-* + the applicable subset of DOM-*.

| Package | Type |
|---|---|
| services/atlas-data/atlas.com/data/wzinput | sub-domain (PATCH/GET /data/wz) |
| services/atlas-data/atlas.com/data/baseline | sub-domain (POST /data/baseline/{publish,restore}) |
| services/atlas-data/atlas.com/data/tenantpurge | sub-domain (DELETE /data/tenants/{id}) |
| services/atlas-data/atlas.com/data/runtime/rest | sub-domain (POST/GET /data/process) + k8s JobCreator |
| services/atlas-data/atlas.com/data/runtime/{ingest,all} | support (entrypoint glue; `all` is dead code) |
| services/atlas-data/atlas.com/data/data/workers | support (10 Worker stubs + registry) |
| services/atlas-data/atlas.com/data/data/{wzsource,runwz}.go | support (WZ fetch + fan-out) |
| services/atlas-data/atlas.com/data/storage/minio | support (MinIO client wrapper) |
| services/atlas-renders/atlas.com/renders/* | new service (handlers are 501 stubs) |

### Critical / blocking findings

#### SEC-01 — `baseline/handler.go:62-90` (restoreInner) is NOT operator-gated

`publishInner` checks `X-Atlas-Operator` at handler.go:37; `restoreInner` does not. Any caller able to reach `POST /api/data/baseline/restore` can DELETE every row for any `tenantId` in the JSON body and re-COPY a canonical dump over it. The PR even has a placeholder test (handler_test.go:29-35) that admits it isn't asserting the operator gate. Recommended fix: add the same 403 short-circuit at the top of `restoreInner`'s returned func; add a positive test that posts without the header and asserts 403.

#### SEC-02 — `baseline/restore.go:38-95` verifies sha256 AFTER mutating the DB

`Restore` streams the tar (restore.go:45-51) into `restoreOneTable` (restore.go:81), which DELETEs the target tenant's rows (restore.go:110) then COPY-FROMs from the unverified stream. Only after the loop does the function compute `actualSum` and compare (restore.go:85-88). On a sha mismatch the target has already been wiped and partially repopulated from untrusted bytes. Recommended fix: download the dump into a scratch file, sha256 the file, verify against the sidecar, THEN open a tar reader on the verified bytes for the destructive transactions. Alternatively buffer in memory (the dumps are small) and verify before any DELETE.

#### Stub leakage (18 entries — user explicitly disallowed)

| File | Line | Symbol | Behavior |
|---|---|---|---|
| services/atlas-data/atlas.com/data/data/workers/item.go | 19-20 | `Item.Run` | `l.Infof("TODO ...")` then `return nil` |
| services/atlas-data/atlas.com/data/data/workers/mob.go | 19-20 | `Mob.Run` | same shape |
| services/atlas-data/atlas.com/data/data/workers/npc.go | 19-20 | `Npc.Run` | same shape |
| services/atlas-data/atlas.com/data/data/workers/reactor.go | 19-20 | `Reactor.Run` | same shape |
| services/atlas-data/atlas.com/data/data/workers/skill.go | 19-20 | `Skill.Run` | same shape |
| services/atlas-data/atlas.com/data/data/workers/quest.go | 19-20 | `Quest.Run` | same shape |
| services/atlas-data/atlas.com/data/data/workers/stringw.go | 19-20 | `String.Run` | same shape |
| services/atlas-data/atlas.com/data/data/workers/mapw.go | 19-20 | `Map.Run` | same shape |
| services/atlas-data/atlas.com/data/data/workers/character.go | 19-20 | `Character.Run` | same shape |
| services/atlas-data/atlas.com/data/data/workers/ui.go | 19-20 | `UI.Run` | same shape |
| services/atlas-data/atlas.com/data/baseline/publish.go | 94-96 | `runCopyOut` | `return fmt.Errorf("not yet implemented")` — every publish 500s |
| services/atlas-data/atlas.com/data/baseline/restore.go | 160-162 | `copyInBinary` | same — every restore 500s (after the broken sha gate) |
| services/atlas-data/atlas.com/data/runtime/rest/watchdog.go | 34-38 | `Watchdog.sweep` | empty body (`_ = ctx`) |
| services/atlas-data/atlas.com/data/runtime/rest/resource.go | 74-83 | `processStatus` | hardcoded `{"jobs": []}` regardless of state |
| services/atlas-data/atlas.com/data/runtime/rest/jobs.go | 59-77 | `defaultTemplate` | hardcoded minimal template; ConfigMap loader promised by plan §12.3 not implemented |
| services/atlas-renders/atlas.com/renders/character/handler.go | 16-22 | `character.Handler` | returns 501 |
| services/atlas-renders/atlas.com/renders/mapr/handler.go | 17-23 | `mapr.Handler` | returns 501 |
| services/atlas-data/atlas.com/data/runtime/all/run.go | 9-13 | `all.Run` | unreferenced dead code; main.go never imports it |

End-to-end the pipeline cannot run: WZ uploads work → publish errors at runCopyOut → restore errors at sha-after-DELETE → worker fan-out logs 10 TODOs and exits success → renders 501. Either implement these or re-scope the PRD/plan as "scaffolding-only" and remove the misleading `workers.Registered` entries and route registrations.

#### SUB-04 / DOM-08 — Manual `json.NewDecoder` in POST handlers

- baseline/handler.go:46 — publish.
- baseline/handler.go:75 — restore.

Both POST routes register via `RegisterHandler` (handler.go:24-25) rather than `RegisterInputHandler[T]`. Per guidelines, POSTs must use the typed input handler so validation + JSON:API decoding go through the shared library. Define a `PublishRequest` / `RestoreRequest` RestModel and switch to `RegisterInputHandler`.

#### SCAFFOLD-01 — atlas-renders not registered in services.json

`grep atlas-renders .github/config/services.json` returns nothing. CI's change-detection reads this file; without an entry the new service never builds in CI.

#### SCAFFOLD-06 — atlas-renders not in docker-compose.core.yml

`grep atlas-renders deploy/compose/docker-compose.core.yml` returns nothing.

#### SCAFFOLD-08 — atlas-renders has no Bruno collection

`services/atlas-renders/.bruno/` does not exist. Required for REST services.

#### atlas-renders main.go:22 has no tenant middleware

`mux.NewRouter()` is used directly with no tenant-header parser, so even once the 501s are replaced, `tenant.MustFromContext(r.Context())` will panic. Stub status hides this today.

### Sub-domain + DOM checklist (applicable items)

| Pkg / ID | Status | Evidence |
|---|---|---|
| wzinput SUB-01 (logic out of handler) | PARTIAL | handler.go:20-83 inlines validation, multipart parsing, MinIO Put. No processor abstraction. |
| wzinput SUB-02 (administrator for writes) | FAIL | `mc.Put` called inside the handler at handler.go:75. |
| wzinput SUB-03 (RegisterInputHandler for PATCH) | FAIL | resource.go:20 uses `RegisterHandler`. Multipart upload mitigates, but request-scope validation is still inline. |
| wzinput SUB-04 (no manual JSON parsing) | PASS | Multipart in / hand-rolled JSON:API out. |
| wzinput zip-slip protection | PASS | validate.go:10-22 rejects `..`, leading `/`, NUL, symlinks (`0o120000`), non-`.wz`. Covered by validate_test.go:28-49. Nested-dir entries (`subdir/Item.wz`) are accepted and propagated to the MinIO key — operational concern, not a vuln. |
| wzinput tenant context | PASS | handler.go:27, status.go:28. |
| wzinput operator gate on `scope=shared` | PASS | scope.go:29 + scope_test.go:42. |
| baseline SUB-03 | FAIL | handler.go:24-25 — both POSTs use `RegisterHandler`. |
| baseline SUB-04 | FAIL | handler.go:46, 75. |
| baseline operator gate on publish | PASS | handler.go:37. |
| baseline operator gate on restore | **FAIL** | See SEC-01. |
| baseline sha verification on restore | **FAIL** | See SEC-02. |
| baseline schema-version refusal | PASS | restore.go:65-67 + handler.go:81 maps to 422. |
| baseline JSON:API response (DOM-18) | FAIL | handler.go:55-57 emits `application/json` with `{"sha256":...}` not a JSON:API doc. |
| tenantpurge SUB-01 | PASS | Logic in `Purge` (purge.go:34); handler is a thin gate. |
| tenantpurge SUB-02 | PASS | DB mutations live in `Purge`. |
| tenantpurge operator gate | PASS | handler.go:35. |
| tenantpurge canonical-UUID refusal | PASS | purge.go:35-37 + purge_test.go:13. |
| tenantpurge best-effort MinIO purge | PASS | purge.go:48-56 — Postgres txn is source of truth; MinIO partial failure only warns. |
| runtime/rest SUB-03 | FAIL | resource.go:23 uses `RegisterHandler`. |
| runtime/rest operator gate on `scope=shared` | PASS | resource.go:42. |
| runtime/rest job template ConfigMap | FAIL (stub) | jobs.go:59-77 hardcodes minimal template. |
| runtime/rest INGEST_IMAGE handling | WARN | jobs.go:71 — empty value silently produces a Pod that ImagePullBackOffs. No fast-fail. |
| runtime/rest processStatus | FAIL (stub) | resource.go:74-83 hardcodes `[]`. |
| runtime/rest Watchdog.sweep | FAIL (stub) | watchdog.go:34-38. |
| runtime/rest restart recovery | PASS | recovery.go + main.go:92-96 + recovery_test.go. |
| workers stubs (×10) | FAIL | See stub table. |
| workers Registry shape | PASS | registry.go:4-15 + workers_test.go:5-29. |
| workers semaphore + errgroup | PASS | runwz.go:23-25. |
| workers `*wz.Image` param mismatch | WARN | runwz.go:42-47 — contract passes `*wz.Image` but WZ root is a directory. Comment acknowledges; real workers must accept `*wz.File` / `*wz.Directory`. |
| Type width drift across service | WARN | workers/worker.go:17-18 declares `uint32`; libs/atlas-tenant/tenant.go:25 returns `uint16`; runtime/rest/resource.go:54-55 casts to `int`. Three widths for one value. |
| renders both handlers 501 | FAIL | character/handler.go:20, mapr/handler.go:21. |
| renders import-lint isolation | PASS | import_lint_test.go:14-37 fails the build if any heavy wz subpackage is pulled in. |
| renders tenant middleware | FAIL | main.go:22-37 uses raw `mux.NewRouter()`; no tenant-header parsing. |
| renders MinIO secret logging | PASS | storage/minio.go:18-27 + storage/config.go:6-23 hold secret in cfg only; never logged. |
| `CanonicalTenantUUID` duplication | WARN | dump.go:13 and tenantpurge/purge.go:17 — two sources of truth for the same magic value. |
| Dead code | WARN | runtime/all/run.go is never imported by main.go. |
| DOM-06 FieldLogger param | PASS | baseline/publish.go:23, baseline/restore.go:33, tenantpurge/purge.go:34 all accept `logrus.FieldLogger`. |
| DOM-07 d.Logger() | PASS | tenantpurge/handler.go:45; baseline/handler.go:50, 79. |
| DOM-12 no os.Getenv in handlers | PASS | jobs.go:45, 71 are inside JobCreator, not handlers. |
| DOM-13 cross-domain orchestration | PASS | Handlers stay in-package. |
| DOM-15 no direct entity creation in handlers | PASS | `db.Exec`/`db.Transaction` confined to `Purge` / `Restorer.Restore`. |
| DOM-20 table-driven tests | PARTIAL | Most tests are one-case-per-Test; validate_test.go is the closest to table-driven (4 funcs). |
| DOM-21 atlas-constants duplication | PASS | New packages don't redeclare world/map/channel/character ids. |
| DOM-22 Dockerfile 4-block pattern (atlas-data) | PASS | All 11 lib direct/indirect references appear 4 times each in services/atlas-data/Dockerfile. |
| DOM-22 Dockerfile 4-block pattern (atlas-renders) | PASS | atlas-wz appears 5 times in services/atlas-renders/Dockerfile. |
| DOM-23 Kafka topic env naming | N/A | No new topics. |

### SEC checklist

| Item | Status | Evidence |
|---|---|---|
| Hardcoded secrets | PASS | All credentials via `os.Getenv("MINIO_ACCESS_KEY")` / `MINIO_SECRET_KEY`; never logged. storage/minio/{client,config}.go and renders/storage/{minio,config}.go. |
| Operator header authenticity | WARN | `X-Atlas-Operator: 1` is a plain header; trust depends on ingress filtering. Neither deploy/k8s/base/atlas-ingress.yaml nor deploy/shared/routes.conf strips or validates it. Document the threat model on each gate. |
| Restore operator gate | **FAIL** | SEC-01. |
| Restore sha verification ordering | **FAIL** | SEC-02. |
| Zip-slip in wzinput | PASS | validate.go:10-22 + tests. |
| Canonical tenant purge refusal | PASS | purge.go:35-37 + purge_test.go:13. |

### SCAFFOLD checklist for atlas-renders (new service)

| ID | Status | Evidence |
|---|---|---|
| SCAFFOLD-01 services.json entry | **FAIL** | absent. |
| SCAFFOLD-02 k8s manifest | PASS | deploy/k8s/base/atlas-renders.yaml. |
| SCAFFOLD-03 Dockerfile | PASS | services/atlas-renders/Dockerfile present, 4-block pattern OK. |
| SCAFFOLD-04 Ingress route | PASS | deploy/shared/routes.conf has 6 `atlas-renders` references. |
| SCAFFOLD-05 Ingress drift-clean | UNVERIFIED | `deploy/scripts/sync-k8s-ingress-routes.sh --check` errors because it points at `deploy/k8s/ingress.yaml` but the file lives at `deploy/k8s/base/atlas-ingress.yaml`. Pre-existing infra problem, surfaced by this task. |
| SCAFFOLD-06 docker-compose | **FAIL** | absent. |
| SCAFFOLD-07 tenant opcode template | N/A | not a packet-handler task. |
| SCAFFOLD-08 Bruno collection | **FAIL** | `services/atlas-renders/.bruno/` does not exist. |

### Backend audit summary

**Blocking (must fix before merge):**

1. SEC-01 — Add `X-Atlas-Operator` gate to baseline restore (handler.go:62).
2. SEC-02 — Verify sha256 BEFORE any DB mutation in baseline restore (restore.go:38-95).
3. SUB-04 / DOM-08 — Switch baseline publish + restore to `RegisterInputHandler[T]` with typed RestModels (handler.go:24-25).
4. Implement the 18 stubs OR re-scope the PRD/plan to "scaffolding only" and remove the misleading registrations.
5. SCAFFOLD-01 — Add atlas-renders entry to `.github/config/services.json`.
6. SCAFFOLD-06 — Add atlas-renders to `deploy/compose/docker-compose.core.yml`.
7. SCAFFOLD-08 — Add `services/atlas-renders/.bruno/` collection.
8. atlas-renders main.go:22 — install tenant-header middleware before the renders are claimed "ready."

**Non-blocking (should fix):**

- `CanonicalTenantUUID` duplicated in baseline/dump.go:13 and tenantpurge/purge.go:17. Consolidate.
- Width drift on MajorVersion/MinorVersion (uint16/uint32/int) — pick one in workers/worker.go:17-18.
- `INGEST_IMAGE` empty fall-through in jobs.go:71 — fast-fail at JobCreator construction.
- Delete or wire `runtime/all/run.go`.
- Move tests to table-driven form per DOM-20.
- Either implement JSON:API conformance on the admin responses or document the carve-out.
- Document the operator-header threat model in package docstrings.
- Fix `deploy/scripts/sync-k8s-ingress-routes.sh` to point at `deploy/k8s/base/atlas-ingress.yaml`.
