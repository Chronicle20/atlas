# Backend Audit — task-288-tenant-status-topic-token (commit b2d368757)

- **Service Path:** services/atlas-tenants/atlas.com/tenants (+ tools/topicguard)
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-09-02
- **Build:** PASS
- **Tests:** all passed (0 failed)
- **Overall:** NEEDS-WORK

## Build & Test Results

```
$ cd services/atlas-tenants/atlas.com/tenants && go build ./... && go test ./... -count=1
ok  	atlas-tenants	0.013s
ok  	atlas-tenants/configuration	0.091s
?   	atlas-tenants/configuration/mock	[no test files]
ok  	atlas-tenants/configuration/seed	0.060s
?   	atlas-tenants/kafka/consumer	[no test files]
?   	atlas-tenants/kafka/message	[no test files]
?   	atlas-tenants/rest	[no test files]
?   	atlas-tenants/scope	[no test files]
ok  	atlas-tenants/tenant	0.057s
?   	atlas-tenants/tenant/mock	[no test files]
?   	atlas-tenants/test	[no test files]

$ cd tools/topicguard && GOWORK=off go build ./... && GOWORK=off go test ./... -count=1
ok  	github.com/Chronicle20/atlas/tools/topicguard	0.708s
  (TestAnalyzer, TestTokenNotInManifest, TestLoadManifest,
   TestParseAllowlistRequiresReason, TestAllowlistEntriesHaveReasons all PASS)
```

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure (DOM-01..05,11,16) | Fired (package-level), but only rules whose own trigger references an actually-changed file are settled here | `tenant` package has `model.go`/`entity.go`/`rest.go`/`provider.go` on disk, none touched by this diff (see Not evaluable) |
| FILE placement (FILE-01..06) | Fired | Changed package `tenant` (kafka.go/kafka_test.go/testmain_test.go), and `tools/topicguard` (analyzer.go) |
| SUB sub-domain (SUB-01..04) | N/A | `tenant` has `model.go` — not a sub-domain package |
| REST (DOM-06..09,12..15,17..19,32) | Fired at package level only | `tenant` package has `resource.go`/`rest.go`/`processor.go`, none touched — see Not evaluable |
| Constants reuse (DOM-21) | Fired | New named const `EventTopicTenantStatus topic.Token` in kafka.go:19, and `resolvedTenantStatusTopic` in testmain_test.go:13 |
| Testing (DOM-10,20,24,33) | Fired | kafka_test.go (new), testmain_test.go changed |
| Cache (DOM-29) | N/A | No `cache.go`, no cached state in changed files |
| Messaging (DOM-30) | Fired | `tenant` package calls `AndEmit`/`message.Buffer.Put` and this diff changes the topic token consumed there (kafka.go:19, processor.go:88) |
| Multi-tenancy (DOM-31) | Fired (package has rest.go) | Evaluated against changed code only — see table; rest.go itself not audited (Not evaluable) |
| Migration hygiene (DOM-34,35) | N/A | No symbol moved between a service and `libs/atlas-*` |
| Deploy & topics (DOM-22,23) | Fired | Diff renames/retypes a topic env var and regenerates `libs/atlas-kafka/gen/topics.yaml` + all deploy surfaces |
| Runtime safety (DOM-26) | Fired | Non-test Go files changed: kafka.go, analyzer.go |
| Channel wire values (DOM-25) | N/A | No channel/packet code touched, no client-interpreted byte |
| Resilience (DOM-27,28) | N/A | No handler or `model.Decorator` changed |
| External clients (EXT-01..04) | N/A | No `requests.*Request[T]` call in changed files |
| Scaffolding (SCAFFOLD-01..09) | N/A | No new `services/atlas-<svc>/` directory |
| Security (SEC-01..04) | N/A | atlas-tenants is not a service handling auth/tokens/redirects/secrets, and this diff touches none |
| patterns-provider.md (foundational) | N/A | No new provider defined/composed; kafka_test.go merely invokes the pre-existing `topic.EnvProvider` |
| patterns-functional.md (foundational) | N/A | `CreateStatusEventProvider`'s curried shape is unchanged by this diff |

## Checklist Results

