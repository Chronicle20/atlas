# Shared Periodic Task Scheduler — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-09-04
Source: `docs/audits/architecture-audit-2026-09-04.md` § M2
---

## 1. Overview

Twenty-two Atlas services each carry a byte-identical-or-nearly-identical copy of a
periodic task scheduler at `services/<svc>/atlas.com/<svc>/tasks/task.go`. Each copy
declares the same two-method `Task` interface (`Run()`, `SleepTime() time.Duration`)
and the same `Register(l, ctx) func(Task)` entry point, and each wraps
`routine.Go` from `libs/atlas-routine` — a 26-line panic-recovering goroutine
launcher that deliberately does not provide a loop.

The copies have drifted into exactly two variants. Eighteen services run a
`select { case <-ctx.Done(): return; case <-time.After(t.SleepTime()): t.Run() }`
loop that stops on shutdown. Four services — **atlas-buffs, atlas-maps,
atlas-reactors, atlas-skills** — run `for { t.Run(); time.Sleep(t.SleepTime()) }`,
which never observes context cancellation. Their periodic goroutines keep running
after SIGTERM until the process is killed. atlas-maps is one of the services pinned
to a single replica because it holds live field state in memory, so an unclean
shutdown there is the most consequential instance of the defect.

This task moves `Task` and `Register` into `libs/atlas-routine` — already the
goroutine supervisor and the only dependency both variants share — deletes the 22
copies, and in the same change hardens the shared loop: the context is now threaded
into `Task.Run(ctx)` so long-running task bodies can abort mid-work, and shutdown
drains in-flight `Run` calls through the teardown `WaitGroup` instead of abandoning
them.

## 2. Goals

Primary goals:

- Exactly one implementation of the periodic-task loop exists in the repository,
  in `libs/atlas-routine`.
- The four divergent services stop their periodic goroutines on context
  cancellation, like the other eighteen.
- `Task.Run` receives the loop's `context.Context`, so a task body can abort
  mid-work on shutdown rather than only between ticks.
- Shutdown waits for an in-flight `Run` to return (bounded), so a task cannot be
  torn down mid-write.
- The 22 per-service `tasks/task.go` files are deleted; no service defines its own
  `Task` interface or `Register` function afterwards.

Non-goals:

- Changing any task's interval, cadence, or business logic.
- Changing `routine.Go` itself.
- Adding a lint/CI guard against a 23rd copy reappearing (explicitly deferred).
- Migrating the self-driven ticker tasks in **atlas-parcel** and **atlas-mts**
  (`Start`/`Stop`/`stopCh`/`WaitGroup` shape) onto the shared scheduler. They
  implement `Run()`/`SleepTime()` but are not `tasks.Register` consumers. See §9.
- Removing the `tasks` Go package from any service — several of them also hold
  task *implementations* (e.g. `atlas-maps/tasks/respawn.go`). Only `task.go` is
  deleted from each.

## 3. User Stories

- As an operator, I want SIGTERM to stop every periodic goroutine in every service,
  so that a rolling deploy of atlas-maps does not leave respawn/weather/jukebox/mist
  ticks mutating in-memory field state while the process is being torn down.
- As a service developer, I want one scheduler in one place, so that fixing a
  scheduler bug fixes it everywhere instead of in one of twenty-two copies.
- As a task author, I want `Run` to receive a context, so that a long DB sweep or
  a fan-out over tenants can abort on shutdown instead of running to completion.
- As a reviewer, I want shutdown to drain an in-flight `Run`, so that "the task was
  killed halfway through a write" is not a failure mode I have to reason about.

## 4. Functional Requirements

### 4.1 Shared scheduler in `libs/atlas-routine`

**FR-1.** `libs/atlas-routine` gains an exported `Task` interface:

```go
type Task interface {
    Run(ctx context.Context)
    SleepTime() time.Duration
}
```

**FR-2.** `libs/atlas-routine` gains an exported `Register` function preserving the
existing curried call shape, with a third parameter for the drain `WaitGroup`:

```go
func Register(l logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup) func(t Task)
```

The curried form is retained deliberately — every existing call site is written as
`tasks.Register(l, ctx)(task)`, so the migration is a package swap plus one added
argument, not a rewrite.

