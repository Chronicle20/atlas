# task-299 — Shared Periodic Task Scheduler — Implementation Plan

Inputs: `prd.md` (v1), `design.md` (v1), `inventory.md` (captured at `31a791e3a`).
Branch: `task-299-shared-task-scheduler`. Base: `31a791e3a`.

Task 1 lands first and is purely additive. Tasks 2–23 are one unit per service and
are mutually independent (each service is its own Go module; no service imports
another's `tasks` package). Task 24 is the repo-wide sweep and gate and lands last.

Each service unit is **atomic**: within that module, deleting `tasks/task.go` before
every `Run()` in the module is converted leaves the module uncompilable. Do not split
a service unit across two commits.

---

## The transformation (reference — each task below restates what it needs)

**Impl signature.** `func (x *T) Run()` → `func (x *T) Run(ctx context.Context)`.
Rules, in precedence order:

1. A body that rooted its own context (`context.Background()`, typically as the otel
   span root) roots it at the passed `ctx` instead:
   `otel...Start(context.Background(), X)` → `otel...Start(ctx, X)`.
2. A body that read a captured `x.ctx` field uses the passed `ctx` instead.
3. A `ctx` struct field left unused after (2) is **removed** (the build fails
   otherwise). Only if that removal makes the constructor's `ctx` parameter dead may
   the parameter be dropped — and then its `main.go` call site is updated in the same
   change. No other constructor signature changes.
4. `envContext func(context.Context) context.Context` closures are untouched; they
   now decorate the passed `ctx`.

**Registration call site.**

```go
// before
routine.Go(l, rt.Context(), func(_ context.Context) {
	tasks.Register(l, rt.Context())(tasks.NewRespawn(l, 10000, envContext))
})
// after
routine.Register(l, rt.Context(), rt.WaitGroup())(tasks.NewRespawn(l, 10000, envContext))
```

The `routine.Go(...)` wrapper is **deleted**, not kept: with `wg.Add(1)` inside
`Register`, an `Add` on a detached goroutine races `Manager.Wait()`. Sites that
already call synchronously need only the argument change. Every affected `main.go`
already imports `routine` and already has `rt.WaitGroup()` in scope. Drop the local
`tasks` import only if it becomes unused — it usually does not, because
`tasks.NewX(...)` lives in the same package.

**Deletion.** Delete `services/<svc>/atlas.com/<mod>/tasks/task.go`. Keep the `tasks`
package directory wherever other files remain.

**Tests.** `task.Run()` → `task.Run(ctx)`, where `ctx` is the context the test already
builds for its tenant/env assertions, or `context.Background()` if it builds none.

**Doc comments.** `tasks.Register` / `tasks.Task` in prose becomes
`routine.Register` / `routine.Task`.

---

## Task 1: Scheduler in `libs/atlas-routine`

### Files

- `libs/atlas-routine/scheduler.go` — **new file**; `Task`, `Register`, `sleepTimeOf`, `drainTimeout`, `minSleep`
- `libs/atlas-routine/scheduler_test.go` — **new file**; external `package routine_test`
- `libs/atlas-routine/scheduler_internal_test.go` — **new file**; internal `package routine`
- `libs/atlas-routine/routine.go` — read-only; `Go` at :15 is the panic-containment primitive the loop runs on
- `libs/atlas-routine/routine_test.go` — read-only; the `waitFor` helper at :19 and the `test.NewNullLogger()` setup shape
- `libs/atlas-routine/go.mod` — read-only; must stay at `logrus` only. `context`/`sync`/`time` are stdlib; **no go.mod change**

Module root (`go build` / `go test` cwd): `libs/atlas-routine`.
Patterns to copy: `libs/atlas-routine/routine_test.go:1-27` (package clause, imports, `waitFor` helper).

**`libs/atlas-routine` must not import `libs/atlas-service`** — `atlas-service/teardown.go:12` imports `atlas-routine`, so the reverse is an import cycle. That is why the `*sync.WaitGroup` is a `Register` parameter.

- [ ] **Step 1: Write the failing tests**

`scheduler_test.go`, external `package routine_test`. Fake task:

```go
type fakeTask struct {
	mu    sync.Mutex
	sleep time.Duration
	runs  []context.Context   // one entry per Run, in order
	fn    func(context.Context) // optional per-test body; nil == no-op
}
```
with `Run(ctx)` appending to `runs` under `mu` then calling `fn` if non-nil, and
`SleepTime()` returning `sleep`. Provide `runCount()` and `lastCtx()` accessors that
take `mu` (the race detector runs on this package).

| test func | setup | assertion |
|---|---|---|
| `TestRegisterSleepFirst` | `sleep = 200ms`, `wg`, `ctx` from `context.WithCancel(context.Background())` | `runCount() == 0` immediately after `Register` returns and again at 50ms; `waitFor` runCount ≥ 1 by 2s |
| `TestRegisterAddsBeforeReturning` | same | immediately after `Register` returns, a `wg.Wait()` launched in a goroutine must NOT complete within 100ms (proves the counter was non-zero at return); then `cancel()` and the wait completes |
| `TestCancellationStopsLoop` | `sleep = 10ms`, logger from `test.NewNullLogger()` | after `cancel()`, `waitFor` the hook to contain an entry whose `Message == "Stopping task execution."`; capture `runCount()`, sleep 100ms, assert `runCount()` unchanged |
| `TestWaitGroupReachesZero` | `sleep = 10ms` | after `cancel()`, `wg.Wait()` in a goroutine closing a channel; `select` on that channel vs `time.After(2 * time.Second)` — timeout is `t.Fatal`, never a hang |
| `TestRunReceivesLiveContext` | `sleep = 10ms`, `fn` records `ctx.Err()` at tick time | the recorded `ctx.Err()` is `nil` during the tick; after `cancel()`, `lastCtx().Err()` is `context.Canceled` |
| `TestAlreadyCancelledContext` | `ctx` cancelled **before** `Register`, `sleep = 10ms` | `wg.Wait()` returns within 2s; `runCount() == 0`; the `"Stopping task execution."` entry was logged |
| `TestPanicStopsOnlyThatTask` | task A `fn` panics with `"boom"`; task B is a plain counter; both `sleep = 10ms`, same `wg` and `ctx` | after A's first run, A's `runCount()` stays at 1 over a 200ms window; B's `runCount()` keeps increasing; a `logrus.ErrorLevel` entry with field `panic == "boom"` was logged; `cancel()` then `wg.Wait()` returns within 2s (A's watchdog released despite the panic) |

`scheduler_internal_test.go`, internal `package routine`. These shorten `drainTimeout`
and read `minSleep`, restoring with `defer`; **they must not call `t.Parallel()`**.

