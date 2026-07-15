//go:build test

package saga

import (
	"context"
	"testing"
	"time"

	cashshopmock "atlas-saga-orchestrator/cashshop/mock"
	compartmentmock "atlas-saga-orchestrator/compartment/mock"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// TestLateEvent_TimeoutRacesCompletion reproduces the task-102 production
// sequence (PRD §1, design §4) deterministically: the timeout path runs to
// terminal first, then the in-flight award_currency_seller success arrives.
func TestLateEvent_TimeoutRacesCompletion(t *testing.T) {
	logger, hook := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, Migration(db))
	SetCache(NewPostgresStore(db, logger))
	t.Cleanup(ResetCache)

	failedEvents := 0
	restore := SetEmitSagaFailedForTest(func(logrus.FieldLogger, context.Context, uuid.UUID, string, uint32, uint32, string, string, string) error {
		failedEvents++
		return nil
	})
	t.Cleanup(func() { SetEmitSagaFailedForTest(restore) })

	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tm)

	// Two-step value-transfer saga: the seller credit is in flight at timeout.
	tx := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("mts-buy-test").
		AddStep("award_currency_seller", Pending, AwardCurrency, AwardCurrencyPayload{CharacterId: 1, AccountId: 42, CurrencyType: 2, Amount: 110}).
		AddStep("move_listing_to_holding", Pending, AwardAsset, AwardItemActionPayload{CharacterId: 1, Item: ItemPayload{TemplateId: 2000000, Quantity: 1}}).
		Build()
	require.NoError(t, err)
	require.NoError(t, GetCache().Put(ctx, s))

	// 1. Timeout fires (invoked directly — no real timers): pending →
	//    compensating → failed → Remove (failed preserved) + one Failed event.
	handleSagaTimeout(logger, ctx, tx, 30*time.Second)
	lc, ok := GetCache().GetLifecycle(ctx, tx)
	require.True(t, ok)
	require.Equal(t, SagaLifecycleFailed, lc)
	require.Equal(t, 1, failedEvents)

	// 2. ~100ms later the seller-credit success arrives on the real processor path.
	refunds := 0
	cs := &cashshopmock.ProcessorMock{
		AwardCurrencyAndEmitFunc: func(txId uuid.UUID, accountId uint32, currencyType uint32, amount int32) error {
			refunds++
			assert.Equal(t, uint32(42), accountId)
			assert.Equal(t, int32(-110), amount, "seller's late payment must be clawed back")
			return nil
		},
	}
	forward := 0
	cp := &compartmentmock.ProcessorMock{
		RequestCreateItemFunc:          func(uuid.UUID, uint32, uint32, uint32, time.Time) error { forward++; return nil },
		RequestCreateItemWithStatsFunc: func(uuid.UUID, uint32, uint32, uint32, time.Time, bool) error { forward++; return nil },
	}
	p := NewProcessor(logger, ctx).WithCashshopProcessor(cs).WithCompartmentProcessor(cp)

	_, accepted := p.AcceptEvent(tx, EventKindCashShopWalletUpdated)
	assert.False(t, accepted, "(a) no forward progress")
	assert.Equal(t, 1, refunds, "(b) exactly one inverse dispatched")
	assert.Equal(t, 0, forward, "(a) next step never dispatched")

	lc, ok = GetCache().GetLifecycle(ctx, tx)
	require.True(t, ok)
	assert.Equal(t, SagaLifecycleFailed, lc, "(c) saga stays terminal")
	assert.Equal(t, 1, failedEvents, "(d) exactly one Failed overall")

	var absorbed bool
	for _, e := range hook.AllEntries() {
		if e.Data["reason"] == SkipReasonSagaTerminal {
			absorbed = true
		}
	}
	assert.True(t, absorbed, "absorb must be logged with saga_terminal reason")

	// 3. Kafka at-least-once: the same event redelivered dispatches nothing (e).
	_, accepted = p.AcceptEvent(tx, EventKindCashShopWalletUpdated)
	assert.False(t, accepted)
	assert.Equal(t, 1, refunds, "(e) marker prevents double-compensation")
	assert.Equal(t, 1, failedEvents)
}

// TestLateEvent_FailureOutcomeAbsorbOnly: a late FAILURE report needs no
// rollback — the step's effect never landed (PRD §4.3).
func TestLateEvent_FailureOutcomeAbsorbOnly(t *testing.T) {
	logger, hook := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, Migration(db))
	SetCache(NewPostgresStore(db, logger))
	t.Cleanup(ResetCache)

	restore := SetEmitSagaFailedForTest(func(logrus.FieldLogger, context.Context, uuid.UUID, string, uint32, uint32, string, string, string) error {
		return nil
	})
	t.Cleanup(func() { SetEmitSagaFailedForTest(restore) })

	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), tm)

	tx := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("award_currency_seller", Pending, AwardCurrency, AwardCurrencyPayload{CharacterId: 1, AccountId: 42, CurrencyType: 2, Amount: 110}).
		Build()
	require.NoError(t, err)
	require.NoError(t, GetCache().Put(ctx, s))
	handleSagaTimeout(logger, ctx, tx, 30*time.Second)

	refunds := 0
	cs := &cashshopmock.ProcessorMock{
		AwardCurrencyAndEmitFunc: func(uuid.UUID, uint32, uint32, int32) error { refunds++; return nil },
	}
	p := NewProcessor(logger, ctx).WithCashshopProcessor(cs)

	// cashshop has no failure kind for AwardCurrency in the acceptance table,
	// so exercise the generic path: a failure-classified kind that matches no
	// step absorbs without dispatch, and a StepCompleted(false) via the
	// commit-time gate also absorbs without dispatch.
	require.NoError(t, p.StepCompleted(tx, false))
	assert.Equal(t, 0, refunds, "failure outcome dispatches nothing")

	var absorbed bool
	for _, e := range hook.AllEntries() {
		if e.Data["reason"] == SkipReasonSagaTerminal {
			absorbed = true
		}
	}
	assert.True(t, absorbed)

	st, ok := GetCache().GetById(ctx, tx)
	require.True(t, ok)
	step, _ := st.StepAt(0)
	assert.False(t, step.LateCompensated(), "no claim on failure outcome")
	assert.Equal(t, Pending, step.Status(), "no step-status mutation")
}
