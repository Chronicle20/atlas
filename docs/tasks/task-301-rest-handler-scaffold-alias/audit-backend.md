# Backend Audit — task-301-rest-handler-scaffold-alias

- **Service Path:** 21 services (atlas-account, atlas-ban, atlas-character, atlas-drop-information, atlas-events, atlas-fame, atlas-map-actions, atlas-messages, atlas-mounts, atlas-mts, atlas-notes, atlas-npc-conversations, atlas-npc-shops, atlas-parcel, atlas-party-quests, atlas-pets, atlas-portal-actions, atlas-quest, atlas-rankings, atlas-reactor-actions, atlas-reward-pools)
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-09-04
- **Build:** PASS
- **Tests:** 21/21 modules `ok`, 0 failed (exact pass/fail counts not tallied by the module runner; every invocation below returned exit 0 with no `FAIL` lines)
- **Overall:** PASS

## Build & Test Results

`go build ./...` run individually for all 21 changed modules (`services/<svc>/atlas.com/<mod>`): zero output, zero errors, all 21 clean.

`go test ./... -count=1` run individually for all 21 changed modules: every package reported `ok` or `[no test files]`; no `FAIL` line anywhere. Longest run was atlas-character (`atlas-character/pending_change` 195.7s, `atlas-character/kafka/consumer/character` 11.5s) — all passed. Full transcripts were inspected inline during the audit session (not re-pasted here for length); representative tail excerpts:

```
ok  	atlas-account	0.014s
ok  	atlas-account/account	1.256s
...
ok  	atlas-character/pending_change	195.697s
...
ok  	atlas-events/event/definition	0.047s
ok  	atlas-fame/fame	0.036s
...
ok  	atlas-reward-pools/reward	0.110s
```

Per instructions, the flagless `tools/verify.sh` (already reported green) was **not** re-run.

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure (DOM-01..05,11,16) | Yes | Every changed `resource.go` package has `model.go` in the same domain package; no new `rest.go`/`entity.go`/`provider.go` content was added by the diff (verified: zero new `Transform`/`RestModel`/`Migration`/`TableName` declarations added — see Checklist Results). |
| FILE placement (FILE-01..06) | Yes | Every changed Go package (no exemption). |
| SUB sub-domain (SUB-01..04) | Yes | Several changed `resource.go` packages (`atlas-account/ban`, `atlas-character/character/session/history`, `atlas-mts/testsupport`, etc.) have `resource.go` with no local `model.go`. |
| REST (DOM-06..09,12..15,17..19,32) | Yes | Every changed package has `resource.go`, `handler.go` (the `rest` package), or registers HTTP routes. |
| Constants reuse (DOM-21) | No | Diff declares no new type, const block, or numeric-literal classification — every changed line is a re-parameterization of existing calls (`db` moved from struct field to closure param) or a pure alias declaration. |
| Testing (DOM-10,20,24,33) | No | Zero `_test.go` files touched (`grep -c '_test\.go$' <changed-files> == 0`); no `Processor`/`Provider`/`Administrator` interface signature changed — only free-standing handler-factory functions changed shape. |
| Cache (DOM-29) | No | No `cache.go` touched, no processor/struct gained cached state. |
| Messaging (DOM-30) | Partially — N/A for this diff's own lines | Changed `resource.go` files do reach `*AndEmit` calls, but every such call site's substance (which processor method, in which order) is byte-identical before/after — only the `db` parameter source moved from `d.DB()` to a closure argument. No new emit path was added or altered. |
| Multi-tenancy (DOM-31) | Yes | Changed code reads tenant/trace state via `d.Context()` passed into `NewProcessor(...)`. |
| Migration hygiene (DOM-34,35) | No | No symbol moved between a service and a `libs/atlas-*` module — `libs/atlas-rest` itself has an empty diff; services import it, they don't extract into it. |
| Deploy & topics (DOM-22,23) | No | No `libs/atlas-*` module added, no Kafka topic env var added/renamed. |
| Runtime safety (DOM-26) | Yes (family), rule N/A | Non-test Go files changed, but zero bare `go` statements added (`grep -n '^\+.*\bgo '` on every hunk: no hits). |
| Channel wire values (DOM-25) | No | Diff touches none of `services/atlas-channel`, `libs/atlas-packet`; no client-interpreted byte touched. |
| Resilience (DOM-27,28) | Yes (family), rules N/A on new content | Several changed handlers write `http.StatusInternalServerError` in DB-backed services, but every such line is unchanged in substance (diff shows it as a pure re-indent, not new code) — DOM-27 pre-dates this diff and is not this diff's finding to make or clear. No `model.Decorator` changed. |
| External clients (EXT-01..04) | No | No changed package calls `requests.RootUrl`/`requests.GetRequest[T]`/`requests.PostRequest[T]` (grep against every changed `resource.go` and `handler.go`: no hits). |
| Scaffolding (SCAFFOLD-01..10) | No | No `services/atlas-<svc>/` directory added, no new channel `Writer`/`Handler`, `deploy/shared/routes.conf` untouched. |
| Security (SEC-01..04) | No | None of the changed lines touch JWT parsing, revocation, redirects, or secrets — `atlas-account`'s password field passes through an unchanged, untouched line (`account/resource.go:81`, outside every diff hunk). |