| test func | setup | assertion |
|---|---|---|
| `TestDrainTimeoutAbandonsBlockedRun` | `drainTimeout = 100ms` (restored by `defer`); task `Run` blocks on a channel the test never closes, ignoring `ctx`; `sleep = 10ms` | after `cancel()`, `wg.Wait()` completes between 100ms and 2s; a `logrus.WarnLevel` entry with field `task == "*routine.blockingTask"` (i.e. `fmt.Sprintf("%T", t)`) and a message containing `"abandoning drain"` was logged. Unblock the task at the end of the test |
| `TestDrainSuccessReleasesEarly` | `drainTimeout = 5s`; `Run` returns on `<-ctx.Done()`; `sleep = 10ms` | after `cancel()`, `wg.Wait()` returns in well under 1s; **no** `WarnLevel` entry was logged |
| `TestNonPositiveSleepTimeClamps` | `SleepTime()` returns `0`; set `minSleep = 50ms` (restored by `defer`) | over a 200ms window `runCount() <= 5` (no busy-spin); a `WarnLevel` entry whose `task` field is the `%T` of the task and whose message contains `"non-positive SleepTime"` was logged |

Run `go test ./...` in `libs/atlas-routine` and confirm the new tests fail to compile
(`Register` undefined) before writing Step 2.

- [ ] **Step 2: Write `scheduler.go`**

`package routine`, imports `context`, `fmt`, `sync`, `time`, `logrus`.

```go
// drainTimeout bounds how long shutdown waits for an in-flight Run to
// return before the scheduler releases the teardown WaitGroup and lets the
// process exit anyway. A var, not a const, only so the scheduler's own
// tests can shorten it; nothing outside this package may set it.
var drainTimeout = 5 * time.Second

// minSleep is the floor applied to a non-positive SleepTime().
var minSleep = 1 * time.Second

// Task is a unit of periodic work. Run receives the scheduler loop's
// context: it is cancelled at shutdown, so a long sweep can abort mid-work
// rather than only between ticks.
type Task interface {
	Run(ctx context.Context)
	SleepTime() time.Duration
}

// Register starts t's periodic loop and returns immediately. The loop is
// sleep-first: the first Run happens one SleepTime() after registration.
// The loop returns when ctx is cancelled.
//
// wg is the teardown WaitGroup (service.Runtime.WaitGroup()). Register
// increments it before returning -- so Register MUST be called
// synchronously from main, never from inside a routine.Go -- and releases
// it when the loop returns, or drainTimeout after cancellation if an
// in-flight Run has not returned by then.
func Register(l logrus.FieldLogger, ctx context.Context, wg *sync.WaitGroup) func(t Task) {
	return func(t Task) {
		wg.Add(1)
		done := make(chan struct{})

		Go(l, ctx, func(_ context.Context) {
			defer close(done)
			for {
				select {
				case <-ctx.Done():
					l.Infof("Stopping task execution.")
					return
				case <-time.After(sleepTimeOf(l, t)):
					t.Run(ctx)
				}
			}
		})

		Go(l, ctx, func(_ context.Context) {
			defer wg.Done()
			select {
			case <-done:
			case <-ctx.Done():
				select {
				case <-done:
				case <-time.After(drainTimeout):
					l.WithField("task", fmt.Sprintf("%T", t)).
						Warnf("Task did not return within %s of shutdown; abandoning drain.", drainTimeout)
				}
			}
		})
	}
}

func sleepTimeOf(l logrus.FieldLogger, t Task) time.Duration {
	d := t.SleepTime()
	if d > 0 {
		return d
	}
	l.WithField("task", fmt.Sprintf("%T", t)).
		Warnf("Task reported a non-positive SleepTime (%s); clamping to %s.", d, minSleep)
	return minSleep
}
```

Two goroutines is deliberate: `wg.Done()` cannot sit on the loop goroutine, because a
`Run` that ignores its context and blocks forever would then hold the teardown
`WaitGroup` open forever and `Manager.Wait()` would hang. The watchdog owns the sole
`Done()`, so double-release is structurally impossible and no `sync.Once` is needed.
`defer close(done)` is inside the function passed to `Go`, so it runs *before* `Go`'s
recover — a panicking `Run` still releases the `WaitGroup`.

- [ ] **Step 3: Verify**

`go build ./... && go test -race ./...` in `libs/atlas-routine` — expect PASS. No
`go.mod` / `go.sum` change; if either file is dirty, something imported outside stdlib
+ logrus and must be undone.

---

## Task 2: atlas-account

### Files

- `services/atlas-account/atlas.com/account/account/task.go` — `Run()` → `Run(ctx context.Context)`
- `services/atlas-account/atlas.com/account/main.go` — the `tasks.Register` call at :84 (wrapped in `routine.Go`)
- `services/atlas-account/atlas.com/account/tasks/task.go` — **delete**

Module root: `services/atlas-account/atlas.com/account`.
Patterns to copy: `libs/atlas-routine/scheduler.go` (Task 1) for the `routine.Task` shape.

Apply the transformation from the plan preamble:

1. `Run()` → `Run(ctx context.Context)`; root any `context.Background()` otel span at
   `ctx`; use `ctx` in place of a captured `x.ctx` field; remove the field if it goes
   unused, and its constructor parameter only if that goes dead too (updating :84).
2. Replace the wrapped registration with a bare synchronous
   `routine.Register(l, rt.Context(), rt.WaitGroup())(...)` — the `routine.Go` wrapper
   is deleted, not kept.
3. Delete `tasks/task.go`; drop the `tasks` import from `main.go` only if unused.

- [ ] **Step 1:** convert the impl and the call site, delete `tasks/task.go`.
- [ ] **Step 2:** `go build ./... && go test ./...` in the module root — expect PASS.
- [ ] **Step 3:** `grep -rn 'tasks\.Register\|) Run()' services/atlas-account` — expect empty.

---

## Task 3: atlas-ban

### Files

- `services/atlas-ban/atlas.com/ban/ban/task.go` — `Run(ctx)`; holds a captured `ctx` field
- `services/atlas-ban/atlas.com/ban/history/task.go` — `Run(ctx)`; holds a captured `ctx` field
- `services/atlas-ban/atlas.com/ban/main.go` — two `tasks.Register` calls at :94 and :97, both wrapped in `routine.Go`
- `services/atlas-ban/atlas.com/ban/tasks/task.go` — **delete**

Module root: `services/atlas-ban/atlas.com/ban`.
Constructors in play: `NewExpiredBanCleanup`, `NewHistoryPurge` — both take a `ctx`
argument today. Drop that parameter only if the field becomes fully unused, and update
:94/:97 in the same change.

Apply the transformation from the plan preamble:

1. Both impls: `Run()` → `Run(ctx context.Context)`; replace reads of the captured
   `b.ctx` / `h.ctx` field with the passed `ctx`; root any `context.Background()` otel
   span at `ctx`; remove the now-unused field.
2. Both call sites: delete the `routine.Go(...)` wrapper, leaving synchronous
   `routine.Register(l, rt.Context(), rt.WaitGroup())(...)`.
3. Delete `tasks/task.go`; drop the `tasks` import only if unused.

- [ ] **Step 1:** convert both impls and both call sites, delete `tasks/task.go`.
- [ ] **Step 2:** `go build ./... && go test ./...` in the module root — expect PASS.
- [ ] **Step 3:** `grep -rn 'tasks\.Register\|) Run()' services/atlas-ban` — expect empty.

---

## Task 4: atlas-buffs

### Files

- `services/atlas-buffs/atlas.com/buffs/tasks/berserk.go` — `Run(ctx)`
- `services/atlas-buffs/atlas.com/buffs/tasks/expiration.go` — `Run(ctx)`
- `services/atlas-buffs/atlas.com/buffs/tasks/periodic.go` — `Run(ctx)`
- `services/atlas-buffs/atlas.com/buffs/tasks/periodic_test.go` — `pt.Run()` at :35 → `pt.Run(ctx)`
- `services/atlas-buffs/atlas.com/buffs/main.go` — three `tasks.Register` calls at :72, :75, :78, all wrapped in `routine.Go`
- `services/atlas-buffs/atlas.com/buffs/tasks/task.go` — **delete** (the `tasks` package directory stays; the three impls live in it)

Module root: `services/atlas-buffs/atlas.com/buffs`.

Apply the transformation from the plan preamble:

1. Three impls: `Run()` → `Run(ctx context.Context)`; root any `context.Background()`
   otel span at `ctx`; use `ctx` in place of any captured field, removing the field if
   it goes unused.
2. `periodic_test.go:35` is `require.NotPanics(t, func() { pt.Run() })` → pass the
   context the test already builds, or `context.Background()`.
3. All three call sites: delete the `routine.Go(...)` wrapper, leaving synchronous
   `routine.Register(l, rt.Context(), rt.WaitGroup())(...)`.
4. Delete `tasks/task.go`. The `tasks` import in `main.go` stays — `tasks.NewX(...)`
   still resolves to the same package.

- [ ] **Step 1:** convert the three impls, the test call, and the three call sites; delete `tasks/task.go`.
- [ ] **Step 2:** `go build ./... && go test ./...` in the module root — expect PASS.
- [ ] **Step 3:** `grep -rn 'tasks\.Register\|) Run()' services/atlas-buffs` — expect empty.

---

## Task 5: atlas-channel

### Files

- `services/atlas-channel/atlas.com/channel/channel/task.go` — `Run(ctx)`; captured `ctx` field; registered at main.go:349
- `services/atlas-channel/atlas.com/channel/character/combo/task.go` — `Run(ctx)`; captured `ctx` field; registered at main.go:363
- `services/atlas-channel/atlas.com/channel/session/task.go` — `Run(ctx)`; `Timeout.Run` at :46 roots its own `context.Background()` otel span. **Not registered anywhere** — convert the signature anyway; the acceptance grep in Task 24 requires zero `) Run()` outside atlas-parcel and atlas-mts
- `services/atlas-channel/atlas.com/channel/main.go` — two `tasks.Register` calls at :349 and :363, both wrapped in `routine.Go`
- `services/atlas-channel/atlas.com/channel/tasks/task.go` — **delete**

Module root: `services/atlas-channel/atlas.com/channel`.
Constructors in play: `NewHeartbeat`, `NewDecayTick` — both take `rt.Context()` today.
Drop that parameter only if the field becomes fully unused, updating :349/:363 in the
same change. `NewDecayTick`'s `service.TenantEnvironment` argument is untouched.

Apply the transformation from the plan preamble:

1. Three impls: `Run()` → `Run(ctx context.Context)`. For `session/task.go`, replace
   `otel.GetTracerProvider().Tracer("atlas-channel").Start(context.Background(), TimeoutTask)`
   with `...Start(ctx, TimeoutTask)`. For the other two, replace reads of the captured
   `ctx` field with the passed `ctx` and remove the field if it goes unused.
2. Both call sites: delete the `routine.Go(...)` wrapper (keeping the long explanatory
   comment above the :363 site verbatim), leaving synchronous
   `routine.Register(l, rt.Context(), rt.WaitGroup())(...)`.
3. Delete `tasks/task.go`; drop the `tasks` import only if unused.

- [ ] **Step 1:** convert the three impls and both call sites, delete `tasks/task.go`.
- [ ] **Step 2:** `go build ./... && go test ./...` in the module root — expect PASS.
- [ ] **Step 3:** `grep -rn 'tasks\.Register\|) Run()' services/atlas-channel` — expect empty.

---

## Task 6: atlas-character

### Files

- `services/atlas-character/atlas.com/character/pending_change/task.go` — `Run(ctx)`; captured `ctx` field; doc comments at :17 and :47 name `tasks.Register` / `tasks.Task`
- `services/atlas-character/atlas.com/character/session/task.go` — `Run(ctx)`; captured `ctx` field
- `services/atlas-character/atlas.com/character/pending_change/task_test.go` — `e.Run()` at :275 → `e.Run(ctx)`
- `services/atlas-character/atlas.com/character/main.go` — two `tasks.Register` calls at :156 and :160, both wrapped in `routine.Go`
- `services/atlas-character/atlas.com/character/tasks/task.go` — **delete**

Module root: `services/atlas-character/atlas.com/character`.

Apply the transformation from the plan preamble:

1. Both impls: `Run()` → `Run(ctx context.Context)`; replace reads of the captured
   `ctx` field with the passed `ctx`; root any `context.Background()` otel span at
   `ctx`; remove the field if it goes unused.
2. `pending_change/task.go` doc comments: `:17` "names the otel span and the
   tasks.Register log line" → `routine.Register`; `:47` "interval is the tasks.Task"
   → `routine.Task`.
3. `task_test.go:275` → `e.Run(ctx)`, using the context the test already builds.
4. Both call sites: delete the `routine.Go(...)` wrapper, leaving synchronous
   `routine.Register(l, rt.Context(), rt.WaitGroup())(...)`.
5. Delete `tasks/task.go`; drop the `tasks` import only if unused.

- [ ] **Step 1:** convert both impls, the doc comments, the test call, and both call sites; delete `tasks/task.go`.
- [ ] **Step 2:** `go build ./... && go test ./...` in the module root — expect PASS.
- [ ] **Step 3:** `grep -rn 'tasks\.Register\|tasks\.Task\|) Run()' services/atlas-character` — expect empty.

