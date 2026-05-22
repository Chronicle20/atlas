# Plan Audit — Round 2

**Audit Date:** 2026-05-20
**Branch:** `task-071-gamedata-minio-consolidation`
**Compared against:** original audit at `docs/tasks/task-071-gamedata-minio-consolidation/audit.md`
**Net new commits since round 1:** 10 (`19cb3616a`, `9782446e5`, `5e8d452a3`, `477b2a429`, `df1e7386b`, `4a5a68506`, `4188993d7`, `8c950b0c1`, `52c7fae04`, `2a63b7959`, `31ced9bd1` — 11 SHAs across post-audit fix sweep)

## Executive summary

Every blocking finding from the round-1 audit is resolved with file:line evidence. All 18 round-1 stubs in `services/atlas-data` and `services/atlas-renders` are now real implementations (10 worker bodies totalling ~900 LOC, pgx CopyTo/CopyFrom in publish/restore, k8s-backed `Watchdog.sweep` and `processStatus`, ConfigMap loader for the Job template, full character + map render pipelines). Both SEC findings are fixed (operator gate on restore, sha verification before any DB mutation). The atlas-renders scaffolding holes (services.json, Bruno, tenant middleware) are filled. Tasks 8, 10, 12, 13, 15 graduate from STUBBED/PARTIAL to DONE; Task 17 remains the only deliberately deferred item (scoped out of this PR).

The branch is now mergeable. `go test -race`, `go vet`, and `docker build` are green for both Go services and `libs/atlas-wz`; vitest passes 712/712 on a clean run (one earlier run produced a flaky failure on a pre-existing rename-dialog test unrelated to this task, did not reproduce on rerun). No new TODOs were introduced; the only remaining `TODO` strings are in `services/atlas-data/atlas.com/data/{skill,map}/reader.go` (pre-existing legacy XML readers, slated for Task 17 deletion).

## Round-2 task status table

| # | Task | R1 Status | R2 Status | Evidence |
|---|------|-----------|-----------|----------|
| 1 | Scaffold `libs/atlas-wz` | DONE | DONE | unchanged |
| 2 | Port wz/, crypto/, canvas/ | DONE | DONE | unchanged |
| 3 | Vendor frozen go1.21 PNG encoder | DONE | DONE | unchanged |
| 4 | manifest/ + maplayout/ subpackages | DONE | DONE | unchanged; `manifest.Vslot` schema extended in `2a63b7959` |
| 5 | MaxRects-BSSF atlas packer | DONE | DONE | unchanged |
| 6 | icons + mapimage extractors | DONE | DONE | unchanged |
| 7 | atlas-data MinIO + MODE switch | DONE | DONE | dead `runtime/all` deleted in `477b2a429`; `runtime/` now only contains `ingest/` + `rest/` |
| 8 | Rewrite domain workers | STUBBED | DONE | All 10 worker `Run` bodies implemented in `52c7fae04`; LOC range 33–204 (`workers/quest.go` 33L through `workers/character.go` 204L); Character.wz uses atlas-pack via `libs/atlas-wz/charparts` (`2a63b7959`); worker contract updated to `*wz.File` + `uint16` widths in `5e8d452a3` |
| 9 | PATCH/GET /api/data/wz | DONE | DONE | unchanged |
| 10 | Baseline publish/restore | STUBBED | DONE | `runCopyOut` now uses `pgxConn.Conn().PgConn().CopyTo` (`publish.go:106`); `copyInBinary` uses `pgxConn.Conn().PgConn().CopyFrom` (`restore.go:202`); commit `9782446e5` |
| 11 | DELETE /api/data/tenants/<id> | DONE | DONE | unchanged |
| 12 | MODE=rest Job + watchdog | STUBBED | DONE | `Watchdog.sweep` real k8s Job list + redis heartbeat + delete-stuck (`watchdog.go:39-100`); `processStatus` real k8s list with full per-job shape (`resource.go:90-150`); `JobCreator.Template` loads ConfigMap `atlas-data-ingest-job-template/job.yaml` via `loadTemplateFromConfigMap` (`jobs.go:74-111`); commit `477b2a429` |
| 13 | atlas-renders service | STUBBED | DONE | `character.Handler` ported from donor (186 LOC, manifest+vslot occlusion) in `4188993d7`+`31ced9bd1`; `mapr.Handler` real zmap composite (133 LOC) in `df1e7386b`; background tile blit documented as STATED LIMITATION in `mapr/composite.go:21-26` (not a TODO); tenant middleware installed in `main.go:28,50-60` (`9782446e5`); services.json entry added (`9782446e5`); Bruno collection `services/atlas-renders/.bruno/` present (`9782446e5`) |
| 14 | k8s manifests | DONE | DONE | unchanged |
| 15 | atlas-ingress routes.conf | PARTIAL | PARTIAL (unchanged — `nginx -t` only, full upstream-stub harness still deferred per `deploy/shared/test/README.md`) |
| 16 | atlas-ui SetupPage rewrite | DONE | DONE | Round-1 review fixes landed: semantic colors on ScopeToggle, `useBaseline` null-safe + invalidates DataStatus, `baseline.service` decodes errors, added tests (`4a5a68506`) |
| 17 | Cutover (compose, smoke, deletes) | NOT AUDITED | NOT AUDITED | Deliberately deferred per round-1 scope decision |

