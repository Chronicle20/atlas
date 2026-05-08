# Redis Leader Election Library Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build `libs/atlas-lock`, a misuse-resistant Go module wrapping `bsm/redislock`, exposing `LeaderElection.Run(ctx, fn)`. Integrate it into `services/atlas-monsters/atlas.com/monsters/main.go` so the six existing sweep tickers run on exactly one elected pod, with a `MONSTER_LEADER_ELECTION_ENABLED` kill-switch.

**Architecture:** New module at `libs/atlas-lock/` mirroring the layout of `libs/atlas-tenant`/`libs/atlas-redis`. The library wraps `bsm/redislock` and owns acquire/renew/release; the only public entry point is `Run(ctx, fn)`. `bsm/redislock` is the dependency boundary — its types never leak into our public API. atlas-monsters imports the new module via a `replace` directive sibling-style, gates the existing `tasks.Register` calls behind `LeaderElection.Run`, and relies on the existing `tasks.Register` ctx-cancel semantics for clean teardown.

**Tech Stack:** Go 1.25.5, `github.com/bsm/redislock` v0.9.4, `github.com/redis/go-redis/v9` v9.19.0, `github.com/sirupsen/logrus` v1.9.4, `github.com/prometheus/client_golang` v1.23.2, `github.com/alicebob/miniredis/v2` v2.37.0 (test), `github.com/stretchr/testify` v1.11.1 (test).

**All paths are absolute under the worktree.** Working directory for every step: `<worktree-root>/`.

---

## Task 1: Scaffold `libs/atlas-lock` module

**Goal:** Create the empty Go module, add it to `go.work`, verify it builds.

**Files:**
- Create: `libs/atlas-lock/go.mod`
- Create: `libs/atlas-lock/doc.go`
- Modify: `go.work`

- [ ] **Step 1: Create the module file**

`libs/atlas-lock/go.mod`:

```
module github.com/Chronicle20/atlas/libs/atlas-lock

go 1.25.5

require (
	github.com/alicebob/miniredis/v2 v2.37.0
	github.com/bsm/redislock v0.9.4
	github.com/prometheus/client_golang v1.23.2
	github.com/redis/go-redis/v9 v9.19.0
	github.com/sirupsen/logrus v1.9.4
	github.com/stretchr/testify v1.11.1
)
```

- [ ] **Step 2: Stub the package doc**

`libs/atlas-lock/doc.go`:

```go
// Package lock provides leader-election semantics on top of a single Redis
// instance.
//
// CORRECTNESS BOUNDARY (single-Redis split-brain caveat):
// During a Redis primary→replica failover the lease key is replicated
// asynchronously. For 1–5 seconds two pods can each believe they hold the
// lease. Use this library only for workloads whose downstream consumers
// already tolerate at-least-once delivery (Atlas sweep tasks emitting Kafka
// events qualify; financial transactions and exclusive resource claims do
// not). Multi-Redis Redlock is out of scope.
package lock
```

- [ ] **Step 3: Add the module to `go.work`**

Open `go.work` and add `./libs/atlas-lock` to the `use (...)` block, alphabetical with the other `./libs/atlas-*` entries (between `./libs/atlas-kafka` and `./libs/atlas-model`).

Diff sketch:

```
 	./libs/atlas-kafka
+	./libs/atlas-lock
 	./libs/atlas-model
```

- [ ] **Step 4: Resolve dependencies and verify it builds**

Run:

```
cd libs/atlas-lock && go mod tidy && go build ./...
```

Expected: clean exit. `go.sum` is created. No source files yet, so `go build` reports `package github.com/Chronicle20/atlas/libs/atlas-lock: build constraints exclude all Go files` is acceptable here only if `doc.go` is missing — with `doc.go` present, build is clean.

If `go mod tidy` rewrites the require block (e.g. to add indirect deps from miniredis/testify), accept the changes.

- [ ] **Step 5: Commit**

```
cd <worktree-root>
git add libs/atlas-lock/go.mod libs/atlas-lock/go.sum libs/atlas-lock/doc.go go.work
git commit -m "feat(atlas-lock): scaffold module with package doc"
```

---

## Task 2: Functional options and config struct

**Goal:** Define `Option`, `config`, and `WithXxx` setters with documented defaults. No `LeaderElection` type yet.

**Files:**
- Create: `libs/atlas-lock/leader.go`
- Create: `libs/atlas-lock/leader_test.go`

- [ ] **Step 1: Write the failing test**

`libs/atlas-lock/leader_test.go`:

```go
package lock

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestOptions_DefaultsApplied(t *testing.T) {
	cfg := config{}
	applyDefaults(&cfg)
	require.Equal(t, 30*time.Second, cfg.ttl)
	require.Equal(t, 10*time.Second, cfg.refreshInterval)
	require.Equal(t, 5*time.Second, cfg.backoff)
	require.Equal(t, 5*time.Second, cfg.gracePeriod)
	require.NotNil(t, cfg.log)
}

func TestOptions_OverridesApplied(t *testing.T) {
	cfg := config{}
	applyDefaults(&cfg)
	WithTTL(2 * time.Minute)(&cfg)
	WithRefreshInterval(20 * time.Second)(&cfg)
	WithBackoff(15 * time.Second)(&cfg)
	WithGracePeriod(10 * time.Second)(&cfg)
	l := logrus.New()
	WithLogger(l)(&cfg)

	require.Equal(t, 2*time.Minute, cfg.ttl)
	require.Equal(t, 20*time.Second, cfg.refreshInterval)
	require.Equal(t, 15*time.Second, cfg.backoff)
	require.Equal(t, 10*time.Second, cfg.gracePeriod)
	require.Same(t, l, cfg.log)
}
```

- [ ] **Step 2: Run test to verify it fails**

```
cd libs/atlas-lock && go test -run 'TestOptions' ./...
```

Expected: FAIL — `undefined: config`, `undefined: applyDefaults`, etc.

- [ ] **Step 3: Implement options**

`libs/atlas-lock/leader.go`:

```go
package lock

import (
	"time"

	"github.com/sirupsen/logrus"
)

// Defaults exposed for documentation; consumers should pass options explicitly.
const (
	DefaultTTL             = 30 * time.Second
	DefaultRefreshInterval = 10 * time.Second // TTL / 3
	DefaultBackoff         = 5 * time.Second
	DefaultGracePeriod     = 5 * time.Second
)

type config struct {
	ttl             time.Duration
	refreshInterval time.Duration
	backoff         time.Duration
	gracePeriod     time.Duration
	log             logrus.FieldLogger
}

// Option mutates a config. Use the WithXxx constructors to obtain Options.
type Option func(*config)

// WithTTL sets the lease TTL. Allowed range: [5s, 5m]. Default: 30s.
func WithTTL(d time.Duration) Option { return func(c *config) { c.ttl = d } }

// WithRefreshInterval sets the renewal cadence. Allowed range: [1s, TTL/2]. Default: TTL/3.
func WithRefreshInterval(d time.Duration) Option {
	return func(c *config) { c.refreshInterval = d }
}

// WithBackoff sets the wait between failed acquire attempts. Allowed range: [1s, 1m]. Default: 5s.
func WithBackoff(d time.Duration) Option { return func(c *config) { c.backoff = d } }

// WithGracePeriod sets how long Run waits for fn to return after lease loss
// before logging a warning and proceeding. Allowed range: [1s, 30s]. Default: 5s.
func WithGracePeriod(d time.Duration) Option { return func(c *config) { c.gracePeriod = d } }

// WithLogger overrides the default logrus.New() logger.
func WithLogger(l logrus.FieldLogger) Option { return func(c *config) { c.log = l } }

func applyDefaults(c *config) {
	c.ttl = DefaultTTL
	c.refreshInterval = DefaultRefreshInterval
	c.backoff = DefaultBackoff
	c.gracePeriod = DefaultGracePeriod
	c.log = logrus.New()
}
```

- [ ] **Step 4: Run tests to verify pass**

```
cd libs/atlas-lock && go test -run 'TestOptions' ./...
```

Expected: PASS for both subtests.

- [ ] **Step 5: Commit**

