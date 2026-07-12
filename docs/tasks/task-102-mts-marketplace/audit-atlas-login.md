# Backend Audit — atlas-login (task-102 scope)

- **Service Path:** services/atlas-login/atlas.com/login
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-07-10
- **Scope:** Only the packages task-102 modified — `socket` (top-level) and `configuration/projection`. Confirmed via `git diff --stat main..HEAD`: exactly two files changed (`socket/init.go`, `configuration/projection/subscriber.go`).
- **Build:** PASS (`go build ./...` rc=0)
- **Vet:** PASS (`go vet ./...` rc=0)
- **Tests:** 9 packages passed, 0 failed (`go test ./... -count=1`)
- **Overall:** PASS

## Build & Test Results

```
go build ./...   -> rc=0 (clean)
go vet ./...     -> rc=0 (clean)
go test ./... -count=1:
  ok  atlas-login/account
  ok  atlas-login/account/mock
  ok  atlas-login/channel
  ok  atlas-login/character
  ok  atlas-login/configuration
  ok  atlas-login/configuration/projection
  ok  atlas-login/listener
  ok  atlas-login/maps/location
  ok  atlas-login/world
  (all other packages: no test files)
```

## Scope Classification

Both modified packages are **support packages** — neither has `model.go` (no DOM checklist) nor a sub-domain `resource.go` action-event pattern (no SUB checklist). DOM-* and SUB-* are therefore N/A. The applicable checklists are **FILE-01..06** (every package), **EXT-*** (only if the package is a JSON:API client — neither is), and **SEC-*** (auth-adjacent — evaluated below).

## Change Review (the two edited lines)

- `socket/init.go:38` — `wg.Add(1)` was moved out of the inner goroutine to before `go func()` at line 39. This is a correct concurrency fix: `WaitGroup.Add` must run before the goroutine that calls `Done()` so a concurrent `Wait()` cannot return early. Strict improvement; no guideline violation.
  - Observation (non-blocking, pre-existing, out of task scope): the **outer** goroutine at `socket/init.go:22` is still not tracked by `wg`, so `Add(1)` at line 38 runs inside an untracked goroutine. Unchanged by this task and not a guideline-checklist item.
- `configuration/projection/subscriber.go:155` — `consumer.ReadEndOffsets` → `consumer.ReadReplayableEndOffsets`. Compiles and is exercised by the passing `projection` tests; used only to seed the CaughtUp gate. No guideline violation.

## FILE Responsibilities Checklist Results

### socket (top-level — `socket/init.go` only)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor only in processor.go | PASS | No `type Processor`/`NewProcessor` defined in package; it consumes `session.NewProcessor` (socket/init.go:42). Nothing misplaced. |
| FILE-02 | RestModel/Transform only in rest.go | PASS | No RestModel/Transform/JSON:API methods defined in package. |
| FILE-03 | Cross-service requests only in requests.go | PASS | No `requests.RootUrl`/`GetRequest`/`PostRequest` in package (grep: 0 matches). |
| FILE-04 | Entity/Migration/TableName only in entity.go | PASS | No GORM entity in package. |
| FILE-05 | Builder/Model/administrator/provider placement | PASS (N/A) | Bootstrap package; none of these responsibilities present. |
| FILE-06 | No package-named catch-all | PASS | Single file `init.go` (not `socket.go`); one responsibility — socket-service bootstrap. |

DOM/SUB: N/A (support). EXT: N/A (no JSON:API client). SEC: no JWT/token/redirect/secret handling in `init.go`; SEC-01..04 not triggered.

### configuration/projection

Files: `apply.go`, `caughtup.go`, `envelope.go`, `loop.go`, `state.go`, `subscriber.go`, `projection_test.go`.

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor only in processor.go | PASS | No `type Processor`/`NewProcessor` in package. |
| FILE-02 | RestModel/Transform only in rest.go | PASS | Package *references* `configuration.RestModel`/`tenant.RestModel` (state.go:82, apply.go:56) but defines no RestModel or Transform/Extract of its own — nothing misplaced. |
| FILE-03 | Cross-service requests only in requests.go | PASS | No REST client calls; it is a Kafka consumer/projection (subscriber.go:70-86). |
| FILE-04 | Entity/Migration/TableName only in entity.go | PASS | No GORM entity in package. |
| FILE-05 | Builder/Model/administrator/provider/state placement | PASS | Cohesive single-purpose files: envelope decode (`envelope.go`), caught-up gate (`caughtup.go`), op computation + `OpKind` enum (`apply.go`), projected `State` (`state.go`), apply loop (`loop.go`), consumer wiring (`subscriber.go`). The domain file taxonomy does not map onto a projection package; each file carries one responsibility. |
| FILE-06 | No package-named catch-all | PASS | No `projection.go`; no single file bundles ≥2 of the FILE responsibilities. |

DOM/SUB: N/A (support). EXT: N/A. SEC: `json.Unmarshal` at state.go:32/59 and envelope.go:46 decodes **Kafka message payloads** (config-status envelopes), not HTTP request bodies — SUB-04's manual-JSON-parsing prohibition applies to `resource.go` handlers only and does not trigger here. No secrets, JWT, tokens, or redirects. SEC-01..04 not triggered.

## Security Review

No package in scope performs authentication, JWT parsing, token revocation, or redirect handling. The `PASSWORD`/`ONE_TIME_PASSWORD` string matches surfaced by grep are semantic status-code constants in the out-of-scope `socket/writer` and `socket/handler` subpackages (e.g. `socket/writer/login_status.go:17`), not hardcoded credentials. SEC-04: no hardcoded secrets in the two modified packages.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- None required by the guidelines. Observation only: the outer goroutine at `socket/init.go:22` remains untracked by the WaitGroup (pre-existing, outside task-102's scope).

### Counts
- Critical: 0
- Important: 0
- Minor: 0
- Observations: 1 (informational)