---

## Task 7: atlas-doors

### Files

- `services/atlas-doors/atlas.com/doors/door/expiry_task.go` — `Run(ctx)`; captured `ctx` field; doc comments at :25 and :66 name `tasks.Task`
- `services/atlas-doors/atlas.com/doors/door/expiry_task_test.go` — `task.Run()` at :103 and :158 → `task.Run(ctx)`
- `services/atlas-doors/atlas.com/doors/main.go` — one `tasks.Register` call at :97, inside the `registerSweepTasks(l, ctx)` closure and wrapped in `routine.Go`; the context argument is a local `ctx` bound from `rt.Context()`
- `services/atlas-doors/atlas.com/doors/tasks/task.go` — **delete**

Module root: `services/atlas-doors/atlas.com/doors`.
`NewExpiryTask` takes a `ctx` argument today; drop it only if the field goes fully
unused, updating :97 in the same change.

Apply the transformation from the plan preamble:

1. `Run()` → `Run(ctx context.Context)`; replace reads of the captured `ctx` field
   with the passed `ctx`; root any `context.Background()` otel span at `ctx`; remove
   the field if it goes unused.
2. Doc comments at :25 and :66: `tasks.Task` → `routine.Task`.
3. `expiry_task_test.go:103,158` → `task.Run(ctx)`.
4. Call site: delete the `routine.Go(...)` wrapper, leaving synchronous
   `routine.Register(l, ctx, rt.WaitGroup())(...)`. `rt.WaitGroup()` must be in scope
   inside `registerSweepTasks` — pass it as a parameter if the closure does not
   already close over `rt`.
5. Delete `tasks/task.go`; drop the `tasks` import only if unused.

- [ ] **Step 1:** convert the impl, doc comments, both test calls, and the call site; delete `tasks/task.go`.
- [ ] **Step 2:** `go build ./... && go test ./...` in the module root — expect PASS.
- [ ] **Step 3:** `grep -rn 'tasks\.Register\|tasks\.Task\|) Run()' services/atlas-doors` — expect empty.

---

## Task 8: atlas-drops

### Files

- `services/atlas-drops/atlas.com/drops/drop/task.go` — `Run(ctx)`; captured `ctx` field
- `services/atlas-drops/atlas.com/drops/main.go` — one `tasks.Register` call at :103, wrapped in `routine.Go`
- `services/atlas-drops/atlas.com/drops/tasks/task.go` — **delete**

Module root: `services/atlas-drops/atlas.com/drops`.

Apply the transformation from the plan preamble:

1. `Run()` → `Run(ctx context.Context)`; replace reads of the captured `ctx` field
   with the passed `ctx`; root any `context.Background()` otel span at `ctx`; remove
   the field if it goes unused, and its constructor parameter only if that goes dead
   (updating :103).
2. Call site: delete the `routine.Go(...)` wrapper, leaving synchronous
   `routine.Register(l, rt.Context(), rt.WaitGroup())(...)`.
3. Delete `tasks/task.go`; drop the `tasks` import only if unused.

- [ ] **Step 1:** convert the impl and the call site, delete `tasks/task.go`.
- [ ] **Step 2:** `go build ./... && go test ./...` in the module root — expect PASS.
- [ ] **Step 3:** `grep -rn 'tasks\.Register\|) Run()' services/atlas-drops` — expect empty.

---

## Task 9: atlas-expressions

### Files

- `services/atlas-expressions/atlas.com/expressions/expression/task.go` — `Run(ctx)`; captured `ctx` field
- `services/atlas-expressions/atlas.com/expressions/expression/task_test.go` — `task.Run()` at :81, :105, :134 → `task.Run(ctx)`
- `services/atlas-expressions/atlas.com/expressions/main.go` — one `tasks.Register` call at :49, wrapped in `routine.Go`
- `services/atlas-expressions/atlas.com/expressions/tasks/task.go` — **delete**

Module root: `services/atlas-expressions/atlas.com/expressions`.

Apply the transformation from the plan preamble:

1. `Run()` → `Run(ctx context.Context)`; replace reads of the captured `ctx` field
   with the passed `ctx`; root any `context.Background()` otel span at `ctx`; remove
   the field if it goes unused.
2. The three test calls → `task.Run(ctx)`, using the context each test already builds
   for its tenant/env assertions. These tests are the cheapest proof that the passed
   context is the one the body now uses.
3. Call site: delete the `routine.Go(...)` wrapper, leaving synchronous
   `routine.Register(l, rt.Context(), rt.WaitGroup())(...)`.
4. Delete `tasks/task.go`; drop the `tasks` import only if unused.

- [ ] **Step 1:** convert the impl, the three test calls, and the call site; delete `tasks/task.go`.
- [ ] **Step 2:** `go build ./... && go test ./...` in the module root — expect PASS.
- [ ] **Step 3:** `grep -rn 'tasks\.Register\|) Run()' services/atlas-expressions` — expect empty.

---

## Task 10: atlas-guilds

### Files

- `services/atlas-guilds/atlas.com/guilds/guild/task.go` — `Run(ctx)`; captured `ctx` field
- `services/atlas-guilds/atlas.com/guilds/main.go` — one `tasks.Register` call at :121, wrapped in `routine.Go`
- `services/atlas-guilds/atlas.com/guilds/tasks/task.go` — **delete**

Module root: `services/atlas-guilds/atlas.com/guilds`.

Apply the transformation from the plan preamble:

1. `Run()` → `Run(ctx context.Context)`; replace reads of the captured `ctx` field
   with the passed `ctx`; root any `context.Background()` otel span at `ctx`; remove
   the field if it goes unused, and its constructor parameter only if that goes dead
   (updating :121).
2. Call site: delete the `routine.Go(...)` wrapper, leaving synchronous
   `routine.Register(l, rt.Context(), rt.WaitGroup())(...)`.
3. Delete `tasks/task.go`; drop the `tasks` import only if unused.

- [ ] **Step 1:** convert the impl and the call site, delete `tasks/task.go`.
- [ ] **Step 2:** `go build ./... && go test ./...` in the module root — expect PASS.
- [ ] **Step 3:** `grep -rn 'tasks\.Register\|) Run()' services/atlas-guilds` — expect empty.

---

## Task 11: atlas-invites

### Files

- `services/atlas-invites/atlas.com/invites/invite/task.go` — `Run(ctx)`; captured `ctx` field
- `services/atlas-invites/atlas.com/invites/main.go` — one `tasks.Register` call at :81, wrapped in `routine.Go`
- `services/atlas-invites/atlas.com/invites/tasks/task.go` — **delete**

Module root: `services/atlas-invites/atlas.com/invites`.

Apply the transformation from the plan preamble:

