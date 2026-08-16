---
title: Backend Audit Checklist
description: The authoritative index of every backend audit rule ID (DOM-*, FILE-*, SUB-*, EXT-*, SCAFFOLD-*, SEC-*), its concise definition, and what triggers it.
---

# Backend Audit Checklist

**This file is the single authoritative definition of the backend audit rule
IDs.** The `backend-guidelines-reviewer` agent owns *how* to review — mindset,
scope, evidence bar, status semantics. This file owns *what* is checked.

Adding, removing, or renumbering a rule means editing **this file and the
linked detail section** — never the agent definition. The agent must not carry
a second copy of any rule.

## How to use this file

The reviewer reads this index in full — it is small on purpose — then loads
**only** the detail documents whose family triggers fire on the review surface.
A diff that touches no REST handler does not load
[patterns-rest-jsonapi.md](patterns-rest-jsonapi.md); a diff that adds no
service does not load [scaffolding-checklist.md](scaffolding-checklist.md).

### Triggering is two-level — do not collapse it

**A family trigger decides which document to open. Each rule's own
`Applies when` decides whether that rule is evaluated.** The two are not the
same, and a family trigger is never narrower than the rules it carries: it is
the *union* of its members' triggers, so that no rule can be skipped just
because it shares a document with rules that did not apply.

Concretely: DOM-06 (`processor.go`) lives in the REST document but applies to
any package with a processor, REST or not. DOM-04/05 (`rest.go`) and DOM-11
(`provider.go`) live in the DOM-structure section but apply to any package with
those files, `model.go` or not.

So:

1. Open a family's document if **any** of its rules' triggers fire.
2. Inside an opened document, evaluate each rule against **its own** trigger.
3. A rule whose own trigger did not fire is `N/A`, with the trigger named as
   evidence — a legitimate disposition, and not the same as "not evaluable".
4. Never mark a rule `N/A` on the strength of the *family* trigger alone.

## Family index — load detail on demand

