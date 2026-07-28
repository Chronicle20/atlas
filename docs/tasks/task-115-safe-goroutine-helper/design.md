# Safe Goroutine Helper (RR-6) — Design

Version: v1
Status: Approved for planning
Created: 2026-07-02
PRD: `docs/tasks/task-115-safe-goroutine-helper/prd.md`

---

## 1. Summary

Introduce `libs/atlas-routine` with one spawn helper, `routine.Go(l, ctx, fn)`, that wraps every goroutine in a `recover()` which logs the panic (value + stack) at Error level and lets the process continue. Blanket-migrate all ~165 bare `go` statements in non-test code under `services/` and `libs/` onto it. Enforce with a **Go AST analyzer** (`tools/goroutineguard`, modeled directly on `tools/rediskeyguard`) wrapped by `tools/goroutine-guard.sh` and wired into CI, plus a new **DOM-25** backend-guidelines checklist item.

This document settles the four open questions the PRD deferred to design (§4), resolves the per-site strategy for the awkward shared-lib sites (§6), and records alternatives considered (§9).

## 2. Component 1 — `libs/atlas-routine`

### 2.1 Module layout

```
libs/atlas-routine/
├── go.mod        # module github.com/Chronicle20/atlas/libs/atlas-routine
├── routine.go    # the helper (~20 lines)
└── routine_test.go
```

Only dependency: `github.com/sirupsen/logrus`. Go version matches the other `libs/atlas-*` modules.

### 2.2 API (final)

```go
package routine

// Go runs fn in a new goroutine, recovering any panic. A recovered panic is
// logged at Error level with the panic value and full stack trace, then
// swallowed — the goroutine ends and the process continues. ctx is passed
// through to fn unmodified; Go itself never inspects or cancels it.
func Go(l logrus.FieldLogger, ctx context.Context, fn func(context.Context))
```

Implementation shape (normative — the plan copies this):

```go
func Go(l logrus.FieldLogger, ctx context.Context, fn func(context.Context)) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				l.WithField("panic", fmt.Sprintf("%v", r)).
					WithField("stack", string(debug.Stack())).
					Errorf("Recovered panic in background goroutine.")
			}
		}()
		fn(ctx)
	}()
}
```

Decisions folded in:

- **Naming (PRD open question 1):** package `routine`, function `Go`. Call sites read `routine.Go(l, ctx, func(ctx context.Context) { ... })` — the closest legal spelling of the original `go func()` idiom. Rejected: `safego` (stutters: `safego.Go`), `spawn` (implies scheduling semantics we don't have).
- **Log format (PRD open question 1):** fixed message **`Recovered panic in background goroutine.`** — the stable, greppable/alertable pattern. Structured fields: `panic` (via `fmt.Sprintf("%v", r)`, so it serializes deterministically regardless of panic value type) and `stack` (`runtime/debug.Stack()`). All fields already on the passed logger (trace ids, tenant, service fields) are preserved, satisfying the observability NFR.
- **ctx is pass-through only.** No cancellation check before running fn — checking would be a behavior change (a bare `go` today runs the body even under a cancelled ctx). FR-1.4's "no behavior beyond spawn+recover+log" wins.
- `runtime.Goexit()` is not a panic (`recover()` returns nil) and is unaffected.

### 2.3 Tests

Using `logrus/hooks/test.NewNullLogger()`:

1. fn executes and receives the exact ctx value passed in (verify via a context key).
2. A panicking fn does not propagate: the test goroutine survives, and sibling work continues.
3. The recorded log entry is Error level, message `Recovered panic in background goroutine.`, `panic` field contains the panic value's string form, `stack` field contains this test file's function name.
4. Deferred functions inside fn run before the helper's recover (ordering guarantee the atlas-lock migration in §6.3 depends on).
5. All under `go test -race`.

### 2.4 Repo wiring

- `go.work`: add `./libs/atlas-routine`.
- Root `Dockerfile`: two `COPY` lines (mod-only block + source block), per the established new-lib recipe.
- No `docker-bake.hcl` change (it enumerates services, not libs).
- Every migrated module's `go.mod` gains `require github.com/Chronicle20/atlas/libs/atlas-routine v0.0.0` + the relative `replace` directive, matching how services already reference `atlas-kafka`/`atlas-model` (verified pattern: `services/atlas-monsters/atlas.com/monsters/go.mod:7,72`).

## 3. Component 2 — Guard: `tools/goroutineguard` + `tools/goroutine-guard.sh`

### 3.1 Analyzer, not grep

The PRD models the guard on `tools/redis-key-guard.sh` — which is **not** a grep script; it builds and runs a `golang.org/x/tools/go/analysis` analyzer (`tools/rediskeyguard/analyzer.go`). The goroutine guard follows the same shape exactly:

```
tools/goroutineguard/
├── go.mod                      # standalone module, GOWORK=off (like rediskeyguard)
├── analyzer.go
├── analyzer_test.go            # analysistest — this is the FR-3.5 self-test
├── testdata/src/bad/bad.go     # go func(){}(), go named(...), marker with empty justification
├── testdata/src/good/good.go   # routine.Go usage; "go func" inside strings/comments;
│                               # //go:generate directive; correctly-marked allow site
└── cmd/goroutineguard/main.go  # singlechecker
```

Detection is a single `inspector.Preorder` over `(*ast.GoStmt)(nil)`. Because it walks the AST, comments, strings, `//go:generate`/`//go:build` directives, and identifiers containing "go" are structurally incapable of matching — FR-3.2's false-positive requirement is satisfied by construction, and both spawn forms (`go func(){}()` and `go pkg.Fn(...)`) are the same node type, so both are caught by construction too. The testdata fixtures prove it anyway.

Skip rules, in order:

1. `pass.Pkg.Path()` has prefix `github.com/Chronicle20/atlas/libs/atlas-routine` → whole package exempt (mirrors rediskeyguard's `libPkgPath` check).
2. File name (via `pass.Fset.Position`) ends `_test.go` → skip (PRD non-goal). `testdata/` under the guard itself is never scanned (it lives in `tools/`, outside the sweep — §3.2).
3. The `go` statement carries a valid allow marker (§3.3) → skip.

Diagnostic message: `goroutineguard: bare go statement; use routine.Go from libs/atlas-routine (or add //goroutine-guard:allow <justification>)`.

### 3.2 Shell wrapper

`tools/goroutine-guard.sh` copies `redis-key-guard.sh` structure with two deltas:

1. **Self-test first (FR-3.5):** `GOWORK=off go test ./...` inside `tools/goroutineguard` before building the binary. A pattern regression (analyzer silently matching nothing) fails `analysistest` and the script — the guard cannot rot into a no-op.
2. **Sweep covers `libs/` too:** the `find ... -name go.mod` loop runs over both `"$ROOT/services"` and `"$ROOT/libs"` (redis-key-guard sweeps services only; this guard's target sites live in libs as well). `tools/` is deliberately not swept — analyzer testdata must be allowed to contain bare `go` statements.

Exit non-zero listing offending `file:line` diagnostics, same UX as redis-key-guard. Verification during execution: run against the pre-migration tree (expect ≈165 findings — this also validates the PRD's count), then post-migration (expect 0).

### 3.3 Allowlist mechanism (PRD open question 2)

**Inline marker comment**, required on the line immediately above (or trailing on the same line as) the `go` statement:

```go
//goroutine-guard:allow test-support: a swallowed panic here would convert a failing test into a silent pass
go func() { ... }()
```

The analyzer resolves the marker via the file's comment map; a marker with an empty justification is itself a diagnostic (FR-3.3's "justification required" is machine-enforced, not convention). Rejected alternatives in §9.3. The audit table (FR-2.6) additionally lists every allow site, so the full allowlist is greppable both in-code (`//goroutine-guard:allow`) and in-docs.

**Initial allowlist — exactly one entry:**

| Site | Justification |
|---|---|
| `libs/atlas-model/testutil/helpers.go` — `ConcurrentRunner.Go` | Test-support concurrency harness. Today a panic in a runner goroutine crashes the test binary → loud failure. Wrapping it in `routine.Go` would log-and-swallow, letting a panicking test **pass silently**. Panic propagation is the desired behavior in test scaffolding. |

This also answers **PRD open question 4**: `testutil` stays on a bare `go` — but via an explicit, justified marker, not a scanner blind spot.

### 3.4 CI wiring

New `goroutine-guard` job in `.github/workflows/pr-validation.yml`, cloned from the `redis-key-guard` job (checkout → setup-go → `./tools/goroutine-guard.sh`), added to the aggregate `needs:` list and the result-check block that gates the PR (currently around lines 84–98, 480, 496).

## 4. Component 3 — Migration strategy

### 4.1 Mechanical transform rules

| Original form | Migrated form |
|---|---|
| `go func() { BODY }()` | `routine.Go(l, ctx, func(ctx context.Context) { BODY })` — body byte-identical; if the body ignores ctx, bind `_ context.Context` |
| `go func(a T) { BODY }(x)` | `routine.Go(l, ctx, func(_ context.Context) { a := x; BODY })` or hoist the binding before the call — whichever keeps the body unchanged; the loop-variable-capture cases (e.g. `manager.go:523`) hoist |
| `go pkg.Fn(args...)` | `routine.Go(l, ctx, func(_ context.Context) { pkg.Fn(args...) })` |

Synchronization stays inside the closure: `defer wg.Done()`, semaphore releases, `defer close(ch)` all move in unchanged. §2.3 test 4 guarantees those defers fire before the helper's recover, so `wg.Wait()`/channel-close protocols cannot deadlock on a panicked worker.

### 4.2 Logger/ctx sourcing rules (FR-2.4)

Priority order, recorded per-site in the audit table:

1. **Both in scope** (the overwhelming majority — handlers, `main.go` tickers, socket/rest/kafka libs): use them.
2. **In a service, not in scope:** plumb from the nearest constructor/caller. Every service has `l := logger.CreateLogger(serviceName)` and a root ctx in `main.go`; plumbing is always possible inside a service.
3. **Shared lib whose public API has no logger** (only `atlas-model` qualifies — §6.1): `logrus.StandardLogger()` + the ctx the site already owns; `context.Background()` only where no ctx exists at all. Documented per-site, never silent.

### 4.3 Audit table (FR-2.6)

`docs/tasks/task-115-safe-goroutine-helper/migration-audit.md`, generated during execution from the pre-migration guard findings. Columns: original `file:line` | form (anon / named-call) | classification (handler-spawned / ticker / lifecycle / lib-internal / test-support) | logger source | ctx source | disposition (migrated / allowlisted+why). Row count must equal the pre-migration guard finding count.

## 5. Component 4 — Guidelines enforcement

- **DOM-25** (next free number; DOM-24 = Kafka producer stubbing) added to both `.claude/agents/backend-guidelines-reviewer.md` and `.claude/skills/backend-dev-guidelines/`: *goroutines in non-test code must be spawned via `routine.Go` from `libs/atlas-routine`; bare `go` statements are banned outside that lib and sites carrying a justified `//goroutine-guard:allow` marker. Verification: `tools/goroutine-guard.sh` exits 0.*
- `CLAUDE.md` Build & Verification: add `tools/goroutine-guard.sh` as item 6 alongside `redis-key-guard.sh` (item 5).
- `docs/architectural-improvements.md`: mark RR-6 resolved by task-115.

## 6. Per-site resolutions for the hard cases

These are the sites the PRD flagged (open question 3) plus the two special-recovery sites; everything else is rule-4.1 mechanical.

### 6.1 `libs/atlas-model` — `model/processor.go` (5 sites), `async/processor.go` (1 site)

`ExecuteForEachSlice`/`ExecuteForEachMap`/`SliceMap`'s parallel workers and `async.AwaitSlice`'s provider spawns are generic combinators whose public API carries no logger. **Resolution: migrate with `logrus.StandardLogger()`** (rule 4.2.3); each site already owns a ctx (`context.WithCancel`/`WithTimeout`) to pass through.

Why not the alternatives: an additive `SetLogger` configurator would be decorative — no existing caller passes one, so the default path is the only real path and the extra public API buys nothing (rejected, §9.4). Allowlisting would leave the *most* dangerous sites unprotected — these combinators run REST/Kafka fan-out work across every service. The std-logger fallback loses per-request fields on this one exceptional path but still lands the Error line + stack in stderr → Loki.

**Documented behavioral consequence** (accepted; strictly better than pod death): a recovered panic in a parallel worker means its error is never sent on the combinator's error channel. `defer wg.Done()` still fires (§4.1), so nothing deadlocks — `ExecuteForEachSlice` returns `nil` for that item, `async.AwaitSlice` times out with `ErrAwaitTimeout`. The panic is visible in logs, not in the return value. Converting panics to returned errors would be a semantic change beyond FR-2.3's "mechanical, body unchanged" and is out of scope.

### 6.2 `libs/atlas-model/testutil` — allowlisted (§3.3). Panic propagation is the point of a test harness.

### 6.3 `libs/atlas-lock/leader.go:155` — the one hand-rolled recover (FR-2.5)

Current recover does three things: log (no stack), `setReason("panic")`, `cancelLeader()`. `routine.Go` replaces the log (and adds the stack trace), but reason/cancel semantics must survive. **Completed-flag pattern** — panic detection without a second recover:

```go
routine.Go(le.cfg.log, leaderCtx, func(c context.Context) {
	defer close(fnDone)
	completed := false
	defer func() {
		if !completed {
			setReason("panic")
			cancelLeader()
		}
	}()
	fn(c)
	completed = true
})
```

Unwind order on panic: inner defer marks reason + cancels → `close(fnDone)` → helper's recover logs with stack. The `lostReason` metric keeps reporting `panic` exactly as today; one recovery idiom remains repo-wide. The sibling renewer goroutine (`leader.go:167`) migrates mechanically.

### 6.4 `libs/atlas-kafka/consumer/manager.go` — 3 sites (`:145`, `:523`, `:558`)

All have `l`/`ctx` in scope; mechanical. `:523` hoists its `p *pending` parameter binding (§4.1 row 2); the semaphore release and `handlerWg.Done()` defers move inside the closures. **`safeHandle` (`:577`) is untouched** — it is inline recovery around a synchronous call with continue-on-panic *handler* policy (returns `cont=true, err`), not a spawn; folding it into `routine.Go` would change handler-failure semantics.

### 6.5 `libs/atlas-socket`, `libs/atlas-rest`, `libs/atlas-seeder`

All sites (`server.go:125,152,173,226`; `server/server.go:171,186`; `handlers.go:49`) have a logger and ctx in scope or one constructor-hop away; mechanical under rule 4.2.1/4.2.2. No public API changes required.

### 6.6 Services (~28, ≈145 sites)

Mechanical. The two shapes worth naming: handler-internal spawns like `atlas-monsters` `processor.go:700` (delayed-effect sleep — the PRD's motivating example; `p.l` and the processor ctx are in scope) and `main.go` ticker/lifecycle registrations like `atlas-channel` `main.go:318,327` (`go tasks.Register(...)` → named-call wrap; `l`, `tdm.Context()` in scope).

## 7. Testing & verification (branch gate)

1. `libs/atlas-routine` unit tests (§2.3) + `tools/goroutineguard` analysistest — both race-clean.
2. Guard against pre-migration tree ≈165 findings; post-migration tree = 0.
3. `go test -race ./...`, `go vet ./...`, `go build ./...` in every changed module.
4. `docker buildx bake all-go-services` (every service `go.mod` is touched — CLAUDE.md rule 4).
5. `tools/redis-key-guard.sh` and `tools/goroutine-guard.sh` both clean from repo root.
6. Audit table row count = pre-migration finding count; every row dispositioned.

## 8. Risks

- **Sheer diff breadth** (~150 files): mitigated by the transform being mechanical (§4.1), per-module test gates, and the guard itself verifying completeness — the tree isn't done until the analyzer says 0.
- **Subtle sync regressions** (a defer accidentally left outside the closure): `go test -race` across all modules + §2.3 test 4 pin the contract; the kafka manager and lock leader (the two protocol-heavy sites) get explicit review in the plan.
- **Guard blind spots**: `analysistest` fixtures cover both statement forms, marker handling, and non-code text; the wrapper runs the fixtures on every invocation, so the self-test cannot be skipped.
- **atlas-model silent-nil on panicked worker** (§6.1): accepted and documented; log line is the detection path.

## 9. Alternatives considered

### 9.1 Helper shape
- **Chosen:** single `routine.Go(l, ctx, fn(ctx))`.
- `fn func()` (no ctx): fewer keystrokes at named-call sites, but drops the ctx-propagation nudge and the PRD fixed the shape.
- Returning a done-channel / accepting a `*sync.WaitGroup`: supervision-flavored scope creep; FR-1.4 forbids it; call sites that need sync already own it inside the closure.

### 9.2 Guard implementation
- **Chosen:** `go/analysis` AST analyzer, twin of `rediskeyguard`.
- grep/regex script: cannot cleanly exclude strings/comments/directives (FR-3.2 risk) and has no typed skip rules; rejected.
- golangci-lint custom plugin: the repo has no golangci-lint in CI; the standalone-analyzer precedent already exists and needs no new toolchain.

### 9.3 Allowlist mechanism
- **Chosen:** inline `//goroutine-guard:allow <justification>` marker, justification machine-required.
- Checked-in baseline file (dispatcher-lint precedent): line numbers rot on rebase, justification lives far from the code; suited to burn-down baselines, not a permanent tiny allowlist.
- Package-path-only exemption (rediskeyguard precedent): the only lib-level exemption we need is atlas-routine itself; per-site granularity requires markers.

### 9.4 atlas-model logger sourcing
- **Chosen:** `logrus.StandardLogger()` at the six lib-internal sites, documented per-site.
- Additive `SetLogger` configurator: no caller would pass it (existing call sites pass none), so it's decorative public API; revisit only if per-request fields on this path ever matter.
- Breaking API change (logger parameter): ~28-service ripple for an exceptional-path log line; rejected.
- Allowlist the sites: leaves the widest-reach parallel executor in the codebase unprotected — defeats the task's primary goal.