1. `Run()` → `Run(ctx context.Context)`; replace reads of the captured `ctx` field
   with the passed `ctx`; root any `context.Background()` otel span at `ctx`; remove
   the field if it goes unused, and its constructor parameter only if that goes dead
   (updating :81).
2. Call site: delete the `routine.Go(...)` wrapper, leaving synchronous
   `routine.Register(l, rt.Context(), rt.WaitGroup())(...)`.
3. Delete `tasks/task.go`; drop the `tasks` import only if unused.

- [ ] **Step 1:** convert the impl and the call site, delete `tasks/task.go`.
- [ ] **Step 2:** `go build ./... && go test ./...` in the module root — expect PASS.
- [ ] **Step 3:** `grep -rn 'tasks\.Register\|) Run()' services/atlas-invites` — expect empty.

---

## Task 12: atlas-login

### Files

- `services/atlas-login/atlas.com/login/session/task.go` — `Run(ctx)`
- `services/atlas-login/atlas.com/login/main.go` — one `tasks.Register` call at :171, wrapped in `routine.Go`
- `services/atlas-login/atlas.com/login/tasks/task.go` — **delete**

Module root: `services/atlas-login/atlas.com/login`.
This impl is not in inventory §C, so it roots its own context rather than capturing
one — expect rule 1 (`context.Background()` → `ctx`), not rule 2.

Apply the transformation from the plan preamble:

1. `Run()` → `Run(ctx context.Context)`; root the `context.Background()` otel span at
   the passed `ctx`.
2. Call site: delete the `routine.Go(...)` wrapper, leaving synchronous
   `routine.Register(l, rt.Context(), rt.WaitGroup())(...)`.
3. Delete `tasks/task.go`; drop the `tasks` import only if unused.

- [ ] **Step 1:** convert the impl and the call site, delete `tasks/task.go`.
- [ ] **Step 2:** `go build ./... && go test ./...` in the module root — expect PASS.
- [ ] **Step 3:** `grep -rn 'tasks\.Register\|) Run()' services/atlas-login` — expect empty.

---

## Task 13: atlas-maps

### Files

- `services/atlas-maps/atlas.com/maps/tasks/respawn.go` — `Run(ctx)`; `Respawn.Run` at :32 roots `context.Background()` at :34 — this is the canonical rule-1 case
- `services/atlas-maps/atlas.com/maps/tasks/weather.go` — `Run(ctx)`
- `services/atlas-maps/atlas.com/maps/tasks/jukebox.go` — `Run(ctx)`
- `services/atlas-maps/atlas.com/maps/tasks/mist_tick.go` — `Run(ctx)`; doc comments at :294 and :345 name `tasks.Register`
- `services/atlas-maps/atlas.com/maps/main.go` — four `tasks.Register` calls at :133, :136, :139, :142, all wrapped in `routine.Go` (see :132-:143)
- `services/atlas-maps/atlas.com/maps/tasks/task.go` — **delete** (the `tasks` package directory stays; all four impls live in it)

Module root: `services/atlas-maps/atlas.com/maps`.
Patterns to copy: `services/atlas-maps/atlas.com/maps/tasks/respawn.go:32-38` is the
canonical before-body quoted in `design.md` §5.1.

The canonical transformation:

```go
// before
func (r *Respawn) Run() {
	ctx, span := otel.GetTracerProvider().Tracer("atlas-maps").Start(context.Background(), RespawnTask)
// after
func (r *Respawn) Run(ctx context.Context) {
	ctx, span := otel.GetTracerProvider().Tracer("atlas-maps").Start(ctx, RespawnTask)
```

Apply the transformation from the plan preamble:

1. All four impls: `Run()` → `Run(ctx context.Context)`; root the
   `context.Background()` otel span at the passed `ctx`; replace reads of any captured
   `ctx` field and remove the field if it goes unused. The `envContext` closures in
   `respawn.go` (`processMapsWithCharacters`) are **untouched** — they now decorate
   the passed `ctx`.
2. Doc comments: `mist_tick.go:294` "registered via tasks.Register in main" →
   `routine.Register`; `:345` "invoked once per tick by tasks.Register's loop" →
   `routine.Register's`.
3. All four call sites: delete the `routine.Go(...)` wrappers, leaving four
   synchronous `routine.Register(l, rt.Context(), rt.WaitGroup())(...)` lines. The
   `envContext` comment block above them at :125-:130 stays verbatim.
4. Delete `tasks/task.go`. The `tasks` import in `main.go` stays.

- [ ] **Step 1:** convert the four impls, the two doc comments, and the four call sites; delete `tasks/task.go`.
- [ ] **Step 2:** `go build ./... && go test ./...` in the module root — expect PASS.
- [ ] **Step 3:** `grep -rn 'tasks\.Register\|) Run()' services/atlas-maps` — expect empty.

---

## Task 14: atlas-merchant

### Files

- `services/atlas-merchant/atlas.com/merchant/frederick/task.go` — `Run(ctx)`; captured `ctx` field
- `services/atlas-merchant/atlas.com/merchant/frederick/notification_task.go` — `Run(ctx)`; captured `ctx` field
- `services/atlas-merchant/atlas.com/merchant/shop/task.go` — `Run(ctx)`; captured `ctx` field
- `services/atlas-merchant/atlas.com/merchant/main.go` — three `tasks.Register` calls at :107, :108, :109 (**not** wrapped in `routine.Go` — argument change only)
- `services/atlas-merchant/atlas.com/merchant/tasks/task.go` — **delete**

Module root: `services/atlas-merchant/atlas.com/merchant`.
Constructors in play: `NewExpirationTask` (shop), `NewNotificationTask`,
`NewCleanupTask` — drop a `ctx` parameter only if the field goes fully unused.

Apply the transformation from the plan preamble:

1. Three impls: `Run()` → `Run(ctx context.Context)`; replace reads of the captured
   `ctx` field with the passed `ctx`; root any `context.Background()` otel span at
   `ctx`; remove the field if it goes unused.
2. Three call sites: `tasks.Register(l, rt.Context())(...)` →
   `routine.Register(l, rt.Context(), rt.WaitGroup())(...)`. Confirm no `routine.Go`
   wrapper is present before editing; if one is, delete it.
3. Delete `tasks/task.go`; drop the `tasks` import only if unused.

- [ ] **Step 1:** convert the three impls and three call sites, delete `tasks/task.go`.
- [ ] **Step 2:** `go build ./... && go test ./...` in the module root — expect PASS.
- [ ] **Step 3:** `grep -rn 'tasks\.Register\|) Run()' services/atlas-merchant` — expect empty.

---

## Task 15: atlas-monsters

