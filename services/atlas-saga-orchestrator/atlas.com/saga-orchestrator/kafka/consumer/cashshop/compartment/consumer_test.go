package compartment

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"

	cashshopCompartment "atlas-saga-orchestrator/kafka/message/cashshop/compartment"
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

// TestHandleAcceptedEvent_CompletesAcceptToCashShopStep verifies that a
// cash-shop compartment ACCEPTED event completes an AcceptToCashShop step.
func TestHandleAcceptedEvent_CompletesAcceptToCashShopStep(t *testing.T) {
	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := mustTenantCtx(t)

	tx := uuid.New()
	s, err := saga.NewBuilder().
		SetTransactionId(tx).
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("s1", saga.Pending, saga.AcceptToCashShop, saga.AcceptToCashShopPayload{TransactionId: tx, CharacterId: 1}).
		AddStep("s2", saga.Pending, saga.AwardAsset, saga.AwardItemActionPayload{CharacterId: 1, Item: saga.ItemPayload{TemplateId: 2000000, Quantity: 1}}).
		Build()
	require.NoError(t, err)
	putTestSaga(t, ctx, s)

	handleAcceptedEvent(logger, ctx, cashshopCompartment.StatusEvent[cashshopCompartment.StatusEventAcceptedBody]{
		Type: cashshopCompartment.StatusEventTypeAccepted,
		Body: cashshopCompartment.StatusEventAcceptedBody{TransactionId: tx},
	})

	got, err := saga.NewProcessor(logger, ctx).GetById(tx)
	require.NoError(t, err)
	assert.Equal(t, saga.Completed, got.Steps()[0].Status(), "AcceptToCashShop step must be completed by ACCEPTED")
}

// TestHandleAcceptedEvent_DoesNotCompleteAwardAssetStep verifies anti-match:
// a cash-shop compartment ACCEPTED event must NOT complete an AwardAsset step.
func TestHandleAcceptedEvent_DoesNotCompleteAwardAssetStep(t *testing.T) {
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

	handleAcceptedEvent(logger, ctx, cashshopCompartment.StatusEvent[cashshopCompartment.StatusEventAcceptedBody]{
		Type: cashshopCompartment.StatusEventTypeAccepted,
		Body: cashshopCompartment.StatusEventAcceptedBody{TransactionId: tx},
	})

	got, err := saga.NewProcessor(logger, ctx).GetById(tx)
	require.NoError(t, err)
	assert.Equal(t, saga.Pending, got.Steps()[0].Status(), "AwardAsset step must not be completed by cash-shop ACCEPTED")

	assertDebugReason(t, hook, saga.SkipReasonActionMismatch)
}