```
git add libs/atlas-lock/leader.go libs/atlas-lock/leader_test.go libs/atlas-lock/go.sum
git commit -m "feat(atlas-lock): functional options with documented defaults"
```

---

## Task 3: `LeaderElection` type, `New()` with validation

**Goal:** Add the public type, name and option-range validation. No `Run` yet.

**Files:**
- Modify: `libs/atlas-lock/leader.go`
- Modify: `libs/atlas-lock/leader_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `leader_test.go`:

```go
import (
	// existing imports plus:
	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

func newTestClient(t *testing.T) (*goredis.Client, *miniredis.Miniredis) {
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rc.Close() })
	return rc, mr
}

func TestNew_RejectsNilClient(t *testing.T) {
	_, err := New(nil, "x")
	require.Error(t, err)
}

func TestNew_RejectsEmptyName(t *testing.T) {
	rc, _ := newTestClient(t)
	_, err := New(rc, "")
	require.Error(t, err)
	_, err = New(rc, "   ")
	require.Error(t, err)
}

func TestNew_RejectsOutOfRangeOptions(t *testing.T) {
	rc, _ := newTestClient(t)

	cases := []struct {
		name string
		opt  Option
	}{
		{"ttl-too-low", WithTTL(time.Second)},
		{"ttl-too-high", WithTTL(10 * time.Minute)},
		{"refresh-too-low", WithRefreshInterval(0)},
		{"refresh-too-high-vs-ttl", WithRefreshInterval(20 * time.Second)}, // > TTL/2 = 15s
		{"backoff-too-low", WithBackoff(0)},
		{"backoff-too-high", WithBackoff(2 * time.Minute)},
		{"grace-too-low", WithGracePeriod(0)},
		{"grace-too-high", WithGracePeriod(2 * time.Minute)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := New(rc, "x", tc.opt)
			require.Error(t, err)
		})
	}
}

func TestNew_AcceptsValidConfig(t *testing.T) {
	rc, _ := newTestClient(t)
	le, err := New(rc, "monsters-sweep",
		WithTTL(30*time.Second),
		WithRefreshInterval(10*time.Second),
		WithBackoff(5*time.Second),
		WithGracePeriod(5*time.Second),
	)
	require.NoError(t, err)
	require.NotNil(t, le)
	require.Equal(t, "atlas:lock:monsters-sweep", le.keyPath())
}
```

- [ ] **Step 2: Run tests to verify they fail**

```
cd libs/atlas-lock && go test -run 'TestNew_' ./...
```

Expected: FAIL — `undefined: New`, `undefined: LeaderElection`, etc.

- [ ] **Step 3: Implement `LeaderElection` and `New`**

Add to `leader.go` (after the options):

```go
import (
	"errors"
	"fmt"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

const keyPrefix = "atlas:lock:"

// LeaderElection runs a callback on exactly one pod for a named lease.
//
// Construction is cheap; only Run blocks. A single LeaderElection instance
// MUST NOT have Run called more than once concurrently. Construct one per
// logical role per pod.
type LeaderElection struct {
	rc   *goredis.Client
	name string
	cfg  config
}

// New constructs a LeaderElection bound to a Redis client and a service-scoped
// lease name. Returns an error for nil clients, empty/whitespace-only names,
// or option values outside the allowed ranges.
func New(rc *goredis.Client, name string, opts ...Option) (*LeaderElection, error) {
	if rc == nil {
		return nil, errors.New("lock: nil redis client")
	}
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("lock: name must be non-empty and not all-whitespace")
	}
	cfg := config{}
	applyDefaults(&cfg)
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.ttl < 5*time.Second || cfg.ttl > 5*time.Minute {
		return nil, fmt.Errorf("lock: TTL %s out of range [5s, 5m]", cfg.ttl)
	}
	if cfg.refreshInterval < time.Second || cfg.refreshInterval > cfg.ttl/2 {
		return nil, fmt.Errorf("lock: RefreshInterval %s out of range [1s, TTL/2]", cfg.refreshInterval)
	}
	if cfg.backoff < time.Second || cfg.backoff > time.Minute {
		return nil, fmt.Errorf("lock: Backoff %s out of range [1s, 1m]", cfg.backoff)
	}
	if cfg.gracePeriod < time.Second || cfg.gracePeriod > 30*time.Second {
		return nil, fmt.Errorf("lock: GracePeriod %s out of range [1s, 30s]", cfg.gracePeriod)
	}
	return &LeaderElection{rc: rc, name: name, cfg: cfg}, nil
}

func (le *LeaderElection) keyPath() string {
	return keyPrefix + le.name
}
```

- [ ] **Step 4: Run tests to verify pass**

```
cd libs/atlas-lock && go test -run 'TestNew_|TestOptions_' ./...
```

Expected: PASS for all.

- [ ] **Step 5: Commit**

```
git add libs/atlas-lock/leader.go libs/atlas-lock/leader_test.go libs/atlas-lock/go.sum
git commit -m "feat(atlas-lock): LeaderElection type with name and option validation"
```

---

## Task 4: Prometheus counters

**Goal:** Declare the four counters in their own file. Verify registration via metric exposition.

**Files:**
- Create: `libs/atlas-lock/metrics.go`
- Modify: `libs/atlas-lock/leader_test.go`

- [ ] **Step 1: Write the failing test**

Append to `leader_test.go`:

```go
import (
	// existing imports plus:
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMetrics_AllCountersExist(t *testing.T) {
	// Force reset to known zero state for a deterministic assertion.
	acquiredTotal.Reset()
	lostTotal.Reset()
	renewFailedTotal.Reset()
	acquireFailedTotal.Reset()

	// Increment each by 1 with representative labels.
	acquiredTotal.WithLabelValues("test").Inc()
	lostTotal.WithLabelValues("test", "released").Inc()
	renewFailedTotal.WithLabelValues("test").Inc()
	acquireFailedTotal.WithLabelValues("test", "held_by_other").Inc()

	require.Equal(t, float64(1), testutil.ToFloat64(acquiredTotal.WithLabelValues("test")))
	require.Equal(t, float64(1), testutil.ToFloat64(lostTotal.WithLabelValues("test", "released")))
	require.Equal(t, float64(1), testutil.ToFloat64(renewFailedTotal.WithLabelValues("test")))
	require.Equal(t, float64(1), testutil.ToFloat64(acquireFailedTotal.WithLabelValues("test", "held_by_other")))
}
```

- [ ] **Step 2: Run test to verify it fails**

```
cd libs/atlas-lock && go test -run 'TestMetrics_AllCountersExist' ./...
```

Expected: FAIL — `undefined: acquiredTotal`, etc.

- [ ] **Step 3: Implement the metrics file**

`libs/atlas-lock/metrics.go`:

```go
package lock

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	acquiredTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_lock_acquired_total",
			Help: "Number of times this pod transitioned from non-leader to leader for a given lease name.",
		},
		[]string{"name"},
	)

	// reason ∈ {renew_failed, context_cancelled, released, panic}
	lostTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_lock_lost_total",
			Help: "Number of times this pod transitioned from leader to non-leader.",
		},
		[]string{"name", "reason"},
	)

	renewFailedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_lock_renew_failed_total",
			Help: "Number of single renewal attempts that failed (does not always cause leader loss).",
		},
		[]string{"name"},
	)

	// reason ∈ {held_by_other, redis_error}
	acquireFailedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_lock_acquire_failed_total",
			Help: "Number of failed acquire attempts.",
		},
		[]string{"name", "reason"},
	)
)
```

- [ ] **Step 4: Run tests to verify pass**

```
cd libs/atlas-lock && go test -run 'TestMetrics_AllCountersExist' ./...
```

Expected: PASS.

- [ ] **Step 5: Commit**

```
git add libs/atlas-lock/metrics.go libs/atlas-lock/leader_test.go libs/atlas-lock/go.sum
git commit -m "feat(atlas-lock): prometheus counters for acquire/lose/renew transitions"
```

---

## Task 5: `Run()` minimal — acquire, invoke fn, release on outer-ctx-cancel

**Goal:** Walking-skeleton `Run`. Acquire the lease, invoke `fn(leaderCtx)` once, wait for outer ctx to cancel, release.

No renewal yet. No panic recovery yet. No grace period yet. No acquire-failure handling yet (let it return error on Redis failure — we'll add classification in Task 9).

**Files:**
- Modify: `libs/atlas-lock/leader.go`
- Modify: `libs/atlas-lock/leader_test.go`

- [ ] **Step 1: Write the failing test**

Append to `leader_test.go`:

```go
import (
	// existing imports plus:
	"context"
	"sync"
)