**Deliberately the largest unit (14 files).** It is not split because the unit is
atomic: deleting `tasks/task.go` while any `Run()` in the module is unconverted leaves
the module uncompilable, and all eight impls are registered from the same eight-line
block in `main.go`. The edits are the same mechanical change repeated, which is the
shape that batches well.

### Files

- `services/atlas-monsters/atlas.com/monsters/monster/task.go` — `Run(ctx)`; captured `ctx` field
- `services/atlas-monsters/atlas.com/monsters/monster/aggro_task.go` — `Run(ctx)`; captured `ctx` field
- `services/atlas-monsters/atlas.com/monsters/monster/picker_task.go` — `Run(ctx)`; captured `ctx` field
- `services/atlas-monsters/atlas.com/monsters/monster/recovery_task.go` — `Run(ctx)`; captured `ctx` field
- `services/atlas-monsters/atlas.com/monsters/monster/drop_timer_task.go` — `Run(ctx)`; captured `ctx` field
- `services/atlas-monsters/atlas.com/monsters/monster/self_destruct_timer_task.go` — `Run(ctx)`; captured `ctx` field
- `services/atlas-monsters/atlas.com/monsters/monster/status_task.go` — `Run(ctx)`; captured `ctx` field
- `services/atlas-monsters/atlas.com/monsters/character/hidden/task.go` — `Run(ctx)`; captured `ctx` field
- `services/atlas-monsters/atlas.com/monsters/monster/aggro_task_test.go` — `tk.Run()` at :93, :135, :160, :197, :224, :327, :367, :394
- `services/atlas-monsters/atlas.com/monsters/monster/picker_task_test.go` — `tk.Run()` at :56, :97, :127, :162
- `services/atlas-monsters/atlas.com/monsters/monster/recovery_task_test.go` — `tk.Run()` at :49, :94, :124, :157, :196, :237
- `services/atlas-monsters/atlas.com/monsters/character/hidden/task_test.go` — `task.Run()` at :37
- `services/atlas-monsters/atlas.com/monsters/main.go` — eight `tasks.Register` calls at :103–:110, all wrapped in `routine.Go`; the context argument is a local `ctx` bound from `rt.Context()`
- `services/atlas-monsters/atlas.com/monsters/tasks/task.go` — **delete**

Module root: `services/atlas-monsters/atlas.com/monsters`.
Constructors in play: `NewMonsterAggroDecayTask`, `NewMonsterSkillPickerSweepTask`,
`NewMonsterRecoveryTask`, `NewDropTimerTask`, `NewSelfDestructTimerTask`,
`NewStatusExpirationTask` — drop a `ctx` parameter only if the field goes fully unused,
updating :103–:110 in the same change.

Apply the transformation from the plan preamble:

1. All eight impls: `Run()` → `Run(ctx context.Context)`; replace reads of the
   captured `ctx` field with the passed `ctx`; root any `context.Background()` otel
   span at `ctx`; remove the field if it goes unused.
2. All 19 test calls above → `Run(ctx)`, using the context each test already builds
   (these tests carry the tenant/env assertions) or `context.Background()`.
3. All eight call sites: delete the `routine.Go(...)` wrappers, leaving eight
   synchronous `routine.Register(l, ctx, rt.WaitGroup())(...)` lines.
4. Delete `tasks/task.go`; drop the `tasks` import only if unused.

- [ ] **Step 1:** convert the eight impls.
- [ ] **Step 2:** convert the 19 test call sites.
- [ ] **Step 3:** convert the eight call sites in `main.go`; delete `tasks/task.go`.
- [ ] **Step 4:** `go build ./... && go test ./...` in the module root — expect PASS.
- [ ] **Step 5:** `grep -rn 'tasks\.Register\|) Run()' services/atlas-monsters` — expect empty.

---

## Task 16: atlas-mounts

### Files

- `services/atlas-mounts/atlas.com/mounts/mount/task.go` — `Run(ctx)`; captured `ctx` field
- `services/atlas-mounts/atlas.com/mounts/mount/task_test.go` — `task.Run()` at :65 → `task.Run(ctx)`
- `services/atlas-mounts/atlas.com/mounts/main.go` — one `tasks.Register` call at :113, wrapped in `routine.Go`
- `services/atlas-mounts/atlas.com/mounts/tasks/task.go` — **delete**

Module root: `services/atlas-mounts/atlas.com/mounts`.

Apply the transformation from the plan preamble:

1. `Run()` → `Run(ctx context.Context)`; replace reads of the captured `ctx` field
   with the passed `ctx`; root any `context.Background()` otel span at `ctx`; remove
   the field if it goes unused, and its constructor parameter only if that goes dead
   (updating :113).
2. `task_test.go:65` → `task.Run(ctx)`.
3. Call site: delete the `routine.Go(...)` wrapper, leaving synchronous
   `routine.Register(l, rt.Context(), rt.WaitGroup())(...)`.
4. Delete `tasks/task.go`; drop the `tasks` import only if unused.

- [ ] **Step 1:** convert the impl, the test call, and the call site; delete `tasks/task.go`.
- [ ] **Step 2:** `go build ./... && go test ./...` in the module root — expect PASS.
- [ ] **Step 3:** `grep -rn 'tasks\.Register\|) Run()' services/atlas-mounts` — expect empty.

---

## Task 17: atlas-pets

### Files

- `services/atlas-pets/atlas.com/pets/pet/task.go` — `Run(ctx)`; captured `ctx` field
- `services/atlas-pets/atlas.com/pets/main.go` — one `tasks.Register` call at :117 (**not** wrapped in `routine.Go` — argument change only)
- `services/atlas-pets/atlas.com/pets/tasks/task.go` — **delete**

Module root: `services/atlas-pets/atlas.com/pets`.

Apply the transformation from the plan preamble:

1. `Run()` → `Run(ctx context.Context)`; replace reads of the captured `ctx` field
   with the passed `ctx`; root any `context.Background()` otel span at `ctx`; remove
   the field if it goes unused, and its constructor parameter only if that goes dead
   (updating :117).
2. Call site: `tasks.Register(l, rt.Context())(...)` →
   `routine.Register(l, rt.Context(), rt.WaitGroup())(...)`. Confirm no `routine.Go`
   wrapper is present before editing; if one is, delete it.
3. Delete `tasks/task.go`; drop the `tasks` import only if unused.

- [ ] **Step 1:** convert the impl and the call site, delete `tasks/task.go`.
- [ ] **Step 2:** `go build ./... && go test ./...` in the module root — expect PASS.
- [ ] **Step 3:** `grep -rn 'tasks\.Register\|) Run()' services/atlas-pets` — expect empty.

---

## Task 18: atlas-rankings

### Files

