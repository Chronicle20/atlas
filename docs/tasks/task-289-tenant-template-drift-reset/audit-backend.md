# Backend Audit — atlas-configurations (task-289 branch diff)

- **Service Path:** services/atlas-configurations/atlas.com/configurations
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-09-02
- **Range:** 9613e7259 (merge-base) .. 0f1b3c75e (HEAD)
- **Build:** PASS
- **Tests:** all packages `ok` (no FAIL), 0 failed
- **Overall:** NEEDS-WORK

## Build & Test Results

```
$ go build ./...          -> exit 0, no output
$ go test ./... -count=1  -> all listed packages "ok" or "[no test files]"; none failed
```
`drift`, `tenants`, `templates`, `templates/characters/preset` all reported `ok`.

## Applicability

| Family | Fired? | Trigger evidence |
|---|---|---|
| FILE-01..06 | Yes | Every changed package (`drift`, `tenants`, `templates/characters/preset`) runs unconditionally. |
| DOM structure (01-05,11,16) | Yes | `tenants` has `rest.go`, `entity.go`, `provider.go` (rest.go/entity.go changed this diff; provider.go present in the changed package). |
| SUB-01..04 | Yes | `tenants` has `resource.go` and **no** `model.go` — classifies as sub-domain per the checklist's literal `model.go` test, despite having `entity.go`/`processor.go`/`administrator.go`/`provider.go`. |
| REST (06-09,12-19,32) | Yes | `tenants` has `resource.go`, `rest.go`, `processor.go`, and registers routes. |
| Constants reuse (DOM-21) | Yes | New fields `AP uint16`/`SP string` added to `templates/characters/preset.Attributes` (`templates/characters/preset/rest.go:50-51`); checked against `libs/atlas-constants` — no `AbilityPoint`/`SkillPoint` type exists there. |
| Testing (DOM-10,20,24,33) | Yes | Diff adds/changes `_test.go` files and adds a method to the `tenants.Processor` interface. |
| Cache (DOM-29) | No | No `cache.go`; no cached state held on `ProcessorImpl` (only per-request `validator`/`templates` deps). |
| Messaging (DOM-30) | No | No `producer.go`; no `AndEmit`/`message.Emit`/`producer.ProviderImpl` call site anywhere in `tenants` or `outbox`. The write path uses `outboxlib.Enqueue(tx, ...)` (transactional outbox), a different symbol than the ones DOM-30's trigger names. |
| Multi-tenancy (DOM-31) | Yes | `tenants` has `rest.go`; changed code (`resource.go`, `reset.go`) synthesizes and reads tenant/env state. |
| Migration hygiene (DOM-34/35) | No | No symbol moved between a service and `libs/atlas-*`. |
| Deploy & topics (DOM-22/23) | No | No `libs/atlas-*` module added; `EnvTenantStatusTopic` is unchanged (pre-existing, confirmed via diff context lines only). |
| Runtime safety (DOM-26) | Yes (vacuous) | Non-test Go files changed; grep for bare `go ` / `go func` across all changed files: zero matches. |
| Channel wire values (DOM-25) | No | Diff does not touch `atlas-channel`/`atlas-packet`; no client-interpreted byte emitted. |
| Resilience (DOM-27/28) | Yes | DB-backed service handlers changed (`resource.go`); new enrichment fallback `baselineFor` added (`processor.go:144`). |
| External clients (EXT-01..04) | No | No `requests.*Request[T]` call added. |
| Scaffolding (SCAFFOLD-01..09) | No | No new `services/atlas-<svc>/` directory. |
| Security (SEC-01..04) | No | Service does not handle auth/tokens/redirects. |

## Checklist Results