## Checklist Results

The core surface of this diff is 19 `rest/handler.go` files converted to type aliases over `libs/atlas-rest/server`, 2 dead `rest` packages deleted (`atlas-events`, `atlas-fame`), and 41 `resource.go` files re-parameterized to take `*gorm.DB` as a handler-factory closure argument instead of reading `d.DB()` off a per-service `HandlerDependency`. `libs/atlas-rest/server` itself is untouched.

### `rest` packages — 19 converted, 2 deleted (support packages)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-32 | Routes register through `server.RegisterHandler`/`RegisterInputHandler[T]` — no bare `http.HandlerFunc`, no manual tenant-header parsing, no custom error-response helpers | PASS | All 19 surviving `rest/handler.go` files declare `type HandlerDependency = server.HandlerDependency`, `type HandlerContext = server.HandlerContext`, `type GetHandler = server.GetHandler`, `var RegisterHandler = server.RegisterHandler`, e.g. `services/atlas-account/atlas.com/account/rest/handler.go:14-25`; identical shape confirmed in all 19 via grep sweep (`services/atlas-ban/.../rest/handler.go:13-24`, `services/atlas-npc-shops/.../rest/handler.go:21-32`, etc.). No local re-implementation of `HandlerDependency`, `ParseInput`, or the middleware chain remains anywhere in the 19. |
| FILE-01..06 | No package-named catch-all; responsibilities placed correctly | PASS | The `rest` packages carry only path-parameter parsing helpers (`ParseAccountId`, `ParseCharacterId`, etc.) plus the aliases — no `Processor`, `RestModel`/`Transform`, `requests.go` content, `entity.go` content, or `builder.go`/`model.go` content bled into these files. Verified: zero new `Transform`/`RestModel`/`Migration`/`TableName` declarations added anywhere in the diff. |
| — (dead-code deletion, not a numbered rule) | Deleted `rest/handler.go` in `atlas-events` and `atlas-fame` had no live callers | PASS | `atlas-events`: `event/definition/resource.go:32-33` already called `server.RegisterHandler(l)(si)` / `server.RegisterInputHandler[PatchInput](l)(si)` directly — the local `rest` package was already dead. `atlas-fame`: `grep -n InitResource services/atlas-fame/atlas.com/fame/main.go` returns no matches — atlas-fame registers no REST routes at all, so its `rest/handler.go` (124 lines) was dead code. `go build ./...` and `go test ./... -count=1` both pass clean for both modules post-deletion. |
| — (behavior delta, disclosed) | `ParseEnvironment` gained in the middleware chain for all 19 | PASS (documented, intentional) | Old local chain: `RetrieveSpan → ParseTenant` (e.g. deleted `services/atlas-fame/atlas.com/fame/rest/handler.go:56-68`, pre-diff). Shared `server.RegisterHandler`: `RetrieveSpan → ParseEnvironment → ParseTenant` (`libs/atlas-rest/server/register.go:11-24`). `ParseEnvironment` short-circuits to a pass-through on an absent `ENVIRONMENT` header (`libs/atlas-rest/server/handler.go:71`, doc comment lines 34-36), so untagged traffic is byte-identical; this delta is explicitly named and accepted as FR-4.1 in `docs/tasks/task-301-rest-handler-scaffold-alias/prd.md:195-205` and `design.md:318-341`, with the shared lib's own tests already covering it (`libs/atlas-rest/server/handler_test.go`, unchanged). Not a silent seam. |
| — (behavior delta, disclosed) | Malformed-body 400 now carries a JSON:API errors document instead of a bare status code | PASS (documented, intentional) | Old local `ParseInput` wrote a bare `w.WriteHeader(http.StatusBadRequest)` (e.g. deleted `services/atlas-account/atlas.com/account/rest/handler.go`, pre-diff, lines 12-28 of the diff hunk). Shared `server.ParseInput` (`libs/atlas-rest/server/context.go:47-66`) calls `writeBadRequestJSONAPI`, which emits a `{"errors":[...]}` document (`libs/atlas-rest/server/error.go:79-101`). Named explicitly in `plan.md:1985` ("two intended behavior deltas"). |
| — (curried-signature change) | `RegisterHandler`/`RegisterInputHandler` dropped the `db *gorm.DB` curry step present in every hand-rolled scaffold | PASS | Old: `RegisterHandler(l) func(db *gorm.DB) func(si) func(name, handler) http.HandlerFunc` (deleted hunk, `services/atlas-account/.../rest/handler.go`). New: `server.RegisterHandler(l) func(si) func(name, handler) http.HandlerFunc` (`libs/atlas-rest/server/register.go:11-13`). Every one of the 174 call sites was updated in the same diff to move `db` into the handler-factory closure instead (see resource.go table below) — confirmed zero remaining `RegisterHandler(l)(db)(si)` / `RegisterInputHandler[...](l)(db)(si)` call shapes anywhere in the 21 services, and zero remaining `d.DB()` / `c.DB()` reads in any changed non-test file. |