func TestRun_AcquireAndReleaseOnOuterCancel(t *testing.T) {
	rc, mr := newTestClient(t)
	le, err := New(rc, "release-test",
		WithTTL(10*time.Second),
		WithRefreshInterval(2*time.Second),
		WithBackoff(time.Second),
	)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	var fnInvocations int
	var mu sync.Mutex

	done := make(chan error, 1)
	go func() {
		done <- le.Run(ctx, func(leaderCtx context.Context) {
			mu.Lock()
			fnInvocations++
			mu.Unlock()
			<-leaderCtx.Done()
		})
	}()

	// Wait until lease is observed in miniredis.
	require.Eventually(t, func() bool {
		return mr.Exists("atlas:lock:release-test")
	}, 2*time.Second, 25*time.Millisecond, "lease should be acquired")

	cancel()
	require.NoError(t, <-done, "Run should return nil on outer ctx cancel")

	mu.Lock()
	require.Equal(t, 1, fnInvocations, "fn invoked exactly once")
	mu.Unlock()

	// Lease should be gone (Released on shutdown).
	require.False(t, mr.Exists("atlas:lock:release-test"), "lease released on shutdown")
}
```

- [ ] **Step 2: Run test to verify it fails**

```
cd libs/atlas-lock && go test -run 'TestRun_AcquireAndReleaseOnOuterCancel' ./...
```

Expected: FAIL — `undefined: (*LeaderElection).Run`.

- [ ] **Step 3: Implement minimal `Run`**

Add to `leader.go`:

```go
import (
	// add to existing imports:
	"context"

	"github.com/bsm/redislock"
)

// Run blocks until ctx is cancelled.
//
// While the lease is held by this pod, fn is invoked once with a child
// context. fn is expected to block on its leaderCtx until the lease is lost
// or the outer ctx is cancelled. On outer-ctx cancel, Run releases the lease
// (best-effort) and returns nil.
//
// This minimal version has no renewal, no panic recovery, and no grace
// period — those are added in subsequent commits. It exists to validate
// the acquire/release skeleton.
func (le *LeaderElection) Run(ctx context.Context, fn func(context.Context)) error {
	locker := redislock.New(le.rc)

	for {
		if ctx.Err() != nil {
			return nil
		}

		rl, err := locker.Obtain(ctx, le.keyPath(), le.cfg.ttl, &redislock.Options{
			RetryStrategy: redislock.NoRetry(),
		})
		if err != nil {
			// Held by other or redis error — back off and retry.
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(le.cfg.backoff):
			}
			continue
		}

		acquiredTotal.WithLabelValues(le.name).Inc()
		le.cfg.log.Infof("Acquired leader for [%s].", le.name)

		leaderCtx, cancelLeader := context.WithCancel(ctx)
		fnDone := make(chan struct{})

		go func() {
			defer close(fnDone)
			fn(leaderCtx)
		}()

		// Wait for outer ctx to cancel or fn to return on its own.
		select {
		case <-ctx.Done():
		case <-fnDone:
		}
		cancelLeader()
		<-fnDone

		// Best-effort release.
		relCtx, relCancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = rl.Release(relCtx)
		relCancel()

		lostTotal.WithLabelValues(le.name, "released").Inc()
		le.cfg.log.Infof("Lost leader for [%s] (reason: released).", le.name)

		if ctx.Err() != nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(le.cfg.backoff):
		}
	}
}
```

- [ ] **Step 4: Run tests to verify pass**

```
cd libs/atlas-lock && go test -race -run 'TestRun_AcquireAndReleaseOnOuterCancel' ./...
```

Expected: PASS. `-race` clean.

- [ ] **Step 5: Commit**

```
git add libs/atlas-lock/leader.go libs/atlas-lock/leader_test.go libs/atlas-lock/go.sum
git commit -m "feat(atlas-lock): minimal Run with acquire-invoke-release skeleton"
```

---

## Task 6: Two-competitors test — only one acquires at a time

**Goal:** Verify the lease enforces mutual exclusion. Two `LeaderElection` instances on the same miniredis with the same name; only one of their `fn` callbacks executes at any moment.

**Files:**
- Modify: `libs/atlas-lock/leader_test.go`

- [ ] **Step 1: Write the failing test**

Append to `leader_test.go`:

```go
import (
	// existing imports plus:
	"sync/atomic"
)

func TestRun_TwoCompetitors_OneAcquires(t *testing.T) {
	rc, _ := newTestClient(t)

	leA, err := New(rc, "competitors",
		WithTTL(5*time.Second), WithRefreshInterval(time.Second), WithBackoff(time.Second),
	)
	require.NoError(t, err)
	leB, err := New(rc, "competitors",
		WithTTL(5*time.Second), WithRefreshInterval(time.Second), WithBackoff(time.Second),
	)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var concurrent int32
	var maxConcurrent int32

	worker := func(le *LeaderElection) error {
		return le.Run(ctx, func(leaderCtx context.Context) {
			n := atomic.AddInt32(&concurrent, 1)
			defer atomic.AddInt32(&concurrent, -1)
			for {
				cur := atomic.LoadInt32(&maxConcurrent)
				if n <= cur || atomic.CompareAndSwapInt32(&maxConcurrent, cur, n) {
					break
				}
			}
			<-leaderCtx.Done()
		})
	}

	doneA := make(chan error, 1)
	doneB := make(chan error, 1)
	go func() { doneA <- worker(leA) }()
	go func() { doneB <- worker(leB) }()

	// Let them race; verify exactly one is running fn at any sampled moment.
	for i := 0; i < 40; i++ {
		require.LessOrEqual(t, atomic.LoadInt32(&concurrent), int32(1), "no overlap permitted")
		time.Sleep(50 * time.Millisecond)
	}

	cancel()
	require.NoError(t, <-doneA)
	require.NoError(t, <-doneB)
	require.Equal(t, int32(1), atomic.LoadInt32(&maxConcurrent), "exactly one fn ran across the whole window")
}
```

- [ ] **Step 2: Run the test (no implementation change required — this exercises behavior already shipped in Task 5)**

```
cd libs/atlas-lock && go test -race -run 'TestRun_TwoCompetitors_OneAcquires' ./...
```

Expected: PASS. The lease is fenced via NX-acquire, so only one of the two `Obtain` calls succeeds at any time. If this fails, return to Task 5 — the bug is there, not here.

- [ ] **Step 3: Commit**

```
git add libs/atlas-lock/leader_test.go
git commit -m "test(atlas-lock): two competitors only one acquires at a time"
```

---

## Task 7: Renewal goroutine — extends lease past TTL, lease loss cancels inner ctx

**Goal:** Add the renewer. Verify (a) renewals keep the lease alive past its initial TTL, (b) when the lease is lost (e.g. expires under load) the leader ctx cancels and `fn` returns.

**Files:**
- Modify: `libs/atlas-lock/leader.go`
- Modify: `libs/atlas-lock/leader_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `leader_test.go`:

```go
func TestRun_RenewalExtendsLeasePastTTL(t *testing.T) {
	rc, mr := newTestClient(t)
	le, err := New(rc, "renew-test",
		WithTTL(5*time.Second),
		WithRefreshInterval(time.Second),
		WithBackoff(time.Second),
	)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- le.Run(ctx, func(leaderCtx context.Context) {
			<-leaderCtx.Done()
		})
	}()

	// Wait for acquire.
	require.Eventually(t, func() bool {
		return mr.Exists("atlas:lock:renew-test")
	}, 2*time.Second, 25*time.Millisecond)

	// Advance miniredis time past the original TTL — renewer should keep the
	// lease alive.
	mr.FastForward(4 * time.Second)
	time.Sleep(1500 * time.Millisecond) // let renewer tick at least once after FastForward
	mr.FastForward(4 * time.Second)
	time.Sleep(1500 * time.Millisecond)

	require.True(t, mr.Exists("atlas:lock:renew-test"), "lease still held after > TTL elapsed")

	cancel()
	require.NoError(t, <-done)
}

func TestRun_LeaseLossCancelsInnerCtx(t *testing.T) {
	rc, mr := newTestClient(t)
	le, err := New(rc, "lose-test",
		WithTTL(5*time.Second),
		WithRefreshInterval(time.Second),
		WithBackoff(time.Second),
	)
	require.NoError(t, err)

	outerCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	innerCtxCancelled := make(chan struct{})

	done := make(chan error, 1)
	go func() {
		done <- le.Run(outerCtx, func(leaderCtx context.Context) {
			<-leaderCtx.Done()
			close(innerCtxCancelled)
		})
	}()

	// Wait for acquire.
	require.Eventually(t, func() bool {
		return mr.Exists("atlas:lock:lose-test")
	}, 2*time.Second, 25*time.Millisecond)

	// Force-expire the lease in miniredis. Next Refresh will return ErrNotObtained.
	mr.FastForward(10 * time.Second)

	// fn's leaderCtx should be cancelled.
	select {
	case <-innerCtxCancelled:
	case <-time.After(5 * time.Second):
		t.Fatal("inner ctx not cancelled within 5s of lease loss")
	}

	cancel()
	require.NoError(t, <-done)
}
```

- [ ] **Step 2: Run tests to verify they fail**

```
cd libs/atlas-lock && go test -race -run 'TestRun_RenewalExtendsLeasePastTTL|TestRun_LeaseLossCancelsInnerCtx' ./...
```

Expected: FAIL — without a renewer, the lease expires after TTL.

- [ ] **Step 3: Add the renewer goroutine and lease-loss handling**

Replace the body of `Run` in `leader.go` with the version below. Note the use of `sync/atomic` for the shared `lostReason` value — multiple goroutines (renewer and main) can write it, so we need synchronization to keep `go test -race` clean.

Add `"sync/atomic"` to the imports.

```go
func (le *LeaderElection) Run(ctx context.Context, fn func(context.Context)) error {
	locker := redislock.New(le.rc)

	for {
		if ctx.Err() != nil {
			return nil
		}

		rl, err := locker.Obtain(ctx, le.keyPath(), le.cfg.ttl, &redislock.Options{
			RetryStrategy: redislock.NoRetry(),
		})
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(le.cfg.backoff):
			}
			continue
		}

		acquiredTotal.WithLabelValues(le.name).Inc()
		le.cfg.log.Infof("Acquired leader for [%s].", le.name)

		leaderCtx, cancelLeader := context.WithCancel(ctx)
		fnDone := make(chan struct{})
		renewerDone := make(chan struct{})

		// First-writer-wins reason; multiple goroutines call setReason.
		var lostReason atomic.Value // string
		setReason := func(r string) { lostReason.CompareAndSwap(nil, r) }

		go func() {
			defer close(fnDone)
			fn(leaderCtx)
		}()

		go func() {
			defer close(renewerDone)
			t := time.NewTicker(le.cfg.refreshInterval)
			defer t.Stop()
			for {
				select {
				case <-leaderCtx.Done():
					return
				case <-t.C:
					rerr := rl.Refresh(ctx, le.cfg.ttl, nil)
					if rerr == nil {
						continue
					}
					if errors.Is(rerr, redislock.ErrNotObtained) {
						setReason("renew_failed")
						le.cfg.log.WithError(rerr).Warnf("Lease lost during refresh for [%s].", le.name)
						cancelLeader()
						return
					}
					renewFailedTotal.WithLabelValues(le.name).Inc()
					le.cfg.log.WithError(rerr).Warnf("Renewal attempt failed for [%s] (transient).", le.name)
				}
			}
		}()

		select {
		case <-ctx.Done():
			setReason("context_cancelled")
		case <-fnDone:
			setReason("released")
		}
		cancelLeader()
		<-fnDone
		<-renewerDone

		relCtx, relCancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = rl.Release(relCtx)
		relCancel()

		reason, _ := lostReason.Load().(string)
		if reason == "" {
			reason = "released"
		}
		lostTotal.WithLabelValues(le.name, reason).Inc()
		le.cfg.log.Infof("Lost leader for [%s] (reason: %s).", le.name, reason)

		if ctx.Err() != nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(le.cfg.backoff):
		}
	}
}
```

- [ ] **Step 4: Run tests to verify pass**

```
cd libs/atlas-lock && go test -race ./...
```

Expected: ALL prior tests still PASS, plus the two new ones.

- [ ] **Step 5: Commit**

```
git add libs/atlas-lock/leader.go libs/atlas-lock/leader_test.go
git commit -m "feat(atlas-lock): renewal goroutine; lease loss cancels inner ctx"
```

---

## Task 8: Panic recovery in `fn`

**Goal:** A panic inside `fn` is recovered, logged, the lease is released, the outer `Run` loop continues. The panic does not propagate.

**Files:**
- Modify: `libs/atlas-lock/leader.go`
- Modify: `libs/atlas-lock/leader_test.go`

- [ ] **Step 1: Write the failing test**

Append to `leader_test.go`:

```go
func TestRun_PanicInFn_RecoveredAndReleased(t *testing.T) {
	rc, mr := newTestClient(t)
	le, err := New(rc, "panic-test",
		WithTTL(5*time.Second),
		WithRefreshInterval(time.Second),
		WithBackoff(time.Second),
	)
	require.NoError(t, err)

	// Reset counter to read it deterministically.
	lostTotal.Reset()

	ctx, cancel := context.WithCancel(context.Background())
	var firstInvocation int32

	done := make(chan error, 1)
	go func() {
		done <- le.Run(ctx, func(leaderCtx context.Context) {
			if atomic.AddInt32(&firstInvocation, 1) == 1 {
				panic("boom")
			}
			<-leaderCtx.Done()
		})
	}()

	require.Eventually(t, func() bool {
		return testutil.ToFloat64(lostTotal.WithLabelValues("panic-test", "panic")) >= 1
	}, 5*time.Second, 50*time.Millisecond, "panic counter recorded")

	require.True(t, mr.Exists("atlas:lock:panic-test") || atomic.LoadInt32(&firstInvocation) >= 2,
		"either re-acquired by 2nd invocation or lease was released after panic")

	cancel()
	require.NoError(t, <-done, "panic must not escape Run")
}
```

- [ ] **Step 2: Run test to verify it fails**

```
cd libs/atlas-lock && go test -race -run 'TestRun_PanicInFn_RecoveredAndReleased' ./...
```

Expected: FAIL — without recovery, the goroutine crashes the test process.

- [ ] **Step 3: Wrap the `fn` goroutine with `recover`**

In `leader.go`, modify the fn goroutine to add panic recovery. (`setReason` and `cancelLeader` are already in scope from Task 7.) Replace:

```go
go func() {
	defer close(fnDone)
	fn(leaderCtx)
}()
```

with:

```go
go func() {
	defer close(fnDone)
	defer func() {
		if r := recover(); r != nil {
			le.cfg.log.WithField("panic", r).Errorf("Leader fn panic for [%s].", le.name)
			setReason("panic")
			cancelLeader()
		}
	}()
	fn(leaderCtx)
}()
```

The recovery cancels `leaderCtx`, which (a) wakes the renewer so it exits, (b) ensures `fnDone` is closed via the outer `defer close(fnDone)`, and (c) ensures `setReason("panic")` wins the `CompareAndSwap` race against the main goroutine's `setReason("released")` (because the panic path writes BEFORE the main goroutine's select resolves on `<-fnDone`).

- [ ] **Step 4: Run tests to verify pass**

```
cd libs/atlas-lock && go test -race ./...
```

Expected: ALL tests PASS, including the new panic test. `-race` clean.

- [ ] **Step 5: Commit**

```
git add libs/atlas-lock/leader.go libs/atlas-lock/leader_test.go
git commit -m "feat(atlas-lock): recover panic in fn; explicit release; loop continues"
```

