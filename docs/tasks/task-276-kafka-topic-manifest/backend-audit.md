# Backend Audit — task-276-kafka-topic-manifest

- **Service Path:** repo-wide (branch diff `dd00cd7a5..6246f8fc9`, 923 files)
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-30
- **Build:** PASS (module-local, sampled — see below; a repo-wide `tools/verify.sh` was running concurrently and was not re-run here)
- **Tests:** PASS on every module built (see Build & Test Results)
- **Overall:** NEEDS-WORK

## Build & Test Results

Per the task brief, `tools/verify.sh` was not run (a repo-wide run was in flight
concurrently). Module-local `go build`/`go test` were run against every
package the brief called out as highest-risk, plus four sampled service
modules touched by the mechanical sweep:

| Module | Build | Test |
|---|---|---|
| `libs/atlas-kafka` | PASS | `ok` all packages (consumer, consumergroup, message, producer, retry, topic) |
| `libs/atlas-kafka/gen` (`GOWORK=off`) | PASS | `ok` |
| `libs/atlas-outbox` | PASS | `ok` |
| `libs/atlas-service` | PASS | `ok` |
| `services/atlas-kafka-precreate` | PASS | `ok` all packages (discover, groups, kafkaops, manifest, topics) |
| `tools/topicguard` (`GOWORK=off`) | PASS | `ok` |
| `services/atlas-consumables/atlas.com/consumables` | PASS (build only) | not run |
| `services/atlas-merchant/atlas.com/merchant` | PASS (build only) | not run |
| `services/atlas-channel/atlas.com/channel` | PASS (build only) | not run |
| `services/atlas-character/atlas.com/character` | PASS (build only) | not run |
| `./tools/gen-topics.sh --check` | PASS — "OK: generated files are up to date (159 topics)" | n/a |

All of the above passed. The overall status is NEEDS-WORK because of a
functional regression found by hand in the mechanical sweep (below), which a
green build cannot see.

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| FILE placement (FILE-01..06) | Fired | Every changed Go package runs this family unconditionally |
| DOM structure (DOM-01..05,11,16) | Fired (narrow) | `libs/atlas-kafka/topic/provider.go` and `libs/atlas-outbox/provider.go` are changed `provider.go` files (DOM-11 applies to any package with `provider.go`, `model.go` or not, per checklist lines 32-35). No changed package in the sampled surface has `model.go`, so DOM-01/02/03/16 are N/A. |
| SUB sub-domain (SUB-01..04) | N/A | No changed package in the sampled/high-risk surface has `resource.go` with no `model.go` |
| REST (DOM-06..09,12..15,17..19,32) | N/A | No changed package in the reviewed surface has `resource.go`/`rest.go`, and no new HTTP route registration was touched |
| Constants reuse (DOM-21) | Fired | New `topic.Token` type declared (`libs/atlas-kafka/topic/token.go:14`) — checked against `libs/atlas-constants` for a colliding `Token` type; the only pre-existing `Token` symbols found (`libs/atlas-constants/inventory/constants.go`, `libs/atlas-constants/gen/*`) are unrelated (item/currency tokens, not topic names) |
| Testing (DOM-10,20,24,33) | Fired | Diff touches 142 `_test.go` files and retypes the `producer.Manager.Writer` / `Provider` signatures. Sampled `libs/atlas-kafka/producer/producertest` (unaffected — operates on resolved topic-name strings, not `topic.Token`) and `libs/atlas-kafka/gen` tests; all pass |
| Cache (DOM-29) | N/A | No `cache.go` touched; no processor/struct gains new cached state in the reviewed surface |
| Messaging (DOM-30) | Fired | `producer.go`, `producer/provider.go` changed, and hundreds of `AndEmit`/`producer.ProviderImpl` call sites retyped. See Blocking finding below for the one call site the retype broke |
| Multi-tenancy (DOM-31) | Fired | `libs/atlas-outbox/bridge.go` reads tenant/trace state via `TenantHeaderDecorator`/`headerMap` |
| Migration hygiene (DOM-34,35) | N/A | No symbol moved between a service and a `libs/atlas-*` module — this diff retypes in place |
| Deploy & topics (DOM-22,23) | Fired | New module `libs/atlas-kafka/gen` added under `libs/`; topic env vars re-rendered in `deploy/k8s/base/env-configmap.yaml` and both overlays |
| Runtime safety (DOM-26) | Fired | Non-test Go files changed throughout; no bare `go` statement found in any new/changed file in the reviewed surface |
| Channel wire values (DOM-25) | Fired (by path) | `services/atlas-channel` is touched, but only via mechanical `topic.Token` retyping of Kafka topic constants — no dispatcher mode, sub-op code, or client-interpreted byte is touched |
| Resilience (DOM-27,28) | N/A | No DB-backed handler error branch or `model.Decorator` changed in the reviewed surface |
| External clients (EXT-01..04) | N/A | The two files matching `requests.RootUrl`/`GetRequest`/`PostRequest` (`atlas-saga-orchestrator/party_quest/requests.go`, `.../reactor/processor.go`) only have their topic constants retyped; no new external call site added |
| Scaffolding (SCAFFOLD-01..09) | N/A | No `services/atlas-<svc>/` directory added; `atlas-kafka-precreate` predates this branch (`abf874fb3`) |
| Security (SEC-01..04) | N/A | `atlas-login` files touched are Kafka consumer/message wiring only — no JWT parse, revocation, redirect, or secret-handling code in the reviewed surface |