### `resource.go` domain / sub-domain packages — 41 files, all touched only by the `db`-closure re-parameterization

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | No new responsibility bled into `resource.go`; no package-named catch-all introduced | PASS | Diffed all 41 changed `resource.go` files against the base commit: every hunk is either (a) the `register* := rest.RegisterHandler(l)(si)` line losing its `(db)` curry, (b) a `handleX(...)` function gaining a `func(db *gorm.DB) rest.{Get,Input}Handler[...]` wrapper layer with one extra level of closure indentation, or (c) `d.DB()` replaced by the closed-over `db`. Zero new `func Transform`, `type RestModel`, `Migration`, or `TableName` declarations were added by the diff (`grep -E '^\+func Transform|^\+type RestModel|^\+func.*Migration|^\+func.*TableName'` across all 41 files' diffs: no matches). Representative: `services/atlas-npc-shops/atlas.com/npc/shops/resource.go` (446-line diff, `handleGetShop`/`handleAddCommodity`/`handleUpdateCommodity`/`handleRemoveCommodity`/`handleDeleteAllCommodities`/`handleDeleteAllShops`/`handleGetShopCharacters` all converted identically). |
| DOM-09 | Every `Transform(` call site checks its error | PASS | `grep -n '_, _ :=.*Transform\|_ = .*Transform('` across all 41 changed `resource.go` files: zero matches. |
| DOM-12 | No `os.Getenv()` in handlers | PASS | `grep -n os.Getenv` across all 41 diffs: zero matches (none added; none in the current file content among the diffed hunks). |
| DOM-15 | No `db.Create`/`db.Save`/`db.Delete` in handlers | PASS | `grep -n '^\+.*db\.Create\|^\+.*db\.Save\|^\+.*db\.Delete'` across all 41 diffs: zero matches added. |
| DOM-30 | Writes emit via `AndEmit` + `message.Buffer`, not a bare `producer.ProviderImpl` from the success path | N/A for this diff's lines | Every `*AndEmit(...)` call site touched by the diff (`DeleteAndEmit`, `CreateAndEmit`, `RecordPinAttemptAndEmit`, `ResolveAndEmit`, `RenameAndEmit`, etc. — `services/atlas-account/atlas.com/account/account/resource.go`, `services/atlas-notes/.../note/resource.go`, `services/atlas-pets/.../pet/resource.go`) is byte-identical in substance before/after; only the trailing `d.DB()` → `db` argument changed. This diff neither introduces nor repairs a DOM-30 violation. |
| DOM-31 | Tenant/trace identifiers travel in context only | PASS | `grep -n 'tenantId\|TENANT_ID\|TenantId'` across all 41 diffs: zero matches. No `RestModel` gained a tenant field (see FILE-01..06 evidence above); `d.Context()` is still the only tenant-carrying value threaded into `NewProcessor(...)`. |
| SUB-01..04 | Business logic outside the handler; writes via administrator; `RegisterInputHandler[T]` for POST; no manual JSON parsing | PASS (unaffected) | `grep -n 'io.ReadAll\|json.NewDecoder\|jsonapi.Unmarshal'` across all 41 diffs: zero matches added — no package regressed to manual body parsing. POST/PATCH routes continue to register through `registerInput`/`registerCreateInput`/etc. (all `RegisterInputHandler[T]`-typed) unchanged from before the diff, e.g. `services/atlas-account/atlas.com/account/account/resource.go:26-36`. |
| DOM-08 | POST/PATCH via `RegisterInputHandler[T]`, not `RegisterHandler` | PASS | Route-to-registrar mapping is unchanged by the diff (same `register`/`registerInput`/`registerCreateInput` names bound to the same HTTP methods); only the registrar *construction* lost its `(db)` curry. E.g. `services/atlas-account/atlas.com/account/account/resource.go:31` (`.Methods(http.MethodPatch)` via `registerInput`) and `:30` (`.Methods(http.MethodDelete)` via `register`). |
| — (compile-level, `d.DB()` removal) | No stray `HandlerDependency{..db..}` literal or `d.DB()`/`c.DB()` read remains in any changed non-test file | PASS | `grep -n 'd\.DB()\|c\.DB()\|HandlerDependency{.*db'` across every changed non-test `.go` file in scope: zero matches. `go build ./...` clean on all 21 modules confirms no dangling reference. |

### Runtime safety / migration hygiene / testing / cache / scaffolding / security / EXT

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-26 | Every goroutine via `routine.Go`; bare `go` needs `//goroutine-guard:allow` | N/A | Zero `go ` / `go func` statements added anywhere in the diff (`grep -n '^\+.*\bgo \|^\+\s*go func'` across all four diff groupings: no matches). |
| DOM-34 / DOM-35 | No aliases/dead symbols left behind after a `libs/atlas-*` extraction | N/A | No symbol moved into or out of a `libs/atlas-*` module in this diff — `libs/atlas-rest` has an empty diff; the 21 services newly *consume* its existing exports via `type X = server.X` aliases, which is the pattern this diff's own PRD (`prd.md`) commissions, not a DOM-34/35 extraction. |
| DOM-10/20/24/33 (Testing) | — | N/A | No `_test.go` touched; no `Processor`/`Provider`/`Administrator` interface re-signed (handler-factory functions are free functions, not interface methods). |
| DOM-29 (Cache) | — | N/A | No `cache.go` touched; no processor/struct gained cached state. |
| SCAFFOLD-01..10 | — | N/A | No `services/atlas-<svc>/` directory added; no new channel `Writer`/`Handler`; `deploy/shared/routes.conf` not in the diff. |
| SEC-01..04 | — | N/A | No JWT/token parsing, revocation, redirect, or secret-handling line is touched by this diff (the one `Password` reference in `atlas-account/account/resource.go:81` sits in an untouched hunk). |
| EXT-01..04 | — | N/A | `grep -l 'requests.RootUrl\|requests.GetRequest\|requests.PostRequest'` against every changed `resource.go` and `handler.go`: zero matches. |

## Security Review

Not applicable — the SEC-* family trigger did not fire (see Applicability table above).

## Not evaluable from the diff

- DOM-13/DOM-14/DOM-17 (no cross-domain orchestration in handlers; handlers call processors not providers; error-to-status mapping) — full line-by-line semantic review of all 41 changed `resource.go` files' pre-existing handler bodies (business logic, not the scaffolding delta) was not performed; verification here was limited to targeted greps (direct provider-package calls, `db.Create`/`Save`/`Delete`, `os.Getenv`) run against the diffed content plus spot-reads of the largest diffs (`atlas-mts/listing`, `atlas-npc-shops/shops`, `atlas-account/account`). Those greps found nothing, and the diffs themselves show only closure-wrapping (no logic lines changed), so a hidden violation would have to be one that already existed unchanged before this branch — out of this diff's own responsibility, but I did not read every one of the 41 files' full handler bodies end-to-end to positively confirm it.
- DOM-27 (transient DB errors → 503 via `server.WriteErrorResponse`) — several changed handlers write `http.StatusInternalServerError` directly (e.g. `services/atlas-mts/atlas.com/mts/testsupport/resource.go`, `services/atlas-notes/atlas.com/notes/note/resource.go`); every such line is unchanged in substance by this diff (confirmed via diff — same content before and after, only re-indented), so whether the pre-existing 500 sites should have been 503-via-`WriteErrorResponse` was not evaluated here; that would require reading the pre-branch state of those handlers, which is outside this diff's own surface.
- Exact aggregate pass/fail test counts across the 21 modules — the module-by-module `go test` runner does not print a single aggregate count; I confirmed zero `FAIL` lines and an `ok`/`[no test files]` result for every package in every module, but did not sum individual test-function counts.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- None. The two behavior deltas (`ParseEnvironment` joining the chain; JSON:API-shaped 400 on malformed bodies) are explicitly disclosed and intended per the PRD/design, and are covered by the shared library's own tests.
