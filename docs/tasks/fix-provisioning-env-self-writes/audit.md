# Backend Audit — fix-provisioning-env-self-writes (commit eb3818018)

- **Service Path:** `libs/atlas-env`, `libs/atlas-rest/server` (shared libraries, not a service)
- **Range audited:** `04f37fa95..eb3818018`
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-19
- **Build:** PASS
- **Tests:** PASS (all packages), 0 failed
- **Overall:** NEEDS-WORK

## Build & Test Results

```
$ cd libs/atlas-env && go build ./... && go test ./... -count=1
ok  	github.com/Chronicle20/atlas/libs/atlas-env	0.003s

$ cd libs/atlas-rest && go build ./... && go test ./... -count=1
ok  	github.com/Chronicle20/atlas/libs/atlas-rest/degrade	0.004s
?   	github.com/Chronicle20/atlas/libs/atlas-rest/examples	[no test files]
ok  	github.com/Chronicle20/atlas/libs/atlas-rest/requests	10.809s
?   	github.com/Chronicle20/atlas/libs/atlas-rest/retry	[no test files]
ok  	github.com/Chronicle20/atlas/libs/atlas-rest/server	0.008s
ok  	github.com/Chronicle20/atlas/libs/atlas-rest/server/paginate	0.005s

# Corroborating: consumers of the changed env.Registry interface also build/test clean
$ cd libs/atlas-service && go build ./... && go test ./... -count=1
ok  	github.com/Chronicle20/atlas/libs/atlas-service	1.051s

$ cd libs/atlas-kafka && go build ./...
(clean, no output)
```

## Applicability

| Family | Fired? | Evidence |
|---|---|---|
| DOM structure (DOM-01..05,11,16) | N/A | Neither `libs/atlas-env` nor `libs/atlas-rest/server` contains `model.go`, `entity.go`, `rest.go`, or `provider.go` (`ls` of both directories, no such files). |
| FILE placement (FILE-01..06) | Fired | Every changed Go package runs this family unconditionally. |
| SUB sub-domain (SUB-01..04) | N/A | Neither package has a `resource.go`. |
| REST (DOM-06..09,12..15,17..19,32) | N/A | Neither package has `resource.go`, `rest.go`, or `processor.go`. `libs/atlas-rest/server` *defines* `RegisterHandler`/`RegisterInputHandler` (in unchanged `register.go`) but DOM-32's own verification procedure (`patterns-rest-jsonapi.md:474`) scopes the check to a service's `resource.go` route lines that call the wrapper — there is no such call site in this diff. |
| Constants reuse (DOM-21) | N/A | Diff declares no new type, const block, or numeric-literal classification — `Record.Provisionable()` and `Registry.IsProvisionable` reuse the existing `PhaseProvisioning`/`PhaseActive` constants (`libs/atlas-env/record.go:5-8`, unchanged by this diff). |
| Testing (DOM-10,20,24,33) | Fired | Diff touches `registry_test.go` and `handler_test.go`, and re-signs the `Registry` interface (adds `IsProvisionable`). |
| Cache (DOM-29) | N/A | No `cache.go`, no cached processor/struct state touched. |
| Messaging (DOM-30) | N/A | No `producer.go`, no `AndEmit`/`message.Emit`/`producer.ProviderImpl` call site in the diff. |
| Multi-tenancy (DOM-31) | Fired (narrow) | `libs/atlas-rest/server/handler.go` is a changed file that carries tenant/environment context-handling code (`ParseTenant`, `ParseEnvironment`) — see Checklist Results. |
| Migration hygiene (DOM-34,35) | N/A | Diff adds new methods in place; it moves no symbol between a service and a `libs/atlas-*` module. |
| Deploy & topics (DOM-22,23) | N/A | No new `libs/atlas-*` module added, no Kafka topic env var touched. |
| Runtime safety (DOM-26) | Fired (trivially clean) | Every non-test Go file changed triggers this; no `go ` statement appears in any diff hunk (`grep -n "^\s*go " record.go registry.go handler.go` → no matches). |
| Channel wire values (DOM-25) | N/A | Diff touches neither `services/atlas-channel` nor `libs/atlas-packet`, and emits no client-interpreted byte. |
| Resilience (DOM-27,28) | N/A | No DB-backed handler writes `http.StatusInternalServerError` in the diff, and no `model.Decorator`/enrichment path changed. |
| External clients (EXT-01..04) | N/A | Diff calls no other atlas service via `requests.*Request[T]`. |
| Scaffolding (SCAFFOLD-01..09) | N/A | Diff adds no `services/atlas-<svc>/` directory, no channel writer/handler, no `routes.conf` change. |
| Security (SEC-01..04) | N/A | Neither `libs/atlas-env` nor `libs/atlas-rest/server` is an auth-related service handling tokens, OAuth callbacks, or secrets (`patterns-security.md` scopes the family to "services that handle authentication, authorization, token issuance or revocation, OAuth-style callbacks, or secrets" — `atlas-login`/`atlas-account` are the named subjects). Widening the REST env gate is a scope/authorization-adjacent change, but it is not a token/redirect/secret surface, so no `SEC-*` rule's own trigger fires. Flagged explicitly rather than silently skipped, per the task instruction to pay particular attention here. |
| patterns-provider.md (foundational) | N/A | Diff defines no provider, composes no provider chain. |
| patterns-functional.md (foundational) | N/A | Diff defines no curried constructor, decorator, or model combinator. |

## Checklist Results

