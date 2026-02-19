package quest

import (
	"atlas-npc-conversations/test"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderFunctionCurrying(t *testing.T) {
	db := test.SetupTestDB(t, MigrateTable)
	defer test.CleanupTestDB(t, db)

	id := uuid.New()
	questId := uint32(1001)

	t.Run("getByIdProvider currying", func(t *testing.T) {
		providerById := getByIdProvider(id)
		assert.NotNil(t, providerById)

		provider := providerById(db)
		assert.NotNil(t, provider)
	})

	t.Run("getByQuestIdProvider currying", func(t *testing.T) {
		providerByQuestId := getByQuestIdProvider(questId)
		assert.NotNil(t, providerByQuestId)

		provider := providerByQuestId(db)
		assert.NotNil(t, provider)
	})

	t.Run("getAllProvider currying", func(t *testing.T) {
		provider := getAllProvider(db)
		assert.NotNil(t, provider)
	})
}

func TestProviderFunctionSignatures(t *testing.T) {
	db := test.SetupTestDB(t, MigrateTable)
	defer test.CleanupTestDB(t, db)

	id := uuid.New()
	questId := uint32(1001)

	t.Run("getByIdProvider returns entity provider", func(t *testing.T) {
		provider := getByIdProvider(id)(db)

		// Provider should be callable and return Entity and error
		entity, err := provider()
		assert.Error(t, err) // Expected - record not found
		assert.Equal(t, Entity{}, entity)
	})

	t.Run("getByQuestIdProvider returns entity provider", func(t *testing.T) {
		provider := getByQuestIdProvider(questId)(db)

		entity, err := provider()
		assert.Error(t, err) // Expected - record not found
		assert.Equal(t, Entity{}, entity)
	})

	t.Run("getAllProvider returns slice provider", func(t *testing.T) {
		provider := getAllProvider(db)

		entities, err := provider()
		assert.NoError(t, err) // Empty result is not an error
		assert.Empty(t, entities)
	})
}

func TestEntityMakeAndToEntity(t *testing.T) {
	t.Run("ToEntity creates valid entity from model", func(t *testing.T) {
		model := createTestModel(t, 1001)
		tenantId := uuid.New()

		entity, err := ToEntity(model, tenantId)
		require.NoError(t, err)

		assert.Equal(t, tenantId, entity.TenantID)
		assert.Equal(t, uint32(1001), entity.QuestID)
		assert.NotEmpty(t, entity.Data)
	})

	t.Run("Make converts entity to model", func(t *testing.T) {
		model := createTestModel(t, 1001)
		tenantId := uuid.New()

		entity, err := ToEntity(model, tenantId)
		require.NoError(t, err)

		// Set a non-nil ID for the entity
		entity.ID = uuid.New()

		convertedModel, err := Make(entity)
		require.NoError(t, err)

		assert.Equal(t, uint32(1001), convertedModel.QuestId())
		assert.Equal(t, "start", convertedModel.StartStateMachine().StartState())
	})

	t.Run("round trip preserves data", func(t *testing.T) {
		originalModel := createTestModel(t, 2002)
		tenantId := uuid.New()

		entity, err := ToEntity(originalModel, tenantId)
		require.NoError(t, err)
		entity.ID = uuid.New()

		roundTripModel, err := Make(entity)
		require.NoError(t, err)

		assert.Equal(t, originalModel.QuestId(), roundTripModel.QuestId())
		assert.Equal(t, originalModel.StartStateMachine().StartState(), roundTripModel.StartStateMachine().StartState())
		assert.Len(t, roundTripModel.StartStateMachine().States(), len(originalModel.StartStateMachine().States()))
	})
}
