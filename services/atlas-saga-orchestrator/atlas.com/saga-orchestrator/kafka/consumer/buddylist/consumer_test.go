package buddylist

import (
	"atlas-saga-orchestrator/saga"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"

	buddylist2 "atlas-saga-orchestrator/kafka/message/buddylist"
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

// TestHandleBuddyCapacityChangedEvent_CompletesIncreaseBuddyCapacityStep verifies
// that a buddy CAPACITY_CHANGE event completes an IncreaseBuddyCapacity step.
func TestHandleBuddyCapacityChangedEvent_CompletesIncreaseBuddyCapacityStep(t *testing.T) {
	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := mustTenantCtx(t)

	tx := uuid.New()
	s, err := saga.NewBuilder().
		SetTransactionId(tx).
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("s1", saga.Pending, saga.IncreaseBuddyCapacity, saga.IncreaseBuddyCapacityPayload{CharacterId: 1, Amount: 5}).
		AddStep("s2", saga.Pending, saga.AwardAsset, saga.AwardItemActionPayload{CharacterId: 1, Item: saga.ItemPayload{TemplateId: 2000000, Quantity: 1}}).
		Build()
	require.NoError(t, err)
	putTestSaga(t, ctx, s)

	handleBuddyCapacityChangedEvent(logger, ctx, buddylist2.StatusEvent[buddylist2.BuddyCapacityChangeStatusEventBody]{
		Type:        buddylist2.StatusEventTypeBuddyCapacityUpdate,
		CharacterId: 1,
		Body:        buddylist2.BuddyCapacityChangeStatusEventBody{Capacity: 25, TransactionId: tx},
	})

	got, err := saga.NewProcessor(logger, ctx).GetById(tx)
	require.NoError(t, err)
	assert.Equal(t, saga.Completed, got.Steps()[0].Status(), "IncreaseBuddyCapacity step must be completed by buddy CAPACITY_CHANGE")
}

// TestHandleBuddyCapacityChangedEvent_DoesNotCompleteAwardAssetStep verifies anti-match.
func TestHandleBuddyCapacityChangedEvent_DoesNotCompleteAwardAssetStep(t *testing.T) {
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

	handleBuddyCapacityChangedEvent(logger, ctx, buddylist2.StatusEvent[buddylist2.BuddyCapacityChangeStatusEventBody]{
		Type:        buddylist2.StatusEventTypeBuddyCapacityUpdate,
		CharacterId: 1,
		Body:        buddylist2.BuddyCapacityChangeStatusEventBody{Capacity: 25, TransactionId: tx},
	})

	got, err := saga.NewProcessor(logger, ctx).GetById(tx)
	require.NoError(t, err)
	assert.Equal(t, saga.Pending, got.Steps()[0].Status(), "AwardAsset step must not be completed by buddy CAPACITY_CHANGE")

	assertDebugReason(t, hook, saga.SkipReasonActionMismatch)
}

// TestHandleBuddyRemovedEvent_PartialAndOutOfOrderAcknowledgementLeavesStepPending
// proves fix round 1 finding 1: a sever_buddies_for_transfer step with N
// buddies is 2N in-flight severances (one command per direction per buddy —
// see handleSeverBuddiesForTransfer in saga/handler.go), and the step must
// NOT complete until every one of them has landed. This drives the real
// production path — saga.RegisterSeveranceTracker (what the handler calls
// before emitting any command) and saga.AcknowledgeSeverance (what this
// consumer calls on every BUDDY_REMOVED) — through handleBuddyRemovedEvent
// itself, feeding acks out of the "natural" forward/reverse order to prove
// the step genuinely tracks the full set rather than completing on count or
// on the first event.
func TestHandleBuddyRemovedEvent_PartialAndOutOfOrderAcknowledgementLeavesStepPending(t *testing.T) {
	logger, _ := logtest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	ctx := mustTenantCtx(t)

	const characterId = uint32(1)
	buddyIds := []uint32{2, 3}

	tx := uuid.New()
	s, err := saga.NewBuilder().
		SetTransactionId(tx).
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("s1", saga.Pending, saga.SeverBuddiesForTransfer, saga.SeverBuddiesForTransferPayload{
			CharacterId: characterId, WorldId: 0, BuddyIds: buddyIds,
		}).
		AddStep("s2", saga.Pending, saga.AwardAsset, saga.AwardItemActionPayload{
			CharacterId: characterId, Item: saga.ItemPayload{TemplateId: 2000000, Quantity: 1},
		}).
		Build()
	require.NoError(t, err)
	putTestSaga(t, ctx, s)

	// Mirrors handleSeverBuddiesForTransfer: register BEFORE any ack arrives.
	saga.RegisterSeveranceTracker(tx, characterId, buddyIds)

	send := func(owner, removed uint32) {
		t.Helper()
		handleBuddyRemovedEvent(logger, ctx, buddylist2.StatusEvent[buddylist2.BuddyRemovedStatusEventBody]{
			Type:        buddylist2.StatusEventTypeBuddyRemoved,
			CharacterId: character.Id(owner),
			Body:        buddylist2.BuddyRemovedStatusEventBody{CharacterId: character.Id(removed), TransactionId: tx},
		})
	}

	assertPending := func(msg string) {
		t.Helper()
		got, err := saga.NewProcessor(logger, ctx).GetById(tx)
		require.NoError(t, err)
		assert.Equal(t, saga.Pending, got.Steps()[0].Status(), msg)
	}

	// 4 severances are expected for BuddyIds=[2,3]: (1,2) (2,1) (1,3) (3,1).
	// Deliver 1 of 4 — the step must remain pending.
	send(1, 2)
	assertPending("step must stay pending after 1 of 4 severances")

	// Redeliver the same one (Kafka at-least-once) — must not be miscounted
	// as a distinct severance.
	send(1, 2)
	assertPending("a redelivered ack must not advance the step")

	// Deliver 2 more, out of forward/reverse order — still 3 of 4.
	send(3, 1)
	assertPending("step must stay pending after 2 of 4 severances")
	send(1, 3)
	assertPending("step must stay pending after 3 of 4 severances")

	// The last of the 4 lands — only now may the step complete.
	send(2, 1)
	got, err := saga.NewProcessor(logger, ctx).GetById(tx)
	require.NoError(t, err)
	assert.Equal(t, saga.Completed, got.Steps()[0].Status(), "step must complete once all 4 severances have landed")
}
