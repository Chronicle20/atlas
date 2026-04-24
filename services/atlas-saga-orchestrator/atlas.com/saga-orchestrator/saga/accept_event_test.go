package saga

import (
	"context"
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func acceptEventTestCtx(t *testing.T) context.Context {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), tm)
}

func newAcceptEventTestProcessor(t *testing.T) (Processor, *logtest.Hook, context.Context) {
	t.Helper()
	logger, hook := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := acceptEventTestCtx(t)
	return NewProcessor(logger, ctx), hook, ctx
}

func putAcceptEventSaga(t *testing.T, ctx context.Context, s Saga) {
	t.Helper()
	require.NoError(t, GetCache().Put(ctx, s))
}

func TestAcceptEvent_SagaNotFound(t *testing.T) {
	p, hook, _ := newAcceptEventTestProcessor(t)
	_, ok := p.AcceptEvent(uuid.New(), EventKindAssetCreated)
	assert.False(t, ok, "AcceptEvent returns false when saga is not found")

	entries := hook.AllEntries()
	require.Len(t, entries, 1)
	assert.Equal(t, logrus.DebugLevel, entries[0].Level)
	assert.Equal(t, skipReasonSagaNotFound, entries[0].Data["reason"])
}

func TestAcceptEvent_NoPendingStep(t *testing.T) {
	p, hook, ctx := newAcceptEventTestProcessor(t)
	tx := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("s1", Completed, SendMessage, SendMessagePayload{CharacterId: 1, Message: "hi"}).
		Build()
	require.NoError(t, err)
	putAcceptEventSaga(t, ctx, s)

	_, ok := p.AcceptEvent(tx, EventKindAssetCreated)
	assert.False(t, ok)

	require.Len(t, hook.AllEntries(), 1)
	assert.Equal(t, skipReasonNoPendingStep, hook.LastEntry().Data["reason"])
}

func TestAcceptEvent_ActionMismatch(t *testing.T) {
	p, hook, ctx := newAcceptEventTestProcessor(t)
	tx := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("s1", Pending, AwardAsset, AwardItemActionPayload{
			CharacterId: 1,
			Item:        ItemPayload{TemplateId: 2070015, Quantity: 1},
			ShowEffect:  true,
		}).
		Build()
	require.NoError(t, err)
	putAcceptEventSaga(t, ctx, s)

	_, ok := p.AcceptEvent(tx, EventKindCharacterStatChanged)
	assert.False(t, ok, "STAT_CHANGED must not match AwardAsset (§9.1 bug)")

	require.Len(t, hook.AllEntries(), 1)
	entry := hook.LastEntry()
	assert.Equal(t, skipReasonActionMismatch, entry.Data["reason"])
	assert.Equal(t, AwardAsset, entry.Data["step_action"])
	assert.Equal(t, "s1", entry.Data["step_id"])
	assert.Equal(t, EventKindCharacterStatChanged, entry.Data["event_kind"])
	assert.Equal(t, tx.String(), entry.Data["transaction_id"])
}

func TestAcceptEvent_Match(t *testing.T) {
	p, _, ctx := newAcceptEventTestProcessor(t)
	tx := uuid.New()
	s, err := NewBuilder().
		SetTransactionId(tx).
		SetSagaType(InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("s1", Pending, AwardAsset, AwardItemActionPayload{
			CharacterId: 1,
			Item:        ItemPayload{TemplateId: 2070015, Quantity: 1},
		}).
		Build()
	require.NoError(t, err)
	putAcceptEventSaga(t, ctx, s)

	decision, ok := p.AcceptEvent(tx, EventKindAssetCreated)
	require.True(t, ok)
	assert.Equal(t, "s1", decision.Step.StepId())
	assert.Equal(t, AwardAsset, decision.Step.Action())
	assert.Equal(t, tx, decision.Saga.TransactionId())
}