## Checklist Results

### `libs/atlas-kafka/topic` (support — `provider.go`, no `model.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-11 | Providers evaluate lazily, not eager-read-wrapped-in-`FixedProvider` | PASS | `libs/atlas-kafka/topic/provider.go:14-21` — `EnvProvider` returns `func() (string, error)` that performs `os.LookupEnv` only when invoked; no `FixedProvider` wrap |
| FILE-05 | Readers in `provider.go` | PASS | `topic/provider.go` holds only the `Provider` type alias and `EnvProvider` |
| FILE-06 | No catch-all file | PASS | `topic/token.go` (type only) and `topic/provider.go` (provider only) are single-responsibility |
| DOM-21 | No redeclaration of an existing `libs/atlas-constants` type | PASS | `libs/atlas-constants/inventory/constants.go` and `libs/atlas-constants/gen/*` define an unrelated item/currency `Token` concept — no topic-name type exists there |

### `libs/atlas-outbox` (support — has `provider.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-11 | Lazy provider | PASS | `libs/atlas-outbox/provider.go:22-32` — `EmitProvider` returns nested closures, no eager evaluation |
| DOM-31 | Tenant/trace travel in context only | PASS | `libs/atlas-outbox/bridge.go:54-71` (`headerMap`) derives tenant/span/env headers from `ctx` via decorators, never from a request field; `bridge.go:26-35` explicitly warns (not silently drops) when tenant headers are absent |
| FILE-06 | No catch-all | PASS | `bridge.go` (buffer enqueue) and `provider.go` (provider) stay single-purpose |

### `libs/atlas-kafka/gen` (support — new module, `package main`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-22 | New `libs/atlas-*` module wired into root `Dockerfile` and `go.work` | FAIL | `go.work:1-95` has no `./libs/atlas-kafka/gen` entry; root `Dockerfile` has no `atlas-kafka/gen` COPY block or `for L in ...` loop entry (`Dockerfile:104-108`). Per the checklist's literal trigger ("diff adds a module under `libs/`") and pass criteria (appear in both `COPY` blocks, the synthesized `go.work` loop, and root `go.work`), this module fails all three. Mitigating context, not an exemption: `tools/verify.sh:181-198` deliberately excludes it from the workspace and instead runs it via its own `GOWORK=off go test`/`gen-topics.sh --check` step, and the same shape already exists for `libs/atlas-constants/gen` (also absent from `go.work` and the Dockerfile loop) — i.e. this is a load-bearing design decision (the module is never imported by any service and is not part of any container image), but the checklist carries no documented carve-out for build-time generator submodules, so it is recorded as a FAIL rather than silently exempted. |
| FILE-06 | No catch-all | PASS | `main.go` (CLI/orchestration), `scan.go` (workspace scan), `render_configmap.go`/`render_overlays.go` (rendering), `splice.go` (marker-block splice) — one responsibility per file |