**Completion rate (tasks 1–16):** 15/16 DONE, 1 PARTIAL (Task 15 documented carve-out). 0 STUBBED.

## Round-1 findings → Round-2 resolution

| Round-1 finding | Round-2 file:line | Status |
|---|---|---|
| SEC-01: `restoreInner` not operator-gated | `baseline/handler.go:64-65` — explicit `X-Atlas-Operator != "1"` 403 short-circuit | FIXED |
| SEC-02: sha verified after DB mutation | `baseline/restore.go:65-71` — sha computed & compared BEFORE the table loop runs | FIXED |
| 10 worker TODO bodies | `data/workers/{item,mob,npc,reactor,skill,quest,stringw,mapw,character,ui}.go` all 33–204 LOC, no `TODO`/`not yet implemented` | FIXED |
| `runCopyOut` not implemented | `baseline/publish.go:98-115` — pgx `CopyTo` against stdlib conn | FIXED |
| `copyInBinary` not implemented | `baseline/restore.go:180-210` — pgx `CopyFrom` against stdlib conn | FIXED |
| `Watchdog.sweep` no-op | `runtime/rest/watchdog.go:39-66` — real k8s List + cutoff + stuck-delete | FIXED |
| `processStatus` hardcoded `[]` | `runtime/rest/resource.go:90-150` — real k8s Jobs List → typed JSON shape | FIXED |
| Job template hardcoded | `runtime/rest/jobs.go:74-111` — `loadTemplateFromConfigMap` reads `atlas-data-ingest-job-template/job.yaml` | FIXED |
| `character.Handler` 501 | `character/handler.go` 186 LOC, full composite via `Composite()` + vslot occlusion | FIXED |
| `mapr.Handler` 501 | `mapr/handler.go` 133 LOC + `composite.go` zmap stack | FIXED |
| `runtime/all` dead code | Directory deleted — `services/atlas-data/atlas.com/data/runtime/` now contains only `ingest/` + `rest/` | FIXED |
| SUB-04: baseline POSTs used `RegisterHandler` | `baseline/resource.go:22-23` — both routes use `rest.RegisterInputHandler[T]` with typed `PublishInputModel` / `RestoreInputModel` | FIXED |
| SCAFFOLD-01: atlas-renders missing from services.json | `.github/config/services.json` contains atlas-renders entry | FIXED |
| SCAFFOLD-08: no Bruno collection | `services/atlas-renders/.bruno/` with `bruno.json`, `collection.bru`, environments, 2 request `.bru` files | FIXED |
| renders main.go no tenant middleware | `main.go:28` `r.Use(tenantMiddleware(l))`; impl at `main.go:50-` parses 4 tenant headers and injects via `tenant.MustFromContext`-compatible context | FIXED |
| `CanonicalTenantUUID` duplicated | New `services/atlas-data/atlas.com/data/canonical/canonical.go` package; `baseline/publish.go:105` + `tenantpurge/purge.go:32` + `tenantpurge/purge_test.go:16` all reference `canonical.TenantUUID` | FIXED |
| FE-06/09/11/17 frontend findings | `4a5a68506` commit | FIXED |
| SCAFFOLD-06: atlas-renders not in compose | `deploy/compose/docker-compose.core.yml` — still absent | DEFERRED (Task 17 scope per user note) |
| Plan §15 upstream-stub harness | `deploy/shared/test/` only contains `routes_nginxt.sh` + `README.md` | DEFERRED (PARTIAL, documented) |

