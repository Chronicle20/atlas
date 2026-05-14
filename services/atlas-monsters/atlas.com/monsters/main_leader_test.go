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

	// leaderAcquired signals when the first leader fn is entered; the test
	// waits for this before starting the standby so that A is deterministically
	// the first holder.
	leaderAcquired := make(chan struct{})

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

	ctxA, cancelA := context.WithCancel(context.Background())
	ctxB, cancelB := context.WithCancel(context.Background())

	doneA := make(chan error, 1)
	doneB := make(chan error, 1)

	// Start A first; signal when it holds the lease so we know A is the leader.
	go func() {
		doneA <- leA.Run(ctxA, func(leaderCtx context.Context) {
			atomic.AddInt32(&registrations, 1)
			n := atomic.AddInt32(&concurrent, 1)
			defer atomic.AddInt32(&concurrent, -1)
			for {
				cur := atomic.LoadInt32(&maxConcurrent)
				if n <= cur || atomic.CompareAndSwapInt32(&maxConcurrent, cur, n) {
					break
				}
			}
			close(leaderAcquired)
			<-leaderCtx.Done()
		})
	}()

	// Wait until A holds the lease before launching B.
	select {
	case <-leaderAcquired:
	case <-time.After(5 * time.Second):
		t.Fatal("A did not acquire leadership within 5s")
	}

	go func() {
		doneB <- leB.Run(ctxB, func(leaderCtx context.Context) {
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
	}()

	// Sample for 1s while A is leader; max concurrent must stay at 1.
	for i := 0; i < 20; i++ {
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