### tenant (domain)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | No redeclaration of a type/const/helper already in `libs/atlas-constants/` | PASS | `grep -rn "TENANT_STATUS\|TenantStatus" libs/atlas-constants/` — no hits; `EventTopicTenantStatus` is a service-local topic token, matches fleet convention (e.g. `services/atlas-dragons/atlas.com/dragons/dragon/kafka.go:12`) |
| DOM-23 | Topic env vars follow `EVENT_TOPIC_*`, `KEY:"KEY"` in base configmap, listed in both overlays, never a literal service-manifest `env:` value | PASS | `deploy/k8s/base/env-configmap.yaml:183` (`EVENT_TOPIC_TENANT_STATUS: "EVENT_TOPIC_TENANT_STATUS"`), `deploy/k8s/overlays/main/kustomization.yaml:213`, `deploy/k8s/overlays/pr/kustomization.yaml:330`, `deploy/k8s/overlays/pr-sparse/kustomization.yaml:493`; `deploy/k8s/base/atlas-tenants.yaml` has no literal `EVENT_TOPIC_TENANT_STATUS` under its `env:` block (only `envFrom`) |
| DOM-30 | Writing operation emits through `AndEmit` + `message.Buffer`, not a direct `producer.ProviderImpl` call from the success path | PASS | `services/atlas-tenants/atlas.com/tenants/tenant/processor.go:88` (`mb.Put(EventTopicTenantStatus, ...)` inside `Create`), consumed transactionally via `message.EmitWithResult(outbox.EmitProvider(...))` in `CreateAndEmit` (`processor.go:112-118`) — unchanged by this diff, still correct after the retype |
| DOM-26 | Every goroutine via `routine.Go`, no bare `go` | PASS | `GUARD_MODULES=.../services/atlas-tenants/atlas.com/tenants ./tools/goroutine-guard.sh` → exit 0, "goroutineguard: 1 module(s), 8 parallel"; no `go ` statement added in kafka.go |
| DOM-31 | Tenant/trace identifiers travel in context only | PASS | kafka.go, kafka_test.go, testmain_test.go read no tenant/trace context, pass no `tenantId`, and open no DB session |
| DOM-10 | Test DB setup calls `database.RegisterTenantCallbacks` | N/A | Neither kafka_test.go nor testmain_test.go opens a GORM DB |
| DOM-20 | Tests are table-driven (`tests := []struct{...}` + `t.Run`) | FAIL | `services/atlas-tenants/atlas.com/tenants/tenant/kafka_test.go:27` (`TestEventTopicTenantStatusIsAnEnvVarName`), `:41` (`TestEventTopicTenantStatusResolvesWhenSet`), `:59` (`TestEventTopicTenantStatusErrorsWhenUnset`) — three standalone `func Test...` bodies, none using `tests := []struct{...}` + `t.Run`. The checklist's playbook carve-out (packet byte-fixtures) does not apply here. |
| DOM-24 | Test packages reaching an emit path stub the producer via `producertest` | N/A | kafka_test.go/testmain_test.go call only `topic.EnvProvider(...)`, never `AndEmit`/`message.Emit`/`producer.Produce`, directly or transitively |
| DOM-33 | Mock updated for every changed Processor/Provider/Administrator interface method | N/A | No interface method added/removed/re-signed by this diff |
| FILE-06 | No package-named catch-all file carrying ≥2 of FILE-01..05's responsibilities | PASS | `kafka.go` carries only a topic-token constant, event-type constants, and a Kafka message-body/provider (none of which are Processor/RestModel/requests/Entity/Builder-Model-administrator-provider-state) — no FILE-01..05 responsibility is duplicated there |
| FILE-01..05 | Processor/RestModel/requests/Entity/Builder-Model-administrator-provider-state each in their own file | N/A | None of kafka.go's changed content is one of these five responsibilities |

### topicguard (support / static-analysis tool, not a service)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-26 | Every goroutine via `routine.Go`, no bare `go` | PASS | `grep -n "^\s*go \|go func" tools/topicguard/analyzer.go` — no match; `tools/` is outside `goroutine-guard.sh`'s `SCAN_ROOTS` (`services`, `libs` only), confirmed by manual grep instead |
| DOM-21 | No redeclaration of an existing shared constant | N/A | `rawEnvTopicPattern` is a regex used by diagnostic 2 only, unrelated to `libs/atlas-constants` domain constants; still referenced at `tools/topicguard/analyzer.go:244`, not dead code |
| FILE-01..06 | Domain package file responsibilities | N/A | `tools/topicguard` has no `model.go`/`resource.go`/`processor.go`/etc.; it is a `go/analysis` checker plus `testdata/` fixtures, not a domain/sub-domain package — the FILE-* vocabulary (Processor, RestModel, Entity, Builder/administrator/provider/state) does not describe this module's shape |
| SUB/EXT/SCAFFOLD/SEC | — | N/A | Not a service package, no HTTP routes, no cross-service client, no new service, no auth/secrets |

## Security Review

N/A — SEC-01..04's trigger ("service handles authentication, authorization, tokens, redirects, or secrets") did not fire. `atlas-tenants` is a tenant-CRUD service; this diff touches only a Kafka topic-token declaration, its test coverage, and a static-analysis tool. No JWT, session, redirect, or secret-handling code is in scope.

## Not evaluable from the diff

- DOM-01 (`builder.go`/`NewBuilder`/`Build()`): would need `services/atlas-tenants/atlas.com/tenants/tenant/builder.go`, not touched by this diff.
- DOM-02/DOM-03 (`entity.go` `ToEntity`/`Make`): would need `tenant/entity.go`, not touched.
- DOM-04/DOM-05 (`rest.go` `Transform`/`TransformSlice`): would need `tenant/rest.go`, not touched.
- DOM-06/DOM-07 (`processor.go`/`resource.go` logger plumbing): would need full read of `tenant/processor.go` and `tenant/resource.go` beyond the three `mb.Put` call sites already inspected for DOM-30; not touched by this diff.
- DOM-08/DOM-09/DOM-12..19/DOM-32 (REST handler conventions): would need `tenant/resource.go`, not touched.
- DOM-11 (`provider.go` lazy evaluation): would need `tenant/provider.go`, not touched.
- DOM-16 (`administrator.go` write functions): would need `tenant/administrator.go`, not touched.
- DOM-31, rest.go's own no-tenant-in-payload compliance: would need `tenant/rest.go`'s `RestModel` fields, not touched by this diff (the changed-code portion of DOM-31 was evaluated directly and passes).

## Summary

### Blocking (must fix)
- DOM-20: `services/atlas-tenants/atlas.com/tenants/tenant/kafka_test.go:27,41,59` — three new test functions are not table-driven (`tests := []struct{...}` + `t.Run`), violating the checklist's literal pass criterion for diffs that add tests.

### Non-Blocking (should fix)
- (none)