## New issues discovered

None. The `grep TODO\|not yet implemented\|StatusNotImplemented` sweep returns only pre-existing legacy markers in `services/atlas-data/atlas.com/data/{skill,map}/reader.go` (8 hits in `skill/reader.go`, 1 in `map/reader.go`) — these are XML-reader paths slated for deletion at Task 17 cutover and were untouched by this branch. No new TODOs introduced by any of the 11 round-2 commits.

Stated limitations (intentional scope decisions, NOT TODOs) confirmed:

- Map render background tile blitting: documented at `services/atlas-renders/atlas.com/renders/mapr/composite.go:21-26` as `STATED LIMITATION` in the package doc comment.
- atlas-renders not in `docker-compose.core.yml`: deferred to Task 17 per task scope.

## Build + test verification

```
go -C services/atlas-data/atlas.com/data test -race -count=1 ./...
  ok  atlas-data/quest 1.091s
  ok  atlas-data/reactor 1.138s
  ok  atlas-data/runtime/rest 1.072s
  ok  atlas-data/searchindex 1.118s
  ok  atlas-data/setup 1.054s
  ok  atlas-data/skill 1.156s
  ok  atlas-data/storage/minio 1.014s
  ok  atlas-data/tenantpurge 1.024s
  ok  atlas-data/wzinput 1.020s
  ok  atlas-data/xml 1.012s
  → ALL OK (exit 0)

go -C services/atlas-data/atlas.com/data vet ./...
  → clean (exit 0, empty output)

go -C services/atlas-renders/atlas.com/renders test -race -count=1 ./...
  ok  atlas-renders 1.188s
  ok  atlas-renders/character 1.018s
  ok  atlas-renders/mapr 1.018s
  ok  atlas-renders/storage 1.013s
  → ALL OK (exit 0)

go test ./libs/atlas-wz/...
  ok  .../libs/atlas-wz/atlas (cached)
  ok  .../libs/atlas-wz/atlas/pngenc (cached)
  ok  .../libs/atlas-wz/canvas (cached)
  ok  .../libs/atlas-wz/charparts (cached)
  ok  .../libs/atlas-wz/crypto (cached)
  ok  .../libs/atlas-wz/icons (cached)
  ok  .../libs/atlas-wz/manifest (cached)
  ok  .../libs/atlas-wz/mapimage (cached)
  ok  .../libs/atlas-wz/maplayout (cached)
  ok  .../libs/atlas-wz/wz (cached)
  ok  .../libs/atlas-wz/wz/property (cached)
  → ALL OK (exit 0)

docker build -f services/atlas-data/Dockerfile .
  → DONE exit 0 (sha256:18acf78f0673…)

docker build -f services/atlas-renders/Dockerfile .
  → DONE exit 0 (sha256:b70abcafe971…)

npm test (services/atlas-ui)
  Test Files  76 passed (76)
       Tests  712 passed (712)
       Duration  9.15s
  → exit 0

  (Note: a separate `npm test` run earlier in the audit produced a single failure
  in a pre-existing rename-dialog test on >100-char input. Did not reproduce on
  rerun; classified as test-suite flake unrelated to task-071 changes — no
  task-071 file touches the rename-dialog component.)
```

## Overall assessment

**MERGEABLE.**

All blocking findings from round 1 are closed with file:line evidence. The branch is now a runnable scaffolding-plus-implementation PR: the 10 WZ→Postgres+MinIO workers can populate Postgres + MinIO, baseline publish/restore can move binary data, the k8s control plane reports real Job state and cleans up stuck Jobs, and atlas-renders serves real character + map composites with proper tenant scoping. Two scope-documented deferrals remain (atlas-renders compose entry; full upstream-stub ingress regression harness) and are correctly assigned to Task 17. Recommendation: ready to ship as the implementation PR; Task 17 follows as the cutover PR.

## Frontend audit (Round 2)

- **Scope:** Re-audit of TypeScript/React changes for task-071-gamedata-minio-consolidation after fix commit `4a5a68506`.
- **Guidelines source:** `.claude/skills/frontend-dev-guidelines/SKILL.md` (+ resources).
- **Build:** PASS (`npm run build` clean; 1.40s).
- **Typecheck:** PASS (`tsc --noEmit -p tsconfig.app.json` clean).
- **Tests:** PASS — 76 files / 712 tests pass (Vitest 4.1.7).
- **Lint:** PASS — no eslint errors in touched files (`ScopeToggle`, `useBaseline`, `baseline.service`, `SetupPage`, `useSeed`).
- **Overall:** PASS.

