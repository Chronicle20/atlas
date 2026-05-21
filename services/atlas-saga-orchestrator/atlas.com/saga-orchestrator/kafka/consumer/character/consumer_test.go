package character

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"

	character2 "atlas-saga-orchestrator/kafka/message/character"
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

// TestHandleCharacterStatChangedEvent_DoesNotCompleteAwardAssetStep verifies
// the §9.1 bug: a STAT_CHANGED event must NOT complete an AwardAsset step.
func TestHandleCharacterStatChangedEvent_DoesNotCompleteAwardAssetStep(t *testing.T) {
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

	handleCharacterStatChangedEvent(logger, ctx, character2.StatusEvent[character2.StatusEventStatChangedBody]{
		Type:          character2.StatusEventTypeStatChanged,
		TransactionId: tx,
		CharacterId:   1,
	})

	got, err := saga.NewProcessor(logger, ctx).GetById(tx)
	require.NoError(t, err)
	assert.Equal(t, saga.Pending, got.Steps()[0].Status(), "AwardAsset step must not be completed by STAT_CHANGED")

	assertDebugReason(t, hook, saga.SkipReasonActionMismatch)
}

// TestHandleCharacterStatChangedEvent_CompletesRebalanceAPStep verifies
// that STAT_CHANGED does complete a RebalanceAP step (happy path).
func TestHandleCharacterStatChangedEvent_CompletesRebalanceAPStep(t *testing.T) {
	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := mustTenantCtx(t)

	tx := uuid.New()
	// Trailing SendMessage step keeps the saga alive in cache post-completion
	// so we can observe the RebalanceAP step status.
	s, err := saga.NewBuilder().
		SetTransactionId(tx).
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("s1", saga.Pending, saga.RebalanceAP, saga.RebalanceAPPayload{
			CharacterId: 1,
		}).
		AddStep("s2", saga.Pending, saga.AwardAsset, saga.AwardItemActionPayload{CharacterId: 1, Item: saga.ItemPayload{TemplateId: 2000000, Quantity: 1}}).
		Build()
	require.NoError(t, err)
	putTestSaga(t, ctx, s)

	handleCharacterStatChangedEvent(logger, ctx, character2.StatusEvent[character2.StatusEventStatChangedBody]{
		Type:          character2.StatusEventTypeStatChanged,
		TransactionId: tx,
		CharacterId:   1,
	})

	got, err := saga.NewProcessor(logger, ctx).GetById(tx)
	require.NoError(t, err)
	assert.Equal(t, saga.Completed, got.Steps()[0].Status(), "RebalanceAP step must be completed by STAT_CHANGED")
}

// TestHandleCharacterJobChangedEvent_CompletesChangeJobOnly verifies that a
// JOB_CHANGED event does NOT advance a non-ChangeJob step (AwardAsset here).
func TestHandleCharacterJobChangedEvent_CompletesChangeJobOnly(t *testing.T) {
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

	handleCharacterJobChangedEvent(logger, ctx, character2.StatusEvent[character2.JobChangedStatusEventBody]{
		Type:          character2.StatusEventTypeJobChanged,
		TransactionId: tx,
		CharacterId:   1,
	})

	got, err := saga.NewProcessor(logger, ctx).GetById(tx)
	require.NoError(t, err)
	assert.Equal(t, saga.Pending, got.Steps()[0].Status(), "AwardAsset step must not be completed by JOB_CHANGED")

	assertDebugReason(t, hook, saga.SkipReasonActionMismatch)
}

// TestHandleCharacterCreationFailedEvent_GatedByAction verifies that a
// CREATION_FAILED event does NOT fail a ChangeJob step (wrong action).
func TestHandleCharacterCreationFailedEvent_GatedByAction(t *testing.T) {
	logger, hook := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := mustTenantCtx(t)

	tx := uuid.New()
	s, err := saga.NewBuilder().
		SetTransactionId(tx).
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("s1", saga.Pending, saga.ChangeJob, saga.ChangeJobPayload{CharacterId: 1, JobId: 400}).
		Build()
	require.NoError(t, err)
	putTestSaga(t, ctx, s)

	handleCharacterCreationFailedEvent(logger, ctx, character2.StatusEvent[character2.StatusEventCreationFailedBody]{
		Type:          character2.StatusEventTypeCreationFailed,
		TransactionId: tx,
		CharacterId:   1,
		Body:          character2.StatusEventCreationFailedBody{Name: "x", Message: "err"},
	})

	got, err := saga.NewProcessor(logger, ctx).GetById(tx)
	require.NoError(t, err)
	assert.Equal(t, saga.Pending, got.Steps()[0].Status(), "ChangeJob step must not be completed by CREATION_FAILED")

	assertDebugReason(t, hook, saga.SkipReasonActionMismatch)
}
