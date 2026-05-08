# atlas-lock

Leader-election semantics on top of a single Redis instance, wrapping
`bsm/redislock`. The public API is one type and one method:

```go
le, err := lock.New(rc, "monsters-sweep",
    lock.WithTTL(30*time.Second),
    lock.WithRefreshInterval(10*time.Second),
    lock.WithBackoff(5*time.Second),
)
if err != nil {
    return err
}

go func() {
    err := le.Run(ctx, func(leaderCtx context.Context) {
        // This block runs ONLY while this pod holds the lease.
        registerSweepTasks(l, leaderCtx)
        <-leaderCtx.Done() // exit when the lease is lost.
    })
    if err != nil {
        l.WithError(err).Errorf("LeaderElection.Run exited with error.")
    }
}()
```

## Correctness boundary — single-Redis split-brain caveat

This library uses a single Redis instance for the lease key. During a Redis
primary→replica failover the lease key is replicated asynchronously. For
1–5 seconds two pods can each believe they hold the lease. Use this library
ONLY for workloads whose downstream consumers already tolerate at-least-once
delivery.

**Suitable workloads:** Sweep tasks emitting Kafka events whose consumers
already handle duplicates (Atlas's primary use case).

**Unsuitable workloads:** Financial transactions, exclusive resource claims
without idempotency at the consumer, anything where duplicate execution is
unsafe.

**Multi-Redis Redlock is out of scope** — Atlas runs a single Redis instance
per environment, and the additional safety isn't worth the operational
complexity for sweep workloads.

## Configuration

| Option | Default | Range | Purpose |
|---|---|---|---|
| `WithTTL` | 30s | [5s, 5m] | Lease TTL |
| `WithRefreshInterval` | 10s (= TTL/3) | [1s, TTL/2] | Renewal cadence |
| `WithBackoff` | 5s | [1s, 1m] | Wait between failed acquire attempts |
| `WithGracePeriod` | 5s | [1s, 30s] | Wait for fn to return after lease loss |
| `WithLogger` | `logrus.New()` | n/a | Override logger |

Out-of-range options return an error from `New`. The constructor does not
silently clamp.

## Observability

Four `promauto` counters labeled by lease `name`:

| Counter | Labels | Meaning |
|---|---|---|
| `atlas_lock_acquired_total` | `name` | Pod transitioned non-leader → leader |
| `atlas_lock_lost_total` | `name`, `reason` | Pod transitioned leader → non-leader. `reason` ∈ {`renew_failed`, `context_cancelled`, `released`, `panic`} |
| `atlas_lock_renew_failed_total` | `name` | A single renewal attempt failed (transient) |
| `atlas_lock_acquire_failed_total` | `name`, `reason` | An acquire attempt failed. `reason` ∈ {`held_by_other`, `redis_error`} |

State transitions are logged at INFO. Renewal attempts at DEBUG. Renewal
failures at WARN.

## Operator recipe

> "Is there a leader for `monsters-sweep` right now?"

```promql
rate(atlas_lock_acquire_failed_total{name="monsters-sweep", reason="held_by_other"}[1m]) > 0
```

If positive, at least one pod is failing to acquire because someone else
holds the lease — i.e., there is a leader.

## Misuse-resistance

- The library exposes no `Acquire`/`Release`/`Refresh` methods. The renewal
  loop is owned by `Run`. Callers cannot forget to renew or release.
- `fn` is invoked with a child context. Lease loss cancels the child;
  outer-ctx cancel cancels the child; the cleanup path in `Run` performs an
  explicit fenced `Release` before returning.
- One `LeaderElection` instance MUST NOT have `Run` called more than once
  concurrently. Construct one per logical role per pod.
- A panic inside `fn` is recovered, logged at ERROR, the lease is released,
  and the outer `Run` loop continues. Panics do not propagate.