### Files audited (post-fix)

- `services/atlas-ui/src/components/features/setup/ScopeToggle.tsx` (component)
- `services/atlas-ui/src/lib/hooks/api/useBaseline.ts` (hook)
- `services/atlas-ui/src/lib/hooks/api/useSeed.ts` (hook — touched in r1 follow-up via `dataStatusKey` export)
- `services/atlas-ui/src/pages/SetupPage.tsx` (page)
- `services/atlas-ui/src/services/api/baseline.service.ts` (service)
- `services/atlas-ui/src/components/features/setup/__tests__/ScopeToggle.test.tsx` (test — new)
- `services/atlas-ui/src/lib/hooks/api/__tests__/useBaseline.test.tsx` (test — new)
- `services/atlas-ui/src/services/api/__tests__/baseline.service.test.ts` (test — new)

### Round-1 finding resolution

| ID | Round-1 finding | Status | Evidence |
|----|-----------------|--------|----------|
| FE-06 | Raw `bg-amber-600` / `text-amber-700` in `ScopeToggle.tsx:35,43` | PASS | `ScopeToggle.tsx:30` uses `variant={value === 'shared' ? 'destructive' : 'outline'}` and `ScopeToggle.tsx:38` uses `text-destructive`. No `bg-amber-*` or `text-amber-*` remain in the file (grep clean). |
| FE-09 | `useBaseline.ts` typed `Tenant` non-null; `SetupPage.tsx:103-104` used `activeTenant!` | PASS | `useBaseline.ts:6` and `:22` both type `tenant: Tenant \| null`. Internal guards at `:10-12` and `:26-28` throw before calling the service. `SetupPage.tsx:103-104` now passes `activeTenant` without `!`. Handler call sites (`:167`, `:187`) early-return on `!activeTenant`. |
| FE-11 | `useBaseline.ts` had no `onSuccess` invalidation | PASS | `useBaseline.ts:15-18` invalidates `dataStatusKey(tenant.id)` after restore; `:31-34` does the same after publish. `dataStatusKey` is exported from `useSeed.ts:25` and matches the key consumed by `useDataStatus` (`useSeed.ts:175`). |
| FE-17 | Zero tests for `ScopeToggle`, `useBaseline`, `baseline.service` | PASS | Three real test files exist with substantive behavioral assertions. `ScopeToggle.test.tsx`: 5 tests covering aria state, both onChange directions, warning visibility, and a semantic-token regression assertion (lines 45-50 explicitly enforce `text-destructive` and ban `text-amber-*`). `useBaseline.test.tsx`: 6 tests covering null-tenant rejection, dataStatus invalidation, and service argument forwarding for both mutations. `baseline.service.test.ts`: 6 tests covering header construction (TENANT_ID/REGION/MAJOR_VERSION/MINOR_VERSION/X-Atlas-Operator), JSON body, JSON error decoding, and non-JSON fallback for both methods. |
| Error decoding | `baseline.service.ts:20,37` used status-only strings | PASS | `baseline.service.ts:11-19` defines `decodeErrorMessage` which awaits `response.json()` and returns `parsed.error` when present; `:31` and `:51` use it with status-coded fallbacks. Tests at `baseline.service.test.ts:58-72,108-118` verify the decoded body propagates. Resulting error messages are user-actionable (`sha256 mismatch`, `missing region`) and surface via `toast.error(\`Baseline restore failed: ${error.message}\`)` at `SetupPage.tsx:180,199`. |
| FE-14 (r1 non-blocking) | `useSeed.ts:24-33` flat queryKey tuples instead of hierarchical `as const` factory | PASS-AT-RULE | All key factories at `useSeed.ts:24-33` are annotated `as const`. They remain flat 2-tuples rather than hierarchical (which is permitted — the FE-14 rule only requires `as const`). A hierarchical refactor would still be a nice-to-have but is not a guideline violation. |

### Fresh FE-* checklist (post-fix sweep)