### `services/atlas-kafka-precreate` (support — CLI batch job, new *logic*, pre-existing service)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..05 | Processor/RestModel/requests/Entity/Builder-Model-Admin-Provider-State placement | N/A | None of these constructs exist in this batch-job service; no `resource.go`, `rest.go`, or DB writer exists |
| FILE-06 | No catch-all | PASS | `internal/manifest/manifest.go` (parse+resolve), `internal/discover/discover.go`, `internal/topics/topics.go`, `internal/groups/groups.go`, `internal/kafkaops/ops.go` are each single-purpose |
| DOM-20 | Table-driven tests | PASS | `internal/manifest/manifest_test.go` uses `tests := []struct{...}` + `t.Run` (confirmed by build/test pass; table-driven shape present in every `_test.go` in this package) |
| DOM-26 | Every goroutine via `routine.Go` | N/A | No `go` statement (bare or wrapped) in `main.go` or `internal/*` |

### Mechanical sweep — Kafka topic retyping (~60 services, `EnvProvider` fail-closed)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| — (correctness, not a numbered rule) | Every `topic.EnvProvider(...)()` call site checks its returned error | PASS (312 non-test call sites swept programmatically) | Every call site matches `t, err := topic.EnvProvider(...)()` / `t, err = ...` followed within 2 lines by an `if err != nil` (or `if rerr != nil`) branch — verified by script over all 312 non-test occurrences in `services/` + `libs/`; zero sites discard the error (`_, _ :=` / `t, _ :=`) |
| — (correctness) | Startup-path call sites (`kafka/consumer/*/consumer.go` `NewConfig`) fail loud (`Fatalf`) rather than silently defaulting | PASS | e.g. `services/atlas-buffs/atlas.com/buffs/kafka/consumer/consumer.go:15-18` — was `t, _ := topic.EnvProvider(l)(token)()` (silent fallback), now `t, err := ...; if err != nil { l.WithError(err).Fatalf(...) }`. This is a one-time process-bootstrap path (`consumer.NewConfig`), not a per-message/per-request path, so a hard fatal on a genuinely misconfigured environment is the correct fail-closed behavior, not a request-path Fatal |
| — (correctness) | Per-request/per-emit call sites propagate the error instead of fataling | PASS | `services/atlas-channel/atlas.com/channel/skill/handler/shadow_stars.go:141-143` and `.../socket/handler/character_attack_projectile.go:156-158` both `return err` on resolution failure — both files are untouched by this diff (the constants they reference were retyped in their own declaring package, which propagates automatically through Go's type system with no call-site edit needed) |
| DOM-30 (messaging) | `producer.ProviderImpl` is never handed a **resolved topic name** cast to `topic.Token` in place of the actual token | **FAIL** | `services/atlas-consumables/atlas.com/consumables/kafka/consumer/pickup/consumer.go:55-62`. `t` at line 55 is obtained from `topic.EnvProvider(l)(mbmsg.EnvCommandTopic)()` — i.e. `t` is the *resolved topic name* (a `string`), not an env-var token. Line 61 then calls `producer.ProviderImpl(l)(ctx)(topic.Token(t))(...)`, and `ProviderImpl` → `ManagerWriterProvider` → `Manager.Writer` (`libs/atlas-kafka/producer/manager.go:67`) performs a **second** `topic.EnvProvider(l)(token)()` resolution — i.e. `os.LookupEnv(t)`, looking up an environment variable literally named after the resolved topic value. This double-resolution bug pre-dates this branch (`git show dd00cd7a5:...pickup/consumer.go` shows the identical `producer.ProviderImpl(l)(ctx)(t)` call, `t` being the same pre-resolved string), but it was previously **masked**: the old `EnvProvider` fell back to returning its input token verbatim when the environment lookup failed, so the second (bogus) lookup silently succeeded by returning `t` unchanged. This branch's `EnvProvider` fail-closed contract (`libs/atlas-kafka/topic/provider.go:14-21`) turns that same call into a hard, permanent failure — `os.LookupEnv(t)` will never find an env var named after a runtime topic value, so every `MONSTER_BOOK.CARD_PICKED_UP` emission from this handler now errors and is only logged (`consumer.go:62`), never actually published. This is exactly the "previously-discarded error now handled wrongly by virtue of a masked pre-existing bug being unmasked" risk the task brief called out. Grepped for the same `topic.Token(<alreadyResolvedVar>)` cast pattern elsewhere in the diff (`grep -rn "topic\.Token("`) — this is the only occurrence in the whole tree. |

## Security Review

SEC-* did not fire on any substantively security-relevant surface: the
`atlas-login` files touched by the sweep (`kafka/consumer/**`,
`kafka/message/**`) are Kafka wiring and message-shape constants only — no
JWT parsing, revocation, redirect, or secret material is present in any file
this diff changes.

## Not evaluable from the diff

- DOM-24 (producertest stub installed by every test package reaching an emit path): the diff touches 142 `_test.go` files across ~60 services; only a handful were sampled directly (`libs/atlas-kafka/producer`, `libs/atlas-kafka/gen`, `services/atlas-kafka-precreate/internal/*`). A full sweep of every changed `_test.go` for `producertest.InstallNoop`/`InstallCapturing` installation was not performed — would require reading the other ~130 test files individually.
- DOM-33 (interface change updates every mock implementation): `producer.Manager.Writer`, `producer.Provider`, and `topic.Provider` all changed signature (`string` → `topic.Token`). Compilation of the four sampled service modules (`atlas-consumables`, `atlas-merchant`, `atlas-channel`, `atlas-character`) confirms no stale mock in those modules, but the remaining ~55 service modules touched by the sweep were not individually built here — the concurrent repo-wide `tools/verify.sh` run was relied on for that coverage per the task brief's instruction not to duplicate it.
- Full DOM-30/AndEmit atomicity review of every retyped `AndEmit` call site: with ~270 non-Fatal `topic.EnvProvider` call sites across services, only the ones matching the specific "resolved-name-recast-as-Token" shape were mechanically found and hand-verified (one confirmed bug). A manual trace of every individual `AndEmit`/`message.Buffer` call site for a *different* class of mishandling (e.g., correct type but wrong topic constant) was not performed — would require reading each call site's business logic.

## Summary

### Blocking (must fix)

- `services/atlas-consumables/atlas.com/consumables/kafka/consumer/pickup/consumer.go:61` — `producer.ProviderImpl(l)(ctx)(topic.Token(t))` passes an already-*resolved* topic name (from `topic.EnvProvider(l)(mbmsg.EnvCommandTopic)()` two lines above) into a parameter that performs its own env-var resolution. `EnvProvider`'s new fail-closed contract (this branch's core change) turns the previously-silent no-op double-resolution into a permanent failure: every `MONSTER_BOOK.CARD_PICKED_UP` emit from the pickup handler now errors and is dropped (only logged). Fix: pass `mbmsg.EnvCommandTopic` (the actual token) to `producer.ProviderImpl`, not `topic.Token(t)`.
- `libs/atlas-kafka/gen` (DOM-22) — new module under `libs/` not wired into `go.work` `use()` or the root `Dockerfile`'s COPY blocks / synthesized-`go.work` `for L in ...` loop. Per the checklist's literal pass criteria this fails, even though the module is deliberately excluded by design (verified via `tools/verify.sh`'s own dedicated `GOWORK=off` drift/test step, and precedented by the identically-shaped `libs/atlas-constants/gen`). Flagged rather than silently exempted because the checklist documents no carve-out for build-time generator submodules of an existing library — worth a deliberate call on whether DOM-22's rule text should be narrowed to exclude this shape, or whether the exclusion needs recording as an explicit exception.

### Non-Blocking (should fix)

- None identified beyond the items above and the "Not evaluable" gaps (which are gaps in review coverage, not confirmed defects).
