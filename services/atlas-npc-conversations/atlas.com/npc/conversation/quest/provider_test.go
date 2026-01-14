package quest

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestProviderFunctionCurrying(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: mockDB,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	tenantId := uuid.New()
	id := uuid.New()
	questId := uint32(1001)

	t.Run("getByIdProvider currying", func(t *testing.T) {
		providerByTenant := getByIdProvider(tenantId)
		assert.NotNil(t, providerByTenant)

		providerById := providerByTenant(id)
		assert.NotNil(t, providerById)

		provider := providerById(gormDB)
		assert.NotNil(t, provider)
	})

	t.Run("getByQuestIdProvider currying", func(t *testing.T) {
		providerByTenant := getByQuestIdProvider(tenantId)
		assert.NotNil(t, providerByTenant)

		providerByQuestId := providerByTenant(questId)
		assert.NotNil(t, providerByQuestId)

		provider := providerByQuestId(gormDB)
		assert.NotNil(t, provider)
	})

	t.Run("getAllProvider currying", func(t *testing.T) {
		providerByTenant := getAllProvider(tenantId)
		assert.NotNil(t, providerByTenant)

		provider := providerByTenant(gormDB)
		assert.NotNil(t, provider)
	})
}

func TestProviderFunctionSignatures(t *testing.T) {
	mockDB, _, err := sqlmock.New()
	require.NoError(t, err)
	defer mockDB.Close()

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: mockDB,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	tenantId := uuid.New()
	id := uuid.New()
	questId := uint32(1001)

	t.Run("getByIdProvider returns entity provider", func(t *testing.T) {
		provider := getByIdProvider(tenantId)(id)(gormDB)

		// Provider should be callable and return Entity and error
		entity, err := provider()
		assert.Error(t, err) // Expected - no mock expectations
		assert.Equal(t, Entity{}, entity)
	})

	t.Run("getByQuestIdProvider returns entity provider", func(t *testing.T) {
		provider := getByQuestIdProvider(tenantId)(questId)(gormDB)

		entity, err := provider()
		assert.Error(t, err) // Expected - no mock expectations
		assert.Equal(t, Entity{}, entity)
	})

	t.Run("getAllProvider returns slice provider", func(t *testing.T) {
		provider := getAllProvider(tenantId)(gormDB)

		entities, err := provider()
		assert.Error(t, err) // Expected - no mock expectations
		assert.Nil(t, entities)
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
