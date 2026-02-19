package quest

import (
	"atlas-npc-conversations/conversation"
	"atlas-npc-conversations/test"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestModel(t *testing.T, questId uint32) Model {
	// Build choices for SendOk dialogue (requires exactly 2 choices)
	okChoice, err := conversation.NewChoiceBuilder().
		SetText("OK").
		SetNextState("end").
		Build()
	require.NoError(t, err)

	escapeChoice, err := conversation.NewChoiceBuilder().
		SetText("End Conversation").
		SetNextState("end").
		Build()
	require.NoError(t, err)

	// Build dialogue with required fields
	dialogue, err := conversation.NewDialogueBuilder().
		SetDialogueType(conversation.SendOk).
		SetText("Quest dialogue text").
		AddChoice(okChoice).
		AddChoice(escapeChoice).
		Build()
	require.NoError(t, err)

	state, err := conversation.NewStateBuilder().
		SetId("start").
		SetDialogue(dialogue).
		Build()
	require.NoError(t, err)

	stateMachine, err := NewStateMachineBuilder().
		SetStartState("start").
		AddState(state).
		Build()
	require.NoError(t, err)

	model, err := NewBuilder().
		SetQuestId(questId).
		SetStartStateMachine(stateMachine).
		Build()
	require.NoError(t, err)

	return model
}

func TestAdministratorFunctionCurrying(t *testing.T) {
	db := test.SetupTestDB(t, MigrateTable)
	defer test.CleanupTestDB(t, db)

	tenantId := uuid.New()
	id := uuid.New()

	t.Run("createQuestConversation currying", func(t *testing.T) {
		createFunc := createQuestConversation(db)
		assert.NotNil(t, createFunc)

		createForTenant := createFunc(tenantId)
		assert.NotNil(t, createForTenant)
	})

	t.Run("updateQuestConversation currying", func(t *testing.T) {
		updateFunc := updateQuestConversation(db)
		assert.NotNil(t, updateFunc)

		updateForId := updateFunc(id)
		assert.NotNil(t, updateForId)
	})

	t.Run("deleteQuestConversation currying", func(t *testing.T) {
		deleteFunc := deleteQuestConversation(db)
		assert.NotNil(t, deleteFunc)
	})
}

func TestAdministratorFunctionSignatures(t *testing.T) {
	db := test.SetupTestDB(t, MigrateTable)
	defer test.CleanupTestDB(t, db)

	tenantId := uuid.New()
	model := createTestModel(t, 1001)

	t.Run("createQuestConversation returns correct types", func(t *testing.T) {
		createFunc := createQuestConversation(db)
		createForTenant := createFunc(tenantId)

		// The function should be callable with a Model and succeed with SQLite
		createdModel, err := createForTenant(model)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, createdModel.Id())
		assert.Equal(t, model.QuestId(), createdModel.QuestId())
	})

	t.Run("updateQuestConversation returns correct types", func(t *testing.T) {
		// First create a conversation to update
		createFunc := createQuestConversation(db)
		createdModel, err := createFunc(tenantId)(model)
		require.NoError(t, err)

		updateFunc := updateQuestConversation(db)
		updateForId := updateFunc(createdModel.Id())

		// Update with the same model data
		updatedModel, err := updateForId(model)
		assert.NoError(t, err)
		assert.Equal(t, createdModel.Id(), updatedModel.Id())
	})

	t.Run("deleteQuestConversation returns correct types", func(t *testing.T) {
		// First create a conversation to delete
		createFunc := createQuestConversation(db)
		createdModel, err := createFunc(tenantId)(model)
		require.NoError(t, err)

		deleteFunc := deleteQuestConversation(db)

		err = deleteFunc(createdModel.Id())
		assert.NoError(t, err)
	})

	t.Run("deleteAllQuestConversations returns correct types", func(t *testing.T) {
		count, err := deleteAllQuestConversations(db)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(0))
	})
}

func TestModelBuilder(t *testing.T) {
	t.Run("valid model creation", func(t *testing.T) {
		model := createTestModel(t, 1001)
		assert.Equal(t, uint32(1001), model.QuestId())
		assert.Equal(t, "start", model.StartStateMachine().StartState())
		assert.Len(t, model.StartStateMachine().States(), 1)
	})

	t.Run("model requires questId", func(t *testing.T) {
		// Build choices for SendOk dialogue (requires exactly 2 choices)
		okChoice, err := conversation.NewChoiceBuilder().
			SetText("OK").
			SetNextState("end").
			Build()
		require.NoError(t, err)

		escapeChoice, err := conversation.NewChoiceBuilder().
			SetText("End Conversation").
			SetNextState("end").
			Build()
		require.NoError(t, err)

		dialogue, err := conversation.NewDialogueBuilder().
			SetDialogueType(conversation.SendOk).
			SetText("Quest dialogue text").
			AddChoice(okChoice).
			AddChoice(escapeChoice).
			Build()
		require.NoError(t, err)

		state, err := conversation.NewStateBuilder().
			SetId("start").
			SetDialogue(dialogue).
			Build()
		require.NoError(t, err)

		stateMachine, err := NewStateMachineBuilder().
			SetStartState("start").
			AddState(state).
			Build()
		require.NoError(t, err)

		_, err = NewBuilder().
			SetStartStateMachine(stateMachine).
			Build()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "questId")
	})

	t.Run("model requires startStateMachine", func(t *testing.T) {
		_, err := NewBuilder().
			SetQuestId(1001).
			Build()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "startStateMachine")
	})

	t.Run("stateMachine requires startState", func(t *testing.T) {
		// Build choices for SendOk dialogue
		okChoice, err := conversation.NewChoiceBuilder().
			SetText("OK").
			SetNextState("end").
			Build()
		require.NoError(t, err)

		escapeChoice, err := conversation.NewChoiceBuilder().
			SetText("End Conversation").
			SetNextState("end").
			Build()
		require.NoError(t, err)

		dialogue, err := conversation.NewDialogueBuilder().
			SetDialogueType(conversation.SendOk).
			SetText("Quest dialogue text").
			AddChoice(okChoice).
			AddChoice(escapeChoice).
			Build()
		require.NoError(t, err)

		state, err := conversation.NewStateBuilder().
			SetId("start").
			SetDialogue(dialogue).
			Build()
		require.NoError(t, err)

		_, err = NewStateMachineBuilder().
			AddState(state).
			Build()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "startState")
	})

	t.Run("stateMachine requires at least one state", func(t *testing.T) {
		_, err := NewStateMachineBuilder().
			SetStartState("start").
			Build()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "state")
	})
}
