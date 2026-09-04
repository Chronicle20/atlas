# Shared Periodic Task Scheduler — Design

Version: v1
Status: Draft
Created: 2026-09-04
Inputs: `prd.md` (v1, approved), `inventory.md` (captured at `31a791e3a`)

---

## 1. Summary

Add `Task` and `Register` to `libs/atlas-routine` (new file `scheduler.go`), delete
the 22 per-service `tasks/task.go` copies, thread the loop's context into
`Task.Run(ctx)`, and drain in-flight `Run` calls through the teardown `WaitGroup`
under a bounded deadline.

The design work that the PRD left open resolves as follows:

| Question | Decision |
|---|---|
| OQ-1 (FR-5.1) panic semantics | **Preserve.** A panic in `Run` ends that task's loop; `routine.Go`'s existing recover logs it with stack. No per-tick recovery. |
| OQ-2 (FR-9) drain deadline | **5s**, an unexported package-level `var drainTimeout` in `atlas-routine` — a documented default, not a `Register` parameter; the `var` (not `const`) exists solely as an in-package test seam. |
| FR-7 non-positive `SleepTime()` | **Clamp** to `minSleep = 1 * time.Second` with a `Warn` naming the task type. Not log-and-stop: FR-6 makes `SleepTime()` config-derived and therefore transiently wrong, and a permanent stop is unrecoverable. |
| OQ-3 (parcel/mts self-driven tickers) | Out of scope, confirmed. Follow-up task, not this one. |
| OQ-4 (package layout) | No renames, no relocations. |
| NFR-4 (first-tick delay) | No task in the four divergent services depends on an immediate first run. See §7. |

Two findings the PRD did not enumerate — both are in scope because the change does
not compile or is not correct without them:

- **F-1 (blocking, §5.3).** 22 of the 41 call sites wrap `tasks.Register(...)` in
  `routine.Go(...)`. Those wrappers must be deleted, or FR-4's "`wg.Add(1)` before
  `Register` returns" is not achieved — the `Add` would land on a detached
  goroutine, racing `Wait()`.
- **F-2 (§5.4).** ~45 in-scope `_test.go` call sites invoke `task.Run()` directly
  across 12 services and must become `task.Run(ctx)`.

---

## 2. Where the code lives, and why

`libs/atlas-routine/scheduler.go`, package `routine`. Confirmed constraints:

- `libs/atlas-service/teardown.go` imports `atlas-routine`, so `atlas-routine` must
  not import `atlas-service`. The `*sync.WaitGroup` is therefore a `Register`
  parameter (PRD §7.4) — the cycle is the reason, not a style preference.
- New imports are `context`, `sync`, `time` — stdlib. `logrus` is already required.
  `libs/atlas-routine/go.mod` and every service `go.mod` are unchanged
  (inventory §F).

### Alternative considered: put the scheduler in `libs/atlas-service`

`atlas-service` owns the `Manager` that holds the `WaitGroup`, so the API could be
`rt.RegisterTask(t)` — no `l`, no `ctx`, no `wg` at the call site, and no way to
pass the *wrong* `WaitGroup`. Rejected here for two reasons: PRD FR-1/FR-2/FR-12 fix
both the home and the call shape, and `atlas-service` drags tracing/kafka/env deps
into what is a 40-line loop with a pure-stdlib test.

The convenience wrapper is a strictly additive follow-up: `atlas-service` may later
add `func (r *Runtime) RegisterTask(t routine.Task)` delegating to
`routine.Register(r.logger, r.tdm.Context(), r.tdm.WaitGroup())(t)`. Not in this
task — adding it now would mean touching all 41 call sites twice.

---

## 3. The scheduler