---

## Task 9: Grace period for `fn` return after lease loss

**Goal:** When `leaderCtx` is cancelled, give `fn` `gracePeriod` to return. If it doesn't, log a WARN and proceed (don't block `Run` indefinitely behind a runaway `fn`).

**Files:**
- Modify: `libs/atlas-lock/leader.go`
- Modify: `libs/atlas-lock/leader_test.go`

- [ ] **Step 1: Write the failing test**

Append to `leader_test.go`:

```go
func TestRun_GracePeriodHonored(t *testing.T) {
	rc, mr := newTestClient(t)
	le, err := New(rc, "grace-test",
		WithTTL(5*time.Second),
		WithRefreshInterval(time.Second),
		WithBackoff(time.Second),
		WithGracePeriod(time.Second),
	)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	fnStarted := make(chan struct{})

	done := make(chan error, 1)
	go func() {
		done <- le.Run(ctx, func(leaderCtx context.Context) {
			close(fnStarted)
			// Ignore leaderCtx.Done() — simulate runaway fn.
			time.Sleep(10 * time.Second)
		})
	}()

	<-fnStarted
	require.Eventually(t, func() bool {
		return mr.Exists("atlas:lock:grace-test")
	}, 2*time.Second, 25*time.Millisecond)

	// Cancel; Run should return within gracePeriod + small slack, not 10s.
	cancel()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not return within gracePeriod + slack — runaway fn blocked it")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```
cd libs/atlas-lock && go test -race -run 'TestRun_GracePeriodHonored' ./...
```

Expected: FAIL — `Run` currently blocks on `<-fnDone` indefinitely.

- [ ] **Step 3: Replace the post-select waiter with a grace-bounded one**

In `leader.go`, replace:

```go
cancelLeader()
<-fnDone
<-renewerDone
```

with:

```go
cancelLeader()
<-renewerDone

graceTimer := time.NewTimer(le.cfg.gracePeriod)
select {
case <-fnDone:
	if !graceTimer.Stop() {
		<-graceTimer.C
	}
case <-graceTimer.C:
	le.cfg.log.Warnf("Leader fn did not return within grace period [%s] for [%s]; proceeding without waiting.",
		le.cfg.gracePeriod, le.name)
}
```

- [ ] **Step 4: Run tests to verify pass**

```
cd libs/atlas-lock && go test -race ./...
```

Expected: ALL tests PASS.

- [ ] **Step 5: Commit**

```
git add libs/atlas-lock/leader.go libs/atlas-lock/leader_test.go
git commit -m "feat(atlas-lock): grace period bounds fn-return wait on lease loss"
```

---

## Task 10: Acquire-failure classification — `held_by_other` vs `redis_error`

**Goal:** When `Obtain` fails, classify as `held_by_other` (`redislock.ErrNotObtained`) or `redis_error` (anything else) and increment `acquire_failed_total{reason}`.

**Files:**
- Modify: `libs/atlas-lock/leader.go`
- Modify: `libs/atlas-lock/leader_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `leader_test.go`:

```go
func TestRun_AcquireFailed_HeldByOther(t *testing.T) {
	rc, _ := newTestClient(t)
	acquireFailedTotal.Reset()

	leA, err := New(rc, "held-by-other",
		WithTTL(5*time.Second), WithRefreshInterval(time.Second), WithBackoff(time.Second),
	)
	require.NoError(t, err)
	leB, err := New(rc, "held-by-other",
		WithTTL(5*time.Second), WithRefreshInterval(time.Second), WithBackoff(time.Second),
	)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	doneA := make(chan error, 1)
	go func() {
		doneA <- leA.Run(ctx, func(leaderCtx context.Context) { <-leaderCtx.Done() })
	}()
	// Let A acquire first.
	time.Sleep(200 * time.Millisecond)

	doneB := make(chan error, 1)
	go func() {
		doneB <- leB.Run(ctx, func(leaderCtx context.Context) { <-leaderCtx.Done() })
	}()

	require.Eventually(t, func() bool {
		return testutil.ToFloat64(acquireFailedTotal.WithLabelValues("held-by-other", "held_by_other")) >= 1
	}, 5*time.Second, 100*time.Millisecond)

	cancel()
	require.NoError(t, <-doneA)
	require.NoError(t, <-doneB)
}

func TestRun_AcquireFailed_RedisError(t *testing.T) {
	rc, mr := newTestClient(t)
	acquireFailedTotal.Reset()

	le, err := New(rc, "redis-err",
		WithTTL(5*time.Second), WithRefreshInterval(time.Second), WithBackoff(time.Second),
	)
	require.NoError(t, err)

	// Stop miniredis so Obtain fails with a connection error.
	mr.Close()

	ctx, cancel := context.WithCancel(context.Background())
	doneRun := make(chan error, 1)
	go func() {
		doneRun <- le.Run(ctx, func(leaderCtx context.Context) { <-leaderCtx.Done() })
	}()

	require.Eventually(t, func() bool {
		return testutil.ToFloat64(acquireFailedTotal.WithLabelValues("redis-err", "redis_error")) >= 1
	}, 5*time.Second, 100*time.Millisecond)

	cancel()
	require.NoError(t, <-doneRun)
}
```

- [ ] **Step 2: Run tests to verify they fail**

```
cd libs/atlas-lock && go test -race -run 'TestRun_AcquireFailed_' ./...
```

Expected: FAIL — current `Run` does not increment `acquireFailedTotal`.

- [ ] **Step 3: Classify acquire errors**

In `leader.go`, replace the `Obtain` error block with:

```go
rl, err := locker.Obtain(ctx, le.keyPath(), le.cfg.ttl, &redislock.Options{
	RetryStrategy: redislock.NoRetry(),
})
if err != nil {
	if errors.Is(err, redislock.ErrNotObtained) {
		acquireFailedTotal.WithLabelValues(le.name, "held_by_other").Inc()
	} else {
		acquireFailedTotal.WithLabelValues(le.name, "redis_error").Inc()
		le.cfg.log.WithError(err).Debugf("Acquire for [%s] failed: %v", le.name, err)
	}
	select {
	case <-ctx.Done():
		return nil
	case <-time.After(le.cfg.backoff):
	}
	continue
}
```

- [ ] **Step 4: Run tests to verify pass**

```
cd libs/atlas-lock && go test -race ./...
```

Expected: ALL tests PASS.

- [ ] **Step 5: Commit**

```
git add libs/atlas-lock/leader.go libs/atlas-lock/leader_test.go
git commit -m "feat(atlas-lock): classify acquire failures (held_by_other vs redis_error)"
```

---

## Task 11: Failover-within-`TTL+backoff` test

**Goal:** When the leader pod releases (graceful shutdown), the standby acquires within `TTL + backoff + epsilon`. This is the user-facing failover SLA.

**Files:**
- Modify: `libs/atlas-lock/leader_test.go`

- [ ] **Step 1: Write the test**

Append to `leader_test.go`:

```go
func TestRun_FailoverAfterGracefulRelease(t *testing.T) {
	rc, _ := newTestClient(t)

	leA, err := New(rc, "failover",
		WithTTL(5*time.Second), WithRefreshInterval(time.Second), WithBackoff(time.Second),
	)
	require.NoError(t, err)
	leB, err := New(rc, "failover",
		WithTTL(5*time.Second), WithRefreshInterval(time.Second), WithBackoff(time.Second),
	)
	require.NoError(t, err)

	ctxA, cancelA := context.WithCancel(context.Background())
	ctxB, cancelB := context.WithCancel(context.Background())
	defer cancelB()

	bAcquired := make(chan struct{})

	doneA := make(chan error, 1)
	go func() {
		doneA <- leA.Run(ctxA, func(leaderCtx context.Context) { <-leaderCtx.Done() })
	}()
	doneB := make(chan error, 1)
	go func() {
		doneB <- leB.Run(ctxB, func(leaderCtx context.Context) {
			close(bAcquired)
			<-leaderCtx.Done()
		})
	}()

	// Let A acquire first.
	time.Sleep(500 * time.Millisecond)

	// A releases gracefully.
	start := time.Now()
	cancelA()
	require.NoError(t, <-doneA)

	// B should acquire within Backoff (small) — graceful release frees the lease immediately.
	select {
	case <-bAcquired:
		require.LessOrEqual(t, time.Since(start), 3*time.Second,
			"failover after graceful release should be within Backoff, well under TTL")
	case <-time.After(8 * time.Second):
		t.Fatal("standby did not acquire after leader released")
	}

	cancelB()
	require.NoError(t, <-doneB)
}
```

