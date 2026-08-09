package lock

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/prometheus/client_golang/prometheus/testutil"
	goredis "github.com/redis/go-redis/v9"
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

// testKey mirrors LeaderElection.keyPath for assertions. These tests never set
// ATLAS_ENV, so every lease lands in the unscoped bucket; TestKeyPath_* below
// pin the scoping behaviour itself.
func testKey(name string) string { return keyPrefix + unscopedEnv + ":" + name }

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
	require.Equal(t, testKey("monsters-sweep"), le.keyPath())
}

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
		return mr.Exists(testKey("release-test"))
	}, 2*time.Second, 25*time.Millisecond, "lease should be acquired")

	cancel()
	require.NoError(t, <-done, "Run should return nil on outer ctx cancel")

	mu.Lock()
	require.Equal(t, 1, fnInvocations, "fn invoked exactly once")
	mu.Unlock()

	// Lease should be gone (Released on shutdown).
	require.False(t, mr.Exists(testKey("release-test")), "lease released on shutdown")
}

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
		return mr.Exists(testKey("renew-test"))
	}, 2*time.Second, 25*time.Millisecond)

	// Advance miniredis time past the original TTL — renewer should keep the
	// lease alive.
	mr.FastForward(4 * time.Second)
	time.Sleep(1500 * time.Millisecond) // let renewer tick at least once after FastForward
	mr.FastForward(4 * time.Second)
	time.Sleep(1500 * time.Millisecond)

	require.True(t, mr.Exists(testKey("renew-test")), "lease still held after > TTL elapsed")

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
		return mr.Exists(testKey("lose-test"))
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

	require.Eventually(t, func() bool {
		return mr.Exists(testKey("panic-test")) || atomic.LoadInt32(&firstInvocation) >= 2
	}, 5*time.Second, 50*time.Millisecond, "either re-acquired by 2nd invocation or lease was released after panic")

	cancel()
	require.NoError(t, <-done, "panic must not escape Run")
}

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
		return mr.Exists(testKey("grace-test"))
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

// TestKeyPath_ScopedByEnv is the regression guard for the defect this scoping
// was added for.
//
// The lease key used to be keyPrefix + name, with no deployment component.
// Every Atlas deployment shares one Redis (REDIS_URL is redis.home:6379 in
// atlas-main and in every ephemeral atlas-pr-NNNN namespace, no DB
// separation), so "monsters-sweep" was a single global lease. The permanent
// deployment acquired it at startup and held it, and every ephemeral
// namespace's pod lost the election forever -- silently running NONE of its
// leader-gated work. In atlas-pr-1255 that meant Poison Mist applied POISON
// to monsters whose HP then never moved, because StatusExpirationTask had
// never been registered.
//
// Distinct ATLAS_ENV values MUST produce distinct keys. That is the whole
// property; the exact spelling is asserted too so a reformat of the key can't
// quietly change what an already-running fleet contends on.
func TestKeyPath_ScopedByEnv(t *testing.T) {
	rc, _ := newTestClient(t)

	t.Setenv(EnvVar, "a628")
	pr, err := New(rc, "monsters-sweep")
	require.NoError(t, err)
	require.Equal(t, "atlas:lock:a628:monsters-sweep", pr.keyPath())

	t.Setenv(EnvVar, "main")
	main, err := New(rc, "monsters-sweep")
	require.NoError(t, err)
	require.Equal(t, "atlas:lock:main:monsters-sweep", main.keyPath())

	require.NotEqual(t, pr.keyPath(), main.keyPath(),
		"two deployments on a shared Redis must not contend on the same lease")
}

// TestKeyPath_UnsetEnvIsMarkedNotOmitted pins that a missing ATLAS_ENV yields a
// visible "unscoped" segment rather than the old unsegmented key. Omitting the
// segment would silently restore the collision AND make the two spellings
// coexist during a rollout; naming the bucket makes it greppable in
// `redis-cli keys atlas:lock:*`.
func TestKeyPath_UnsetEnvIsMarkedNotOmitted(t *testing.T) {
	rc, _ := newTestClient(t)

	t.Setenv(EnvVar, "")
	le, err := New(rc, "monsters-sweep")
	require.NoError(t, err)
	require.Equal(t, "atlas:lock:unscoped:monsters-sweep", le.keyPath())
}

// TestKeyPath_WhitespaceEnvTreatedAsUnset pins that a padded value (a trivially
// easy thing to get out of a YAML block scalar) does not become part of the
// key. " a628" and "a628" naming different leases would split one deployment's
// fleet into two leaders.
func TestKeyPath_WhitespaceEnvTreatedAsUnset(t *testing.T) {
	rc, _ := newTestClient(t)

	t.Setenv(EnvVar, "   ")
	le, err := New(rc, "monsters-sweep")
	require.NoError(t, err)
	require.Equal(t, "atlas:lock:unscoped:monsters-sweep", le.keyPath())
}

// TestKeyPath_CapturedAtConstruction pins that the scope is read once in New.
// Re-reading it per keyPath call would let an env mutation move the lease out
// from under the renewer while fn still believed it was leader -- the renewer
// would refresh a key nobody holds and the real lease would expire unnoticed.
func TestKeyPath_CapturedAtConstruction(t *testing.T) {
	rc, _ := newTestClient(t)

	t.Setenv(EnvVar, "a628")
	le, err := New(rc, "monsters-sweep")
	require.NoError(t, err)

	t.Setenv(EnvVar, "somethingelse")
	require.Equal(t, "atlas:lock:a628:monsters-sweep", le.keyPath(),
		"the key must not change under a running election")
}