```go
package routine

// drainTimeout bounds how long shutdown waits for an in-flight Run to
// return before the scheduler releases the teardown WaitGroup and lets the
// process exit anyway (FR-9). A var, not a const, only so the scheduler's
// own tests can shorten it; nothing outside this package may set it.
var drainTimeout = 5 * time.Second

// minSleep is the floor applied to a non-positive SleepTime() (FR-7).
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

### 3.1 Why two goroutines

The `wg.Done()` cannot sit on the loop goroutine: if it did, a `Run` that ignores
its context and blocks forever would hold the teardown `WaitGroup` open forever, and
`Manager.Wait()` would hang. Today such a task is simply abandoned at exit; FR-9
forbids regressing "always exits" into "can hang".

So the loop goroutine only signals — `defer close(done)` — and a second, tiny
watchdog goroutine owns the `wg.Done()`. The watchdog releases as soon as the loop
returns, or `drainTimeout` after cancellation, whichever comes first. Only the
watchdog ever calls `Done`, so no `sync.Once` is needed and double-release is
structurally impossible.

`defer close(done)` is inside the function passed to `Go`, so it runs *before*
`Go`'s recover: a panicking `Run` still releases the `WaitGroup`. Panic cannot hang
shutdown.

The watchdog always terminates once `ctx` is cancelled, so it is not a leak; if the
process never shuts down, it lives as long as the loop it watches — one extra
parked goroutine per registered task, at most 8 in any service (atlas-monsters).

**Alternatives rejected.** (a) `defer wg.Done()` on the loop goroutine — simplest,
violates FR-9. (b) Run each tick on its own goroutine and have the loop select on
tick-completion vs `ctx.Done()` — one goroutine per tick forever, and it moves the
panic boundary per tick, silently answering OQ-1 the other way. (c) Force
cancellation of a blocked `Run` — impossible in Go; `Run` must cooperate, which is
exactly what threading `ctx` buys.

### 3.2 Why 5 seconds

The drain is a grace window for a cooperative `Run` to notice `ctx.Done()` and
unwind, not a budget for finishing work. `Manager.Wait()` already fires every
`TeardownFunc` concurrently on `doneChan` close before `cancel()`, so the drain runs
in parallel with, not after, existing teardown — 5s does not stack onto it. It sits
well inside a standard 30s pod termination grace period, and the deadline only
matters at all for a task that ignores its context, which the warning names so it
gets fixed.

### 3.3 Preserved behavior

- `l.Infof("Stopping task execution.")` on cancellation, verbatim (NFR-3).
- `SleepTime()` re-read every iteration (FR-6).
- Sleep-first ordering, taken verbatim from the 18-service majority (FR-3).
- Panic containment via `routine.Go` (FR-5), including its existing Error log with
  panic value and stack.
- A `ctx` already cancelled at `Register` time: the loop's first select takes
  `ctx.Done()`, logs, returns; the watchdog releases immediately. No tick runs.

### 3.4 On OQ-1

Per-tick recovery would let a sweep survive a bad tenant row instead of dying
silently. It is rejected for now because it is a behavior change across 22 services
delivered inside a refactor, and because a panicking `Run` is a bug: today it stops
the loop and logs an Error with a stack, which is the signal that gets it fixed;
per-tick recovery converts that into an unbounded error-log loop that reads as
noise. It can be added later purely inside `scheduler.go` with no API change and no
call-site churn — deliberately keeping that option open is part of the reason the
loop body stays this small.

---

## 4. Shutdown interaction

`Manager.Wait()` is unchanged (FR-10):

```
<-termChan → close(doneChan) → TeardownFuncs fire concurrently → cancel() → waitGroup.Wait()
```

Task loops now join that `WaitGroup`, so `Wait()` returns once every loop has
returned or every watchdog has timed out — bounded by `drainTimeout` regardless of
task behavior.

**Known interaction, accepted.** `TeardownFunc`s fire concurrently with the drain,
so a `TeardownFunc` that closes a DB/Redis handle can close it while an in-flight
`Run` is mid-query. The query then returns an error rather than crashing, and
`routine.Go` contains a panic if one occurs. This is not a regression: today the
same `Run` is abandoned mid-query at process exit. Ordering teardown behind the
drain would be a change to `atlas-service`'s teardown contract and is out of scope.

---

## 5. Migration

### 5.1 Interface change (42 in-scope implementations)

`func (x *T) Run()` → `func (x *T) Run(ctx context.Context)`.

The canonical body transformation, from `atlas-maps/tasks/respawn.go`:

```go
// before
func (r *Respawn) Run() {
	ctx, span := otel.GetTracerProvider().Tracer("atlas-maps").Start(context.Background(), RespawnTask)

// after
func (r *Respawn) Run(ctx context.Context) {
	ctx, span := otel.GetTracerProvider().Tracer("atlas-maps").Start(ctx, RespawnTask)
```

Rules, in order of precedence:

1. A body that rooted its own context (`context.Background()`, typically as the otel
   span root) roots it at the passed `ctx` instead.
2. A body that read a captured `r.ctx` field uses the passed `ctx` (34 files,
   inventory §C).
3. A `ctx` struct field left unused after (2) is **removed** — the build fails
   otherwise. Only if that removal makes the constructor's `ctx` parameter dead may
   the parameter be dropped, and then its call site is updated in the same change
   (FR-11). No other constructor signature changes.
4. `envContext func(context.Context) context.Context` closures are untouched; they
   now decorate the passed `ctx` (NFR-5).

**NFR-5 holds by construction, not by inspection.** The `ctx` a task now receives is
`rt.Context()` — which is the *same value* that was previously captured at
construction (every call site passes `rt.Context()` or a local bound from it,
inventory §D) and carries no tenant/env values (it descends from
`context.Background()` in `GetTeardownManager`). Tenant and environment still enter
exclusively via `tenant.WithContext` / `envContext` inside the body. The only
semantic difference is that the context is now *cancellable*.

**Accepted consequence.** A sweep in flight when SIGTERM arrives now aborts with
`context.Canceled` instead of running to completion, which will produce error logs
during shutdown that did not appear before. This is the intent of FR-11; every
affected sweep is idempotent and re-runs on the next tick after restart.

### 5.2 Registration call sites (41)

```go
// before
tasks.Register(l, rt.Context())(task)
// after
routine.Register(l, rt.Context(), rt.WaitGroup())(task)
```

All 22 in-scope `main.go` files already import `atlas-routine` (verified: none of
them appear in the list of mains lacking that import), and every one has
`rt.WaitGroup()` in scope. The local `tasks` import is dropped wherever it becomes
unused — note it often does *not* become unused, because `tasks.NewRespawn(...)` and
friends live in the same package (FR-13 keeps the package).

### 5.3 Finding F-1 — unwrap `routine.Go(...)` around `Register` (22 sites)

22 of the 41 call sites look like:

```go
routine.Go(l, rt.Context(), func(_ context.Context) {
	tasks.Register(l, rt.Context())(tasks.NewRespawn(l, 10000, envContext))
})
```

in atlas-account, atlas-ban ×2, atlas-buffs ×3, atlas-channel, atlas-character,
atlas-drops, atlas-expressions, atlas-guilds, atlas-invites, atlas-login,
atlas-maps ×4, atlas-mounts, atlas-reactors, atlas-rps, atlas-skills, atlas-world.

The wrapper is already pointless — `Register` never blocks; it launches a goroutine
and returns. With FR-4 it becomes wrong: `wg.Add(1)` would execute on a detached
goroutine, so `Register` returning no longer implies the counter was incremented,
and the `Add` races `Wait()`. `sync.WaitGroup` documents that an `Add` which starts
when the counter is zero must happen before `Wait`.

**Every one of these 22 wrappers is deleted**, leaving a bare synchronous
`routine.Register(l, rt.Context(), rt.WaitGroup())(...)` in `main`. This is a
required part of the change, not a cleanup. `atlas-doors`'s
`registerSweepTasks(l, ctx)` closure and the other unwrapped sites already call
synchronously and need only the argument change.

### 5.4 Finding F-2 — test call sites (~45, 12 services)

`task.Run()` in `_test.go` becomes `task.Run(ctx)`, where `ctx` is the context the
test already builds (most of these tests construct one for the tenant/env
assertions) or `context.Background()`. Affected: atlas-buffs, atlas-rankings,
atlas-summons, atlas-expressions, atlas-doors, atlas-character, atlas-monsters
(picker/aggro/recovery/hidden), atlas-rps, atlas-mounts, atlas-world. The
atlas-parcel test call sites are out of scope and unchanged; `atlas-configurations`'
`s.Run()` is a seeder, unrelated.

These tests are also the cheapest available proof of FR-11 rule (2): a test that
previously exercised a captured `ctx` now proves the passed one is used.

### 5.5 Deletions and doc comments

- Delete the 22 `services/*/atlas.com/*/tasks/task.go` (inventory §A). Keep the
  `tasks` package directory wherever other files remain (FR-13).
- FR-15 doc comments: `atlas-maps/tasks/mist_tick.go:294,345`,
  `atlas-doors/door/expiry_task.go:25,66`,
  `atlas-character/pending_change/task.go:17,47`,
  `atlas-rps/game/task.go:20`, `atlas-parcel/parcel/task.go`'s cross-reference —
  `tasks.Register`/`tasks.Task` becomes `routine.Register`/`routine.Task`.
- One markdown file outside `docs/tasks/` references the old name and is updated:
  `services/atlas-rps/docs/domain.md:135`.

### 5.6 Sequencing and execution shape

Three stages:

1. **`libs/atlas-routine`** — `scheduler.go` + tests. Purely additive; nothing else
   in the repo compiles differently. Lands first.
2. **Per service, 22 independent units.** Each unit is: impls → `Run(ctx)`, tests →
   `Run(ctx)`, `main.go` call sites (unwrap + `wg` arg + imports), delete
   `tasks/task.go`, doc comments, then `go build ./... && go test ./...` in that
   module. Each service is its own Go module and no service imports another's
   `tasks` package, so the 22 units are genuinely independent and safely parallel —
   but each unit is atomic: deleting `task.go` before its impls are converted leaves
   that module uncompilable.
3. **Repo sweep + gate.** The §8 greps, then flagless `tools/verify.sh` (NFR-7 — 23
   modules change, so a module-local build is not evidence).

**Codemod evaluation (`docs/codemod-vs-agents.md`).** The transformation is
templated and repeated 22 times, which triggers the rule, so it is evaluated here
rather than at the second dispatch. Verdict: **agent fan-out, no rewriter.** The
mechanical share (`Run()` → `Run(ctx context.Context)`, the `Register` argument, the
import edits, the file deletions) is genuinely AST- or `sed`-shaped, but the rule
requires *at most one* judgment step per site, and this change has four that a
rewriter cannot decide: which context a body should root (§5.1 rules 1–2), whether a
struct field became dead, whether a constructor parameter may then be dropped, and
what `ctx` each of ~45 tests should pass. Stage 2's per-service units are also small
— the largest, atlas-monsters, is 8 impls plus its tests — so they sit inside a
single implementer budget rather than blowing past it. A scripted pre-pass may still
do the two purely syntactic parts repo-wide (delete the 22 `task.go`; rewrite the
`tasks.Register(l, X)(` prefix), leaving agents only the judgment work.

---

## 6. Testing

`libs/atlas-routine/scheduler_test.go` (external `package routine_test`, matching
the existing `routine_test.go`) covers NFR-6, with a fake task whose `SleepTime` is
in the low milliseconds and whose `Run` records the contexts it received:

| Case | Assertion |
|---|---|
| Cancellation stops the loop | after `cancel()`, no further `Run`, and `"Stopping task execution."` was logged |
| WaitGroup reaches zero | `wg.Wait()` returns after `cancel()` (guarded by a test timeout, so a regression fails rather than hangs) |
| `Run` receives a live context | the `ctx` handed to `Run` is not done during the tick, and *is* done after the parent is cancelled |
| Sleep-first | no `Run` before the first interval elapses |
| `Add` precedes return | `wg` counter is non-zero the instant `Register` returns (a second `Add(1)`/`Done()` pair around a `Wait` in the test, or a `Wait` that must not return early) |
| Already-cancelled ctx | zero `Run` calls; loop exits; `wg` released |
| Panic (OQ-1) | a `Run` that panics stops that task's loop, is logged at Error by `Go`, releases the `wg`, and does not stop a second registered task |

`scheduler_internal_test.go` (internal `package routine`) covers the two cases that
need the timing seams — it shortens `drainTimeout` and reads `minSleep`, restoring
both with `defer`; these tests must not call `t.Parallel()`:

| Case | Assertion |
|---|---|
| Drain timeout (FR-9) | a `Run` that ignores its context and blocks: `wg.Wait()` returns within the shortened `drainTimeout`, and a Warn naming the task type (`%T`) was logged |
| Drain success | a `Run` that returns on `ctx.Done()`: `wg` is released as soon as it returns, well before the deadline, and no Warn is logged |
| Non-positive `SleepTime` (FR-7) | `SleepTime() == 0` does not busy-spin — `Run` call count over a window is bounded by the clamp, and the clamp Warn names the task |

Per-service test changes are the §5.4 signature updates; no new per-service tests
are required by the PRD, and existing tenant/env assertions carry over unchanged.

**Race.** `go test -race` on `libs/atlas-routine` is required (acceptance criteria)
and is covered by flagless `tools/verify.sh`.

---

## 7. First-tick delay (NFR-4)

Per-task confirmation for the four services that lose their immediate first `Run`:

| Task | Delay | Assessment |
|---|---|---|
| buffs `Expiration` | 10s | Expiry sweep over buff state that is empty at process start; nothing to expire on tick 0. |
| buffs `PeriodicTick` | 1s | Drives periodic-effect rows for live characters; none are connected 1s into boot. |
| buffs `BerserkTick` | 1s | Same. |
| maps `Respawn` | 10s | The most visible: after a restart, monsters appear up to 10s later. Field state is in-memory and empty at boot, so tick 0 spawned into a map with no characters; `GetMapsWithCharacters` returns nothing until characters arrive, which itself takes longer than 10s after a restart. |
| maps `Weather` / `Jukebox` | 1s | Registry-driven, empty at boot. |
| maps `MistTick` | 1s | Expires mists past lifetime; no mists exist at boot. |
| reactors `CooldownCleanup` | 60s | In-memory cooldown registry, empty at boot; nothing to clean on tick 0. |
| skills `ExpirationTask` | 1s | The one DB-backed case: expired cooldowns can exist in the table at boot, and they now persist 1s longer. Immaterial against a 1s cadence. |

**No task depends on running before its first interval elapses.** Every one of the
eight sweeps a registry or table that is either empty at boot or tolerant of one
extra interval of staleness. The general reason: these are *sweeps*, not
initializers — none of them seeds state that later ticks depend on.

---

## 8. Acceptance evidence

Each PRD acceptance criterion maps to a command whose output is the evidence:

```
find services -path '*/tasks/task.go'                      # expect: empty
grep -rn 'func Register(l logrus.FieldLogger, ctx context.Context) func' services   # expect: empty
grep -rn 'tasks\.Register' services                         # expect: empty (incl. doc comments)
grep -rn ') Run()' services --include='*.go'                # expect: only atlas-parcel (2) + atlas-mts
grep -rn 'routine\.Register' services | wc -l               # expect: 41
grep -rn -B2 'routine\.Register' services --include=main.go | grep -c 'routine.Go'  # expect: 0  (F-1)
go test -race ./... in libs/atlas-routine                   # expect: PASS
tools/verify.sh                                             # flagless, expect: exit 0
```

The final acceptance item — atlas-maps exits cleanly on SIGTERM with all four tasks
registered — is covered by the `libs/atlas-routine` drain tests plus a manual
`docker compose` SIGTERM against atlas-maps, checking for four
`"Stopping task execution."` lines, no drain Warn, and process exit.

---

## 9. Risks

| Risk | Mitigation |
|---|---|
| A missed `routine.Go` wrapper (F-1) silently reintroduces the `Add`/`Wait` race — it compiles and usually works | The §8 grep asserts zero wrapped sites; it is part of acceptance, not review judgment |
| A `Run` body keeps using a captured `ctx` field the compiler no longer forces it to drop (e.g. field still used elsewhere in the struct) | Rule 2 applies to every one of the 34 files in inventory §C by name; per-service review checks the field, not just the build |
| 22 parallel units, 23 modules — a unit that builds locally but breaks the repo gate | Stage 3's flagless `tools/verify.sh` is the only completion evidence (NFR-7) |
| Shutdown log noise from newly-cancelled sweeps reads as a regression to an operator | Accepted and documented (§5.1); sweeps are idempotent |
| The drain masks a task that ignores its context — it now delays shutdown 5s instead of exiting instantly | The Warn names the task type (`%T`), making it actionable rather than invisible |
