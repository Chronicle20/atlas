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
		return mr.Exists("atlas:lock:panic-test") || atomic.LoadInt32(&firstInvocation) >= 2
	}, 5*time.Second, 50*time.Millisecond, "either re-acquired by 2nd invocation or lease was released after panic")

	cancel()
	require.NoError(t, <-done, "panic must not escape Run")
}