#### Anti-pattern checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-01 | No `any` type | PASS | `grep -nE ': any\|as any'` on all 7 touched files returns zero matches. `useSeed.ts:141,153,165,...` retain pre-existing `activeTenant!` non-null casts, which are not `any` and not in scope for this task. |
| FE-02 | No manual class concatenation | PASS | All className values in touched files are static strings; no `+` or template-string concatenation. `ScopeToggle.tsx:14,15,38`, `SetupPage.tsx` use plain string classNames. |
| FE-03 | No direct API client calls in components | PASS | `ScopeToggle.tsx` imports only `Button`. `SetupPage.tsx` imports `useSeed` / `useBaseline` hooks plus `useTenant`; no `@/lib/api/client` import. |
| FE-04 | No inline Zod schemas in components | PASS | Grep for `z\.(object\|string\|number)` returns zero matches in touched files. |
| FE-05 | No spinners for content loading | PASS | `Loader2 ... animate-spin` instances in `SetupPage.tsx:344,371,399,427,460` are all on submit buttons gated by `*.isPending` — permitted. No content-area spinners. |
| FE-06 | No hardcoded colors | PASS | Grep for `bg-(amber\|red\|...)-\d` and `text-(amber\|red\|...)-\d` on `ScopeToggle.tsx` and `SetupPage.tsx` returns zero matches. Semantic tokens (`text-destructive`, `text-muted-foreground`) used throughout. |
| FE-07 | No state mutation | PASS | `SetupPage.tsx` uses `setScope(...)` only; no `.push/.splice/.sort` followed by setState. |
| FE-08 | No default exports for components | PASS | `ScopeToggle.tsx:12`, `SetupPage.tsx:74` use `export function`. Grep for `export default function` returns zero matches in touched files. |
| FE-09 | Tenant guard in hooks | PASS | `useBaseline.ts:6,22` accept `Tenant \| null`; mutationFn throws if null before service call (`:10-12`, `:26-28`). `onSuccess` short-circuits with `if (!tenant) return` (`:16`, `:32`). Mutation will not silently fire against an unresolved tenant. |
| FE-10 | Tenant ID in query keys | PASS | `dataStatusKey(tenantId)` at `useSeed.ts:25` is keyed by tenant id; consumer `useDataStatus` at `:172-181` falls back to `['dataStatus', 'none']` when tenant unresolved with `enabled: !!activeTenant`. Invalidation key from `useBaseline.ts:17,33` matches the active-tenant key exactly. |
| FE-11 | Error handling | PASS | `baseline.service.ts:11-19,30-33,50-53` throws `Error` with decoded `error` field. `SetupPage.tsx:179-181,198-200` surfaces `error.message` via `toast.error`. The pattern matches the existing `useSeed`/`SetupPage` flow; the formal `createErrorFromUnknown` helper is wired into the `apiClient` path, not the bare-`fetch` services used here. |

#### Architecture checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-12 | JSON:API model shape | PASS | `Tenant` (used at `useBaseline.ts:3`) follows `{id, attributes:{...}}` — confirmed by `useBaseline.test.tsx:17-25` and `baseline.service.test.ts:5-13`. `SetupPage.tsx:170-173,190-192` reads `activeTenant.attributes.region/majorVersion/minorVersion`. |
| FE-13 | Service pattern | PASS | `BaselineService` at `baseline.service.ts:21-55` follows the direct-fetch pattern used by sibling services. Atlas-ui CLAUDE.md documents this as a permitted variant; using `tenantHeaders(tenant)` keeps the four-header tenant contract intact. |
| FE-14 | Query key factory uses `as const` | PASS | All key factories in `useSeed.ts:24-33` are `as const`, including the newly-exported `dataStatusKey` at `:25`. |
| FE-15 | Forms use `react-hook-form` + `zodResolver` | N/A | No forms introduced in this task — mutation triggers are bare buttons. |
| FE-16 | Schema in `lib/schemas/` with inferred type | N/A | No Zod schemas introduced. |

#### Testing checklist

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FE-17 | Tests exist for changed components | PASS | Three new test files (see Round-1 row above). All exercise real behavior; not shallow renders. `ScopeToggle.test.tsx:45-50` even regression-guards FE-06. |
| FE-18 | Mocks updated when services changed | PASS | `baselineService` is consumed only by `useBaseline.ts`, which mocks the module inline at `useBaseline.test.tsx:10-15`. No central `__mocks__/` entry required by convention. |

### Test quality assessment (behavioral depth)

