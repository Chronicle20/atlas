# Backend Audit — atlas-channel (task-244-listener-drain-socket-close)

- **Service Path:** services/atlas-channel/atlas.com/channel (+ libs/atlas-socket)
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-20
- **Build:** PASS
- **Tests:** PASS (all packages), 0 failed
- **Overall:** NEEDS-WORK

## Build & Test Results

```
cd services/atlas-channel/atlas.com/channel && go build ./...   -> exit 0, no output
cd services/atlas-channel/atlas.com/channel && go test ./... -count=1  -> all `ok`, zero `FAIL` lines
cd libs/atlas-socket && go build ./...   -> exit 0, no output
cd libs/atlas-socket && go test ./... -count=1   -> ok (server, crypto), no test files in handler/packet/request/response/writer
cd services/atlas-login/atlas.com/login && go build ./... -> exit 0 (sanity check only — atlas-login is out of scope but consumes the widened libs/atlas-socket interface unedited)
```

## Applicability

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure (DOM-01..05,11,16) | No | No `model.go`/`entity.go`/`rest.go`/`provider.go` in any in-scope package (grep confirmed empty for `listener`, `socket`, `configuration/projection`) |
| FILE placement (FILE-01..06) | Yes | Every changed Go package — no exemptions |
| SUB sub-domain (SUB-01..04) | No | No `resource.go` in any in-scope package |
| REST (DOM-06..09,12..15,17..19,32) | No | No `resource.go`/`rest.go`/`processor.go`, no HTTP route registration in the diff |
| Constants reuse (DOM-21) | Yes (checked, nothing to flag) | Diff declares new types (`WaitGrouper`, `dualWaitGroup`) and a sentinel error (`ErrDraining`) |
| Testing (DOM-10,20,24,33) | Yes | Diff touches `_test.go` files extensively |
| Cache (DOM-29) | No | No `cache.go`, no cached processor/struct state introduced |
| Messaging (DOM-30) | No | No `producer.go`, no direct `AndEmit`/`message.Emit`/`producer.ProviderImpl` call added by this diff |
| Multi-tenancy (DOM-31) | Checked, N/A | No `rest.go` in scope; all tenant/`server.Key.TenantId` flow is internal (unexported registry/apply-loop), never a public REST field/path/query param |
| Migration hygiene (DOM-34,35) | Checked, N/A | `Run` is decomposed into `Bind`/`Serve` **within the same package** (`libs/atlas-socket/server.go`) — not a move between a service and a lib. `Run` remains genuinely called (`services/atlas-login/atlas.com/login/socket/init.go:62`), so it is not a leftover delegation wrapper |
| Deploy & topics (DOM-22,23) | No | No new `libs/atlas-*` module, no topic env var added/renamed (`git diff --stat -- deploy/ go.work` empty; grep for `_TOPIC_` only hits struct field names/test literals) |
| Runtime safety (DOM-26) | Yes | Non-test Go files changed in `libs/atlas-socket` and `services/atlas-channel` |
| Channel wire values (DOM-25) | Yes, checked, N/A | Diff touches `services/atlas-channel`; the only wire-facing call (`socket/drain.go`'s `chatpkt.WorldMessageWriter` / `writer.WorldMessagePopUpBody`) reuses pre-existing, unmodified writer-options-table machinery (`services/atlas-channel/atlas.com/channel/socket/writer/world_message.go`, untouched by this diff) |
| Resilience (DOM-27,28) | No | No DB-backed handler error branches, no `model.Decorator`/enrichment path touched |
| External clients (EXT-01..04) | No | Zero `requests.RootUrl`/`requests.GetRequest[T]`/`requests.PostRequest[T]` call sites in any changed package |
| Scaffolding (SCAFFOLD-01..09) | No | No new service directory, no new channel Writer/Handler registration, `deploy/shared/routes.conf` untouched |
| Security (SEC-01..04) | No | No auth/token/redirect/secret-handling code touched |

## Checklist Results

### libs/atlas-socket (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | File placement | N/A | No Processor/RestModel/requests/entity/Builder markers present; `server.go` is the pre-existing single-purpose server file, not a new catch-all — `grep` for every FILE-01..05 marker returns zero hits |
| DOM-21 | No redeclaration of an existing `libs/atlas-constants` type | PASS | `WaitGrouper` (`libs/atlas-socket/server.go:103-106`) is a novel minimal interface unique to this task; `grep -rn "WaitGrouper" libs/atlas-constants/` returns no hits |
| DOM-26 | Every goroutine via `routine.Go`; bare `go` needs `//goroutine-guard:allow` | PASS | `libs/atlas-socket/server.go:163` (listener-close watcher) and `:201`/`:233` (session + handle dispatch) all use `routine.Go`; `tools/goroutine-guard.sh` run directly, exit 0, "goroutineguard: 89 module(s)" with no FAIL output |
| DOM-20 | Tests are table-driven | FAIL | `libs/atlas-socket/bind_serve_test.go` has 4 discrete `func Test...` (lines 34, 53, 71, 101) — none use `tests := []struct{...}` + `t.Run` |
| DOM-24 | Test package reaching an emit path installs `producertest`/no-op producer | N/A | `bind_serve_test.go`'s tests exercise `Bind`/`Serve` bind/accept/waitgroup behavior only; no `AndEmit`/`message.Emit`/`producer.Produce` call reachable from any test in this package |
| DOM-33 | Mock updated for changed Processor/Provider/Administrator interface | N/A | `WaitGrouper` is not a `Processor`/`Provider`/`Administrator` interface; `Run`'s signature widened from `*sync.WaitGroup` to `socket.WaitGrouper`, which `*sync.WaitGroup` satisfies with no caller changes (confirmed by `atlas-login` building unedited) |

### services/atlas-channel/atlas.com/channel/listener (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | File placement | N/A/PASS | No Processor/RestModel/requests/entity/Builder/db-write/db-read markers in `handle.go` or `registry.go` (`grep` empty); `State` enum + `Handle`/`HandlerHandle` structs living together in `handle.go` do not match `model.go`'s definition ("immutable domain objects with private fields and accessor methods" — `Handle`'s fields are exported and mutated directly under `r.mu`), so this is not the Processor+RestModel+requests-style collapse FILE-06 targets |
| DOM-26 | goroutine via `routine.Go` | PASS | `listener/registry.go:238` (phase-3 wait) and `:290` (`DrainAll` fan-out) both use `routine.Go`; `tools/goroutine-guard.sh` exit 0 |
| **RUNTIME — Kafka-handler rollback leak** | `Registry.Add`'s rollback (`registry.go:122-134`) only deletes the registry entry and decrements `refs` — it never calls `deps.RemoveHandler` for any `HandlerHandle` the failing `body` already registered with `consumer.GetManager().RegisterHandler` before the failure. `main.go`'s `buildListener` registers ~20 Kafka consumer handlers (`account2.InitHandlers` … `worldbroadcastConsumer.InitHandlers`, `main.go:445-618`) *before* the socket bind at the tail (`main.go:624`) that this diff newly makes fallible and rollback-triggering. A bind failure after any handler registration leaks those consumer handlers permanently (no `Handle` survives to ever call `RemoveHandler` on them) — a `producer`/`consumer` resource leak, not merely a goroutine leak, but squarely a graceful-shutdown/resource-lifecycle correctness gap this diff newly makes reachable. | FAIL (Important) | `services/atlas-channel/atlas.com/channel/listener/registry.go:122-134` (rollback branch, unchanged by this diff but newly reachable); `services/atlas-channel/atlas.com/channel/main.go:624-630` (bind failure now propagates and fires that rollback, per `dbb7fcb1e`/`345593a61`). Pre-existing: the same gap already existed for a failure mid-`InitHandlers` loop before this diff (confirmed against `0b6917b6e:main.go:600-625`), so this is not a net-new code path, but the diff is what makes the previously-unreachable-via-bind failure mode reachable in production for the first time. Not addressed by any test — `TestRegistry_AddReturnsErrorAndLeavesNoEntryWhenBodyFails` (`registry_test.go:330`) only exercises a `body` that registers zero handlers before failing. |
| DOM-20 | Tests are table-driven | FAIL | `listener/registry_test.go` has 12 discrete `func Test...` (lines 50-455) — none use `tests := []struct{...}` + `t.Run` |
| DOM-24 | Emit-path stub | N/A | `registry_test.go` uses `nopDeps()` and per-test `h.Kick`/`h.Sessions` fakes (e.g. `registry_test.go:91,379,440`) rather than the real `session.Processor`; no test in this package reaches `AndEmit`/`message.Emit` |
| DOM-33 | Mock updated for changed interface | N/A | `listener.Dependencies` is a plain callback struct, not a `Processor`/`Provider`/`Administrator` interface — its field removal (`SessionsForKey`/`SendShutdownNotice`/`DestroySession`) is not subject to DOM-33; no mock package implements `Dependencies` |

### services/atlas-channel/atlas.com/channel/socket (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | File placement | N/A | No Processor/RestModel/requests/entity/Builder markers in `drain.go`/`init.go` (`grep` empty) |
| DOM-26 | goroutine via `routine.Go` | PASS | `socket/init.go:104` and `:140` use `routine.Go`. The `:140-146` registration-loop goroutine is tracked by neither waitgroup — matches the pre-image exactly (`git show 0b6917b6e:.../init.go` has the identical unwrapped construct) and is an explicit PRD non-goal; not re-reported per task instructions. |
| DOM-25 | Client-interpreted wire values resolved from a tenant table | N/A | `drain.go`'s `chatpkt.WorldMessageWriter` / `writer.WorldMessagePopUpBody(shutdownNotice)` (`socket/drain.go:52`) reuse `services/atlas-channel/atlas.com/channel/socket/writer/world_message.go`, an existing, unmodified writer-options-table-backed helper — no new Go literal wire byte introduced |
| DOM-20 | Tests are table-driven | FAIL | `socket/drain_test.go` (1 func, line 19), `socket/init_test.go` (6 funcs, lines 47-158) — none table-driven |
| DOM-24 | Emit-path stub | N/A | `drain_test.go`'s single test asserts a type-assertion error path and returns before reaching `session.Processor.Destroy`'s `p.kp(...)` emit (`session/processor.go:427`); `init_test.go`'s tests exercise bind-failure/waitgroup-fanout only |
| DOM-33 | Mock updated for changed interface | N/A | `CreateSocketService`'s signature widened (`*sync.WaitGroup`→`socket.WaitGrouper`, plus a new `sessionWg` param, plus a `(net.Listener, error)` return) but it is a package function, not a `Processor`/`Provider`/`Administrator` interface method; its only in-repo call site (`main.go:624`) was updated in the same diff |

### services/atlas-channel/atlas.com/channel/configuration/projection (support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | File placement | N/A | No Processor/RestModel/requests/entity/Builder markers in `loop.go` (`grep` empty); `state.go` (holding `State` struct) is untouched by this diff |
| DOM-26 | goroutine via `routine.Go` | N/A | `loop.go`'s diff adds no new goroutine spawn — it only adds a `pending`/`retries` retry map and threads `h.Ctx` into `AddBody` |
| DOM-20 | Tests are table-driven | FAIL | `projection_test.go` has 17 discrete `func Test...` (lines 25-404+) — none table-driven |
| DOM-24 | Emit-path stub | N/A | `AddBody` in these tests is a test-supplied closure (`projection_test.go:327`, `:380`) that never reaches Kafka |
| DOM-33 | Mock updated for changed interface | N/A | `ApplyLoop.execute` changed its own return type (`error`), not a `Processor`/`Provider`/`Administrator` interface method |

### services/atlas-channel/atlas.com/channel (main, support)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01..06 | File placement | N/A | `main.go` carries no Processor/RestModel/requests/entity/Builder markers (`grep` empty) — pure wiring file, consistent with its established role |
| DOM-26 | goroutine via `routine.Go` | PASS | No new bare `go` introduced by `main.go`'s diff; `tools/goroutine-guard.sh` exit 0 covers the whole repo including this file |

## Not evaluable from the diff

- None. Every FILE/RUNTIME/MULTITENANCY/CONSTANTS/TESTING/MIGRATION/CHANNEL-WIRE/DEPLOY item that fired its trigger was settled either from the changed files themselves, one targeted grep, or the repo-root `tools/goroutine-guard.sh` script — no item required reading outside the diff's touched files plus the two symbol lookups already covered above (`session.Processor.Destroy`, `socket/writer/world_message.go`).

## Summary

### Blocking (must fix)
- DOM-20: `libs/atlas-socket/bind_serve_test.go`, `services/atlas-channel/atlas.com/channel/listener/registry_test.go`, `services/atlas-channel/atlas.com/channel/socket/drain_test.go`, `services/atlas-channel/atlas.com/channel/socket/init_test.go`, `services/atlas-channel/atlas.com/channel/configuration/projection/projection_test.go` — none of the ~30 new/changed test functions across this diff use the `tests := []struct{...}` + `t.Run` table-driven pattern the checklist's DOM-20 verification procedure requires (no packet-fixture-playbook exception applies to any of them).
- RUNTIME (unnumbered — `Registry.Add` rollback / consumer-handler leak): `services/atlas-channel/atlas.com/channel/listener/registry.go:122-134` combined with `services/atlas-channel/atlas.com/channel/main.go:624-630` — a socket-bind failure now propagates into `Registry.Add`'s rollback for the first time in production (that was the point of `dbb7fcb1e`/`345593a61`), but the rollback never deregisters the ~20 Kafka consumer handlers `buildListener` already registered via `consumer.GetManager().RegisterHandler` earlier in the same `body` call. Those handlers leak permanently — no surviving `Handle` will ever call `RemoveHandler` on them. Untested: `TestRegistry_AddReturnsErrorAndLeavesNoEntryWhenBodyFails` only covers a `body` that registers zero handlers before failing.

### Non-Blocking (should fix)
- None beyond the above two items — everything else disposed as PASS or N/A with cited evidence.

### Not applicable
- DOM structure, SUB-*, REST, Cache, Messaging, Resilience, EXT-*, Scaffolding, Security, Deploy & topics — see Applicability table for the trigger that did not fire.