- [ ] **Step 2: Run the test (no implementation change required — exercises behavior already shipped)**

```
cd libs/atlas-lock && go test -race -run 'TestRun_FailoverAfterGracefulRelease' ./...
```

Expected: PASS. If it fails, the bug is in Tasks 5/7 (release-on-shutdown or acquire-loop), not here.

- [ ] **Step 3: Commit**

```
git add libs/atlas-lock/leader_test.go
git commit -m "test(atlas-lock): standby acquires after leader graceful release"
```

---

## Task 12: README and split-brain documentation

**Goal:** Author the README with the explicit single-Redis split-brain caveat. Verify by grep that all required phrases are present (so future scrubs can detect drift).

**Files:**
- Create: `libs/atlas-lock/README.md`

- [ ] **Step 1: Write the README**

`libs/atlas-lock/README.md`:

````markdown
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
````

- [ ] **Step 2: Verify all required documentation phrases are present**

```
cd libs/atlas-lock
grep -F "Redlock is out of scope" README.md && \
  grep -F "single-Redis split-brain" README.md && \
  grep -F "atlas_lock_acquired_total" README.md && \
  grep -F "atlas_lock_lost_total" README.md && \
  grep -F "atlas_lock_renew_failed_total" README.md && \
  grep -F "atlas_lock_acquire_failed_total" README.md
```

Expected: every grep matches; exit 0.

- [ ] **Step 3: Final lib-level verification**

```
cd libs/atlas-lock && go test -race ./... && go vet ./...
```

Expected: tests PASS, `go vet` clean.

- [ ] **Step 4: Commit**

```
git add libs/atlas-lock/README.md
git commit -m "docs(atlas-lock): README with split-brain caveat and operator recipe"
```

---

## Task 13: atlas-monsters dependency wiring

**Goal:** Add `libs/atlas-lock` as a direct dependency of atlas-monsters, mirroring the sibling-lib `replace` pattern. No code change to `main.go` yet.

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/go.mod`

- [ ] **Step 1: Add direct require + replace**

Edit `services/atlas-monsters/atlas.com/monsters/go.mod`:

In the first `require (...)` block, add:

```
	github.com/Chronicle20/atlas/libs/atlas-lock v0.0.0
```

After the existing `replace` directives (after the `replace ... atlas-tracing => ...` line), add:

```
replace github.com/Chronicle20/atlas/libs/atlas-lock => ../../../../libs/atlas-lock
```

- [ ] **Step 2: Verify the build resolves**

```
cd services/atlas-monsters/atlas.com/monsters && go mod tidy && go build ./...
```

Expected: clean. `go.sum` updated.

- [ ] **Step 3: Verify existing tests still pass**

```
cd services/atlas-monsters/atlas.com/monsters && go test -race ./...
```

Expected: existing tests PASS unchanged. (No new behavior yet — this step is a regression check on the build/dep wiring only.)

- [ ] **Step 4: Commit**

```
git add services/atlas-monsters/atlas.com/monsters/go.mod services/atlas-monsters/atlas.com/monsters/go.sum
git commit -m "chore(atlas-monsters): add atlas-lock dependency"
```

---

## Task 14: atlas-monsters env-loader helpers

**Goal:** Implement env-var loaders for `MONSTER_LEADER_ELECTION_ENABLED`, `MONSTER_LEADER_TTL`, `MONSTER_LEADER_REFRESH`, `MONSTER_LEADER_BACKOFF` following the task-060 warn-and-default pattern. Pure functions, easy to unit-test.

**Files:**
- Create: `services/atlas-monsters/atlas.com/monsters/leaderconfig.go`
- Create: `services/atlas-monsters/atlas.com/monsters/leaderconfig_test.go`

- [ ] **Step 1: Write the failing test**

`services/atlas-monsters/atlas.com/monsters/leaderconfig_test.go`:

```go
package main

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestLeaderEnabled_DefaultsTrue(t *testing.T) {
	t.Setenv("MONSTER_LEADER_ELECTION_ENABLED", "")
	require.True(t, leaderEnabled(logrus.New()))
}

func TestLeaderEnabled_ParsesFalse(t *testing.T) {
	t.Setenv("MONSTER_LEADER_ELECTION_ENABLED", "false")
	require.False(t, leaderEnabled(logrus.New()))
}

func TestLeaderEnabled_BadValueWarnsAndReturnsTrue(t *testing.T) {
	t.Setenv("MONSTER_LEADER_ELECTION_ENABLED", "maybe")
	require.True(t, leaderEnabled(logrus.New()))
}

func TestLeaderTTL_DefaultsTo30s(t *testing.T) {
	t.Setenv("MONSTER_LEADER_TTL", "")
	require.Equal(t, 30*time.Second, leaderTTL(logrus.New()))
}

func TestLeaderTTL_OutOfRangeFallsBack(t *testing.T) {
	t.Setenv("MONSTER_LEADER_TTL", "1s")
	require.Equal(t, 30*time.Second, leaderTTL(logrus.New()))
	t.Setenv("MONSTER_LEADER_TTL", "10m")
	require.Equal(t, 30*time.Second, leaderTTL(logrus.New()))
}

func TestLeaderTTL_ValidValueAccepted(t *testing.T) {
	t.Setenv("MONSTER_LEADER_TTL", "60s")
	require.Equal(t, 60*time.Second, leaderTTL(logrus.New()))
}

func TestLeaderRefresh_DefaultsToTTLOver3(t *testing.T) {
	t.Setenv("MONSTER_LEADER_TTL", "30s")
	t.Setenv("MONSTER_LEADER_REFRESH", "")
	require.Equal(t, 10*time.Second, leaderRefresh(logrus.New(), 30*time.Second))
}

func TestLeaderRefresh_OutOfRangeFallsBack(t *testing.T) {
	t.Setenv("MONSTER_LEADER_REFRESH", "0s")
	require.Equal(t, 10*time.Second, leaderRefresh(logrus.New(), 30*time.Second))
	t.Setenv("MONSTER_LEADER_REFRESH", "20s") // > TTL/2 = 15s
	require.Equal(t, 10*time.Second, leaderRefresh(logrus.New(), 30*time.Second))
}

func TestLeaderBackoff_DefaultsTo5s(t *testing.T) {
	t.Setenv("MONSTER_LEADER_BACKOFF", "")
	require.Equal(t, 5*time.Second, leaderBackoff(logrus.New()))
}

func TestLeaderBackoff_OutOfRangeFallsBack(t *testing.T) {
	t.Setenv("MONSTER_LEADER_BACKOFF", "0s")
	require.Equal(t, 5*time.Second, leaderBackoff(logrus.New()))
	t.Setenv("MONSTER_LEADER_BACKOFF", "5m")
	require.Equal(t, 5*time.Second, leaderBackoff(logrus.New()))
}
```

- [ ] **Step 2: Run tests to verify they fail**

```
cd services/atlas-monsters/atlas.com/monsters && go test -run 'TestLeader' .
```

Expected: FAIL — `undefined: leaderEnabled`, etc.

- [ ] **Step 3: Implement the loader**

`services/atlas-monsters/atlas.com/monsters/leaderconfig.go`:

```go
package main

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	envLeaderEnabled = "MONSTER_LEADER_ELECTION_ENABLED"
	envLeaderTTL     = "MONSTER_LEADER_TTL"
	envLeaderRefresh = "MONSTER_LEADER_REFRESH"
	envLeaderBackoff = "MONSTER_LEADER_BACKOFF"

	defaultLeaderTTL     = 30 * time.Second
	defaultLeaderRefresh = 10 * time.Second
	defaultLeaderBackoff = 5 * time.Second
)