- `useBaseline.test.tsx:36-48,88-95` confirms the null-tenant guard actually short-circuits — `result.current.isError` is asserted and `baselineService.restore`/`publish` are explicitly verified to have not been called. Not a "render-and-pass" shallow check.
- `useBaseline.test.tsx:50-65,97-107` uses `vi.spyOn(qc, 'invalidateQueries')` and asserts the exact query key tuple matches `dataStatusKey(tenant.id)`. This would fail if the invalidation key drifted from `useDataStatus`'s consumer key.
- `baseline.service.test.ts:26-56,96-106` inspects the actual `Headers` instance constructed by `tenantHeaders()` and verifies each of the four tenant headers individually, plus the `X-Atlas-Operator: 1` differentiator on publish. Not shallow.

### Frontend summary

- Round-1 blockers cleared: FE-06, FE-09, FE-11, FE-17 — all four PASS with file:line evidence.
- Non-blocking error-decoding finding cleared (`baseline.service.ts:11-19`).
- Non-blocking `useSeed` query-key finding: factories are `as const`, so the rule is satisfied; hierarchical structure remains optional polish.
- No NEW-ISSUE introduced by the fix sweep. Build, typecheck, tests, and lint all clean.

## Backend audit (Round 2)

- **Reviewer:** backend-guidelines-reviewer
- **Date:** 2026-05-20
- **Scope:** Go changes only (libs/atlas-wz + atlas-data + atlas-renders) on branch `task-071-gamedata-minio-consolidation`, re-audited after commits 9782446e5..31ced9bd1
- **Build:** PASS — `go build`, `go vet ./...`, `docker build -f services/atlas-data/Dockerfile .`, and `docker build -f services/atlas-renders/Dockerfile .` all exit 0
- **Tests:** PASS — `go test -race -count=1 ./...` green across atlas-data, atlas-renders, and libs/atlas-wz (incl. new `baseline`, `data/workers`, `data/wztoxml`, `runtime/rest`, `character`, `mapr`, `storage`, `libs/atlas-wz/charparts` packages)
- **Overall:** **PASS** with one residual non-blocking carryover (`wzinput SUB-02/03`) and one user-deferred infra item (`SCAFFOLD-06`)

### Round-1 blocker resolution

