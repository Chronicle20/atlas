package npc

import (
	"atlas-npc-conversations/conversation"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, func()) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: mockDB,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	cleanup := func() {
		mockDB.Close()
	}

	return gormDB, mock, cleanup
}

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
	gormDB, _, cleanup := setupMockDB(t)
	defer cleanup()

	tenantId := uuid.New()
	id := uuid.New()

	t.Run("createNpcConversation currying", func(t *testing.T) {
		createFunc := createNpcConversation(gormDB)
		assert.NotNil(t, createFunc)

		createForTenant := createFunc(tenantId)
		assert.NotNil(t, createForTenant)
	})

	t.Run("updateNpcConversation currying", func(t *testing.T) {
		updateFunc := updateNpcConversation(gormDB)
		assert.NotNil(t, updateFunc)

		updateForTenant := updateFunc(tenantId)
		assert.NotNil(t, updateForTenant)

		updateForId := updateForTenant(id)
		assert.NotNil(t, updateForId)
	})

	t.Run("deleteNpcConversation currying", func(t *testing.T) {
		deleteFunc := deleteNpcConversation(gormDB)
		assert.NotNil(t, deleteFunc)

		deleteForTenant := deleteFunc(tenantId)
		assert.NotNil(t, deleteForTenant)
	})

	t.Run("deleteAllNpcConversations currying", func(t *testing.T) {
		deleteAllFunc := deleteAllNpcConversations(gormDB)
		assert.NotNil(t, deleteAllFunc)
	})
}

func TestAdministratorFunctionSignatures(t *testing.T) {
	gormDB, _, cleanup := setupMockDB(t)
	defer cleanup()

	tenantId := uuid.New()
	id := uuid.New()
	model := createTestModel(t, 1001)

	t.Run("createNpcConversation returns correct types", func(t *testing.T) {
		// Verify the function signature by checking we can call it
		createFunc := createNpcConversation(gormDB)
		createForTenant := createFunc(tenantId)

		// The function should be callable with a Model
		// We expect an error due to no mock expectations, but that's OK
		_, err := createForTenant(model)
		assert.Error(t, err) // Expected - no mock expectations set up
	})

	t.Run("updateNpcConversation returns correct types", func(t *testing.T) {
		updateFunc := updateNpcConversation(gormDB)
		updateForTenant := updateFunc(tenantId)
		updateForId := updateForTenant(id)

		_, err := updateForId(model)
		assert.Error(t, err) // Expected - no mock expectations set up
	})

	t.Run("deleteNpcConversation returns correct types", func(t *testing.T) {
		deleteFunc := deleteNpcConversation(gormDB)
		deleteForTenant := deleteFunc(tenantId)

		err := deleteForTenant(id)
		assert.Error(t, err) // Expected - no mock expectations set up
	})

	t.Run("deleteAllNpcConversations returns correct types", func(t *testing.T) {
		deleteAllFunc := deleteAllNpcConversations(gormDB)

		count, err := deleteAllFunc(tenantId)
		assert.Error(t, err) // Expected - no mock expectations set up
		assert.Equal(t, int64(0), count)
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