| Family | Open the document when (union of member triggers) | Detail document |
|---|---|---|
| **DOM structure** (DOM-01..05, 11, 16) | Changed package has `model.go`, `entity.go`, `rest.go`, or `provider.go` | [file-responsibilities.md](file-responsibilities.md#audit-verification--domain-structure-dom-0105-dom-11-dom-16) |
| **FILE placement** (FILE-01..06) | Any changed Go package — no exemptions | [file-responsibilities.md](file-responsibilities.md#audit-verification--file-0106) |
| **SUB sub-domain** (SUB-01..04) | Changed package has `resource.go` but no `model.go` | [file-responsibilities.md](file-responsibilities.md#audit-verification--sub-0104) |
| **REST** (DOM-06..09, 12..15, 17..19, 32) | Changed package has `resource.go`, `rest.go`, or `processor.go`, or registers HTTP routes | [patterns-rest-jsonapi.md](patterns-rest-jsonapi.md#audit-verification--rest-checks) |
| **Constants reuse** (DOM-21) | Diff declares a new type, named const block, or numeric-literal classification | [anti-patterns.md](anti-patterns.md#audit-verification--dom-21-shared-domain-types) |
| **Testing** (DOM-10, 20, 24, 33) | Diff touches a `_test.go`, changes a `Processor`/`Provider`/`Administrator` interface, or a changed package reaches an emit path | [testing-guide.md](testing-guide.md#audit-verification--dom-10-dom-20-dom-24-dom-33) |
| **Cache** (DOM-29) | Changed package has `cache.go`, or a processor/struct holds cached state | [patterns-cache.md](patterns-cache.md#audit-verification--dom-29) |
| **Messaging** (DOM-30) | Changed package has `producer.go`, or calls `AndEmit` / `message.Emit` / `producer.ProviderImpl` | [patterns-kafka.md](patterns-kafka.md#audit-verification--dom-30) |
| **Multi-tenancy** (DOM-31) | Changed package has `rest.go`, or changed code reads tenant/trace state, passes a `tenantId`, or opens a DB session (`tenant.MustFromContext`, `db.WithContext(ctx)`) | [patterns-multitenancy-context.md](patterns-multitenancy-context.md#audit-verification--dom-31) |
| **Migration hygiene** (DOM-34, 35) | Diff moves, extracts, or re-homes symbols between a service and a `libs/atlas-*` module | [anti-patterns.md](anti-patterns.md#audit-verification--dom-34-dom-35-library-migration-hygiene) |
| **Deploy & topics** (DOM-22, 23) | Diff adds a `libs/atlas-*` module, or adds/renames a Kafka topic env var | [patterns-deploy.md](patterns-deploy.md) |
| **Runtime safety** (DOM-26) | Any non-test Go file changed | [anti-patterns.md](anti-patterns.md#audit-verification--dom-26-goroutines) |
| **Channel wire values** (DOM-25) | Diff touches `services/atlas-channel` or `libs/atlas-packet`, or a domain service emits a byte a client interprets | [anti-patterns.md](anti-patterns.md#audit-verification--dom-25-client-interpreted-wire-values) |
| **Resilience** (DOM-27, 28) | DB-backed service handlers, or `model.Decorator` / enrichment paths changed | [patterns-resilience.md](patterns-resilience.md#audit-verification--dom-27-dom-28) |
| **External clients** (EXT-01..04) | Changed package calls `requests.RootUrl` / `requests.GetRequest[T]` / `requests.PostRequest[T]` for another atlas service | [cross-service-implementation.md](cross-service-implementation.md#audit-verification--ext-0104) |
| **Scaffolding** (SCAFFOLD-01..09) | Diff adds a `services/atlas-<svc>/` directory, registers a new atlas-channel `Writer`/`Handler`, or changes `deploy/shared/routes.conf` | [scaffolding-checklist.md](scaffolding-checklist.md#audit-verification--scaffold-0109) |
| **Security** (SEC-01..04) | Service handles authentication, authorization, tokens, redirects, or secrets | [patterns-security.md](patterns-security.md) |

## Foundational guidelines — no numbered rules

These two documents carry general architectural guidance that is deliberately
*not* mechanised into an audit rule: it describes how the code is shaped, not a
condition a reviewer can grade PASS/FAIL from a diff. They are the guideline of
record when a finding needs one and no numbered rule covers it, and they may be
cited as the documented exception that exempts a deviation. Open the ones whose
subject the diff touches; they are not subject to the "if it is not in the
checklist it does not exist" clause, which governs *numbered rules* only.

| Document | Open when |
|---|---|
| [patterns-provider.md](patterns-provider.md) | Changed code defines or composes providers |
| [patterns-functional.md](patterns-functional.md) | Changed code defines curried constructors, decorators, or model combinators |

**No document is loaded unconditionally.** This index is the only thing a
review reads before classifying the surface; every other document — foundational
ones included — waits for its trigger. If you find yourself wanting a document
loaded for "any Go diff", that is a sign its enforceable content belongs in the
tables below as a numbered rule with its own `Applies when`.

---

## DOM-* — domain package rules

| ID | Rule | Applies when |
|---|---|---|
| DOM-01 | `builder.go` exists with `NewBuilder()`, fluent setters, and a validating `Build()` | package has `model.go` |
| DOM-02 | `Model.ToEntity()` defined in `entity.go` | package has `model.go` + `entity.go` |
| DOM-03 | `Make(Entity) (Model, error)` defined in `entity.go` | package has `model.go` + `entity.go` |
| DOM-04 | `Transform(Model) (RestModel, error)` defined in `rest.go` | package has `rest.go` |
| DOM-05 | `TransformSlice` defined in `rest.go` and used by list handlers — no inline loops in `resource.go` | package has `rest.go` |
| DOM-06 | Processor constructor takes `logrus.FieldLogger`, never `*logrus.Logger` | package has `processor.go` |
| DOM-07 | Handlers pass `d.Logger()` into `NewProcessor`, never `logrus.StandardLogger()` | package has `resource.go` |
| DOM-08 | POST/PATCH routes register via `RegisterInputHandler[T]`, not `RegisterHandler` | package registers POST/PATCH routes |
| DOM-09 | Every `Transform(` call site checks its error — no `_, _ :=` / `_ =` | `resource.go` calls `Transform` |
| DOM-10 | Test DB setup calls `database.RegisterTenantCallbacks(l, db)` | tests open a GORM DB directly |
| DOM-11 | Providers evaluate lazily via `database.Query` / `database.SliceQuery`, not eager reads wrapped in `FixedProvider` | package has `provider.go` |
| DOM-12 | No `os.Getenv()` in handlers | package has `resource.go` |
| DOM-13 | Handlers contain no cross-domain orchestration — that belongs in the processor layer | package has `resource.go` |
| DOM-14 | Handlers call processor methods, never provider functions directly | package has `resource.go` |
| DOM-15 | No `db.Create` / `db.Save` / `db.Delete` in handlers — writes go processor → administrator | package has `resource.go` |
| DOM-16 | `administrator.go` holds the write functions for a domain with create/update/delete | package has `model.go` and performs writes |
| DOM-17 | Domain errors map to specific HTTP status: validation → 400, not-found → 404, conflict → 409 | package has `resource.go` |
| DOM-18 | REST models implement the JSON:API interface (`GetName()`, `GetID()`, `SetID()`) | package has `rest.go` |
| DOM-19 | Request models are flat — no nested `Data`/`Type`/`Attributes` structs | package has `rest.go` |
| DOM-20 | Tests are table-driven (`tests := []struct{...}` + `t.Run`) | diff adds or changes tests |
| DOM-21 | No redeclaration of a type, helper, or numeric constant that already exists in `libs/atlas-constants/` | diff declares a new type, const block, or numeric-literal classification |
| DOM-22 | A new `libs/atlas-*` module is wired into the shared root `Dockerfile` and `go.work` | diff adds a module under `libs/` |
| DOM-23 | Kafka topic env vars follow `COMMAND_TOPIC_*` / `EVENT_TOPIC_*`, live in the base env-configmap as `KEY: "KEY"`, are re-listed in **both** overlays' `atlas-env` generator (which uses `behavior: replace` — a base key an overlay omits is absent at runtime in that environment), and are never redeclared as literal `env:` values in a service manifest | diff adds or renames a topic env var |
| DOM-24 | Test packages that reach an emit path install the shared `producertest` stub (or inject a no-op producer per test) | a changed test package reaches `AndEmit` / `message.Emit` / `producer.Produce`, directly or transitively |
| DOM-25 | Bytes the client interprets (dispatcher modes, sub-op codes, message/fail-reason codes) are resolved from a tenant writer-options table, never written as Go literals; domain services emit semantic keys, not client bytes | diff touches channel/socket/packet code, or a domain service event carries a client-interpreted byte |
| DOM-26 | Every goroutine is spawned via `routine.Go(l, ctx, fn)`; a bare `go` statement needs a justified `//goroutine-guard:allow` marker | any non-test Go file changed |
| DOM-27 | In DB-backed services, handler error branches use `server.WriteErrorResponse(...)` so transient DB errors surface as 503, not a bare 500 | changed handler writes `http.StatusInternalServerError` and the service calls `database.Connect` |
| DOM-28 | Fallible enrichment/decorator paths degrade loudly — `model.ErrDecorator` + `degrade.Observe(...)` — never `if err != nil { return m }` | diff changes a `model.Decorator` or an enrichment fallback that fetches remote data |
| DOM-29 | Caches are application-scoped singletons reached through a `GetCache()` accessor — never constructed in a processor constructor or held as per-instance processor state | package has `cache.go`, or a processor/struct holds cached state |
| DOM-30 | Business operations emit through the `AndEmit` + `message.Buffer` pattern, so the write and its events stay atomic — not a direct `producer.ProviderImpl(...)` call from the operation path | changed package emits Kafka messages |
| DOM-31 | Tenant and trace identifiers travel in context only — never a field on a REST model, a request body, or a public path/query parameter | package has `rest.go`, or changed code reads or passes tenant/trace state |
| DOM-32 | Routes register through `server.RegisterHandler` / `server.RegisterInputHandler[T]` — no bare `http.HandlerFunc` route bodies, no manual tenant-header parsing, no custom error-response helpers | package registers HTTP routes |
| DOM-33 | An interface change updates every mock implementation of that interface in the same diff | diff adds, removes, or re-signs a method on a `Processor` / `Provider` / `Administrator` interface |
| DOM-34 | No type aliases, re-exports, or delegating wrappers left behind when a symbol moves to a shared library — every call site imports the new home directly | diff moves or extracts symbols between a service and a `libs/atlas-*` module |
| DOM-35 | Symbols the extraction left unreferenced are deleted — no dead constants, structs, functions, imports, or variables | same |

## FILE-* — file responsibilities

Runs on **every** changed Go package — domain, sub-domain, and support alike.
A REST-client or reader package with no `model.go` is not exempt; that is
exactly where collapsed-file violations hide.

| ID | Rule | Applies when |
|---|---|---|
| FILE-01 | `Processor` interface, constructor, and `ProcessorImpl` methods live in `processor.go` or a `processor_<group>.go` split | any changed Go package |
| FILE-02 | `RestModel`, `Transform`/`Extract`, and the JSON:API methods live in `rest.go` | any changed Go package |
| FILE-03 | Cross-service request functions live in `requests.go` | any changed Go package |
| FILE-04 | Entity struct, `Migration`, and `TableName` live in `entity.go` | any changed Go package |
| FILE-05 | Builder in `builder.go`, domain `Model` in `model.go`, writes in `administrator.go`, readers in `provider.go`, enums in `state.go` | any changed Go package |
| FILE-06 | No package-named catch-all file carrying ≥2 of the responsibilities above | any changed Go package |

## SUB-* — sub-domain (action-event) packages

| ID | Rule | Applies when |
|---|---|---|
| SUB-01 | Business logic lives in the package's own processor or the parent's, never in the handler | package has `resource.go`, no `model.go` |
| SUB-02 | Writes go through an administrator (own or parent) — no `db.Create` / `db.Save` in `resource.go` | same |
| SUB-03 | POST endpoints use `RegisterInputHandler[T]` | same, and registers POST |
| SUB-04 | No manual JSON parsing — no `json.NewDecoder`, `json.Unmarshal`, `io.ReadAll` in `resource.go` | same |

## EXT-* — external atlas-service HTTP clients

| ID | Rule | Applies when |
|---|---|---|
| EXT-01 | The target REST model implements `SetToOneReferenceID` and `SetToManyReferenceIDs` (even as no-ops) | package calls another atlas service via `requests.*Request[T]` |
| EXT-02 | An `httptest`-backed integration test serves a representative JSON:API fixture and asserts a populated domain struct — `FakeClient` mocks alone do not satisfy this | same |
| EXT-03 | Only genuine 404s map to a domain "not found"; transport/decode/5xx failures bubble up with their original error | same |
| EXT-04 | Service URL composed via `requests.RootUrl(<DOMAIN>)`, not hardcoded DNS | same |

## SCAFFOLD-* — new service / new channel feature

| ID | Rule | Applies when |
|---|---|---|
| SCAFFOLD-01 | `.github/config/services.json` has a `go-service` entry for the service | diff adds `services/atlas-<svc>/` |
| SCAFFOLD-02 | Kubernetes base manifest exists and is listed in the base `kustomization.yaml` resources | same |
| SCAFFOLD-03 | Build registration complete: `docker-bake.hcl` `go_services` entry and a `go.work` `use()` entry (there is no per-service Dockerfile) | same |
| SCAFFOLD-04 | Ingress location block present in `deploy/shared/routes.conf` (REST services only) | same, service exposes REST |
| SCAFFOLD-05 | Generated routes template regenerated from the shared source and committed | `deploy/shared/routes.conf` changed |
| SCAFFOLD-06 | docker-compose entry present alongside peers | diff adds `services/atlas-<svc>/` |
| SCAFFOLD-07 | New atlas-channel `Writer` / `Handler` constants are seeded in every targeted tenant opcode template | diff registers a new channel writer/handler |
| SCAFFOLD-08 | Bruno collection present (REST services only) | diff adds `services/atlas-<svc>/` with REST |
| SCAFFOLD-09 | Overlay enumerations, `ATLAS_DB_NAMES`, and DB bootstrap complete — machine-checked by `tools/service-registration-guard.sh` | diff adds `services/atlas-<svc>/` |

## SEC-* — security (auth-related services)

| ID | Rule | Applies when |
|---|---|---|
| SEC-01 | JWT parsing is verified — no `ParseUnverified` on a trust path | service handles tokens |
| SEC-02 | Revocation/logout reads claims only from a validated token | service handles tokens |
| SEC-03 | Redirect targets are validated — no open redirect | service has callback/redirect handlers |
| SEC-04 | No hardcoded keys, passwords, or secrets in source | service handles secrets |

---

## Rule ownership

| Concern | Owner |
|---|---|
| Rule IDs, definitions, triggers | this file |
| Verification procedure, examples, rationale for a rule | the linked detail document |
| Reviewer mindset, scope, evidence bar, phases, status semantics, artifacts | `.claude/agents/backend-guidelines-reviewer.md` |

A rule appears in exactly one place in this table's first column. If you find a
second copy of a rule's definition anywhere, delete it and link here.
