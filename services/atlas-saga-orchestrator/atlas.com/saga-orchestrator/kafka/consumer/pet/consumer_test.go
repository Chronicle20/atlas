package pet

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"

	pet2 "atlas-saga-orchestrator/kafka/message/pet"
	"atlas-saga-orchestrator/saga"
)

func mustTenantCtx(t *testing.T) context.Context {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), tm)
}

func putTestSaga(t *testing.T, ctx context.Context, s saga.Saga) {
	t.Helper()
	require.NoError(t, saga.GetCache().Put(ctx, s))
}

func assertDebugReason(t *testing.T, hook *logtest.Hook, want string) {
	t.Helper()
	for _, e := range hook.AllEntries() {
		if r, ok := e.Data["reason"]; ok && r == want {
			return
		}
	}
	t.Fatalf("expected a debug log with reason=%q; got: %+v", want, hook.AllEntries())
}

// TestHandleClosenessChangedEvent_CompletesGainClosenessStep verifies that a
// CLOSENESS_CHANGED event completes a GainCloseness step.
func TestHandleClosenessChangedEvent_CompletesGainClosenessStep(t *testing.T) {
	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := mustTenantCtx(t)

	tx := uuid.New()
	s, err := saga.NewBuilder().
		SetTransactionId(tx).
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("s1", saga.Pending, saga.GainCloseness, saga.GainClosenessPayload{PetId: 1, Amount: 10}).
		AddStep("s2", saga.Pending, saga.AwardAsset, saga.AwardItemActionPayload{CharacterId: 1, Item: saga.ItemPayload{TemplateId: 2000000, Quantity: 1}}).
		Build()
	require.NoError(t, err)
	putTestSaga(t, ctx, s)

	handleClosenessChangedEvent(logger, ctx, pet2.StatusEvent[pet2.ClosenessChangedStatusEventBody]{
		Type:    pet2.StatusEventTypeClosenessChanged,
		PetId:   1,
		OwnerId: 2,
		Body:    pet2.ClosenessChangedStatusEventBody{Closeness: 50, Amount: 10, TransactionId: tx},
	})

	got, err := saga.NewProcessor(logger, ctx).GetById(tx)
	require.NoError(t, err)
	assert.Equal(t, saga.Completed, got.Steps()[0].Status(), "GainCloseness step must be completed by CLOSENESS_CHANGED")
}

// TestHandleClosenessChangedEvent_DoesNotCompleteAwardAssetStep verifies anti-match.
func TestHandleClosenessChangedEvent_DoesNotCompleteAwardAssetStep(t *testing.T) {
	logger, hook := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := mustTenantCtx(t)

	tx := uuid.New()
	s, err := saga.NewBuilder().
		SetTransactionId(tx).
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("s1", saga.Pending, saga.AwardAsset, saga.AwardItemActionPayload{
			CharacterId: 1,
			Item:        saga.ItemPayload{TemplateId: 2070015, Quantity: 1},
		}).
		Build()
	require.NoError(t, err)
	putTestSaga(t, ctx, s)

	handleClosenessChangedEvent(logger, ctx, pet2.StatusEvent[pet2.ClosenessChangedStatusEventBody]{
		Type:    pet2.StatusEventTypeClosenessChanged,
		PetId:   1,
		OwnerId: 2,
		Body:    pet2.ClosenessChangedStatusEventBody{Closeness: 50, Amount: 10, TransactionId: tx},
	})

	got, err := saga.NewProcessor(logger, ctx).GetById(tx)
	require.NoError(t, err)
	assert.Equal(t, saga.Pending, got.Steps()[0].Status(), "AwardAsset step must not be completed by CLOSENESS_CHANGED")

	assertDebugReason(t, hook, saga.SkipReasonActionMismatch)
}
