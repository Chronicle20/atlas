package npcshop

import (
	"atlas-saga-orchestrator/saga"
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"

	npcshop "atlas-saga-orchestrator/kafka/message/npcshop"
)

func newTenantContext(t *testing.T) context.Context {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), tm)
}

func putTestSaga(t *testing.T, ctx context.Context, s saga.Saga) {
	t.Helper()
	require.NoError(t, saga.GetCache().Put(ctx, s))
}

// TestHandleEnteredEvent_IgnoresWrongType guards the shared-topic fan-out: every
// handler registered on EVENT_TOPIC_NPC_SHOP_STATUS sees every event, so the
// type check is what stops an EXITED from completing an open_npc_shop step.
func TestHandleEnteredEvent_IgnoresWrongType(t *testing.T) {
	l, hook := test.NewNullLogger()
	ctx := newTenantContext(t)

	handleEnteredEvent(l, ctx, npcshop.StatusEvent[npcshop.StatusEventEnteredBody]{
		TransactionId: uuid.New(),
		CharacterId:   1234,
		Type:          npcshop.StatusEventTypeExited,
	})

	if len(hook.Entries) != 0 {
		t.Errorf("handler acted on a non-ENTERED event: %v", hook.Entries)
	}
}

// TestHandleEnteredEvent_NilTransactionIgnored: the ordinary NPC-talk path
// produces ENTER with uuid.Nil, and every one of those events lands here. It
// must never advance a saga.
func TestHandleEnteredEvent_NilTransactionIgnored(t *testing.T) {
	l, hook := test.NewNullLogger()
	ctx := newTenantContext(t)

	handleEnteredEvent(l, ctx, npcshop.StatusEvent[npcshop.StatusEventEnteredBody]{
		TransactionId: uuid.Nil,
		CharacterId:   1234,
		Type:          npcshop.StatusEventTypeEntered,
	})

	if len(hook.Entries) != 0 {
		t.Errorf("handler acted on a uuid.Nil transaction: %v", hook.Entries)
	}
}

// TestHandleEnterErrorEvent_NilTransactionIgnored
func TestHandleEnterErrorEvent_NilTransactionIgnored(t *testing.T) {
	l, hook := test.NewNullLogger()
	ctx := newTenantContext(t)

	handleEnterErrorEvent(l, ctx, npcshop.StatusEvent[npcshop.StatusEventEnterErrorBody]{
		TransactionId: uuid.Nil,
		CharacterId:   1234,
		Type:          npcshop.StatusEventTypeEnterError,
		Body:          npcshop.StatusEventEnterErrorBody{Reason: npcshop.EnterErrorShopNotFound},
	})

	if len(hook.Entries) != 0 {
		t.Errorf("handler acted on a uuid.Nil transaction: %v", hook.Entries)
	}
}

// TestHandleEnteredEvent_CompletesOpenNpcShopStep verifies the ENTERED→success
// polarity: an ENTERED status event completes a pending open_npc_shop step.
func TestHandleEnteredEvent_CompletesOpenNpcShopStep(t *testing.T) {
	l, _ := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)
	ctx := newTenantContext(t)

	tx := uuid.New()
	s, err := saga.NewBuilder().
		SetTransactionId(tx).
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("s1", saga.Pending, saga.OpenNpcShop, saga.OpenNpcShopPayload{CharacterId: 1234, NpcTemplateId: 9090000}).
		AddStep("s2", saga.Pending, saga.AwardAsset, saga.AwardItemActionPayload{CharacterId: 1234, Item: saga.ItemPayload{TemplateId: 2000000, Quantity: 1}}).
		Build()
	require.NoError(t, err)
	putTestSaga(t, ctx, s)

	handleEnteredEvent(l, ctx, npcshop.StatusEvent[npcshop.StatusEventEnteredBody]{
		TransactionId: tx,
		CharacterId:   1234,
		Type:          npcshop.StatusEventTypeEntered,
		Body:          npcshop.StatusEventEnteredBody{NpcTemplateId: 9090000},
	})

	got, err := saga.NewProcessor(l, ctx).GetById(tx)
	require.NoError(t, err)
	assert.Equal(t, saga.Completed, got.Steps()[0].Status(), "open_npc_shop step must be completed by npcshop ENTERED")
}

// TestHandleEnterErrorEvent_FailsOpenNpcShopStep verifies the
// ENTER_ERROR→failure polarity: an ENTER_ERROR status event drives
// StepCompleted(..., false) for a pending open_npc_shop step (task-221 FR-4.4
// — the cash item must survive). It asserts via the emitted logs rather than
// final saga state: OpenNpcShop has no compensator entry yet (that lands in
// task-221 Task 9), so the compensator's no-op default case reverts the step
// back to Pending immediately after marking it Failed — an implementation
// detail of the not-yet-built compensation step, not of this handler's
// success/failure polarity.
func TestHandleEnterErrorEvent_FailsOpenNpcShopStep(t *testing.T) {
	l, hook := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)
	ctx := newTenantContext(t)

	tx := uuid.New()
	s, err := saga.NewBuilder().
		SetTransactionId(tx).
		SetSagaType(saga.InventoryTransaction).
		SetInitiatedBy("test").
		AddStep("s1", saga.Pending, saga.OpenNpcShop, saga.OpenNpcShopPayload{CharacterId: 1234, NpcTemplateId: 9090000}).
		Build()
	require.NoError(t, err)
	putTestSaga(t, ctx, s)

	handleEnterErrorEvent(l, ctx, npcshop.StatusEvent[npcshop.StatusEventEnterErrorBody]{
		TransactionId: tx,
		CharacterId:   1234,
		Type:          npcshop.StatusEventTypeEnterError,
		Body:          npcshop.StatusEventEnterErrorBody{NpcTemplateId: 9090000, Reason: npcshop.EnterErrorShopNotFound},
	})

	foundMarkedFailed := false
	for _, e := range hook.AllEntries() {
		if e.Message == "Marked earliest pending step as [failed]." && e.Data["step_id"] == "s1" {
			foundMarkedFailed = true
		}
	}
	assert.True(t, foundMarkedFailed, "open_npc_shop step must be driven to Failed by npcshop ENTER_ERROR: %+v", hook.AllEntries())
}
