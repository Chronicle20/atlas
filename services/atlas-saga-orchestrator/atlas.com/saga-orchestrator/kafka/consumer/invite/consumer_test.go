package invite

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	invitec "github.com/Chronicle20/atlas/libs/atlas-constants/invite"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"

	invite2 "atlas-saga-orchestrator/kafka/message/invite"
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

// TestHandleInviteAcceptedEvent_CompletesCreateInviteStep verifies that an
// INVITE ACCEPTED event completes a CreateInvite step.
func TestHandleInviteAcceptedEvent_CompletesCreateInviteStep(t *testing.T) {
	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := mustTenantCtx(t)

	tx := uuid.New()
	s, err := saga.NewBuilder().
		SetTransactionId(tx).
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("s1", saga.Pending, saga.CreateInvite, saga.CreateInvitePayload{InviteType: "GUILD", OriginatorId: 1, TargetId: 2}).
		AddStep("s2", saga.Pending, saga.AwardAsset, saga.AwardItemActionPayload{CharacterId: 1, Item: saga.ItemPayload{TemplateId: 2000000, Quantity: 1}}).
		Build()
	require.NoError(t, err)
	putTestSaga(t, ctx, s)

	handleAcceptedStatusEvent(logger, ctx, invite2.StatusEvent[invite2.AcceptedEventBody]{
		Type:          invitec.StatusTypeAccepted,
		TransactionId: tx,
	})

	got, err := saga.NewProcessor(logger, ctx).GetById(tx)
	require.NoError(t, err)
	assert.Equal(t, saga.Completed, got.Steps()[0].Status(), "CreateInvite step must be completed by INVITE ACCEPTED")
}

// TestHandleInviteRejectedEvent_DoesNotFailUnrelatedStep verifies anti-match on
// the failure path: a saga whose active step is AwardAsset (not CreateInvite)
// must NOT be failed by an INVITE REJECTED event.
func TestHandleInviteRejectedEvent_DoesNotFailUnrelatedStep(t *testing.T) {
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

	handleRejectedStatusEvent(logger, ctx, invite2.StatusEvent[invite2.RejectedEventBody]{
		Type:          invitec.StatusTypeRejected,
		TransactionId: tx,
	})

	got, err := saga.NewProcessor(logger, ctx).GetById(tx)
	require.NoError(t, err)
	assert.Equal(t, saga.Pending, got.Steps()[0].Status(), "AwardAsset step must remain Pending under INVITE REJECTED")

	assertDebugReason(t, hook, saga.SkipReasonActionMismatch)
}