**FR-3.** The loop is **sleep-first** for all services:

```
for {
    select {
    case <-ctx.Done():
        <log "Stopping task execution."> ; return
    case <-time.After(t.SleepTime()):
        t.Run(ctx)
    }
}
```

This is the existing 18-service behavior, adopted verbatim. The first `Run` of every
task now happens one `SleepTime()` after registration.

**FR-4.** `Register` calls `wg.Add(1)` synchronously (before launching the
goroutine, so the counter is incremented before `Register` returns) and the loop
goroutine calls `wg.Done()` on exit via `defer`.

**FR-5.** The loop goroutine is launched through `routine.Go`, preserving the
existing panic-recovery behavior: a panic inside `Run` is logged with stack and
swallowed, and the process continues.

> **FR-5.1 (open — see §9 OQ-1).** Today a panic inside `Run` kills the whole loop,
> because `routine.Go`'s recover sits outside the `for`. Whether the shared
> scheduler should instead recover per-tick and continue ticking is a behavior
> change; the design phase decides. Default if unresolved: preserve today's
> behavior exactly (panic ends that task's loop).

**FR-6.** `SleepTime()` is read on every iteration (as today), so a task whose
interval is derived from mutable configuration continues to observe changes.

**FR-7.** A non-positive `SleepTime()` must not produce a busy spin. Behavior for
`<= 0` must be explicitly defined and tested (design decides between: clamp to a
documented minimum, or log-and-stop). Today's copies pass it straight to
`time.After`/`time.Sleep`, where `<= 0` fires immediately.

### 4.2 Shutdown drain

**FR-8.** `service.Manager.Wait()` (in `libs/atlas-service/teardown.go`) already
calls `m.cancel()` and then `m.waitGroup.Wait()`. Because FR-4 registers each task
loop on that same `WaitGroup` (call sites pass `rt.WaitGroup()`, FR-12), shutdown
will not complete until every task loop has returned — and a loop returns only
after any in-flight `Run(ctx)` has returned.

**FR-9.** The drain must be **bounded**. A `Run` body that ignores its context and
blocks forever must not hang shutdown indefinitely; today such a task is simply
abandoned and the process exits. The scheduler must log a clearly identifiable
warning when a `Run` has not returned within a drain deadline, and must not block
`Wait()` past that deadline. The deadline value and the mechanism (e.g. the loop
goroutine returning while a detached `Run` finishes on its own) are a design
decision; the requirement is that shutdown cannot regress from "always exits" to
"can hang".

**FR-10.** The drain must not change teardown ordering relative to other
`TeardownFunc` registrations, which fire concurrently on `doneChan` close.

### 4.3 Task interface migration

**FR-11.** All 44 files implementing `Run()` / `SleepTime()` change their `Run`
signature to `Run(ctx context.Context)`.

- Where an implementation currently reads a `ctx` field captured at construction
  (34 of the 44 files do), the body must use the **passed** `ctx` instead.
- Constructor signatures are **not** changed in this task, even where the `ctx`
  parameter becomes unused — except that a struct field left entirely unused after
  the change must be removed (otherwise the build/lint fails). Where removing the
  field makes the constructor's `ctx` parameter dead, the parameter may be dropped
  and its call site updated; this is the only permitted constructor change.
- Tenant/environment-decorating closures (`func(ctx context.Context) context.Context`,
  passed to several constructors) keep working — they now decorate the passed `ctx`.

**FR-12.** All 41 real `tasks.Register(...)` call sites (44 grep hits minus 3 doc
comments in `atlas-maps/tasks/mist_tick.go` ×2 and
`atlas-character/pending_change/task.go` ×1) change to
`routine.Register(l, <ctx>, rt.WaitGroup())(task)`. Every one of the 22 services
bootstraps via `service.Bootstrap` and has `rt.WaitGroup()` in scope in `main.go`
(verified across all 22).

**FR-13.** All 22 `services/*/atlas.com/*/tasks/task.go` files are deleted. The
`tasks` package directory is kept wherever it contains other files.

**FR-14.** Import blocks are updated: services drop the local `tasks` import where
it becomes unused, and add the `routine` import where it is not already present.
Every one of the 22 services already has the `atlas-routine` `require` +
`replace` directive in its `go.mod` (verified), so no module wiring changes are
needed.

### 4.4 Doc comments

**FR-15.** Doc comments that reference the old scheduler by its old name must be
updated, not left stale. Known instances: `atlas-maps/tasks/mist_tick.go:294`,
`atlas-maps/tasks/mist_tick.go:345`, `atlas-doors/door/expiry_task.go:25`,
`atlas-doors/door/expiry_task.go:66`, `atlas-character/pending_change/task.go:17`,
`atlas-character/pending_change/task.go:47`, and the cross-reference in
`atlas-parcel/parcel/task.go`'s `ExpiryTask` comment.

## 5. API Surface

No REST or Kafka surface changes. The only public surface is the Go API of
`libs/atlas-routine`:

| Symbol | Before | After |
|---|---|---|
| `routine.Go` | `func(l, ctx, fn func(context.Context))` | unchanged |
| `Task` | 22 per-service copies; `Run()` | `routine.Task`; `Run(ctx context.Context)` |
| `Register` | 22 per-service copies; `(l, ctx) func(Task)` | `routine.Register(l, ctx, wg) func(Task)` |

Breaking for every in-repo caller by design; there are no external consumers of the
per-service `tasks` packages.

## 6. Data Model

No database entities, migrations, or persisted state are touched.

## 7. Service Impact

### 7.1 The four divergent services (behavior fix + first-tick delay)

md5 `48fe2ca06ba55eb1b190e7e6a24fc82a`:

| Service | Registered tasks | First-tick delay introduced |
|---|---|---|
| atlas-buffs | `NewExpiration` (10000ms), `NewPeriodicTick` (1000ms), `NewBerserkTick` (1000ms) | 10s / 1s / 1s |
| atlas-maps | `NewRespawn` (10000ms), `NewWeather` (1s), `NewJukebox` (1s), `NewMistTick` (1000ms) | 10s / 1s / 1s / 1s |
| atlas-reactors | `NewCooldownCleanup` (60s) | 60s |
| atlas-skills | `NewExpirationTask` (1000ms) | 1s |

These four gain correct shutdown. They also lose their immediate first `Run` at
startup — accepted (see §8 NFR-4).

### 7.2 The eighteen majority services (mechanical)

md5 `e84da3270e946ddc3e4741a4ef79cadf`: atlas-account, atlas-ban, atlas-channel,
atlas-character, atlas-doors, atlas-drops, atlas-expressions, atlas-guilds,
atlas-invites, atlas-login, atlas-merchant, atlas-monsters, atlas-mounts,
atlas-pets, atlas-rankings, atlas-rps, atlas-summons, atlas-world.

Loop semantics unchanged; they gain `Run(ctx)` and the drain.

### 7.3 Task implementations by service (44 files)

atlas-monsters 8, atlas-maps 4, atlas-merchant 3, atlas-channel 3, atlas-buffs 3,
atlas-world 2, atlas-summons 2, atlas-parcel 2, atlas-character 2, atlas-ban 2,
and one each in atlas-skills, atlas-rps, atlas-reactors, atlas-rankings,
atlas-pets, atlas-mounts, atlas-login, atlas-invites, atlas-guilds,
atlas-expressions, atlas-drops, atlas-doors, atlas-account.

The two **atlas-parcel** files implement the interface but drive themselves via
`routine.Go` + their own ticker; they are out of scope (§2) and their `Run` need
not change unless the shared `Task` interface is asserted against them.

### 7.4 `libs/atlas-routine`

Grows from 26 lines to the scheduler plus `Task`. Its `go.mod` gains no new
dependency (`context`, `sync`, `time` are stdlib; `logrus` is already required).
It must **not** import `libs/atlas-service` — that would create an import cycle,
since `atlas-service` imports `atlas-routine`. This is why the `WaitGroup` is a
parameter rather than obtained from the teardown manager.

## 8. Non-Functional Requirements

**NFR-1 (correctness).** After this change, `grep -rn "for {" ` over the scheduler
must find no loop lacking a `ctx.Done()` arm, and no service may define a local
`Task`/`Register` pair.

**NFR-2 (shutdown).** Every service must still terminate on SIGTERM. FR-9's bounded
drain is the guard: adding a drain must not convert a clean exit into a hang.
atlas-maps, single-replica with in-memory field state, is the acceptance case.

**NFR-3 (observability).** The existing `l.Infof("Stopping task execution.")` line
on cancellation is preserved. The drain-timeout warning (FR-9) must name the task
type so an operator can identify which task blocked.

**NFR-4 (startup timing).** The four divergent services accept a one-interval
startup delay for their first tick. The worst case is atlas-reactors' cooldown
cleanup at 60s; expired reactor cooldowns simply persist for up to one extra minute
after boot. atlas-maps' respawn moves from immediate to +10s. No task is known to
depend on running before its first interval elapses; the design phase must confirm
this per task and flag any that does.

**NFR-5 (multi-tenancy).** Tasks that fan out across tenants do so through
constructor-supplied environment closures; threading `ctx` through `Run` must not
change which tenant context a task body sees.

**NFR-6 (testing).** `libs/atlas-routine` gains unit tests covering: cancellation
stops the loop; `wg` reaches zero after cancellation; `Run` receives a context that
is cancelled when the parent is; sleep-first ordering (no `Run` before the first
interval); `SleepTime() <= 0` handling (FR-7); drain-timeout behavior (FR-9); panic
behavior (FR-5.1).

**NFR-7 (verification).** Flagless `tools/verify.sh` must exit 0. The change touches
23 Go modules, so a module-local build is not sufficient evidence.

## 9. Open Questions

- **OQ-1 (FR-5.1).** Should a panic in `Run` end that task's loop (today's
  behavior, since `routine.Go`'s recover wraps the whole loop) or be recovered
  per-tick so the task keeps ticking? Per-tick recovery is arguably correct for a
  periodic sweep but is a real behavior change. Design decides; default is preserve.
- **OQ-2 (FR-9).** What is the drain deadline, and where does it live — a package
  default in `atlas-routine`, or a `Register` parameter? A single documented
  default is preferred over a knob nobody tunes.
- **OQ-3.** Should `atlas-parcel` (`ExpiryTask`, `NotificationTask`) and
  `atlas-mts` (`task/periodic.go`) — the third, self-driven ticker pattern with
  `Start`/`Stop`/`stopCh`/`WaitGroup` — be migrated onto the shared scheduler in a
  follow-up? They already have drain semantics the shared scheduler is now
  acquiring, so consolidating them is the natural next step, but it is out of scope
  here.
- **OQ-4.** Should the `tasks` package be renamed or its remaining
  implementation files relocated in the services where `task.go` was its only
  scheduler content? Default: no, leave package layout alone.

## 10. Acceptance Criteria

- [ ] `libs/atlas-routine` exports `Task` (with `Run(ctx context.Context)`) and
      `Register(l, ctx, wg) func(Task)`.
- [ ] The shared loop is sleep-first and returns on `ctx.Done()`, logging
      `"Stopping task execution."`.
- [ ] `Register` increments the supplied `WaitGroup` before returning and
      decrements it when the loop exits.
- [ ] Shutdown drains an in-flight `Run` and is bounded by a deadline that logs a
      named warning rather than hanging.
- [ ] `find services -path '*/tasks/task.go'` returns zero results.
- [ ] No service package declares its own `Task` interface or `Register` scheduler
      function (repo-wide grep is clean).
- [ ] All 41 registration call sites use `routine.Register(l, ctx, rt.WaitGroup())`.
- [ ] All 44 (or 42, excluding the two out-of-scope atlas-parcel files) task
      implementations compile against `Run(ctx context.Context)` and use the passed
      context where they previously used a captured one; no unused `ctx` struct
      field remains.
- [ ] Stale doc comments referencing the old per-service scheduler are updated
      (FR-15 list).
- [ ] `libs/atlas-routine` tests cover NFR-6's seven cases and pass with `-race`.
- [ ] Flagless `tools/verify.sh` exits 0.
- [ ] Manual or test-based confirmation that atlas-maps exits cleanly on SIGTERM
      with all four of its tasks registered.
