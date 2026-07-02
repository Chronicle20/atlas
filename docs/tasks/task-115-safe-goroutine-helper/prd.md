# Safe Goroutine Helper (RR-6) — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-02
---

## 1. Overview

Every Atlas Go service relies on `safeHandle` (`libs/atlas-kafka/consumer/manager.go:577-585`) to recover panics raised inside Kafka handlers. That recovery only covers the handler's own goroutine: any `go` statement executed inside a handler — or anywhere else in a service — spawns an unprotected goroutine, and a panic there crashes the whole pod. A concrete example is the delayed-effect goroutine in `services/atlas-monsters/atlas.com/monsters/monster/processor.go:700-707` (`go func() { time.Sleep(...); applyAnimationDelayedEffect(...) }()`).

The exposure is repo-wide, not incidental. A sweep of non-test Go files under `services/` and `libs/` finds **126 anonymous `go func` sites** and **39 named-call `go` statements** (e.g. `go tasks.Register(...)` ticker loops in service `main.go` files, `go applyWeatherEffects(...)` in atlas-channel's map consumer) — roughly **165 bare goroutine spawns** with no panic recovery. Only two hand-rolled `recover()` sites exist in shared libs: `safeHandle` itself and `libs/atlas-lock/leader.go:158`.

This task introduces a small shared library exposing a single recover-wrapping spawn helper, blanket-migrates every bare `go` statement in non-test code onto it, and adds two layers of enforcement (a CI guard script modeled on `tools/redis-key-guard.sh`, and a DOM-* item in the backend guidelines checklist) so new bare goroutines cannot regress in.

Note on naming: the backlog item proposed `go.Safe(l, fn)`, but `go` is a Go keyword and cannot be a package identifier. The library uses package `routine` (working name; final name settled in design).

## 2. Goals

Primary goals:
- No goroutine in non-test Atlas code can crash a pod via an unrecovered panic.
- One shared, unit-tested helper is the only sanctioned way to spawn a goroutine outside the helper library itself.
- Regression is mechanically prevented (CI guard) and review-enforced (DOM-* checklist item).
- Existing ad-hoc recovery (`libs/atlas-lock/leader.go`) is unified onto the helper; no duplicate recovery idioms remain.

Non-goals:
- Changing `safeHandle`'s continue-on-panic policy or any handler semantics.
- Adding retry/backoff, supervision trees, or restart-on-panic behavior to spawned work.
- Panic metrics/alerting beyond structured error logging (Loki picks up the log line).
- RR-7 (REST request cancellation) and RR-8 (decorator degradation) from `docs/architectural-improvements.md`.
- Migrating `go` statements in `_test.go` files.

## 3. User Stories

- As an operator, I want a panic in a background goroutine to log an error with a stack trace instead of killing the pod, so that one bad message or race doesn't take down a channel server mid-session.
- As a service developer, I want a single obvious helper for spawning background work, so that I don't have to remember to hand-roll `defer recover()` in every closure.
- As a reviewer, I want CI to fail when a bare `go` statement is introduced outside the helper library, so that panic-safety doesn't depend on review vigilance.

## 4. Functional Requirements

### FR-1. New shared library `libs/atlas-routine`

1. New Go module `libs/atlas-routine` (module path `github.com/Chronicle20/atlas/libs/atlas-routine`), package `routine`.
2. Exactly **one spawn shape** (per scope decision): a function taking a `logrus.FieldLogger`, a `context.Context`, and the work function; it starts the goroutine, defers a `recover()`, and passes the context through to the work function. Working signature (design may refine names, not the shape):
   ```go
   func Go(l logrus.FieldLogger, ctx context.Context, fn func(context.Context))
   ```
3. On panic: recover, log at Error level via the supplied logger, include the panic value and a full stack trace (`runtime/debug.Stack()`). The panic is swallowed — the goroutine ends, nothing is rethrown, the process continues.
4. No behavior beyond spawn+recover+log: no retry, no restart, no metrics, no goroutine registry.
5. Unit tests must cover: fn runs and receives the given context; a panicking fn does not propagate (subtest survives); the panic is logged with the panic value and a stack trace; `go test -race` clean.
6. Repo wiring for a new lib: one `./libs/atlas-routine` line in `go.work`, two `COPY` lines in the repo-root `Dockerfile` (mod-only block + source block). No `docker-bake.hcl` change (that list enumerates services, not libs).

### FR-2. Blanket migration of bare `go` statements

1. Every bare `go` statement in non-test `.go` files under `services/` and `libs/` is replaced with the helper — both forms:
   - anonymous: `go func() { ... }()` → `routine.Go(l, ctx, func(ctx context.Context) { ... })`
   - named call: `go tasks.Register(...)` → `routine.Go(l, ctx, func(_ context.Context) { tasks.Register(...) })` (or equivalent)
2. The only permitted bare `go` statements after migration are inside `libs/atlas-routine` itself and any explicitly allowlisted site (FR-4.3); the allowlist starts as small as possible and every entry carries a written justification.
3. Migration is mechanical, not semantic: the work body is unchanged. Sites that already manage a `sync.WaitGroup`, mutex, or teardown hook keep that logic inside the wrapped fn (e.g. `defer wg.Done()` moves inside the closure passed to the helper).
4. Sites with no `logrus.FieldLogger` or `context.Context` in scope must have one plumbed from the nearest constructor/caller. If a shared-lib site genuinely cannot accept a logger without breaking its public API, the resolution (API change vs. allowlist entry) is decided in design and recorded per-site — never silently skipped.
5. `libs/atlas-lock/leader.go:158`'s hand-rolled recover is replaced by the helper (FR-goal: one recovery idiom). `safeHandle` in atlas-kafka remains as-is — it is inline recovery around a synchronous call, not a spawn — but the `go func` that invokes it (`manager.go:551-570`) migrates like any other site.
6. The migration produces a per-site audit table (in the task folder, e.g. `migration-audit.md`) listing every original site (file:line), its classification (handler-spawned / ticker / lifecycle / lib-internal), and its disposition (migrated / allowlisted+why).

### FR-3. CI guard script `tools/goroutine-guard.sh`

1. New guard script modeled on `tools/redis-key-guard.sh`: scans non-test `.go` files under `services/` and `libs/`, fails (non-zero exit, listing offending file:line) on any bare `go` statement — both `go func` and named-call forms — outside `libs/atlas-routine` and the allowlist.
2. Must not false-positive on non-statement text (comments, strings, `go:generate`/`go:build` directives, words like "go" in identifiers). Must catch the statement forms actually present in the repo today; verified by running it against a pre-migration tree (expect ~165 findings) and the post-migration tree (expect 0).
3. Allowlist mechanism (inline marker comment or a checked-in list file — design decides, mirroring whatever redis-key-guard does) with justification required per entry.
4. Wired into CI wherever `tools/redis-key-guard.sh` runs, and runnable locally from the repo root.
5. Guard has a self-test or fixture check so a regression in the script itself (e.g. a grep pattern typo silently matching nothing) is detectable.

### FR-4. Guidelines enforcement (DOM-* item)

1. Add a DOM-* checklist item to the backend guidelines: goroutines in non-test code must be spawned via the atlas-routine helper; bare `go` statements are banned outside the helper lib and the documented allowlist.
2. Update the `backend-guidelines-reviewer` agent definition and the `backend-dev-guidelines` skill content to include the new item.
3. Update `CLAUDE.md`'s Build & Verification section to list `tools/goroutine-guard.sh` alongside `tools/redis-key-guard.sh`.
4. Update `docs/architectural-improvements.md` to mark RR-6 resolved by this task.

## 5. API Surface

No REST/Kafka surface changes. The only new public API is the Go package API of `libs/atlas-routine` (FR-1.2). No JSON:API resources, no new endpoints, no new events.

## 6. Data Model

None. No entities, no migrations, no tenant-scoped data.

## 7. Service Impact

Blanket migration touches essentially the whole Go codebase:

- **New:** `libs/atlas-routine` (module, package, tests).
- **Shared libs with bare `go` sites:** `atlas-kafka` (consumer manager), `atlas-lock` (leader loop — also loses its hand-rolled recover), `atlas-socket`, `atlas-rest`, `atlas-model` (`async/`, `model/`; `testutil` is test-support — disposition decided in design), `atlas-seeder`.
- **Services:** all ~28 Go services have at least one site. Heaviest: atlas-channel (13 files), atlas-maps (5), atlas-login (4), atlas-monsters (3); long tail of 1-2 files each, plus the 39 named-call sites concentrated in service `main.go` ticker registration.
- **Repo plumbing:** `go.work`, root `Dockerfile` (2 COPY lines), `tools/goroutine-guard.sh`, CI workflow that runs the guards, `CLAUDE.md`, backend guidelines skill + reviewer agent definition, `docs/architectural-improvements.md`.

Because every migrated service gains a dependency on the new lib, every service's `go.mod` is touched → per CLAUDE.md rule 4, **`docker buildx bake all-go-services` is mandatory** before this branch is done, in addition to per-module `go test -race`, `go vet`, `go build`, and both guard scripts.

## 8. Non-Functional Requirements

- **Zero behavior change** apart from panic containment: spawned work runs the same body, same ordering, same synchronization. No added latency on the spawn path beyond a deferred recover.
- **Race safety:** helper and all migrated sites clean under `go test -race`.
- **Observability:** panic log line is structured (logger fields preserved from the passed logger), Error level, includes panic value + stack; greppable pattern stable enough to alert on later (exact message fixed in design).
- **Multi-tenancy:** no impact; the helper does not touch tenant context beyond passing `ctx` through.
- **Compatibility:** no public API of any existing lib changes except where FR-2.4 forces a logger/context parameter to be plumbed; each such change is called out in the plan.

## 9. Open Questions

Deferred to the design phase (none block the PRD):

1. Final package/function naming (`routine.Go` vs. alternatives) and the exact panic log message format.
2. Allowlist mechanism: inline marker comment vs. checked-in baseline file (follow redis-key-guard precedent).
3. Per-site strategy for the handful of shared-lib sites where plumbing a logger may ripple into public APIs (`atlas-model/async`, `atlas-socket`) — plumb, wrap at a higher level, or allowlist with justification.
4. Whether `libs/atlas-model/testutil`'s site counts as test-support (exempt) or non-test code (migrate).

## 10. Acceptance Criteria

- [ ] `libs/atlas-routine` exists with the single spawn helper, wired into `go.work` and the root `Dockerfile`; unit tests prove panic containment, context propagation, and stack-trace logging; `go test -race ./...` and `go vet ./...` clean in the module.
- [ ] Zero bare `go` statements in non-test `.go` files under `services/` and `libs/` outside `libs/atlas-routine` and the justified allowlist; `tools/goroutine-guard.sh` exits 0 from the repo root and is wired into CI next to `redis-key-guard.sh`.
- [ ] Guard script demonstrably catches both `go func` and named-call forms (fixture/self-test), and produced ~165 findings against the pre-migration tree.
- [ ] Per-site migration audit table committed in the task folder covering all pre-migration sites with dispositions.
- [ ] `libs/atlas-lock/leader.go` uses the shared helper; no hand-rolled goroutine `recover()` remains outside `libs/atlas-routine` and `safeHandle`.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed module; `docker buildx bake all-go-services` succeeds from the repo root; `tools/redis-key-guard.sh` still clean.
- [ ] DOM-* item added to backend guidelines skill + `backend-guidelines-reviewer` agent; `CLAUDE.md` verification section lists the new guard; `docs/architectural-improvements.md` RR-6 marked resolved.