### drift (support package — no `model.go`, no `resource.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor code in processor.go | N/A | No `Processor`/`ProcessorImpl` symbol anywhere in `drift/*.go`. |
| FILE-02 | RestModel/Transform in rest.go | N/A | No `RestModel` type in `drift`; package operates on marshaled JSON only (`drift/doc.go:1-23`). |
| FILE-03 | Cross-service requests in requests.go | N/A | No `requests.*` call in `drift`. |
| FILE-04 | Entity/Migration/TableName in entity.go | N/A | No `Entity` type in `drift`. |
| FILE-05 | Builder/Model/administrator/provider placement | N/A | None of these exist in `drift`; it is a pure function package by design (`drift/doc.go:1-23`). |
| FILE-06 | No catch-all file bundling ≥2 responsibilities | PASS | `canonical.go` (canonicalize/prune), `merge.go` (section validate/merge), `revision.go` (hash/sections/aggregate/compare) — each file is a single cohesive responsibility; no `drift.go` catch-all exists. |
| DOM-21 | No redeclaration of `libs/atlas-constants` types | N/A | `drift` declares no domain type, alias, or numeric-literal classification — only string slices/consts describing JSON section names (`drift/doc.go:37-69`), not a match for anything in `libs/atlas-constants`. |
| DOM-20 | Table-driven tests | PASS (mostly), WARN on 1 file | `canonical_test.go`, `merge_test.go` use `tests := []struct{...}` + `t.Run` (confirmed via grep, 3/2 `t.Run` sites). `crosstype_test.go` (3 `Test*` funcs, 0 `t.Run`) is single-scenario regression coverage per function, not an enumerable case set — WARN, non-blocking, since the letter of DOM-20 requires the table shape unconditionally. |
| DOM-10 | Test DB via `database.RegisterTenantCallbacks` | N/A | `drift` package tests build no GORM DB at all — pure functions over `Doc`/JSON. |
| DOM-24 | Producer stub in tests reaching an emit path | N/A | `drift` reaches no emit path. |