### libs/atlas-env (support package — no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-05 | Builder/model/writes/readers/enums placed per file table | PASS | `record.go:17-38` holds only the `Record` model struct and two boolean predicate methods (`Active`, `Provisionable`) — no builder, write, or reader logic added or misplaced. `registry.go:15-99` (interface) and `179-193` (impl) hold only registry read logic — no writes. |
| FILE-06 | No catch-all file carrying ≥2 responsibilities | PASS | `record.go` carries only the `Record` model + its own predicates (a single responsibility from the FILE-01..05 table); `registry.go` carries only `Registry`/`MapRegistry`/`legacyRegistry` (a single responsibility). Neither collapses Processor+RestModel+requests as `wallet.go` did (task-102). |
| DOM-20 | Tests use `tests := []struct{...}` + `t.Run` | FAIL (Important) | `libs/atlas-env/registry_test.go:30-45` (`TestRecordProvisionableAcrossPhases`) declares `cases := []struct{ phase string; want bool }{...}` and iterates with a bare `for _, c := range cases` loop calling `t.Errorf` — it never calls `t.Run`. Per-case subtests are required by DOM-20's own pass criteria (`testing-guide.md:322`); without `t.Run`, a failing case reports only "Provisionable() phase=... = ..., want ..." with no subtest name to `go test -run`, and the whole function aborts on the first per-case check only if `t.Fatal` were used (it isn't, so this one specifically continues, but the missing `t.Run` is still the documented shape violation). |
| DOM-33 | Every mock of a changed interface updated in the same diff | PASS | `env.Registry` gained `IsProvisionable(e Id) bool` (`registry.go:31`). Its only two implementations in the repo are `*MapRegistry` (`registry.go:189-198`, added in this diff) and `legacyRegistry` (`registry.go:271`, added in this diff — `func (legacyRegistry) IsProvisionable(Id) bool { return true }`). `grep -rln "IsActive" --include='*.go' .` across the whole repo turns up no third hand-rolled `env.Registry` implementation (`libs/atlas-service/envregistry_test.go` only calls `IsActive`/`IsOwner` on a real `*MapRegistry`, it does not implement the interface itself). Corroborated by `go build ./...` passing clean in both `libs/atlas-service` and `libs/atlas-kafka`, the two other modules that consume `env.Registry`. |
| DOM-21 | No redeclaration of an existing constant/type | N/A | `Provisionable()`/`IsProvisionable` reuse `PhaseProvisioning`/`PhaseActive`, declared before this diff (`record.go:5-6`) — no new const/type in the diff. |

### libs/atlas-rest/server (support package — no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-06 | No catch-all file carrying ≥2 responsibilities | PASS | `handler.go` (changed) holds only three middleware functions (`RetrieveSpan`, `ParseEnvironment`, `ParseTenant`) — none of the FILE-01..05 responsibility types (Processor/RestModel/requests/entity/builder-model-admin-provider). No new responsibility was folded in by this diff. |
| DOM-31 | Tenant/trace identifiers travel in context only | PASS | The diff's own change is confined to the environment-id gate: `handler.go:51` reads `id := env.Id(r.Header.Get(env.Key))` (pre-existing line, unchanged) and, on passing the new `IsProvisionable` gate, `handler.go:60` puts it on the context via `env.WithContext(ctx, id)` — never onto a `RestModel`, request body, or exposed elsewhere. `ParseTenant` (unchanged by this diff, `handler.go:67-140`) likewise only ever writes into `tenant.WithContext` (`handler.go:130`). No REST model or query parameter carrying tenant/env identity was added. |
| DOM-20 | Tests use `tests := []struct{...}` + `t.Run` | PASS | The three new tests (`handler_test.go`, `TestParseEnvironmentAdmitsAProvisioningEnvironment`, `TestParseEnvironmentRejectsADeactivatingEnvironment`, `TestParseEnvironmentRejectsADeletedEnvironment`) are each a single-scenario test, not a multi-case table — DOM-20's table-driven requirement only binds when a `[]struct{...}` shape is used (as `registry_test.go` does, and fails to follow through with `t.Run`). No table was declared here, so there is nothing to run as subtests. |
| DOM-33 | Mocks updated for interface change | N/A (already covered) | `env.Registry`'s only implementations are audited under `libs/atlas-env` above; `libs/atlas-rest/server` only calls `env.CurrentRegistry().IsProvisionable(id)` (`handler.go:51`), it does not implement the interface. |

## Not evaluable from the diff

- Whether every REST service's downstream `scope.AuthorizeWrite`/`scope.Strict` layer (referenced by `handler.go:43-44`'s comment as the actual confinement mechanism for a PROVISIONING-phase caller) correctly rejects a PROVISIONING environment writing outside its own rows — this diff does not touch that layer (confirmed present at, e.g., `services/atlas-configurations/atlas.com/configurations/templates/administrator.go:29`), and auditing it would mean reading every consuming service's scope-layer call sites, which is outside this diff's changed-file surface.
- Whether `atlas-pr-bootstrap`'s service-config self-write flow (named in `handler.go:41-42`'s comment as the motivating caller) actually exercises the PROVISIONING path end-to-end — no `atlas-pr-bootstrap` file appears in this diff's changed-file list.

## Summary

### Blocking (must fix)
- DOM-20: `libs/atlas-env/registry_test.go:30-45` — `TestRecordProvisionableAcrossPhases` builds a `cases := []struct{...}` table but iterates with a bare loop instead of `t.Run(...)` per case, violating DOM-20's table-driven pattern.

### Non-Blocking (should fix)
- (none)