func leaderEnabled(l logrus.FieldLogger) bool {
	v := strings.TrimSpace(os.Getenv(envLeaderEnabled))
	if v == "" {
		return true
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		l.Warnf("[%s] value %q not a boolean; defaulting to true.", envLeaderEnabled, v)
		return true
	}
	return b
}

func leaderTTL(l logrus.FieldLogger) time.Duration {
	return parseDurationInRange(l, envLeaderTTL, defaultLeaderTTL, 5*time.Second, 5*time.Minute)
}

func leaderRefresh(l logrus.FieldLogger, ttl time.Duration) time.Duration {
	def := ttl / 3
	if def < time.Second {
		def = time.Second
	}
	return parseDurationInRange(l, envLeaderRefresh, def, time.Second, ttl/2)
}

func leaderBackoff(l logrus.FieldLogger) time.Duration {
	return parseDurationInRange(l, envLeaderBackoff, defaultLeaderBackoff, time.Second, time.Minute)
}

func parseDurationInRange(l logrus.FieldLogger, env string, def, lo, hi time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(env))
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		l.Warnf("[%s] value %q not a duration; using default %s.", env, v, def)
		return def
	}
	if d < lo || d > hi {
		l.Warnf("[%s] value %s out of range [%s, %s]; using default %s.", env, d, lo, hi, def)
		return def
	}
	return d
}
```

- [ ] **Step 4: Run tests to verify pass**

```
cd services/atlas-monsters/atlas.com/monsters && go test -race -run 'TestLeader' .
```

Expected: ALL PASS.

- [ ] **Step 5: Commit**

```
git add services/atlas-monsters/atlas.com/monsters/leaderconfig.go services/atlas-monsters/atlas.com/monsters/leaderconfig_test.go
git commit -m "feat(atlas-monsters): env loaders for leader election config"
```

---

## Task 15: atlas-monsters main.go integration — gate the six tasks

**Goal:** Replace `main.go:88-93` with a leader-gated registration. When the kill-switch is off (`MONSTER_LEADER_ELECTION_ENABLED=false`), register the six tasks at `tdm.Context()` exactly as before.

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/main.go`

- [ ] **Step 1: Edit main.go**

Replace lines 88-93 (the six `tasks.Register(...)` calls) with:

```go
registerSweepTasks := func(l logrus.FieldLogger, ctx context.Context) {
	tasks.Register(l, ctx)(monster.NewRegistryAudit(l, time.Second*30))
	tasks.Register(l, ctx)(monster.NewStatusExpirationTask(l, ctx, time.Second))
	tasks.Register(l, ctx)(monster.NewDropTimerTask(l, ctx, time.Second))
	tasks.Register(l, ctx)(monster.NewMonsterAggroDecayTask(l, ctx, monster.AggroSweepInterval))
	tasks.Register(l, ctx)(monster.NewMonsterSkillPickerSweepTask(l, ctx, monster.MonsterSkillPickerSweepInterval))
	tasks.Register(l, ctx)(monster.NewMonsterRecoveryTask(l, ctx, monster.MonsterRecoveryInterval))
}

if leaderEnabled(l) {
	ttl := leaderTTL(l)
	le, err := lock.New(rc, "monsters-sweep",
		lock.WithTTL(ttl),
		lock.WithRefreshInterval(leaderRefresh(l, ttl)),
		lock.WithBackoff(leaderBackoff(l)),
		lock.WithLogger(l),
	)
	if err != nil {
		l.WithError(err).Fatal("Unable to construct LeaderElection.")
	}
	go func() {
		err := le.Run(tdm.Context(), func(leaderCtx context.Context) {
			registerSweepTasks(l, leaderCtx)
			<-leaderCtx.Done()
		})
		if err != nil {
			l.WithError(err).Errorf("LeaderElection.Run exited with error.")
		}
	}()
} else {
	l.Warnf("MONSTER_LEADER_ELECTION_ENABLED=false — sweep tasks run unconditionally on this pod.")
	registerSweepTasks(l, tdm.Context())
}
```

Add to the existing import block:

```go
import (
	// existing imports plus:
	"context"

	lock "github.com/Chronicle20/atlas/libs/atlas-lock"
	"github.com/sirupsen/logrus"
)
```

(`context` and `logrus` may already be transitively present; add explicitly so the closure type checks.)

- [ ] **Step 2: Build and check the integration compiles**

```
cd services/atlas-monsters/atlas.com/monsters && go build ./...
```

Expected: clean.

- [ ] **Step 3: Run existing tests; they must still pass**

```
cd services/atlas-monsters/atlas.com/monsters && go test -race ./...
```

Expected: existing tests PASS without modification. New `leaderconfig_test.go` from Task 14 also PASS.

- [ ] **Step 4: Verify the diff is what we want**

Run:

```
cd services/atlas-monsters/atlas.com/monsters && git diff main.go
```

Expected diff:
- Six `tasks.Register(l, tdm.Context())(...)` lines removed.
- `registerSweepTasks` closure introduced.
- `if leaderEnabled(l)` branch wrapping `lock.New(...)` + `le.Run(...)` in a goroutine.
- `else` branch calling `registerSweepTasks(l, tdm.Context())`.
- Import block extended.

- [ ] **Step 5: Commit**

```
git add services/atlas-monsters/atlas.com/monsters/main.go services/atlas-monsters/atlas.com/monsters/go.sum
git commit -m "$(cat <<'EOF'
feat(atlas-monsters): gate sweep tasks behind leader election

Six in-process sweep tickers in main.go (RegistryAudit,
StatusExpirationTask, DropTimerTask, MonsterAggroDecayTask,
MonsterSkillPickerSweepTask, MonsterRecoveryTask) were emitting
every event N times when running with replicas > 1. Each iterates
global state and emits Kafka events as a side effect, so duplicate
emission was structural.

Gate them all behind a single shared LeaderElection lease named
"monsters-sweep" using the new libs/atlas-lock module. The existing
tasks.Register obeys ctx-cancel, so loss of leader naturally tears
down all six task goroutines via leaderCtx; re-acquire spawns a
fresh generation. No changes to the Task interface or any task body.

Kill-switch via MONSTER_LEADER_ELECTION_ENABLED=false preserves the
previous unconditional behavior for docker-compose and emergency
rollback.

Single-Redis split-brain caveat: during Redis primary→replica
failover, two pods may briefly believe they hold the lease.
Acceptable because Atlas Kafka consumers must already tolerate
at-least-once delivery. Multi-Redis Redlock is out of scope.
EOF
)"
```

---

## Task 16: atlas-monsters integration tests (kill-switch + leader-election)

**Goal:** Two integration tests — one verifying the kill-switch path registers tasks unconditionally, one verifying two `LeaderElection` instances against a shared miniredis observe mutual exclusion of a fake task.

**Files:**
- Create: `services/atlas-monsters/atlas.com/monsters/main_leader_test.go`

- [ ] **Step 1: Write the integration tests**

`services/atlas-monsters/atlas.com/monsters/main_leader_test.go`:

