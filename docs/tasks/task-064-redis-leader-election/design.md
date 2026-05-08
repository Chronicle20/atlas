# Redis Leader Election Library — Design

Status: Draft
Phase: 2 (design)
Created: 2026-05-08
Companion: `prd.md` (this folder)

---

## 1. Context recap

atlas-monsters registers six in-process sweep tickers in `services/atlas-monsters/atlas.com/monsters/main.go:88-93`. Each iterates global state (the in-memory `MonsterRegistry`, statuses, drop timers) and emits Kafka events as a side effect. Running the service with `replicas > 1` causes every event to be emitted N times. The PRD scopes the fix to a shared "elect-one-pod" primitive backed by Redis (the only datastore Atlas already mandates per service; works on both k8s and docker-compose).

This design fixes the implementation strategy, the public API, and the call-site shape. It also catalogues the alternatives so the choice is auditable later.

## 2. Decisions at a glance

| # | Decision | Choice |
|---|---|---|
| D-1 | Library boundary | New module `libs/atlas-lock` (not an extension of `libs/atlas-redis`) |
| D-2 | Implementation | Wrap `github.com/bsm/redislock` (do not reimplement NX-renew-release) |
| D-3 | Public API | `New(rc, name, opts...)` + `Run(ctx, fn)` + functional options |
| D-4 | Lease key | `atlas:lock:<name>` — service-scoped; rejection of empty/whitespace |
| D-5 | Task gating in atlas-monsters | `fn(leaderCtx)` registers the six tasks via `tasks.Register(l, leaderCtx)`; loss-of-leader cancels `leaderCtx`, all six goroutines exit naturally |
| D-6 | Kill switch | `MONSTER_LEADER_ELECTION_ENABLED` env var (default `true`); when `false`, register the six tasks at the outer `tdm.Context()` exactly as today |
| D-7 | Failover-during-fn | In-flight `Run()` calls of individual tasks finish naturally; the next `<-time.After(SleepTime)` tick is skipped because `leaderCtx` is already cancelled |
| D-8 | Panic-in-fn | Recovered inside library, logged ERROR, lease explicitly released, outer `Run` loop continues |
| D-9 | Observability | Four `promauto` counters in the library, labeled by `name` (and `reason` where applicable), no `service` label |
| D-10 | Split-brain | Documented in package doc + library README + integration PR description; no code mitigation (single-Redis design choice) |
| D-11 | RegistryAudit placement | Inside the gate (PRD §9 Q3 answered "yes") |
| D-12 | Metrics prefix option | Not exposed; library uses fixed `atlas_lock_*` prefix (PRD §9 Q4 answered "no") |
| D-13 | Renewal-failure threshold | Inherit `bsm/redislock` default; do not expose as a library knob in v1 |

## 3. D-1: Library boundary — `libs/atlas-lock` vs. extending `libs/atlas-redis`

Per the user's "audit existing libs before designing a new one" rule, this section enumerates `libs/atlas-*` and explains why none of them is the right home.

### 3.1 Existing libraries inspected

| Lib | Purpose | Overlap with leader election? |
|---|---|---|
| `atlas-redis` | Generic Redis registry (`Registry[K,V]`, `TenantRegistry[K,V]`), `Coalesced`/`TenantCoalesced` cache, key-namespacing helpers, **and a `Lock` type for short critical sections** (`lock.go`) | Closest neighbor — has a `Lock` type already. Detail in §3.2. |
| `atlas-tenant` | Tenant model + `tenant.WithContext` | None — no Redis primitives. |
| `atlas-retry` | Generic exponential-backoff retry | Tangential — could be reused by the acquire loop, but does not own state. |
| `atlas-kafka` | Kafka consumer/producer | None. |
| `atlas-database` | GORM connection wiring | None. |
| `atlas-model`, `atlas-rest`, `atlas-service`, `atlas-tracing`, `atlas-saga`, `atlas-script-core`, `atlas-socket`, `atlas-packet`, `atlas-opcodes`, `atlas-object-id`, `atlas-constants` | Domain and transport plumbing | None. |

### 3.2 Why not extend `libs/atlas-redis`

`libs/atlas-redis/lock.go` already defines a `Lock` type:

```go
// libs/atlas-redis/lock.go (current)
type Lock struct { client *goredis.Client; namespace string; ttl time.Duration }
func NewLock(...) *Lock
func (l *Lock) Acquire(ctx, key) (bool, error)   // SET NX EX, value=hardcoded "1"
func (l *Lock) Release(ctx, key) error           // DEL — anyone can release anyone's lock
func (l *Lock) Extend(ctx, key) (bool, error)    // EXPIRE
```

This is used by `services/atlas-messengers/atlas.com/messengers/messenger/processor.go:65` for a short-lived "create messenger" critical section: `atlas.NewLockWithTTL(client, "messenger-create", 10*time.Second)`. The caller acquires, performs ~one Redis read + one Redis write inside, then releases. Renewal is not needed; ownership is best-effort.

This existing API is **incorrect** as a general distributed lock — the hardcoded value `"1"` means any pod can `Release` any lease, so there is no fencing. But for atlas-messengers' "best-effort coalesce" usage it happens to be OK because the contention is rare and the cost of a wrong release is bounded.

We do **not** extend this type for leader election because:

1. **Different responsibility.** Leader election is a long-lived auto-renewing lease with a callback contract. The existing `Lock` is a short-lived NX-acquire-then-Release primitive. Mashing both onto one type produces a Frankenstein API surface where 70% of methods don't apply to either caller.
2. **Different correctness boundary.** The existing `Lock` is intentionally cheap-and-loose; promoting it to fenced-lease semantics either breaks atlas-messengers (which doesn't expect ownership tokens) or forks behavior with a flag, which is worse than just a separate type.
3. **Misuse-resistance goal.** PRD §3 user story 4 says "should be impossible to use incorrectly." The existing `Lock` exposes `Acquire`/`Release`/`Extend` directly — exactly the API surface the PRD says the new library MUST NOT expose to callers.
4. **Migration risk.** Replacing or refactoring the existing `Lock` to add fencing changes atlas-messengers' wire format (the value at rest in Redis), which is a separable cleanup task with its own tradeoffs. That cleanup is **not** in scope for task-063.

**Conclusion:** `libs/atlas-lock` is a new module. The existing `atlas-redis/lock.go` stays as-is for its single short-lived-critical-section caller. A future task can decide whether to migrate atlas-messengers onto `libs/atlas-lock` (using a separate `lock.Mutex`-style API if we add one) or simply fix `atlas-redis/lock.go` to use ownership tokens. Out of scope here.

### 3.3 Module layout

```
libs/atlas-lock/
├── go.mod
├── go.sum
├── README.md
├── leader.go              # LeaderElection type, New, Run, options
├── leader_test.go         # miniredis-based unit tests
├── metrics.go             # promauto counter declarations
└── doc.go                 # package doc with split-brain caveat
```

Module name: `github.com/Chronicle20/atlas/libs/atlas-lock`. Go version: 1.25.5 (matches sibling libs).

Direct deps:
- `github.com/bsm/redislock` (latest tagged release compatible with go-redis/v9)
- `github.com/redis/go-redis/v9` (already pinned across the repo at v9.19.0)
- `github.com/sirupsen/logrus` v1.9.4
- `github.com/prometheus/client_golang` v1.23.2

Test-only deps:
- `github.com/alicebob/miniredis/v2` v2.37.0
- `github.com/stretchr/testify` v1.11.1

`replace` directive in any consumer's `go.mod`: `replace github.com/Chronicle20/atlas/libs/atlas-lock => ../../../../libs/atlas-lock` (mirroring the existing pattern).

## 4. D-2: Wrap `bsm/redislock` vs. roll our own

### 4.1 What `bsm/redislock` gives us for free

- NX-acquire with random ownership token (per-Client UUID) — the fencing the existing `atlas-redis/lock.go` lacks.
- Lua-scripted refresh: only refreshes if the value still matches our token. (Prevents the "stolen-then-refreshed" foot-gun.)
- Lua-scripted release: same fencing — only deletes if the value matches our token.
- Tracks `go-redis/v9` versions; MIT licensed; ~500 LOC; no transitive deps beyond go-redis.

### 4.2 Roll-our-own — what we'd have to build

`SET NX PX` is two lines. The Lua scripts for fenced refresh and release are five lines each. The bookkeeping for ownership tokens is trivial. The renewal goroutine, the backoff loop, the panic recovery, the metrics — all of those are ours to write either way. The save by adopting `bsm/redislock` is roughly: ~25 LOC of correctness-critical Lua/SET-NX logic that's been battle-tested by other projects.

### 4.3 Decision and risk

Wrap `bsm/redislock`. We only depend on its `Client.Obtain` and `Lock.Refresh`/`Lock.Release` primitives. Public API surface of our library deliberately does not expose `*redislock.Lock` — the only way the dependency leaks into our types is the `Lock` we hold internally. This means a future swap (to handcrafted Lua, or to a different distributed-lock library) is a one-package change with no consumer-visible diff.

Risk: the project becomes unmaintained. Mitigation: the wrapped surface is small (~3 methods); if upstream stalls, vendoring or rewriting our wrapper internals takes a day. Worth the upfront save.

## 5. D-3, D-4: Public API

```go
// Package lock provides leader-election semantics on top of a single Redis
// instance. See doc.go for the single-Redis split-brain caveat.
package lock

import (
    "context"
    "errors"
    "time"

    goredis "github.com/redis/go-redis/v9"
    "github.com/sirupsen/logrus"
)

// LeaderElection runs a callback on exactly one pod for a named lease.
//
// Construction is cheap; only Run blocks. A LeaderElection instance
// MUST NOT have Run called more than once concurrently. Construct one
// per logical role per pod.
type LeaderElection struct {
    rc   *goredis.Client
    name string
    cfg  config
}

type config struct {
    ttl             time.Duration
    refreshInterval time.Duration
    backoff         time.Duration
    gracePeriod     time.Duration
    log             logrus.FieldLogger
}

type Option func(*config)

func WithTTL(d time.Duration) Option              // [5s, 5m]; default 30s
func WithRefreshInterval(d time.Duration) Option  // [1s, TTL/2]; default TTL/3
func WithBackoff(d time.Duration) Option          // [1s, 1m]; default 5s
func WithGracePeriod(d time.Duration) Option      // [1s, 30s]; default 5s
func WithLogger(l logrus.FieldLogger) Option

// New constructs a LeaderElection. Returns an error for empty or whitespace-only
// names, or for option values outside their allowed ranges.
func New(rc *goredis.Client, name string, opts ...Option) (*LeaderElection, error)

// Run blocks until ctx is cancelled.
//
// While the lease is held by this pod, fn is invoked once with a child
// context. The child context is cancelled when (a) the outer ctx is
// cancelled, (b) renewal fails (lease lost), or (c) fn panics (after
// the panic is recovered).
//
// fn is expected to return promptly when its ctx is cancelled. If fn
// is still running gracePeriod after ctx-cancel, Run logs a warning and
// proceeds with the acquire loop without waiting; the orphaned fn keeps
// running but its lease is gone.
//
// On lease loss, Run releases (best-effort) and re-enters the acquire
// loop after Backoff. On clean shutdown (outer ctx cancelled), Run
// performs an explicit, fenced Release before returning.
func (le *LeaderElection) Run(ctx context.Context, fn func(ctx context.Context)) error
```

### 5.1 Why no `Acquire` / `Release` / `Refresh` exposed

PRD §4.2 mandates this. Two reasons:

1. **Misuse-resistance.** Exposing `Acquire`/`Release` lets callers forget to refresh, forget to release, or call them in the wrong order. The whole point of the wrapper is to remove those failure modes.
2. **Implementation freedom.** Today we wrap `bsm/redislock`. Tomorrow we may swap the primitive. If `Run` is the only public entry point, swapping is invisible to consumers.

### 5.2 Why not return a result channel or status accessor

PRD §3 user story 4 plus §4.2's "thread-safe; one Run per instance" combine to push toward the simplest possible contract: blocking `Run`, callback-driven, no state to query. If a caller wants to know "am I the leader right now?" the answer is "you are if and only if `fn` is currently running on your goroutine" — which is structural, not a runtime query. Counters in §10 cover the operator-level question "who's the leader?".

### 5.3 Lease key shape (D-4)

`atlas:lock:<name>`. `<name>` is verbatim from the caller (after trim-and-non-empty validation). The library does not derive a hostname-or-region prefix; cross-environment isolation is a Redis-instance-level concern (each env has its own Redis, so namespacing is implicit).

Examples:
- atlas-monsters: `atlas:lock:monsters-sweep`
- (future) atlas-buffs: `atlas:lock:buffs-sweep`
- (future) atlas-drops: `atlas:lock:drops-sweep`

Library validates `name`: rejects empty, rejects all-whitespace, accepts everything else. It is the caller's responsibility to use a name that is unique across services.

## 6. Internals — the acquire-renew-release state machine

```
                       outer ctx cancelled
                              │
                              ▼
                       ┌──────────────┐
                       │   RETURN     │  (release if held)
                       └──────────────┘
                              ▲
                              │
                ┌─────────────┴──────────────┐
                │                            │
       ┌────────┴────────┐          ┌────────┴────────┐
       │   ACQUIRE-LOOP  │ ──────►  │   LEADER-LOOP   │
       │ (no lease yet)  │ acquire  │ (lease held)    │
       └────────┬────────┘  ok      └────────┬────────┘
                │                            │
                │ acquire fails              │ renewal fails
                │ (held / redis err)         │ OR fn returns
                │                            │ OR panic
                ▼                            ▼
            backoff                       release
                │                            │
                └────────┬───────────────────┘
                         ▼
                    (loop back)
```

### 6.1 Goroutine layout inside `Run`

`Run` runs on the caller's goroutine — it does not spawn a goroutine for itself. Inside the LEADER-LOOP block:

- One goroutine is spawned for `fn(leaderCtx)`. This is the user's work.
- One goroutine is spawned for the renewal ticker. It calls `redislock.Lock.Refresh` every `RefreshInterval`. On any non-recoverable error, it cancels `leaderCtx` and exits. On a transient error, it increments the `renew_failed_total` counter and continues; `bsm/redislock` will eventually fail-cancel us out of the lease via the lease's own TTL.

The two goroutines coordinate via `leaderCtx.Done()`. After both have exited (fn-done channel + renewer exited), `Run` performs:

1. `redislock.Lock.Release()` (fenced; ignores error if the lease has already expired).
2. Increment `lost_total{reason}` with the appropriate reason.
3. Sleep `Backoff`.
4. Re-enter ACQUIRE-LOOP.

### 6.2 Grace period

When `leaderCtx` is cancelled, the renewer exits immediately. The fn goroutine is given `GracePeriod` (default 5s) to return. If it does not, `Run` logs `WARN` and proceeds without waiting — the orphan fn keeps running but its lease is gone (and its outer-ctx-derived child contexts are still healthy, so it can finish if it wants). This avoids `Run` becoming permanently stuck behind a runaway `fn`.

### 6.3 Panic recovery

The fn goroutine is wrapped in:

```go
go func() {
    defer func() {
        if r := recover(); r != nil {
            le.cfg.log.WithField("panic", r).Errorf("Leader fn panic for [%s].", le.name)
            cancel(leaderCtx)
        }
        close(fnDone)
    }()
    fn(leaderCtx)
}()
```

The recovery does **not** escape `Run` — the outer loop continues. If the `fn` would otherwise blast the process via `os.Exit` or `runtime.Goexit`, the library makes no guarantees; `defer` with recover handles only `panic`. (This matches Go runtime semantics; we don't try to be cleverer.)

### 6.4 Acquire failures

`bsm/redislock.Client.Obtain` returns one of:

- `nil, nil` — lock acquired (we get a `*redislock.Lock`)
- `nil, redislock.ErrNotObtained` — held by another pod
- `nil, <other error>` — Redis problem (timeout, connection, scripting)

We classify and emit `acquire_failed_total{name, reason}` accordingly:

- `held_by_other` — `redislock.ErrNotObtained`
- `redis_error` — anything else

The acquire loop retries forever until either `outer ctx` is cancelled or acquisition succeeds. Backoff between attempts: `Backoff` (default 5s). No jitter in v1 — the loops on different pods are dephased by their startup-time differences, which is enough for our scale (handful of pods per service).

## 7. D-5, D-6, D-7: atlas-monsters integration

### 7.1 The shape of the change

`main.go:88-93` today:

```go
tasks.Register(l, tdm.Context())(monster.NewRegistryAudit(l, time.Second*30))
tasks.Register(l, tdm.Context())(monster.NewStatusExpirationTask(l, tdm.Context(), time.Second))
tasks.Register(l, tdm.Context())(monster.NewDropTimerTask(l, tdm.Context(), time.Second))
tasks.Register(l, tdm.Context())(monster.NewMonsterAggroDecayTask(l, tdm.Context(), monster.AggroSweepInterval))
tasks.Register(l, tdm.Context())(monster.NewMonsterSkillPickerSweepTask(l, tdm.Context(), monster.MonsterSkillPickerSweepInterval))
tasks.Register(l, tdm.Context())(monster.NewMonsterRecoveryTask(l, tdm.Context(), monster.MonsterRecoveryInterval))
```

After:

```go
registerSweepTasks := func(l logrus.FieldLogger, ctx context.Context) {
    tasks.Register(l, ctx)(monster.NewRegistryAudit(l, time.Second*30))
    tasks.Register(l, ctx)(monster.NewStatusExpirationTask(l, ctx, time.Second))
    tasks.Register(l, ctx)(monster.NewDropTimerTask(l, ctx, time.Second))
    tasks.Register(l, ctx)(monster.NewMonsterAggroDecayTask(l, ctx, monster.AggroSweepInterval))
    tasks.Register(l, ctx)(monster.NewMonsterSkillPickerSweepTask(l, ctx, monster.MonsterSkillPickerSweepInterval))
    tasks.Register(l, ctx)(monster.NewMonsterRecoveryTask(l, ctx, monster.MonsterRecoveryInterval))
}

if leaderEnabled() {
    le, err := lock.New(rc, "monsters-sweep", buildLockOptions(l)...)
    if err != nil { l.WithError(err).Fatal("Unable to construct LeaderElection.") }

    go func() {
        if err := le.Run(tdm.Context(), func(leaderCtx context.Context) {
            registerSweepTasks(l, leaderCtx)
            <-leaderCtx.Done()
        }); err != nil {
            l.WithError(err).Errorf("LeaderElection.Run exited with error.")
        }
    }()
} else {
    l.Warnf("MONSTER_LEADER_ELECTION_ENABLED=false — sweep tasks run unconditionally on this pod.")
    registerSweepTasks(l, tdm.Context())
}
```

The crucial property: **`tasks.Register` already obeys `<-ctx.Done()`**. Look at `tasks/task.go:16-30` — its loop is:

```go
for {
    select {
    case <-ctx.Done():
        l.Infof("Stopping task execution.")
        return
    case <-time.After(t.SleepTime()):
        t.Run()
    }
}
```

When `leaderCtx` is cancelled, every one of the six task goroutines hits `<-ctx.Done()` on the next select cycle and exits cleanly. No new code needed inside `tasks.Register` to make this work.

In-flight `Run()` calls finish naturally (they hold the goroutine; the next `<-time.After` is what would have been cancelled). This satisfies PRD §4.4.

On re-acquire, `fn` runs again and calls `registerSweepTasks(l, newLeaderCtx)`. Six fresh goroutines are spawned. The previous generation is gone.

### 7.2 The `<-leaderCtx.Done()` block inside fn

`fn` registers six goroutine-based tasks (fire-and-forget) and then must block — otherwise `fn` returns immediately, `Run` releases the lease, and the six goroutines run untethered with the wrong ctx-loss semantics.

Blocking on `<-leaderCtx.Done()` is the contract: fn returns iff the lease is lost. This matches the §6 state machine.

### 7.3 Why not pass the leader gate inside each task

Considered alternative: each `*Task.Run()` checks an `IsLeader()` flag at the top and bails out if false. Rejected because:

- Pierces task isolation. Every task author has to remember to add the gate.
- Lossier failure mode: a task can do "half" the sweep before the gate reads `false` mid-call. The ctx-cancel approach skips entire ticks, which is cleaner.
- Cardinality of tests doubles (each task gets a "bypass when not leader" test).

### 7.4 Why not a single supervisor goroutine driving all six tasks inline

Considered alternative: replace `tasks.Register` with a hand-rolled select-loop inside `fn` that drives all six task `Run()` methods on their own intervals. Rejected because:

- Reimplements `tasks.Register` for no benefit.
- Couples leader-election to the sweep ticker semantics — future tasks can no longer use `tasks.Register` if they want the same gating.
- The chosen approach is composable: any `tasks.Task`-shaped type "just works" under the gate.

### 7.5 Kill switch (D-6)

`MONSTER_LEADER_ELECTION_ENABLED` env var.

- Unset or `true` → leader gate active.
- `false` → six tasks register at `tdm.Context()` exactly as today (current behavior preserved).
- Any other value → log warning, treat as `true`.

The kill switch protects:
- Single-pod docker-compose deployments where Redis hiccups during pod startup. If Redis is briefly unavailable, the acquire loop will retry forever — but that's the right behavior in production. In docker-compose, an operator can set `=false` to skip the dance entirely.
- Emergency rollback if leader election misbehaves in production. Set the env var, re-deploy, behavior is identical to pre-task-063.

### 7.6 What does NOT change in atlas-monsters

- HTTP routes (`monster.InitResource`, `world.InitResource`, `/metrics`, `/debug/consumers`) — unchanged.
- Kafka consumers (`monster2.InitConsumers`, `_map.InitConsumers`) — unchanged. These are pod-local: each pod consumes its assigned partitions and acts on them. Per-pod consumers are correct by design.
- Kafka producers — unchanged.
- The `Task` interface in `tasks/task.go` — unchanged. PRD §4.4 explicitly preserves "the public behavior of each task — its interval, its `Run()` semantics — must be unchanged."
- `monster/registry.go`, `monster/aggro_task.go`, `monster/picker_task.go`, `monster/recovery_task.go`, `monster/status_task.go`, `monster/drop_timer_task.go`, `monster/task.go` (RegistryAudit) — unchanged.

## 8. Configuration

### 8.1 Env vars (atlas-monsters consumer)

| Env var | Purpose | Default | Validation |
|---|---|---|---|
| `MONSTER_LEADER_ELECTION_ENABLED` | Kill switch (§7.5) | `true` | Parse as boolean; non-bool → warn + treat as true |
| `MONSTER_LEADER_TTL` | Lease TTL | `30s` | `time.ParseDuration`; out-of-range [5s, 5m] → warn + use default |
| `MONSTER_LEADER_REFRESH` | Renewal cadence | `10s` (= TTL/3) | `time.ParseDuration`; out-of-range [1s, TTL/2] → warn + use default |
| `MONSTER_LEADER_BACKOFF` | Acquire-retry sleep | `5s` | `time.ParseDuration`; out-of-range [1s, 1m] → warn + use default |

Validation matches the atlas-monsters task-060 pattern (env loader logs `WARN` and falls back to default).

### 8.2 Library option ranges

The library's `WithXxx` validators enforce the same ranges. Out-of-range values returned from `New(...)` as an error (not silently corrected) — caller-side validation is the consumer's responsibility, library-side validation is a defense-in-depth check.

## 9. D-9: Observability

### 9.1 Counters

```go
// libs/atlas-lock/metrics.go
var (
    acquiredTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "atlas_lock_acquired_total",
            Help: "Number of times this pod transitioned from non-leader to leader for a given lease name.",
        },
        []string{"name"},
    )
    lostTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "atlas_lock_lost_total",
            Help: "Number of times this pod transitioned from leader to non-leader.",
        },
        []string{"name", "reason"}, // reason ∈ {renew_failed, context_cancelled, released, panic}
    )
    renewFailedTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "atlas_lock_renew_failed_total",
            Help: "Number of single renewal attempts that failed (does not always cause leader loss).",
        },
        []string{"name"},
    )
    acquireFailedTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "atlas_lock_acquire_failed_total",
            Help: "Number of failed acquire attempts (held by other pod, or Redis error).",
        },
        []string{"name", "reason"}, // reason ∈ {held_by_other, redis_error}
    )
)
```

Cardinality: 1 series per `name` per pod per counter (or 4 with reason). Trivially bounded — Atlas has on the order of a dozen services.

No `service` label. The pod's own service name is implicit in the deployment-level label set Prometheus applies (`job`, `pod`, etc.); duplicating it in a series label would explode cardinality without clarity gain.

### 9.2 Logging

State transitions are logged at `Info`:

- `Acquired leader for [<name>].` — on transition to leader.
- `Lost leader for [<name>] (reason: <reason>).` — on transition from leader.
- `Released leader for [<name>] on shutdown.` — on outer-ctx cancel.

Renewal attempts are logged at `Debug` only. Renewal failures (the individual ones, not the lease-loss event) are logged at `Warn`.

Acquire-loop retries: the first failure per loss-cycle is logged at `Info`; subsequent retries against the same root cause are logged at `Debug` to avoid a noise flood when Redis is down.

### 9.3 Operator-facing query (recipe)

> "Is there a leader for `monsters-sweep` right now?"
>
> `rate(atlas_lock_acquire_failed_total{name="monsters-sweep", reason="held_by_other"}[1m]) > 0`
>
> If the rate is positive, at least one pod is failing to acquire because someone else holds it — i.e., there is a leader. If the rate is zero across all pods AND `atlas_lock_acquired_total` is also zero recently, the lease is free (no one is currently leading).

## 10. Testing strategy

### 10.1 Library unit tests (`libs/atlas-lock/leader_test.go`)

All driven by `miniredis` — no real Redis needed.

| Test | Scenario | Asserts |
|---|---|---|
| `Test_Run_AcquireThenReleaseOnShutdown` | One LE; outer ctx cancelled | fn invoked once; lease key gone after Run returns |
| `Test_Run_TwoCompetitors_OneAcquires` | Two LEs same name; both call Run | exactly one fn runs at a time; standby's fn never runs while leader is up |
| `Test_Run_RenewalExtendsLeasePastTTL` | One LE; sleep > TTL; verify lease still held | renewer kept the lease alive |
| `Test_Run_LeaseLossCancelsInnerCtx` | One LE; force `miniredis.FastForward(TTL+1s)` | inner ctx cancelled; fn returned |
| `Test_Run_PanicInFn_RecoveredAndReleased` | fn panics; outer ctx cancelled later | Run does not propagate panic; lease released after panic; counter `lost_total{reason="panic"}` incremented |
| `Test_Run_OuterCtxCancelTriggersRelease` | outer ctx cancelled mid-leader | lease key gone within < TTL |
| `Test_Run_GracePeriodHonored` | fn ignores ctx-cancel for >GracePeriod | Run logs warn, proceeds without waiting forever |
| `Test_New_RejectsEmptyName` | `New(rc, "", ...)`, `New(rc, "  ", ...)` | both return error |
| `Test_New_OutOfRangeOptions` | TTL=1s, TTL=10m, RefreshInterval=TTL | each returns a structured error |
| `Test_Run_FailoverWithinTTLPlusBackoff` | Two LEs; kill leader's connection; assert standby acquires | lease takeover within `TTL + backoff + epsilon` |
| `Test_Run_AcquireFailedClassification` | redis disconnected at Obtain | `acquire_failed_total{reason="redis_error"}` increments |

`go test -race ./...` clean.

### 10.2 atlas-monsters integration tests

Existing tests must continue to pass without modification. The kill switch (default in test contexts will be `false`, set explicitly) means no test gains a Redis dependency it did not previously have.

New tests:
- `main_test.go` (or equivalent fixture-style test) — wire two LEs against a shared miniredis with name=`monsters-sweep`, drive a fake task with a counter, verify the counter only advances on one LE at a time.
- Kill-switch test — `MONSTER_LEADER_ELECTION_ENABLED=false`, no Redis available, verify the six tasks register and run.

### 10.3 What is NOT tested

- Real Redis failover (Sentinel/cluster). That's an integration concern at deployment time. The library makes no claim to behave correctly across real Redis primary→replica handover beyond the documented split-brain caveat. We don't simulate it.
- Mass concurrent contention (>10 LEs on the same name). Out of scope; production uses 1–3 LEs per name.
- Long-running soak. Manual verification per PRD §10 acceptance criterion.

## 11. Failure modes and degraded operation

| Failure | Library behavior | Caller-visible behavior |
|---|---|---|
| Redis unreachable at startup | Acquire loop retries forever on `Backoff` cadence | Pod stays up. HTTP, Kafka consumers, Kafka producers all healthy. Sweep tasks idle until Redis returns. |
| Redis dies mid-leader | Renewer fails → leaderCtx cancelled → fn returns → Release best-effort (will fail; that's OK; lease will TTL out anyway) → re-enter acquire loop | Sweep tasks pause. Worst-case `TTL + backoff` (default 35s) before another pod acquires. |
| Redis recovers after blip | Acquire-loop attempt succeeds | Sweep tasks resume on whichever pod won the race. |
| `fn` panics | Recovered; ERROR logged; lease explicitly released; loop continues on Backoff | Sweep tasks pause for `Backoff` (default 5s) before re-acquire attempt. |
| Pod crash (SIGKILL) | No graceful release; lease TTLs out after `TTL` | Standby acquires after `TTL + backoff` (worst case 35s). |
| Pod graceful shutdown (SIGTERM) | Outer ctx cancelled → Run releases → returns | Standby acquires within Backoff (~5s). |
| Network partition (pod loses Redis but lives) | Renewer fails → leaderCtx cancelled → fn returns → no successful Release; lease TTLs out | Standby pod acquires after `TTL + backoff`. Original pod re-enters acquire loop and will re-acquire when partition heals. |
| Two Redis primaries (split-brain after failover) | Each pod refreshes against its own primary; both believe they're leader for ~1–5s | Documented caveat (§12). Sweep tasks emit duplicates briefly; downstream Kafka consumers must tolerate this (already required for at-least-once Kafka semantics). |
| Misconfigured `name` | `New` returns error | Caller must `Fatal` (atlas-monsters does). |

## 12. Single-Redis split-brain — documentation requirements

The caveat from PRD §8.4 must appear in:

1. **`libs/atlas-lock/doc.go`** — package-level Go doc comment.
2. **`libs/atlas-lock/README.md`** — top-section "Correctness boundary" block, including a list of unsuitable workloads (financial transactions, exclusive resource claims with no idempotency).
3. **`services/atlas-monsters` integration commit message** — explicit acknowledgment that the chosen workloads are at-least-once-tolerant.
4. **PR description for task-063** — link to PRD §8.4 plus a one-paragraph summary so future adopters reading the PR history don't blindly copy the pattern.

The wording must include the phrase "Redlock is out of scope" so future readers find the link.

## 13. PRD open questions — resolutions

| PRD §9 question | Resolution in this design |
|---|---|
| Library boundary (extend atlas-redis vs. new module) | New module `libs/atlas-lock`. See §3. |
| API for multi-role pods | Acceptable: caller constructs N `LeaderElection` instances and runs N `Run` goroutines. Revisit if N grows; not designed-in for v1. |
| Renewal-failure semantics — expose threshold? | No. Inherit `bsm/redislock` policy. The single renewal-failure counter (`renew_failed_total`) is observable; the lost-leader counter (`lost_total{reason="renew_failed"}`) tells operators when a streak ended in lease loss. |
| RegistryAudit inside or outside the gate? | Inside. Aligns with PRD §9 provisional answer. Outside-gate diagnostic visibility is achievable via per-pod Prometheus pod-level metrics; the lease-gated audit is correct and matches "exactly one pod runs sweep work." |
| Metrics-prefix option? | No. Library uses fixed `atlas_lock_*` with `name` label. Dashboard-authoring ergonomics are fine with one shared family of series. |

## 14. Out-of-scope / follow-up work

The following are explicitly out of scope for task-063 and tracked in `docs/TODO.md` as separate tasks (per PRD §10 acceptance criterion):

- **Per-service adoption** of `libs/atlas-lock` for the 14 candidate services in PRD §7.3 (atlas-buffs, atlas-ban, atlas-drops, atlas-pets, atlas-skills, atlas-reactors, atlas-maps, atlas-merchant, atlas-guilds, atlas-account, atlas-world, atlas-invites, atlas-expressions, atlas-character).
- **Migration of `atlas-redis/lock.go`** to use ownership tokens (or migration of atlas-messengers onto a `lock.Mutex` short-lived API). Decision deferred — depends on whether other short-lived-critical-section callers appear.
- **A short-lived `Mutex`-style API** in `libs/atlas-lock` for transactional critical sections distinct from leader election. Designed-in only when a use case appears.
- **Multi-Redis Redlock**. Out of scope per PRD §2 non-goals.
- **Per-tenant leader election**. Out of scope per PRD §2 non-goals.
- **Replacing sweep tasks with event-driven alternatives** (Redis keyspace notifications, Kafka deadlines). Out of scope per PRD §2 non-goals.

## 15. Summary

Task-063 introduces `libs/atlas-lock`, a misuse-resistant leader-election library that wraps `bsm/redislock`. The single public entry point is `Run(ctx, fn)`. atlas-monsters' six sweep tasks are gated by a single shared `monsters-sweep` lease via the trivial composition `fn(leaderCtx) { tasks.Register(l, leaderCtx)(...); <-leaderCtx.Done() }` — leveraging the existing `tasks.Register` ctx-cancel semantics with zero changes to the `Task` interface or any task body. Kill switch via env var preserves docker-compose single-pod behavior. Four `atlas_lock_*` counters give operators failover visibility. The single-Redis split-brain caveat is documented in code, README, commit, and PR. Per-service adoption follow-ups are tracked in `docs/TODO.md`.