### tenants (classified sub-domain per literal `model.go` absence; also runs DOM-04/05/11 per the checklist's explicit "model.go or not" carve-out)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor interface/constructor/ProcessorImpl methods in processor.go or processor_<group>.go | **FAIL (Important)** | `ResetById` (`tenants/reset.go:36`) and `validateReset` (`tenants/reset.go:166`) are `*ProcessorImpl` methods living in a bare topic-named file, `reset.go` — exactly the anti-pattern the rule names (`custody.go`/`register.go`). The compliant split name would be `processor_reset.go`. |
| FILE-02 | RestModel/Transform/GetName/GetID/SetID in rest.go | PASS (partial) | `GetName`/`GetID`/`SetID` all in `tenants/rest.go:35-46`. No `Transform`/`Extract` exists anywhere in the package (see DOM-04 below). |
| FILE-03 | Cross-service requests in requests.go | N/A | No `requests.*Request[T]` call added. |
| FILE-04 | Entity/Migration/TableName in entity.go | PASS | `tenants/entity.go:12-42` (unchanged this diff, but present and correctly placed). |
| FILE-05 | Builder/Model/administrator/provider placement | N/A (Builder/Model) / PASS (administrator/provider) | No `model.go`/`builder.go` in this package at all (see DOM-01 N/A below). Writes (`update`, `delete`) are in `administrator.go`; readers in `provider.go`. |
| FILE-06 | No catch-all bundling ≥2 responsibilities | PASS | No file named `tenants.go`; `resource.go` carries only route registration + handlers (its own designated role), though see DOM-19/32 below for two responsibilities (request-model shape, error formatting) that leaked into it via the new reset endpoint. |
| SUB-01 | Business logic not in handler | PASS | All reset business logic (`ResetById`, `validateReset`) lives on `*ProcessorImpl` (`tenants/reset.go:36,166`); `handleResetConfigurationTenant` only wires dependencies and maps errors to statuses (`tenants/resource.go:223-289`). |
| SUB-02 | No `db.Create`/`db.Save`/`db.Delete` in resource.go | PASS | `grep -n "db\.Create\|db\.Save\|db\.Delete" tenants/resource.go` → zero matches. |
| SUB-03 | POST routes via `RegisterInputHandler[T]` | **FAIL (Important)** | `r.HandleFunc("/{tenantId}/reset", rest.RegisterHandler(l)(si)("reset_configuration_tenant", handleResetConfigurationTenant(db))).Methods(http.MethodPost)` — `tenants/resource.go:36` — POST registered via bare `RegisterHandler`, not `RegisterInputHandler[T]`. |
| SUB-04 | No manual JSON parsing in resource.go | **FAIL (Important)** | `err := json.NewDecoder(r.Body).Decode(&body)` — `tenants/resource.go:78`, inside `parseResetSections`. |
| DOM-01 | builder.go + NewBuilder + Build() | N/A | Package has no `model.go` — trigger does not fire. |
| DOM-02/03 | `ToEntity`/`Make(Entity)` | N/A (DOM-02) / evaluated below (DOM-03 needs model.go+entity.go, not met) | No `model.go`; `Make(Entity) (RestModel, error)` exists (`tenants/processor.go:272`) but targets `RestModel` directly rather than an intermediate `Model` — DOM-03's own trigger (`model.go` + `entity.go`) does not fire since there is no `model.go`. |
| DOM-04 | `func Transform(Model)(RestModel,error)` in rest.go | **FAIL (Important)** | `grep -n "^func Transform" tenants/*.go` → zero matches anywhere in the package. Per the checklist's explicit note ("DOM-04/05 ... apply to any package with those files, model.go or not"), this fires regardless of the missing `model.go`. |
| DOM-05 | `TransformSlice` used by list handlers | **FAIL (Important)** | No `TransformSlice` exists; `handleGetConfigurationTenants` (`tenants/resource.go:88-109`) builds its list straight from `viewProcessor(d, db).AllViewProvider(page)()` with no transform step at all. |
| DOM-06 | Processor ctor takes `logrus.FieldLogger` | PASS | `func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor` — `tenants/processor.go:93`. |
| DOM-07 | Handlers pass `d.Logger()` | PASS | Every `NewProcessor(` call site in `resource.go` (lines 46, 147, 171, 211, 241, 250) passes `d.Logger()`; none pass `logrus.StandardLogger()`. |
| DOM-08 | POST/PATCH via `RegisterInputHandler[T]` | **FAIL (Important)** | Same evidence as SUB-03: `tenants/resource.go:36` registers `POST /{tenantId}/reset` via `rest.RegisterHandler`, not `RegisterInputHandler[T]`. PATCH (`resource.go:34`) and the other POST (`resource.go:32`) correctly use `RegisterInputHandler[RestModel]`. |
| DOM-09 | Every `Transform(` call checked | N/A | `grep -n "Transform(" tenants/resource.go` → zero matches; rule's own trigger does not fire. |
| DOM-11 | Providers lazy via `database.Query`/`SliceQuery` | **FAIL (Important)** | `byIdEntityProvider` and `byRegionVersionEntityProvider` (`tenants/provider.go:23-51`) eagerly call `.First(&result)` then wrap the already-fetched value in `model.FixedProvider[Entity](result)` (`provider.go:32`, `provider.go:48`) — the exact anti-pattern DOM-11 names. Pre-existing (file unchanged this diff), read here because `reset.go`/`processor.go` call these functions directly and correctness of the reset path's environment scoping depends on their contract. |
| DOM-12 | No `os.Getenv` in resource.go | PASS | `grep -n "os.Getenv" tenants/resource.go` → zero matches. |
| DOM-13 | No cross-domain orchestration in handlers | PASS | `handleResetConfigurationTenant` wires only the tenant's own processor plus `templates.NewProcessor` for baseline resolution (`tenants/resource.go:250-252`) — same shape the existing PATCH handler already uses (`resource.go:147-148`); no decision logic lives in the handler. |
| DOM-14 | Handlers call processor methods only | PASS | `resource.go:241` calls `.GetById(`, `resource.go:254` calls `p.ResetById(` — both processor methods, no direct provider call. |
| DOM-15 | No `db.Create`/`Save`/`Delete` in resource.go | PASS | Same evidence as SUB-02. |
| DOM-16 | administrator.go holds domain writes | N/A | Rule requires `model.go`, which this package lacks; writes are nonetheless correctly isolated in `administrator.go`. |
| DOM-17 | Domain errors map to correct HTTP status | PASS | `ErrTenantNotFound`→404, `ErrNoBaselineTemplate`→409 (conflict), `drift.ErrUnknownSection`→400 (`tenants/resource.go:263-270`). The 422-for-validationFailureError and 403-for-cross-environment-write-as-defense-in-depth arms are per the explicit prior ruling (not relitigated) and are present verbatim at `resource.go:265-266` and `resource.go:272-281`. |
| DOM-18 | RestModel implements GetName/GetID/SetID | PASS | `tenants/rest.go:35-46`; `ViewRestModel` (`rest.go:70-77`) gets all three by embedding `RestModel`. |
| DOM-19 | Request models flat, no nested Data/Attributes | **FAIL (Important)** | `resetRequest` (`tenants/resource.go:68-74`) is `Data.Attributes.Sections` — precisely the nested envelope shape the rule forbids — and it is hand-decoded rather than run through the framework, so the flattening `RegisterInputHandler[T]` would otherwise provide never happens. It also is not defined in `rest.go` at all. |
| DOM-20 | Table-driven tests | PASS | `reset_test.go` (11 `t.Run`), `resource_reset_test.go` (12 `t.Run`), `view_test.go` (8 `t.Run`) are all subtest/table-shaped. |
| DOM-24 | Producer stub for emit-reaching tests | N/A | `grep -rn "AndEmit\|message.Emit\|producer.Produce\|producer.ProviderImpl" tenants/ outbox/` → zero matches; the write path uses `outboxlib.Enqueue` (a DB-only transactional-outbox insert), never a live producer call. |
| DOM-27 | DB-backed handlers use `server.WriteErrorResponse` for 500s | PASS | `tenants/resource.go:284` (`default:` arm) uses `server.WriteErrorResponse(d.Logger())(w)(err)`; no direct `w.WriteHeader(http.StatusInternalServerError)` anywhere in the changed handler. Classifier registration (`server.RegisterTransientErrorClassifier`) confirmed pre-existing at `main.go:56` (outside the diff, checked because DOM-27(b) requires it). |
| DOM-28 | Enrichment fallback degrades loudly (`model.ErrDecorator` + `degrade.Observe`) | **FAIL (Important)** | `baselineFor` (`tenants/processor.go:144-160`, new this diff) fetches cross-domain data via `p.templates.GetByRegionAndVersion` and, on a non-`ErrRecordNotFound` failure, only logs `p.l...Warn(...)` (`processor.go:151-155`) and returns `ok=false` — no `model.ErrDecorator`, no `degrade.Observe(...)` call. `grep -rln "degrade.Observe\|model.ErrDecorator" services/atlas-configurations/` returns nothing: this service has never adopted the required metric emission, and this new fallback repeats the omission. |
| DOM-29 | Cache is an app-scoped singleton | N/A | No `cache.go`; `ProcessorImpl.templates`/`.validator` are per-request collaborators set via `With*`, not cached state. |
| DOM-30 | DB write emits via `AndEmit`+`message.Buffer` | N/A (own trigger absent) — see transaction-boundary note below | No `producer.ProviderImpl(`/`producer.Produce(` call site in `tenants`; the outbox-insert path is verified atomic by inspection (see note). |
| DOM-31 | Tenant/trace identity travels in context only | PASS | `tenants.RestModel`/`ViewRestModel` carry no `TenantId`/`tenant_id` JSON field (`tenants/rest.go:13-33,70-77`); `{tenantId}` in the URL (`resource.go:36`) names the *administered configuration resource's own id*, not the caller's multi-tenancy identity — the actual environment/scope check runs off `env.MustFromContext(ctx)` (`administrator.go:34,65`; `provider.go:18,26,40`), never a client-supplied value. The synthesized `tenantlib.WithContext` value (`resource.go:243-244`) is consumed only by the atlas-data-backed preset validator and uses a disjoint context key space from `atlas-env` (`libs/atlas-tenant/processor.go:12-16,90-97` vs. `libs/atlas-env/env.go:36`), so it cannot alter environment-scope enforcement. See the dedicated section below. |
| DOM-32 | Routes resolve to `server.Register(Input)Handler`; no hand-rolled error writer | **FAIL (Important)** | `writeJSONAPIError` (`tenants/resource.go:53-63`) is a new, local error-response helper — exactly the anti-pattern named in the rule ("a local `writeError`/`respondWithError` helper"). It is used at `resource.go:229,264,266,268,270,279-281`. |
| DOM-33 | Mocks updated for interface changes | PASS | `tenants/mock/processor.go` adds `WithTemplatesFunc`, `ViewByIdProviderFunc`, `AllViewProviderFunc`, `ResetByIdFunc` and matching methods (lines 15,19-20,26,38-43,45-57,115-120) — every new `Processor` method is mocked in the same diff. |

