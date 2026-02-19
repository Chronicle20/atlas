package npc

import (
	"atlas-npc-conversations/conversation"
	"atlas-npc-conversations/test"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestModel(t *testing.T, npcId uint32) Model {
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
		SetText("Hello, adventurer!").
		AddChoice(okChoice).
		AddChoice(escapeChoice).
		Build()
	require.NoError(t, err)

	state, err := conversation.NewStateBuilder().
		SetId("start").
		SetDialogue(dialogue).
		Build()
	require.NoError(t, err)

	model, err := NewBuilder().
		SetNpcId(npcId).
		SetStartState("start").
		AddState(state).
		Build()
	require.NoError(t, err)

	return model
}

func TestAdministratorFunctionCurrying(t *testing.T) {
	db := test.SetupTestDB(t, MigrateTable)
	defer test.CleanupTestDB(t, db)

	tenantId := uuid.New()
	id := uuid.New()

	t.Run("createNpcConversation currying", func(t *testing.T) {
		createFunc := createNpcConversation(db)
		assert.NotNil(t, createFunc)

		createForTenant := createFunc(tenantId)
		assert.NotNil(t, createForTenant)
	})

	t.Run("updateNpcConversation currying", func(t *testing.T) {
		updateFunc := updateNpcConversation(db)
		assert.NotNil(t, updateFunc)

		updateForId := updateFunc(id)
		assert.NotNil(t, updateForId)
	})

	t.Run("deleteNpcConversation currying", func(t *testing.T) {
		deleteFunc := deleteNpcConversation(db)
		assert.NotNil(t, deleteFunc)
	})
}

func TestAdministratorFunctionSignatures(t *testing.T) {
	db := test.SetupTestDB(t, MigrateTable)
	defer test.CleanupTestDB(t, db)

	tenantId := uuid.New()
	model := createTestModel(t, 1001)

	t.Run("createNpcConversation returns correct types", func(t *testing.T) {
		createFunc := createNpcConversation(db)
		createForTenant := createFunc(tenantId)

		// The function should be callable with a Model and succeed with SQLite
		createdModel, err := createForTenant(model)
		assert.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, createdModel.Id())
		assert.Equal(t, model.NpcId(), createdModel.NpcId())
	})

	t.Run("updateNpcConversation returns correct types", func(t *testing.T) {
		// First create a conversation to update
		createFunc := createNpcConversation(db)
		createdModel, err := createFunc(tenantId)(model)
		require.NoError(t, err)

		updateFunc := updateNpcConversation(db)
		updateForId := updateFunc(createdModel.Id())

		// Update with the same model data
		updatedModel, err := updateForId(model)
		assert.NoError(t, err)
		assert.Equal(t, createdModel.Id(), updatedModel.Id())
	})

	t.Run("deleteNpcConversation returns correct types", func(t *testing.T) {
		// First create a conversation to delete
		createFunc := createNpcConversation(db)
		createdModel, err := createFunc(tenantId)(model)
		require.NoError(t, err)

		deleteFunc := deleteNpcConversation(db)

		err = deleteFunc(createdModel.Id())
		assert.NoError(t, err)
	})

	t.Run("deleteAllNpcConversations returns correct types", func(t *testing.T) {
		count, err := deleteAllNpcConversations(db)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(0))
	})
}

func TestModelBuilder(t *testing.T) {
	t.Run("valid model creation", func(t *testing.T) {
		model := createTestModel(t, 1001)
		assert.Equal(t, uint32(1001), model.NpcId())
		assert.Equal(t, "start", model.StartState())
		assert.Len(t, model.States(), 1)
	})

	t.Run("model requires npcId", func(t *testing.T) {
		dialogue := &conversation.DialogueModel{}
		state, err := conversation.NewStateBuilder().
			SetId("start").
			SetDialogue(dialogue).
			Build()
		require.NoError(t, err)

		_, err = NewBuilder().
			SetStartState("start").
			AddState(state).
			Build()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "npcId")
	})

	t.Run("model requires startState", func(t *testing.T) {
		dialogue := &conversation.DialogueModel{}
		state, err := conversation.NewStateBuilder().
			SetId("start").
			SetDialogue(dialogue).
			Build()
		require.NoError(t, err)

		_, err = NewBuilder().
			SetNpcId(1001).
			AddState(state).
			Build()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "startState")
	})

	t.Run("model requires at least one state", func(t *testing.T) {
		_, err := NewBuilder().
			SetNpcId(1001).
			SetStartState("start").
			Build()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "state")
	})
}
