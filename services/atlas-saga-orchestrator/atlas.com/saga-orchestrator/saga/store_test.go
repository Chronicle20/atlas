package saga

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func newStoreTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, Migration(db))
	return db
}

func newStoreTestCtx(t *testing.T) context.Context {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), tm)
}

func newTestStoreSaga(t *testing.T) Saga {
	t.Helper()
	s, err := NewBuilder().
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("s1", Pending, AwardCurrency, AwardCurrencyPayload{CharacterId: 1, AccountId: 2, CurrencyType: 2, Amount: 100}).
		Build()
	require.NoError(t, err)
	return s
}

// TestPostgresStore_TryTransitionBumpsVersion: an optimistic Put built on a
// pre-terminal read must fail with VersionConflictError once the terminal
// transition commits (design §3.3b).
func TestPostgresStore_TryTransitionBumpsVersion(t *testing.T) {
	logger, _ := test.NewNullLogger()
	store := NewPostgresStore(newStoreTestDB(t), logger)
	ctx := newStoreTestCtx(t)
	s := newTestStoreSaga(t)

	require.NoError(t, store.Put(ctx, s))
	_, ok := store.GetById(ctx, s.TransactionId()) // tracks version 1
	require.True(t, ok)

	require.True(t, store.TryTransition(ctx, s.TransactionId(), SagaLifecyclePending, SagaLifecycleCompensating))

	err := store.Put(ctx, s) // built on the stale (pre-transition) version
	var vce *VersionConflictError
	require.ErrorAs(t, err, &vce)
}

// TestPostgresStore_PutCannotResurrectTerminal: a Put built on a FRESH read of
// a terminal saga updates saga_data but cannot regress status (design §3.3c).
func TestPostgresStore_PutCannotResurrectTerminal(t *testing.T) {
	logger, _ := test.NewNullLogger()
	store := NewPostgresStore(newStoreTestDB(t), logger)
	ctx := newStoreTestCtx(t)
	s := newTestStoreSaga(t)

	require.NoError(t, store.Put(ctx, s))
	require.True(t, store.TryTransition(ctx, s.TransactionId(), SagaLifecyclePending, SagaLifecycleCompensating))
	require.True(t, store.TryTransition(ctx, s.TransactionId(), SagaLifecycleCompensating, SagaLifecycleFailed))

	fresh, ok := store.GetById(ctx, s.TransactionId()) // re-tracks post-bump version
	require.True(t, ok)
	marked, err := fresh.WithStepLateCompensated(0)
	require.NoError(t, err)
	require.NoError(t, store.Put(ctx, marked)) // succeeds: fresh version

	lc, ok := store.GetLifecycle(ctx, s.TransactionId())
	require.True(t, ok)
	assert.Equal(t, SagaLifecycleFailed, lc, "Put must not regress failed status")

	reread, ok := store.GetById(ctx, s.TransactionId())
	require.True(t, ok)
	st, _ := reread.StepAt(0)
	assert.True(t, st.LateCompensated(), "saga_data update must still land")
}

// TestPostgresStore_RemovePreservesFailed: Remove collapses active/compensating
// to completed but must not erase the failed audit state (design defect 3).
func TestPostgresStore_RemovePreservesFailed(t *testing.T) {
	logger, _ := test.NewNullLogger()
	store := NewPostgresStore(newStoreTestDB(t), logger)
	ctx := newStoreTestCtx(t)

	failed := newTestStoreSaga(t)
	require.NoError(t, store.Put(ctx, failed))
	require.True(t, store.TryTransition(ctx, failed.TransactionId(), SagaLifecyclePending, SagaLifecycleCompensating))
	require.True(t, store.TryTransition(ctx, failed.TransactionId(), SagaLifecycleCompensating, SagaLifecycleFailed))
	assert.True(t, store.Remove(ctx, failed.TransactionId()))
	lc, ok := store.GetLifecycle(ctx, failed.TransactionId())
	require.True(t, ok)
	assert.Equal(t, SagaLifecycleFailed, lc)

	active := newTestStoreSaga(t)
	require.NoError(t, store.Put(ctx, active))
	assert.True(t, store.Remove(ctx, active.TransactionId()))
	lc, ok = store.GetLifecycle(ctx, active.TransactionId())
	require.True(t, ok)
	assert.Equal(t, SagaLifecycleCompleted, lc)
}
