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

Each row's **Applies when** column is the trigger. If a trigger does not fire,
the rule is `N/A` with the trigger named as evidence — that is a legitimate
disposition and is not the same as "not evaluable".

## Family index — load detail on demand

| Family | Applies when | Detail document |
|---|---|---|
| **DOM structure** (DOM-01..05, 11, 16) | Changed package has `model.go` | [file-responsibilities.md](file-responsibilities.md#audit-verification--domain-structure-dom-0105-dom-11-dom-16) |
| **FILE placement** (FILE-01..06) | Any changed Go package — no exemptions | [file-responsibilities.md](file-responsibilities.md#audit-verification--file-0106) |
| **SUB sub-domain** (SUB-01..04) | Changed package has `resource.go` but no `model.go` | [file-responsibilities.md](file-responsibilities.md#audit-verification--sub-0104) |
| **REST** (DOM-06..09, 12..15, 17..19) | Changed package has `resource.go` or `rest.go`, or registers HTTP routes | [patterns-rest-jsonapi.md](patterns-rest-jsonapi.md#audit-verification--rest-checks) |
| **Constants reuse** (DOM-21) | Diff declares a new type, named const block, or numeric-literal classification | [anti-patterns.md](anti-patterns.md#audit-verification--dom-21-shared-domain-types) |
| **Testing** (DOM-10, 20, 24) | Diff touches a `_test.go`, or a changed package reaches an emit path | [testing-guide.md](testing-guide.md#audit-verification--dom-10-dom-20-dom-24) |
| **Deploy & topics** (DOM-22, 23) | Diff adds a `libs/atlas-*` module, or adds/renames a Kafka topic env var | [patterns-deploy.md](patterns-deploy.md) |
| **Runtime safety** (DOM-26) | Any non-test Go file changed | [anti-patterns.md](anti-patterns.md#audit-verification--dom-26-goroutines) |
| **Channel wire values** (DOM-25) | Diff touches `services/atlas-channel` or `libs/atlas-packet`, or a domain service emits a byte a client interprets | [anti-patterns.md](anti-patterns.md#anti-pattern-hardcoding-client-interpreted-wire-values) |
| **Resilience** (DOM-27, 28) | DB-backed service handlers, or `model.Decorator` / enrichment paths changed | [patterns-resilience.md](patterns-resilience.md#audit-verification--dom-27-dom-28) |
| **External clients** (EXT-01..04) | Changed package calls `requests.RootUrl` / `requests.GetRequest[T]` / `requests.PostRequest[T]` for another atlas service | [cross-service-implementation.md](cross-service-implementation.md#audit-verification--ext-0104) |
| **Scaffolding** (SCAFFOLD-01..09) | Diff adds a `services/atlas-<svc>/` directory, or registers a new atlas-channel `Writer`/`Handler` | [scaffolding-checklist.md](scaffolding-checklist.md#audit-verification--scaffold-0109) |
| **Security** (SEC-01..04) | Service handles authentication, authorization, tokens, redirects, or secrets | [patterns-security.md](patterns-security.md) |

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
| DOM-23 | Kafka topic env vars follow `COMMAND_TOPIC_*` / `EVENT_TOPIC_*`, live in the base env-configmap as `KEY: "KEY"`, and are never redeclared as literal `env:` values in a service manifest | diff adds or renames a topic env var |
| DOM-24 | Test packages that reach an emit path install the shared `producertest` stub (or inject a no-op producer per test) | a changed test package reaches `AndEmit` / `message.Emit` / `producer.Produce`, directly or transitively |
| DOM-25 | Bytes the client interprets (dispatcher modes, sub-op codes, message/fail-reason codes) are resolved from a tenant writer-options table, never written as Go literals; domain services emit semantic keys, not client bytes | diff touches channel/socket/packet code, or a domain service event carries a client-interpreted byte |
| DOM-26 | Every goroutine is spawned via `routine.Go(l, ctx, fn)`; a bare `go` statement needs a justified `//goroutine-guard:allow` marker | any non-test Go file changed |
| DOM-27 | In DB-backed services, handler error branches use `server.WriteErrorResponse(...)` so transient DB errors surface as 503, not a bare 500 | changed handler writes `http.StatusInternalServerError` and the service calls `database.Connect` |
| DOM-28 | Fallible enrichment/decorator paths degrade loudly — `model.ErrDecorator` + `degrade.Observe(...)` — never `if err != nil { return m }` | diff changes a `model.Decorator` or an enrichment fallback that fetches remote data |

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