- `services/atlas-rankings/atlas.com/rankings/tasks/recompute.go` — `Run(ctx)`; captured `ctx` field
- `services/atlas-rankings/atlas.com/rankings/tasks/recompute_test.go` — `task.Run()` at :129, :152, :180, :215, :249, :271 (the last is `task.Run() // must not panic` — keep the comment)
- `services/atlas-rankings/atlas.com/rankings/main.go` — one `tasks.Register` call at :74 (**not** wrapped in `routine.Go`); the context argument is a local `ctx` bound from `rt.Context()`
- `services/atlas-rankings/atlas.com/rankings/tasks/task.go` — **delete** (the `tasks` package directory stays; `recompute.go` lives in it)

Module root: `services/atlas-rankings/atlas.com/rankings`.
`NewRecomputeTask` takes a `ctx` argument today; drop it only if the field goes fully
unused, updating :74 in the same change.

Apply the transformation from the plan preamble:

1. `Run()` → `Run(ctx context.Context)`; replace reads of the captured `ctx` field
   with the passed `ctx`; root any `context.Background()` otel span at `ctx`; remove
   the field if it goes unused.
2. The six test calls → `task.Run(ctx)`, using the context each test already builds.
3. Call site: `tasks.Register(l, ctx)(...)` →
   `routine.Register(l, ctx, rt.WaitGroup())(...)`. Confirm no `routine.Go` wrapper is
   present before editing; if one is, delete it.
4. Delete `tasks/task.go`. The `tasks` import in `main.go` stays.

- [ ] **Step 1:** convert the impl, the six test calls, and the call site; delete `tasks/task.go`.
- [ ] **Step 2:** `go build ./... && go test ./...` in the module root — expect PASS.
- [ ] **Step 3:** `grep -rn 'tasks\.Register\|) Run()' services/atlas-rankings` — expect empty.

---

## Task 19: atlas-reactors

### Files

- `services/atlas-reactors/atlas.com/reactors/tasks/cooldown_cleanup.go` — `Run(ctx)`
- `services/atlas-reactors/atlas.com/reactors/main.go` — one `tasks.Register` call at :68, wrapped in `routine.Go`
- `services/atlas-reactors/atlas.com/reactors/tasks/task.go` — **delete** (the `tasks` package directory stays; `cooldown_cleanup.go` lives in it). This is a **variant-1** copy — the loop has no `ctx.Done()` check — so this service also gains cancellation-aware shutdown

Module root: `services/atlas-reactors/atlas.com/reactors`.
This impl is not in inventory §C, so expect rule 1 (`context.Background()` → `ctx`),
not rule 2.

Apply the transformation from the plan preamble:

1. `Run()` → `Run(ctx context.Context)`; root the `context.Background()` otel span at
   the passed `ctx`.
2. Call site: delete the `routine.Go(...)` wrapper, leaving synchronous
   `routine.Register(l, rt.Context(), rt.WaitGroup())(...)`.
3. Delete `tasks/task.go`. The `tasks` import in `main.go` stays.

- [ ] **Step 1:** convert the impl and the call site, delete `tasks/task.go`.
- [ ] **Step 2:** `go build ./... && go test ./...` in the module root — expect PASS.
- [ ] **Step 3:** `grep -rn 'tasks\.Register\|) Run()' services/atlas-reactors` — expect empty.

---

## Task 20: atlas-rps

### Files

- `services/atlas-rps/atlas.com/rps/game/task.go` — `Run(ctx)`; doc comment at :20 says "implements the tasks.Task interface structurally (Run + SleepTime)"
- `services/atlas-rps/atlas.com/rps/game/task_test.go` — `task.Run()` at :125, :167, :211 → `task.Run(ctx)`
- `services/atlas-rps/atlas.com/rps/game/env_wiring_test.go` — `task.Run()` at :47 → `task.Run(ctx)`
- `services/atlas-rps/atlas.com/rps/main.go` — one `tasks.Register` call at :58, wrapped in `routine.Go`
- `services/atlas-rps/atlas.com/rps/tasks/task.go` — **delete**
- `services/atlas-rps/docs/domain.md` — :135 says "Implements the `tasks.Task` interface (`Run`, `SleepTime`) structurally, without importing `atlas-rps/tasks`" → `routine.Task`, and the trailing clause becomes "without importing `atlas-routine`'s registration helper" or is dropped, since `atlas-rps/tasks` no longer exists

Module root: `services/atlas-rps/atlas.com/rps`.
This impl is not in inventory §C, so expect rule 1 (`context.Background()` → `ctx`),
not rule 2.

Apply the transformation from the plan preamble:

1. `Run()` → `Run(ctx context.Context)`; root the `context.Background()` otel span at
   the passed `ctx`.
2. Doc comment at :20: `tasks.Task` → `routine.Task`.
3. The four test calls → `task.Run(ctx)`; `env_wiring_test.go` asserts the env wiring,
   so pass the context that test already builds.
4. Call site: delete the `routine.Go(...)` wrapper, leaving synchronous
   `routine.Register(l, rt.Context(), rt.WaitGroup())(...)`.
5. Delete `tasks/task.go`; drop the `tasks` import only if unused.
6. Update `services/atlas-rps/docs/domain.md:135`.

- [ ] **Step 1:** convert the impl, doc comment, four test calls, and the call site; delete `tasks/task.go`.
- [ ] **Step 2:** update `services/atlas-rps/docs/domain.md:135`.
- [ ] **Step 3:** `go build ./... && go test ./...` in the module root — expect PASS.
- [ ] **Step 4:** `grep -rn 'tasks\.Register\|tasks\.Task\|) Run()' services/atlas-rps` — expect empty.

---

## Task 21: atlas-skills

### Files

- `services/atlas-skills/atlas.com/skills/tasks/expiration.go` — `Run(ctx)`
- `services/atlas-skills/atlas.com/skills/main.go` — one `tasks.Register` call at :98, wrapped in `routine.Go`
- `services/atlas-skills/atlas.com/skills/tasks/task.go` — **delete** (the `tasks` package directory stays; `expiration.go` lives in it). This is a **variant-1** copy — the loop has no `ctx.Done()` check — so this service also gains cancellation-aware shutdown

Module root: `services/atlas-skills/atlas.com/skills`.
This impl is not in inventory §C, so expect rule 1 (`context.Background()` → `ctx`),
not rule 2.

Apply the transformation from the plan preamble:

1. `Run()` → `Run(ctx context.Context)`; root the `context.Background()` otel span at
   the passed `ctx`.
2. Call site: delete the `routine.Go(...)` wrapper, leaving synchronous
   `routine.Register(l, rt.Context(), rt.WaitGroup())(...)`.
3. Delete `tasks/task.go`. The `tasks` import in `main.go` stays.