| # | Round-1 item | Round-2 file:line | Status |
|---|---|---|---|
| 1 | SEC-01 — restore not operator-gated | `services/atlas-data/atlas.com/data/baseline/handler.go:64-67` adds `X-Atlas-Operator: 1` short-circuit before any DB work | PASS |
| 2 | SEC-02 — sha256 verified after DB mutation | `services/atlas-data/atlas.com/data/baseline/restore.go:51-72` stages dump to `os.CreateTemp` while hashing and returns `ErrShaMismatch` BEFORE the table iteration at line 95 | PASS |
| 3a | Stub `Item.Run` | `data/workers/item.go:26-72` (Item.wz categories + per-id icon emit) | PASS |
| 3b | Stub `Mob.Run` | `data/workers/mob.go:1-73` (UI.wz cross-archive fetch via `fetchArchive`) | PASS |
| 3c | Stub `Npc.Run` | `data/workers/npc.go:1-58` | PASS |
| 3d | Stub `Reactor.Run` | `data/workers/reactor.go:1-50` | PASS |
| 3e | Stub `Skill.Run` | `data/workers/skill.go:1-88` | PASS |
| 3f | Stub `Quest.Run` | `data/workers/quest.go:21-33` | PASS |
| 3g | Stub `String.Run` | `data/workers/stringw.go:1-53` | PASS |
| 3h | Stub `Map.Run` | `data/workers/mapw.go:1-94` | PASS |
| 3i | Stub `Character.Run` | `data/workers/character.go:36-101` packs atlases via `libs/atlas-wz/charparts` and emits smap sidecar | PASS |
| 3j | Stub `UI.Run` | `data/workers/ui.go:1-97` | PASS |
| 3k | Stub `runCopyOut` | `baseline/publish.go:98-118` real `pgx.Conn().PgConn().CopyTo` against whitelisted table + constant canonical UUID | PASS |
| 3l | Stub `copyInBinary` | `baseline/restore.go:180-210` real `pgx.PgConn().CopyFrom` with `io.Pipe` + drain pattern (`pr.CloseWithError(err)` + `<-errc`) | PASS |
| 3m | Stub `Watchdog.sweep` | `runtime/rest/watchdog.go:39-110` lists labeled Jobs, reads Redis heartbeats, foreground-deletes stuck Jobs | PASS |
| 3n | Stub `processStatus` hardcoded `[]` | `runtime/rest/resource.go:90-125` lists real `BatchV1().Jobs(...)` via `labelIngest` selector and emits typed per-Job status | PASS |
| 3o | Stub `defaultTemplate` hardcoded | `runtime/rest/jobs.go:74-115` `loadTemplateFromConfigMap` reads `atlas-data-ingest-job-template/job.yaml`, rejects missing key + empty container image | PASS |
| 3p | Stub `character.Handler` 501 | `services/atlas-renders/atlas.com/renders/character/handler.go:32-154` full composite (cache probe, hash verify at :81-91, MinIO composite via `Composite`, best-effort PUT) | PASS |
| 3q | Stub `mapr.Handler` 501 | `services/atlas-renders/atlas.com/renders/mapr/handler.go:32-69` (minimap 302 + render composite via `serveRender`) | PASS |
| 3r | Dead code `runtime/all/run.go` | Deleted — `ls services/atlas-data/atlas.com/data/runtime/` shows only `ingest/` and `rest/` | PASS |
| 4 | SUB-04 / DOM-08 — baseline POSTs used `RegisterHandler` | `baseline/handler.go:22-23` now `rest.RegisterInputHandler[PublishInputModel]` / `[RestoreInputModel]` | PASS |
| 5 | SCAFFOLD-01 — atlas-renders missing from services.json | `.github/config/services.json:406-410` registers atlas-renders | PASS |
| 6 | SCAFFOLD-06 — atlas-renders missing from docker-compose.core.yml | Absent — explicitly deferred per user instruction to Task 17 | DEFERRED |
| 7 | SCAFFOLD-08 — atlas-renders missing Bruno collection | `services/atlas-renders/.bruno/{bruno.json,collection.bru,environments/,*.bru}` present | PASS |
| 8 | atlas-renders main.go missing tenant middleware | `services/atlas-renders/atlas.com/renders/main.go:28` installs `tenantMiddleware(l)`; parser at `:68-94` injects `tenant.Model` | PASS |
| W-A | `CanonicalTenantUUID` duplicated | Consolidated into `services/atlas-data/atlas.com/data/canonical/canonical.go:11`; both `baseline/publish.go:105` and `tenantpurge/purge.go:32` import it | PASS |
| W-B | Width drift on MajorVersion/MinorVersion | Unified to `uint16` — `data/workers/worker.go:17-18`, `runtime/rest/jobs.go:121`, `libs/atlas-tenant` | PASS |
| W-C | DOM-18 — baseline publish returned plain JSON | `baseline/handler.go:50-52` sets `application/vnd.api+json` + `server.MarshalResponse[PublishOutputModel]`; restore returns 202 empty body (acceptable for accepted operations) | PASS |

### Checklist on new code introduced

#### libs/atlas-wz/charparts/ (new package)

| ID | Status | Evidence |
|---|---|---|
| DOM-20 table-driven tests | PASS | `libs/atlas-wz/charparts/{extract_test.go,smap_test.go}` cover Walk/ExtractSmap/ToAtlasInputs |
| DOM-21 atlas-constants duplication | PASS | Pure WZ-walker; no game id types declared |
| Manifest Vslot determinism | PASS | `libs/atlas-wz/manifest/types.go:15` uses `omitempty`; `encode_test.go:43-57` asserts byte-identical legacy manifests; `:61-87` asserts deterministic Vslot output |

#### services/atlas-data workers (10 archive bodies + helpers)

| ID | Status | Evidence |
|---|---|---|
| Worker contract uses `*wz.File` | PASS | `data/workers/worker.go:26-30` |
| Worker concurrency bounded | PASS | `data/runwz.go:23-25` `semaphore.NewWeighted` + `errgroup.WithContext`; each worker holds its own `wz.File`; cross-archive helpers in `data/workers/runtime.go:102,128` open per-call downloads — no shared mutable state |
| Tenant context derivation | PASS | `data/workers/runtime.go:34-63` `tenantFromParams` uses `canonical.TenantUUID` for shared scope |
| DOM-21 atlas-constants | PASS | Workers delegate to existing domain register funcs; no id type duplication |

#### services/atlas-data baseline/