### templates/characters/preset (support package — data-only)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-02 | RestModel in rest.go | PASS | `Attributes`/`RestModel` defined in `templates/characters/preset/rest.go:27-61`. |
| DOM-04 | `func Transform(` in rest.go | **FAIL (Important, low-impact)** | `grep -n "^func Transform" templates/characters/preset/*.go` → zero matches. `preset.RestModel` is a nested field on `templates.RestModel`, never a top-level JSON:API resource, so the practical exposure is small, but the rule fires mechanically on this changed `rest.go` regardless. |
| DOM-21 | No redeclaration of constants-lib types | PASS | New `AP uint16`/`SP string` fields (`rest.go:50-51`) checked against `libs/atlas-constants` — no matching type exists there to reuse. |
| DOM-20 | Table-driven tests | WARN | `TestAttributesCarriesAPAndSP` (`rest_test.go:9-30`) covers two scenarios (zero-value, set-value) inline without `t.Run`/table struct — non-blocking, low duplication cost. |

## Transaction boundary and multi-tenancy scrutiny (destructive write path)

1. **Atomicity.** `ResetById` wraps the row update, the history insert, and the outbox enqueue in one `database.ExecuteTransaction(p.db, func(db *gorm.DB) error { ... })` block (`tenants/reset.go:124-140`). `update(...)` (`administrator.go:58-91`) does the `AuthorizeWrite` check, `HistoryEntity` insert, and `Entity` save on the *same* `db` handle; `enqueueTenantStatus(db, tenantId, sanitized)` (`reset.go:139`, `processor.go:36-54`) writes the outbox row via `outboxlib.Enqueue(tx, ...)` on that same handle. A failure in any step returns a non-nil error from the closure, which `database.ExecuteTransaction`/`db.Transaction` rolls back as a unit (`libs/atlas-database/transaction.go:9-14`). All three writes commit or fail together.
2. **Producer seam.** The actual Kafka publish is decoupled from this transaction by design (transactional outbox: a separate relay reads the `outbox` table and produces). No `producer.ProviderImpl`/`AndEmit` call exists in `tenants`, so DOM-30's specific trigger does not fire, but the atomicity guarantee DOM-30 exists to protect is satisfied by construction here — the outbox row is inside the same SQL transaction as the row it describes.
3. **Cross-tenant/cross-environment safety of the synthesized context.** `handleResetConfigurationTenant` (`resource.go:223-289`) builds a `tenantlib.Model` from `{URL tenantId, stored.Region, stored.MajorVersion, stored.MinorVersion}` purely to let the atlas-data-backed preset validator run (mirrors the existing PATCH path, `resource.go:132-146`). Verified this cannot cross a tenant or environment boundary:
   - `libs/atlas-tenant/processor.go:12-16` keys its context values off `"TENANT_ID"`/`"REGION"`/... string keys; `libs/atlas-env/env.go:36` uses a distinct private `ctxKey` type. The two context spaces cannot collide, so `env.MustFromContext(ctx)` used by `scope.Strict`/`scope.AuthorizeWrite` is unaffected by `tenantlib.WithContext`.
   - The `tenants` GORM `Entity` (`entity.go:16-27`) carries no `tenant_id` column, so `libs/atlas-database`'s automatic tenant-filter callback (`libs/atlas-database/tenant_scope.go:65,89`, `hasTenantColumn`) is a no-op for every query/create against the `tenants` table regardless of what is in `ctx`.
   - `HistoryEntity` (`entity.go:32-42`) *does* have a `TenantId` column (an unrelated, pre-existing naming coincidence — it stores the owning tenant-config-row's own id, not a multi-tenancy tenant), but its `db.Create(he)` call (`administrator.go:76`) runs on the transaction's `db` handle, which never receives `.WithContext(ctx)` — so `tenant.FromContext(ctx)` inside the create callback fails and the callback is skipped, leaving `HistoryEntity.TenantId` exactly as explicitly set (`administrator.go:71`, `= e.Id`).
   - Both the ad-hoc `GetById` pre-read (`resource.go:241`) and `ResetById`'s own internal read (`reset.go:43`) go through `byIdEntityProvider`, which is `scope.Strict`-scoped by the caller's *environment* (`provider.go:26-31`) — a cross-environment tenant id fails both reads identically as `gorm.ErrRecordNotFound`, which `ResetById` maps to `ErrTenantNotFound`→404 (`reset.go:44-51`). No environment or tenant boundary can be crossed by supplying an arbitrary `{tenantId}`.
   - Minor inefficiency (not a rule violation): the extra `GetById` pre-read at `resource.go:241` duplicates the read `ResetById` performs internally at `reset.go:43`; if the first read fails (not found/cross-environment), the code silently falls through without the synthesized tenant context and lets `ResetById`'s own read reproduce the same 404. Functionally correct, redundant I/O only.

## Not evaluable from the diff

none

## Summary

### Blocking (must fix)
- FILE-01: `ResetById`/`validateReset` (`tenants/reset.go:36,166`) are `*ProcessorImpl` methods in a bare topic-named file; move to `processor.go` or rename to `processor_reset.go`.
- DOM-04/DOM-05: no `Transform`/`TransformSlice` exists anywhere in `tenants` (`tenants/rest.go`), despite the checklist's explicit "applies model.go or not" carve-out.
- DOM-08/SUB-03: `POST /{tenantId}/reset` registered via `rest.RegisterHandler` instead of `RegisterInputHandler[T]` (`tenants/resource.go:36`).
- SUB-04: manual `json.NewDecoder(r.Body).Decode(...)` in `resource.go:78` instead of routing the body through the input-handler framework.
- DOM-19: `resetRequest` (`resource.go:68-74`) is a nested `Data.Attributes` request struct, the exact shape DOM-19 forbids, and lives outside `rest.go`.
- DOM-32: hand-rolled `writeJSONAPIError` helper (`resource.go:53-63`) — the named anti-pattern ("local writeError/respondWithError helper").
- DOM-11: `byIdEntityProvider`/`byRegionVersionEntityProvider` (`tenants/provider.go:32,48`) eagerly read then wrap in `FixedProvider`, defeating lazy composition (pre-existing, exercised by this diff's new reset path).
- DOM-28: `baselineFor` (`tenants/processor.go:144-160`, new) degrades a cross-domain fetch failure with only a `Warn` log — no `model.ErrDecorator`/`degrade.Observe` metric, which this service has never adopted anywhere.
- DOM-04 (templates/characters/preset): no `Transform` in `templates/characters/preset/rest.go` (low practical impact — nested, non-top-level model).

### Non-Blocking (should fix)
- DOM-20 (WARN): `drift/crosstype_test.go` and `templates/characters/preset/rest_test.go` are multi-scenario tests written without the `tests := []struct{...}` + `t.Run` shape.
- Redundant `GetById` pre-read in `handleResetConfigurationTenant` (`resource.go:241`) duplicates the read `ResetById` performs internally — functionally correct, just an avoidable extra query.
- Already-accepted, not relitigated: `baselineFor`'s non-`ErrRecordNotFound` degrade branch is untested; `PreservesTheFR44Set`'s Id/Environment assertions are non-diagnostic (Make() always overwrites both).