- [ ] **Step 1:** convert the impl and the call site, delete `tasks/task.go`.
- [ ] **Step 2:** `go build ./... && go test ./...` in the module root — expect PASS.
- [ ] **Step 3:** `grep -rn 'tasks\.Register\|) Run()' services/atlas-skills` — expect empty.

---

## Task 22: atlas-summons

### Files

- `services/atlas-summons/atlas.com/summons/summon/expiry_task.go` — `Run(ctx)`; captured `ctx` field
- `services/atlas-summons/atlas.com/summons/summon/beholder_task.go` — `Run(ctx)`; captured `ctx` field
- `services/atlas-summons/atlas.com/summons/summon/expiry_task_test.go` — `task.Run()` at :65
- `services/atlas-summons/atlas.com/summons/summon/beholder_task_test.go` — `task.Run()` at :118, :213
- `services/atlas-summons/atlas.com/summons/summon/env_test.go` — `task.Run()` at :77, :109
- `services/atlas-summons/atlas.com/summons/main.go` — two `tasks.Register` calls at :105 and :106 (**not** wrapped in `routine.Go`); the context argument is a local `ctx` bound from `rt.Context()`
- `services/atlas-summons/atlas.com/summons/tasks/task.go` — **delete**

Module root: `services/atlas-summons/atlas.com/summons`.
`NewExpiryTask` and `NewBeholderTask` take a `ctx` argument today; drop it only if the
field goes fully unused, updating :105/:106 in the same change.

Apply the transformation from the plan preamble:

1. Both impls: `Run()` → `Run(ctx context.Context)`; replace reads of the captured
   `ctx` field with the passed `ctx`; root any `context.Background()` otel span at
   `ctx`; remove the field if it goes unused.
2. The five test calls → `task.Run(ctx)`. `env_test.go` asserts env wiring, so pass
   the context that test already builds.
3. Both call sites: `tasks.Register(l, ctx)(...)` →
   `routine.Register(l, ctx, rt.WaitGroup())(...)`. Confirm no `routine.Go` wrapper is
   present before editing; if one is, delete it.
4. Delete `tasks/task.go`; drop the `tasks` import only if unused.

- [ ] **Step 1:** convert both impls, the five test calls, and both call sites; delete `tasks/task.go`.
- [ ] **Step 2:** `go build ./... && go test ./...` in the module root — expect PASS.
- [ ] **Step 3:** `grep -rn 'tasks\.Register\|) Run()' services/atlas-summons` — expect empty.

---

## Task 23: atlas-world

### Files

- `services/atlas-world/atlas.com/world/broadcast/task.go` — `Run(ctx)`; captured `ctx` field
- `services/atlas-world/atlas.com/world/channel/task.go` — `Run(ctx)`; captured `ctx` field
- `services/atlas-world/atlas.com/world/broadcast/task_env_test.go` — `sweep.Run()` at :55 → `sweep.Run(ctx)`
- `services/atlas-world/atlas.com/world/main.go` — two `tasks.Register` calls: :142 (argument `rt.Context()`) and :150 (argument a local `ctx`), both wrapped in `routine.Go`
- `services/atlas-world/atlas.com/world/tasks/task.go` — **delete**

Module root: `services/atlas-world/atlas.com/world`.
Constructors in play: `NewSweep` (broadcast), `NewExpiration` (channel) — drop a `ctx`
parameter only if the field goes fully unused, updating :142/:150 in the same change.

Apply the transformation from the plan preamble:

1. Both impls: `Run()` → `Run(ctx context.Context)`; replace reads of the captured
   `ctx` field with the passed `ctx`; root any `context.Background()` otel span at
   `ctx`; remove the field if it goes unused.
2. `task_env_test.go:55` → `sweep.Run(ctx)`, using the context the test already builds
   for its env assertion.
3. Both call sites: delete the `routine.Go(...)` wrappers, keeping each site's existing
   context argument (`rt.Context()` at :142, the local `ctx` at :150) and adding
   `rt.WaitGroup()`.
4. Delete `tasks/task.go`; drop the `tasks` import only if unused.

- [ ] **Step 1:** convert both impls, the test call, and both call sites; delete `tasks/task.go`.
- [ ] **Step 2:** `go build ./... && go test ./...` in the module root — expect PASS.
- [ ] **Step 3:** `grep -rn 'tasks\.Register\|) Run()' services/atlas-world` — expect empty.

---

## Task 24: Repo sweep and the full gate

### Files

- `tools/verify.sh` — read-only; the flagless run is the completion evidence
- `services/` — read-only sweep target; this task edits nothing unless a grep below finds a straggler
- `libs/atlas-routine/` — read-only; the `-race` run in Step 2 happens here

No module root: the greps run from the worktree root and `tools/verify.sh` covers all
23 changed modules. A module-local build is not evidence at this scale.

- [ ] **Step 1: Acceptance greps**

Run each and record its output. Every one must match its stated expectation:

| command | expect |
|---|---|
| `find services -path '*/tasks/task.go'` | empty |
| `grep -rn 'func Register(l logrus.FieldLogger, ctx context.Context) func' services` | empty |
| `grep -rn 'tasks\.Register' services` | empty (calls *and* doc comments) |
| `grep -rn 'tasks\.Task' services` | empty (doc comments) |
| `grep -rn ') Run()' services --include='*.go'` | only `atlas-parcel/parcel/task.go`, `atlas-parcel/parcel/notification_task.go`, and `atlas-mts/task/periodic.go` — all out of scope |
| `grep -rn 'routine\.Register' services \| wc -l` | `41` |
| `grep -rn -B2 'routine\.Register' services --include=main.go \| grep -c 'routine.Go'` | `0` (this is the F-1 guard: a surviving `routine.Go` wrapper reintroduces the `wg.Add` / `Wait` race, and it compiles and usually works, so only this grep catches it) |
| `git status --porcelain -- '*/go.mod' '*/go.sum'` | empty — the change is stdlib-only |

Fix any straggler in the owning service's module and re-run.

- [ ] **Step 2: Race test on the scheduler**

`go test -race ./...` in `libs/atlas-routine` — expect PASS.

- [ ] **Step 3: The gate**

Flagless `tools/verify.sh` from the worktree root, exit 0. `--quick` / `--no-docker`
do **not** count: 23 modules changed and the bake plus `-race` is the point.

- [ ] **Step 4: Manual SIGTERM check (record the result; not a blocker for the gate)**

`docker compose` up atlas-maps, send SIGTERM, and confirm: four
`"Stopping task execution."` lines, no `"abandoning drain"` Warn, and process exit.
If docker is unavailable in the execution environment, say so explicitly rather than
claiming the check passed — the `libs/atlas-routine` drain tests are the automated
substitute.