```go
package main

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	lock "github.com/Chronicle20/atlas/libs/atlas-lock"
	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func newTestRedis(t *testing.T) *goredis.Client {
	mr := miniredis.RunT(t)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rc.Close() })
	return rc
}

// Kill-switch: when leaderEnabled returns false, "register" runs unconditionally
// on the supplied ctx without any Redis interaction.
func TestKillSwitch_RunsUnconditionally(t *testing.T) {
	t.Setenv("MONSTER_LEADER_ELECTION_ENABLED", "false")
	require.False(t, leaderEnabled(logrus.New()))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var ran int32
	register := func(l logrus.FieldLogger, ctx context.Context) {
		atomic.AddInt32(&ran, 1)
	}

	if leaderEnabled(logrus.New()) {
		t.Fatal("kill-switch should be off")
	}
	register(logrus.New(), ctx)

	require.Equal(t, int32(1), atomic.LoadInt32(&ran))
}

// Leader election: with two LEs on the same miniredis with the same name, only
// one of them runs the sweep "register" callback at a time. After cancelling
// the leader, the standby takes over and runs its callback.
func TestLeaderElection_OnlyOneRegisters(t *testing.T) {
	t.Setenv("MONSTER_LEADER_ELECTION_ENABLED", "true")
	rc := newTestRedis(t)

	leA, err := lock.New(rc, "monsters-sweep-test",
		lock.WithTTL(5*time.Second),
		lock.WithRefreshInterval(time.Second),
		lock.WithBackoff(time.Second),
	)
	require.NoError(t, err)
	leB, err := lock.New(rc, "monsters-sweep-test",
		lock.WithTTL(5*time.Second),
		lock.WithRefreshInterval(time.Second),
		lock.WithBackoff(time.Second),
	)
	require.NoError(t, err)

	var concurrent int32
	var maxConcurrent int32
	var registrations int32

	work := func(le *lock.LeaderElection, ctx context.Context) error {
		return le.Run(ctx, func(leaderCtx context.Context) {
			atomic.AddInt32(&registrations, 1)
			n := atomic.AddInt32(&concurrent, 1)
			defer atomic.AddInt32(&concurrent, -1)
			for {
				cur := atomic.LoadInt32(&maxConcurrent)
				if n <= cur || atomic.CompareAndSwapInt32(&maxConcurrent, cur, n) {
					break
				}
			}
			<-leaderCtx.Done()
		})
	}

	ctxA, cancelA := context.WithCancel(context.Background())
	ctxB, cancelB := context.WithCancel(context.Background())

	doneA := make(chan error, 1)
	doneB := make(chan error, 1)
	go func() { doneA <- work(leA, ctxA) }()
	go func() { doneB <- work(leB, ctxB) }()

	// Sample for 2s; max concurrent must stay at 1.
	for i := 0; i < 40; i++ {
		require.LessOrEqual(t, atomic.LoadInt32(&concurrent), int32(1), "no overlap permitted")
		time.Sleep(50 * time.Millisecond)
	}
	require.Equal(t, int32(1), atomic.LoadInt32(&maxConcurrent))

	// Cancel A; B should take over (registrations == 2 within failover window).
	cancelA()
	require.NoError(t, <-doneA)

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&registrations) >= 2
	}, 8*time.Second, 100*time.Millisecond, "standby should acquire after leader release")

	cancelB()
	require.NoError(t, <-doneB)
}
```

- [ ] **Step 2: Run tests to verify they fail OR pass**

```
cd services/atlas-monsters/atlas.com/monsters && go test -race -run 'TestKillSwitch_|TestLeaderElection_' .
```

Expected: PASS. The behavior is already implemented — these tests are coverage on top.

- [ ] **Step 3: Final atlas-monsters verification**

```
cd services/atlas-monsters/atlas.com/monsters && go test -race ./...
```

Expected: ALL tests PASS — preexisting + Task 14 + Task 16. If a test fails, return to Tasks 14–15 to fix the underlying behavior, not the test.

- [ ] **Step 4: Commit**

```
git add services/atlas-monsters/atlas.com/monsters/main_leader_test.go
git commit -m "test(atlas-monsters): kill-switch and leader-election integration coverage"
```

---

## Task 17: docs/TODO.md follow-up entries

**Goal:** PRD §10 acceptance criterion: catalogue every other Atlas service whose sweep tasks have the same multi-pod hazard, with a TODO entry per service linking back to this task.

**Files:**
- Modify: `docs/TODO.md`

- [ ] **Step 1: Locate `docs/TODO.md` and append the section**

Search for an existing section template:

```
grep -n '^##' docs/TODO.md | head -20
```

Use the project's prevailing format. Append a new section near the top of "open work" (or wherever the project conventionally places per-service rollouts):

```markdown
## Leader-election adoption (depends on task-064)

Each entry below is a per-service follow-up task — adopt `libs/atlas-lock`
for that service's sweep tickers so the Deployment can scale beyond one
replica without duplicating Kafka emission. See PRD §7.3 of
`docs/tasks/task-064-redis-leader-election/prd.md` for the catalogue.

- [ ] atlas-buffs — gate `NewExpiration`, `NewPoisonTick` (`services/atlas-buffs/atlas.com/buffs/main.go:63-64`)
- [ ] atlas-ban — gate `NewExpiredBanCleanup`, `NewHistoryPurge` (`services/atlas-ban/atlas.com/ban/main.go:79-80`)
- [ ] atlas-drops — gate `NewExpirationTask` (`services/atlas-drops/atlas.com/drops/main.go:92`)
- [ ] atlas-pets — gate `NewHungerTask` (`services/atlas-pets/atlas.com/pets/main.go:89`)
- [ ] atlas-skills — gate `NewExpirationTask` (`services/atlas-skills/atlas.com/skills/main.go:77`)
- [ ] atlas-reactors — gate `NewCooldownCleanup` (`services/atlas-reactors/atlas.com/reactors/main.go:68`)
- [ ] atlas-maps — gate `NewRespawn`, `NewWeather`, `NewMistTick` (`services/atlas-maps/atlas.com/maps/main.go:105-107`)
- [ ] atlas-merchant — gate `NewExpirationTask`, `NewCleanupTask`, `NewNotificationTask` (`services/atlas-merchant/atlas.com/merchant/main.go:79-81`)
- [ ] atlas-guilds — gate `NewTransitionTimeout` (`services/atlas-guilds/atlas.com/guilds/main.go:99`)
- [ ] atlas-account — gate `NewTransitionTimeout` (`services/atlas-account/atlas.com/account/main.go:76`)
- [ ] atlas-world — gate `NewExpiration` (`services/atlas-world/atlas.com/world/main.go:90`)
- [ ] atlas-invites — gate `NewInviteTimeout` (`services/atlas-invites/atlas.com/invites/main.go:80`)
- [ ] atlas-expressions — gate `NewRevertTask` (`services/atlas-expressions/atlas.com/expressions/main.go:49`)
- [ ] atlas-character — review `NewTimeout` (`services/atlas-character/atlas.com/character/main.go:102`); gate iff the work is global, not per-pod-session

The following two services are **review-and-decline** — listed for completeness, not for adoption:
- atlas-login — `NewTimeout` is per-pod session timeout, do NOT gate
- atlas-channel — `NewHeartbeat` is per-pod state by design, do NOT gate
```

- [ ] **Step 2: Verify the file is valid markdown**

```
grep -c '^- \[ \] atlas-' docs/TODO.md
```

Expected: ≥ 14 (the candidate set per PRD §7.3).

- [ ] **Step 3: Commit**

```
git add docs/TODO.md
git commit -m "docs(TODO): catalogue 14 per-service follow-up adoption tasks for atlas-lock"
```

---

## Final verification

After Task 17, run the whole-tree verification (NOT a separate task — just sanity):

```
cd <worktree-root>
cd libs/atlas-lock && go test -race ./... && go vet ./... && cd -
cd services/atlas-monsters/atlas.com/monsters && go test -race ./... && go build ./... && go vet ./... && cd -
git status --short  # expect clean
```

If anything fails, the failing module's tests pinpoint the regression.

Cross-check `prd.md` §10 acceptance criteria against committed work:

- [ ] New module `libs/atlas-lock` exists with `LeaderElection`, `New`, `Run`, options. (Tasks 1–4, 7–10)
- [ ] Unit tests cover §4.9 list. (Tasks 2–11)
- [ ] `go build ./...` and `go test -race ./...` clean. (Verified above)
- [ ] `services/atlas-monsters/atlas.com/monsters/main.go` constructs `LeaderElection` for `monsters-sweep` using existing `*goredis.Client`; six tasks inside the `Run` callback. (Task 15)
- [ ] `MONSTER_LEADER_ELECTION_ENABLED=false` causes `main.go` to register the six tasks unconditionally. (Tasks 14–15; verified by Task 16)
- [ ] All four observability counters in §4.8 emitted and labeled. (Task 4)
- [ ] `docs/TODO.md` updated with one entry per service in §7.3 (excluding atlas-login and atlas-channel). (Task 17)
- [ ] PR description and library README clearly document the single-Redis split-brain caveat. (Task 12 covers README; PR description is a manual step at PR creation time)
- [ ] `go test -race ./...` clean in both modules. (Verified above)
