package npcconversation

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

	npc "atlas-saga-orchestrator/kafka/message/npc"
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

// TestHandleStartedEvent_IgnoresWrongType guards the shared-topic fan-out:
// every handler registered on EVENT_TOPIC_NPC_CONVERSATION_STATUS sees every
// event, so the type check is what stops a START_ERROR from completing a
// conversation-start step.
func TestHandleStartedEvent_IgnoresWrongType(t *testing.T) {
	l, hook := test.NewNullLogger()
	ctx := newTenantContext(t)

	handleStartedEvent(l, ctx, npc.ConversationStatusEvent[npc.StatusEventStartedBody]{
		TransactionId: uuid.New(),
		CharacterId:   1234,
		Type:          npc.StatusEventTypeStartError,
	})

	if len(hook.Entries) != 0 {
		t.Errorf("handler acted on a non-STARTED event: %v", hook.Entries)
	}
}

// TestHandleStartedEvent_NilTransactionIgnored: the ordinary NPC-talk path
// produces commands with uuid.Nil, and any status event carrying that value
// must never advance a saga.
func TestHandleStartedEvent_NilTransactionIgnored(t *testing.T) {
	l, hook := test.NewNullLogger()
	ctx := newTenantContext(t)

	handleStartedEvent(l, ctx, npc.ConversationStatusEvent[npc.StatusEventStartedBody]{
		TransactionId: uuid.Nil,
		CharacterId:   1234,
		Type:          npc.StatusEventTypeStarted,
	})

	if len(hook.Entries) != 0 {
		t.Errorf("handler acted on a uuid.Nil transaction: %v", hook.Entries)
	}
}

// TestHandleStartErrorEvent_NilTransactionIgnored
func TestHandleStartErrorEvent_NilTransactionIgnored(t *testing.T) {
	l, hook := test.NewNullLogger()
	ctx := newTenantContext(t)

	handleStartErrorEvent(l, ctx, npc.ConversationStatusEvent[npc.StatusEventStartErrorBody]{
		TransactionId: uuid.Nil,
		CharacterId:   1234,
		Type:          npc.StatusEventTypeStartError,
		Body:          npc.StatusEventStartErrorBody{Reason: npc.StartErrorNoConversationAuthored},
	})

	if len(hook.Entries) != 0 {
		t.Errorf("handler acted on a uuid.Nil transaction: %v", hook.Entries)
	}
}

// TestHandleStartedEvent_CompletesItemConversationStep verifies the
// STARTED→success polarity for a start_item_conversation step.
func TestHandleStartedEvent_CompletesItemConversationStep(t *testing.T) {
	l, _ := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)
	ctx := newTenantContext(t)

	tx := uuid.New()
	s, err := saga.NewBuilder().
		SetTransactionId(tx).
		SetSagaType(saga.ScriptedItemUse).
		SetInitiatedBy("test").
		AddStep("s1", saga.Pending, saga.StartItemConversation, saga.StartItemConversationPayload{
			CharacterId:   1234,
			ItemId:        2430008,
			NpcTemplateId: 2084002,
		}).
		AddStep("s2", saga.Pending, saga.DestroyAssetFromSlot, saga.DestroyAssetFromSlotPayload{CharacterId: 1234, Slot: 5}).
		Build()
	require.NoError(t, err)
	putTestSaga(t, ctx, s)

	handleStartedEvent(l, ctx, npc.ConversationStatusEvent[npc.StatusEventStartedBody]{
		TransactionId: tx,
		CharacterId:   1234,
		Type:          npc.StatusEventTypeStarted,
		Body:          npc.StatusEventStartedBody{NpcTemplateId: 2084002, SourceId: 2430008},
	})

	got, err := saga.NewProcessor(l, ctx).GetById(tx)
	require.NoError(t, err)
	assert.Equal(t, saga.Completed, got.Steps()[0].Status(), "start_item_conversation step must be completed by STARTED")
}

// TestHandleStartedEvent_CompletesNpcConversationStep verifies the same
// polarity for a start_npc_conversation step.
func TestHandleStartedEvent_CompletesNpcConversationStep(t *testing.T) {
	l, _ := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)
	ctx := newTenantContext(t)

	tx := uuid.New()
	s, err := saga.NewBuilder().
		SetTransactionId(tx).
		SetSagaType(saga.RemoteNpcUse).
		SetInitiatedBy("test").
		AddStep("s1", saga.Pending, saga.StartNpcConversation, saga.StartNpcConversationPayload{
			CharacterId:   4321,
			NpcTemplateId: 9090002,
		}).
		AddStep("s2", saga.Pending, saga.AwardAsset, saga.AwardItemActionPayload{CharacterId: 4321, Item: saga.ItemPayload{TemplateId: 2000000, Quantity: 1}}).
		Build()
	require.NoError(t, err)
	putTestSaga(t, ctx, s)

	handleStartedEvent(l, ctx, npc.ConversationStatusEvent[npc.StatusEventStartedBody]{
		TransactionId: tx,
		CharacterId:   4321,
		Type:          npc.StatusEventTypeStarted,
		Body:          npc.StatusEventStartedBody{NpcTemplateId: 9090002, SourceId: 9090002},
	})

	got, err := saga.NewProcessor(l, ctx).GetById(tx)
	require.NoError(t, err)
	assert.Equal(t, saga.Completed, got.Steps()[0].Status(), "start_npc_conversation step must be completed by STARTED")
}

// TestHandleStartErrorEvent_FailsConversationStep verifies the
// START_ERROR→failure polarity: a START_ERROR status event drives
// StepCompleted(..., false) for a pending conversation-start step, so the
// item is never consumed by a following destroy step.
func TestHandleStartErrorEvent_FailsConversationStep(t *testing.T) {
	l, hook := test.NewNullLogger()
	l.SetLevel(logrus.DebugLevel)
	ctx := newTenantContext(t)

	tx := uuid.New()
	s, err := saga.NewBuilder().
		SetTransactionId(tx).
		SetSagaType(saga.ScriptedItemUse).
		SetInitiatedBy("test").
		AddStep("s1", saga.Pending, saga.StartItemConversation, saga.StartItemConversationPayload{
			CharacterId:   1234,
			ItemId:        2430008,
			NpcTemplateId: 2084002,
		}).
		Build()
	require.NoError(t, err)
	putTestSaga(t, ctx, s)

	handleStartErrorEvent(l, ctx, npc.ConversationStatusEvent[npc.StatusEventStartErrorBody]{
		TransactionId: tx,
		CharacterId:   1234,
		Type:          npc.StatusEventTypeStartError,
		Body:          npc.StatusEventStartErrorBody{NpcTemplateId: 2084002, SourceId: 2430008, Reason: npc.StartErrorNoConversationAuthored},
	})

	foundMarkedFailed := false
	for _, e := range hook.AllEntries() {
		if e.Message == "Marked earliest pending step as [failed]." && e.Data["step_id"] == "s1" {
			foundMarkedFailed = true
		}
	}
	assert.True(t, foundMarkedFailed, "start_item_conversation step must be driven to Failed by START_ERROR: %+v", hook.AllEntries())
}
