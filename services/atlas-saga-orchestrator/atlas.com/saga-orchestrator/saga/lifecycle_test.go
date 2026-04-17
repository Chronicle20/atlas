package saga

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func newTestCtx() context.Context {
	te, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	return tenant.WithContext(context.Background(), te)
}

func TestIsValidTransition(t *testing.T) {
	cases := []struct {
		from, to SagaLifecycleState
		want     bool
	}{
		{SagaLifecyclePending, SagaLifecycleCompensating, true},
		{SagaLifecyclePending, SagaLifecycleCompleted, true},
		{SagaLifecycleCompensating, SagaLifecycleFailed, true},

		{SagaLifecyclePending, SagaLifecyclePending, false},
		{SagaLifecyclePending, SagaLifecycleFailed, false},
		{SagaLifecycleCompensating, SagaLifecycleCompleted, false},
		{SagaLifecycleCompensating, SagaLifecyclePending, false},
		{SagaLifecycleCompleted, SagaLifecycleFailed, false},
		{SagaLifecycleFailed, SagaLifecyclePending, false},
	}
	for _, c := range cases {
		got := IsValidTransition(c.from, c.to)
		assert.Equal(t, c.want, got, "from=%s to=%s", c.from, c.to)
	}
}

func TestInMemoryCache_PutSetsPendingLifecycle(t *testing.T) {
	ResetCache()
	ctx := newTestCtx()

	s, err := NewBuilder().SetSagaType(CharacterCreation).SetInitiatedBy("test").Build()
	assert.NoError(t, err)
	assert.NoError(t, GetCache().Put(ctx, s))

	state, ok := GetCache().GetLifecycle(ctx, s.TransactionId())
	assert.True(t, ok)
	assert.Equal(t, SagaLifecyclePending, state)
}

func TestInMemoryCache_PutPreservesLifecycleOnUpdate(t *testing.T) {
	ResetCache()
	ctx := newTestCtx()

	s, _ := NewBuilder().SetSagaType(CharacterCreation).SetInitiatedBy("test").Build()
	_ = GetCache().Put(ctx, s)

	assert.True(t, GetCache().TryTransition(ctx, s.TransactionId(), SagaLifecyclePending, SagaLifecycleCompensating))

	// A second Put (updated saga) must not reset the lifecycle.
	_ = GetCache().Put(ctx, s)
	state, _ := GetCache().GetLifecycle(ctx, s.TransactionId())
	assert.Equal(t, SagaLifecycleCompensating, state)
}

func TestInMemoryCache_TryTransition_InvalidTransitionRejected(t *testing.T) {
	ResetCache()
	ctx := newTestCtx()
	s, _ := NewBuilder().SetSagaType(CharacterCreation).SetInitiatedBy("test").Build()
	_ = GetCache().Put(ctx, s)

	// Pending → Failed is not a valid transition (must go through Compensating).
	assert.False(t, GetCache().TryTransition(ctx, s.TransactionId(), SagaLifecyclePending, SagaLifecycleFailed))

	state, _ := GetCache().GetLifecycle(ctx, s.TransactionId())
	assert.Equal(t, SagaLifecyclePending, state)
}

func TestInMemoryCache_TryTransition_WrongFromRejected(t *testing.T) {
	ResetCache()
	ctx := newTestCtx()
	s, _ := NewBuilder().SetSagaType(CharacterCreation).SetInitiatedBy("test").Build()
	_ = GetCache().Put(ctx, s)

	// Move into Compensating first.
	assert.True(t, GetCache().TryTransition(ctx, s.TransactionId(), SagaLifecyclePending, SagaLifecycleCompensating))

	// Now a Pending → Completed call should fail because the saga is no longer Pending.
	assert.False(t, GetCache().TryTransition(ctx, s.TransactionId(), SagaLifecyclePending, SagaLifecycleCompleted))
}

func TestInMemoryCache_TryTransition_MissingSaga(t *testing.T) {
	ResetCache()
	ctx := newTestCtx()
	assert.False(t, GetCache().TryTransition(ctx, uuid.New(), SagaLifecyclePending, SagaLifecycleCompensating))
}

// TestInMemoryCache_TryTransition_ConcurrentWinner stresses the concurrent case
// from PRD §4.7 / plan Phase 2.4: two goroutines racing to transition the same
// saga Pending → Compensating. Exactly one must win.
func TestInMemoryCache_TryTransition_ConcurrentWinner(t *testing.T) {
	const goroutines = 128

	for trial := 0; trial < 10; trial++ {
		ResetCache()
		ctx := newTestCtx()
		s, _ := NewBuilder().SetSagaType(CharacterCreation).SetInitiatedBy("test").Build()
		_ = GetCache().Put(ctx, s)

		var wg sync.WaitGroup
		var winners int64

		start := make(chan struct{})
		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				if GetCache().TryTransition(ctx, s.TransactionId(), SagaLifecyclePending, SagaLifecycleCompensating) {
					atomic.AddInt64(&winners, 1)
				}
			}()
		}
		close(start)
		wg.Wait()

		assert.EqualValues(t, 1, winners, "trial %d: expected exactly one winner, got %d", trial, winners)

		state, _ := GetCache().GetLifecycle(ctx, s.TransactionId())
		assert.Equal(t, SagaLifecycleCompensating, state)
	}
}

// TestInMemoryCache_TryTransition_RaceBetweenBranches covers the timer-vs-StepCompleted
// race: one goroutine tries Pending → Compensating (failure path) while another
// tries Pending → Completed (success path). Exactly one must succeed.
func TestInMemoryCache_TryTransition_RaceBetweenBranches(t *testing.T) {
	for trial := 0; trial < 10; trial++ {
		ResetCache()
		ctx := newTestCtx()
		s, _ := NewBuilder().SetSagaType(CharacterCreation).SetInitiatedBy("test").Build()
		_ = GetCache().Put(ctx, s)

		var wg sync.WaitGroup
		var wins int64
		start := make(chan struct{})

		for i := 0; i < 32; i++ {
			wg.Add(2)
			go func() {
				defer wg.Done()
				<-start
				if GetCache().TryTransition(ctx, s.TransactionId(), SagaLifecyclePending, SagaLifecycleCompensating) {
					atomic.AddInt64(&wins, 1)
				}
			}()
			go func() {
				defer wg.Done()
				<-start
				if GetCache().TryTransition(ctx, s.TransactionId(), SagaLifecyclePending, SagaLifecycleCompleted) {
					atomic.AddInt64(&wins, 1)
				}
			}()
		}
		close(start)
		wg.Wait()

		assert.EqualValues(t, 1, wins, "trial %d: expected exactly one total winner across both branches, got %d", trial, wins)
	}
}
