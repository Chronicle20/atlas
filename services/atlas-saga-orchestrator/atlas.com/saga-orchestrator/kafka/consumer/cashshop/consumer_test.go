package cashshop

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"

	cashshop2 "atlas-saga-orchestrator/kafka/message/cashshop"
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

// TestHandleWalletUpdatedEvent_CompletesAwardCurrencyStep verifies that a
// WALLET UPDATED event completes an AwardCurrency step.
func TestHandleWalletUpdatedEvent_CompletesAwardCurrencyStep(t *testing.T) {
	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := mustTenantCtx(t)

	tx := uuid.New()
	s, err := saga.NewBuilder().
		SetTransactionId(tx).
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("s1", saga.Pending, saga.AwardCurrency, saga.AwardCurrencyPayload{CharacterId: 1, AccountId: 10, CurrencyType: 1, Amount: 100}).
		AddStep("s2", saga.Pending, saga.SendMessage, saga.SendMessagePayload{CharacterId: 1, Message: "pinned"}).
		Build()
	require.NoError(t, err)
	putTestSaga(t, ctx, s)

	handleWalletUpdatedEvent(logger, ctx, cashshop2.StatusEvent[cashshop2.StatusEventUpdatedBody]{
		Type:      cashshop2.StatusEventTypeUpdated,
		AccountId: 10,
		Body:      cashshop2.StatusEventUpdatedBody{TransactionId: tx, Credit: 100},
	})

	got, err := saga.NewProcessor(logger, ctx).GetById(tx)
	require.NoError(t, err)
	assert.Equal(t, saga.Completed, got.Steps()[0].Status(), "AwardCurrency step must be completed by WALLET UPDATED")
}

// TestHandleWalletUpdatedEvent_DoesNotCompleteAwardAssetStep verifies anti-match:
// a WALLET UPDATED event must NOT complete an AwardAsset step.
func TestHandleWalletUpdatedEvent_DoesNotCompleteAwardAssetStep(t *testing.T) {
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

	handleWalletUpdatedEvent(logger, ctx, cashshop2.StatusEvent[cashshop2.StatusEventUpdatedBody]{
		Type:      cashshop2.StatusEventTypeUpdated,
		AccountId: 10,
		Body:      cashshop2.StatusEventUpdatedBody{TransactionId: tx, Credit: 100},
	})

	got, err := saga.NewProcessor(logger, ctx).GetById(tx)
	require.NoError(t, err)
	assert.Equal(t, saga.Pending, got.Steps()[0].Status(), "AwardAsset step must not be completed by WALLET UPDATED")

	assertDebugReason(t, hook, saga.SkipReasonActionMismatch)
}