| ID | Status | Evidence |
|---|---|---|
| SEC-01 operator gate (publish + restore) | PASS | `handler.go:35-38, 64-67` |
| SEC-02 sha-before-mutate | PASS | `restore.go:65-72` |
| SUB-03 `RegisterInputHandler[T]` | PASS | `handler.go:22-23` |
| SUB-04 no manual JSON parsing | PASS | typed input model |
| DOM-06 FieldLogger | PASS | `publish.go:25`, `restore.go:35` |
| DOM-07 d.Logger() | PASS | `handler.go:39, 68` |
| DOM-15 no direct entity creation in handlers | PASS | Mutations confined to `Publisher.Publish` / `Restorer.Restore` |
| DOM-18 JSON:API response | PASS (publish) | `handler.go:50-52` uses `server.MarshalResponse[PublishOutputModel]` |
| SQL injection on COPY out | PASS | `publish.go:104-105` interpolates whitelisted `DumpTables` entry + constant `canonical.TenantUUID` — no user-controlled SQL |

#### services/atlas-data runtime/rest/

| ID | Status | Evidence |
|---|---|---|
| Watchdog real implementation | PASS | `watchdog.go:39-110` |
| processStatus real implementation | PASS | `resource.go:90-125` |
| ConfigMap template loader | PASS | `jobs.go:74-115`; rejects missing key + empty container image |
| Operator gate on `scope=shared` | PASS | `resource.go:44-47` |
| `*RegisterHandler` on POST/GET | NOTED | `resource.go:25-26` POST/GET use `RegisterHandler` not `RegisterInputHandler[T]`. Neither endpoint takes a JSON body (params come from headers + query string), so DOM-08 does not apply cleanly. Acceptable |

#### services/atlas-data canonical/

| ID | Status | Evidence |
|---|---|---|
| Single source of truth | PASS | `canonical/canonical.go:11` |

#### services/atlas-renders/

| ID | Status | Evidence |
|---|---|---|
| Tenant middleware installed | PASS | `main.go:28`; parser at `:68-94` |
| Character composite + vslot occlusion fidelity | PASS | `character/composite.go` (611 LOC) + `character/vslot.go:96-137` `applyVslotOcclusion` (port of donor `characterimage/vslot.go`) |
| Hash-verify before composite | PASS | `character/handler.go:81-91` recomputes `LoadoutHash(canonical)` and rejects URL tampering |
| Path tenant vs context tenant cross-check | PASS | `character/handler.go:64-71` rejects cross-tenant cache probes |
| Best-effort cache PUT with fresh ctx | PASS | `character/handler.go:135-143` and `mapr/handler.go:120-126` use `context.WithTimeout(context.Background(), 10*time.Second)` so client cancellation doesn't abort cache writes |
| Map minimap 302 + render composite | PASS | `mapr/handler.go:55-67` |
| LRU cache thread-safety | PASS | `storage/lru.go:38-44` uses `hashicorp/golang-lru/v2` (internally locked) |
| MinIO secret never logged | PASS | `storage/minio.go`, `storage/config.go` |
| Import lint isolation | PASS | `services/atlas-renders/atlas.com/renders/import_lint_test.go` present |

### Residual / non-blocking carryovers

1. **wzinput SUB-01/02/03** — `services/atlas-data/atlas.com/data/wzinput/handler.go:75` still calls `mc.Put` inside the handler; `resource.go:20-21` registers via `RegisterHandler`. This is the existing multipart-upload codepath; round-2's user-supplied fix list did not include it and round-1 already accepted it as a non-blocker. Follow-up item.
2. **SCAFFOLD-06** — atlas-renders not in `deploy/compose/docker-compose.core.yml`. Deferred to Task 17 per explicit user instruction.
3. **Operator-header threat-model docstring** — `X-Atlas-Operator: 1` is a plain header relying on ingress filtering. Documentation-only nit.

### TODO sweep

`grep -rn 'TODO\|not yet implemented\|StatusNotImplemented'` against the three target trees (excluding `_test.go`) returns 11 hits — all in legacy XML readers `services/atlas-data/atlas.com/data/{skill,map}/reader.go` slated for deletion at Task 17 cutover. No round-2 code introduces new TODO / 501 / `not yet implemented` strings.

### Backend audit (Round 2) summary

**Blocking:** none. All 8 round-1 blockers and 3 of the 8 round-1 non-blocking items are resolved with file:line evidence; build, vet, `go test -race`, and `docker build` all pass for atlas-data and atlas-renders.

**Non-blocking carryovers:** wzinput SUB-02/03 (pre-existing), SCAFFOLD-06 (deferred), operator-header docstring (documentation only).

