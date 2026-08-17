//go:build test

package saga

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"

	compmock "atlas-saga-orchestrator/compartment/mock"
)

// Conversation-first ordering means the only compensable path is "dialogue
// opened, destroy failed". Compensation is a UI teardown (END_CONVERSATION),
// not an item restore.
//
// TestScriptedItemUseCompensationEndsTheConversation verifies the
// scripted_item_use reverse-walk (task-230): when consume_scripted_item fails,
// the already-completed start_item_conversation is inverted with an
// END_CONVERSATION command so the player is not left standing in a dialogue
// for an item they still hold.
func TestScriptedItemUseCompensationEndsTheConversation(t *testing.T) {
	logger, _ := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	var gotCharacter, gotNpc uint32
	var called int
	origEnd := SetEmitNpcConversationEndForTest(func(_ logrus.FieldLogger, _ context.Context, _ uuid.UUID, characterId uint32, npcTemplateId uint32) error {
		called++
		gotCharacter, gotNpc = characterId, npcTemplateId
		return nil
	})
	t.Cleanup(func() { SetEmitNpcConversationEndForTest(origEnd) })

	compP := &compmock.ProcessorMock{
		RequestCreateItemFunc: func(_ uuid.UUID, _ uint32, _ uint32, _ uint32, _ time.Time) error {
			return nil
		},
	}

	s, err := NewBuilder().
		SetTransactionId(uuid.New()).
		SetSagaType(ScriptedItemUse).
		SetInitiatedBy("scripted-item-use-compensation-test").
		AddStep("start_item_conversation", Completed, StartItemConversation, StartItemConversationPayload{
			CharacterId:   1234,
			ItemId:        2430008,
			NpcTemplateId: 2084002,
		}).
		AddStep("consume_scripted_item", Failed, DestroyAssetFromSlot, DestroyAssetFromSlotPayload{
			CharacterId: 1234,
			Slot:        5,
		}).
		Build()
	assert.NoError(t, err, "saga build should not fail")

	compensator := NewCompensator(logger, testTenantContext()).
		WithCompartmentProcessor(compP)

	compensator.DispatchCashItemUseRollbacks(s)

	assert.Equal(t, 1, called, "END_CONVERSATION emitted %d times, want 1")
	assert.Equal(t, uint32(1234), gotCharacter, "emitted for character %d", gotCharacter)
	assert.Equal(t, uint32(2084002), gotNpc, "emitted for npc %d", gotNpc)
}

// TestScriptedItemUseCompensationSkipsUncompletedConversation verifies that a
// conversation start that never completed is NOT ended — the reverse-walk
// only inverts Completed mutations.
func TestScriptedItemUseCompensationSkipsUncompletedConversation(t *testing.T) {
	logger, _ := test.NewNullLogger()

	var called int
	origEnd := SetEmitNpcConversationEndForTest(func(_ logrus.FieldLogger, _ context.Context, _ uuid.UUID, characterId uint32, npcTemplateId uint32) error {
		called++
		return nil
	})
	t.Cleanup(func() { SetEmitNpcConversationEndForTest(origEnd) })

	s, err := NewBuilder().
		SetTransactionId(uuid.New()).
		SetSagaType(ScriptedItemUse).
		SetInitiatedBy("scripted-item-use-compensation-test").
		AddStep("start_item_conversation", Failed, StartItemConversation, StartItemConversationPayload{
			CharacterId:   88222,
			ItemId:        2430008,
			NpcTemplateId: 2084002,
		}).
		Build()
	assert.NoError(t, err)

	NewCompensator(logger, testTenantContext()).DispatchCashItemUseRollbacks(s)

	assert.Equal(t, 0, called, "a dialogue that never opened must not be ended")
}

// TestRemoteNpcUseCompensationEndsTheConversation verifies the
// remote_npc_use reverse-walk (task-230): the StartNpcConversation twin of
// the scripted-item case above.
func TestRemoteNpcUseCompensationEndsTheConversation(t *testing.T) {
	logger, _ := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	var gotCharacter, gotNpc uint32
	var called int
	origEnd := SetEmitNpcConversationEndForTest(func(_ logrus.FieldLogger, _ context.Context, _ uuid.UUID, characterId uint32, npcTemplateId uint32) error {
		called++
		gotCharacter, gotNpc = characterId, npcTemplateId
		return nil
	})
	t.Cleanup(func() { SetEmitNpcConversationEndForTest(origEnd) })

	compP := &compmock.ProcessorMock{
		RequestCreateItemFunc: func(_ uuid.UUID, _ uint32, _ uint32, _ uint32, _ time.Time) error {
			return nil
		},
	}

	s, err := NewBuilder().
		SetTransactionId(uuid.New()).
		SetSagaType(RemoteNpcUse).
		SetInitiatedBy("remote-npc-use-compensation-test").
		AddStep("start_npc_conversation", Completed, StartNpcConversation, StartNpcConversationPayload{
			CharacterId:   4321,
			NpcTemplateId: 9090002,
		}).
		AddStep("consume_remote_npc_item", Failed, DestroyAssetFromSlot, DestroyAssetFromSlotPayload{
			CharacterId: 4321,
			Slot:        3,
		}).
		Build()
	assert.NoError(t, err, "saga build should not fail")

	compensator := NewCompensator(logger, testTenantContext()).
		WithCompartmentProcessor(compP)

	compensator.DispatchCashItemUseRollbacks(s)

	assert.Equal(t, 1, called, "END_CONVERSATION emitted %d times, want 1")
	assert.Equal(t, uint32(4321), gotCharacter, "emitted for character %d", gotCharacter)
	assert.Equal(t, uint32(9090002), gotNpc, "emitted for npc %d", gotNpc)
}
