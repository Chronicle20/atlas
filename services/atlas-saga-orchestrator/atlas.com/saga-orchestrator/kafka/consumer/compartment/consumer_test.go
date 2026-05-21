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

	"atlas-saga-orchestrator/kafka/message/compartment"
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

// assertDebugReason asserts at least one debug entry in the hook has the
// given `reason` field. Avoids brittle index coupling.
func assertDebugReason(t *testing.T, hook *logtest.Hook, want string) {
	t.Helper()
	for _, e := range hook.AllEntries() {
		if r, ok := e.Data["reason"]; ok && r == want {
			return
		}
	}
	t.Fatalf("expected a debug log with reason=%q; got: %+v", want, hook.AllEntries())
}

// TestHandleCompartmentErrorEvent_DoesNotFailUnrelatedStep proves that a
// COMPARTMENT_ERROR event does NOT fail a step whose action is not compartment-related.
func TestHandleCompartmentErrorEvent_DoesNotFailUnrelatedStep(t *testing.T) {
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

	handleCompartmentErrorEvent(logger, ctx, compartment.StatusEvent[compartment.ErrorEventBody]{
		Type:          compartment.StatusEventTypeError,
		TransactionId: tx,
		CharacterId:   1,
		Body: compartment.ErrorEventBody{
			ErrorCode:     "SOME_ERROR",
			TransactionId: tx,
		},
	})

	got, err := saga.NewProcessor(logger, ctx).GetById(tx)
	require.NoError(t, err)
	assert.Equal(t, saga.Pending, got.Steps()[0].Status(), "AwardAsset step must not be failed by COMPARTMENT_ERROR")

	assertDebugReason(t, hook, saga.SkipReasonActionMismatch)
}

// TestHandleCompartmentAcceptedEvent_CompletesAcceptToCharacterStep proves
// that a COMPARTMENT_ACCEPTED event advances an AcceptToCharacter step.
func TestHandleCompartmentAcceptedEvent_CompletesAcceptToCharacterStep(t *testing.T) {
	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := mustTenantCtx(t)

	tx := uuid.New()
	// Add a trailing pending step so the saga doesn't terminate (and get
	// evicted from the cache) after the AcceptToCharacter step completes.
	s, err := saga.NewBuilder().
		SetTransactionId(tx).
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("s1", saga.Pending, saga.AcceptToCharacter, saga.AcceptToCharacterPayload{
			TransactionId: tx,
			CharacterId:   1,
			InventoryType: 2,
			TemplateId:    2070015,
		}).
		AddStep("s2", saga.Pending, saga.AwardAsset, saga.AwardItemActionPayload{CharacterId: 1, Item: saga.ItemPayload{TemplateId: 2000000, Quantity: 1}}).
		Build()
	require.NoError(t, err)
	putTestSaga(t, ctx, s)

	handleCompartmentAcceptedEvent(logger, ctx, compartment.StatusEvent[compartment.AcceptedEventBody]{
		Type:          compartment.StatusEventTypeAccepted,
		TransactionId: uuid.New(), // envelope has a different id; inner body wins
		CharacterId:   1,
		CompartmentId: uuid.New(),
		Body: compartment.AcceptedEventBody{
			TransactionId: tx,
		},
	})

	got, err := saga.NewProcessor(logger, ctx).GetById(tx)
	require.NoError(t, err)
	assert.Equal(t, saga.Completed, got.Steps()[0].Status())
}
