package npc

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
	npcId := uint32(1001)

	t.Run("getByIdProvider currying", func(t *testing.T) {
		providerByTenant := getByIdProvider(tenantId)
		assert.NotNil(t, providerByTenant)

		providerById := providerByTenant(id)
		assert.NotNil(t, providerById)

		provider := providerById(gormDB)
		assert.NotNil(t, provider)
	})

	t.Run("getByNpcIdProvider currying", func(t *testing.T) {
		providerByTenant := getByNpcIdProvider(tenantId)
		assert.NotNil(t, providerByTenant)

		providerByNpcId := providerByTenant(npcId)
		assert.NotNil(t, providerByNpcId)

		provider := providerByNpcId(gormDB)
		assert.NotNil(t, provider)
	})

	t.Run("getAllProvider currying", func(t *testing.T) {
		providerByTenant := getAllProvider(tenantId)
		assert.NotNil(t, providerByTenant)

		provider := providerByTenant(gormDB)
		assert.NotNil(t, provider)
	})

	t.Run("getAllByNpcIdProvider currying", func(t *testing.T) {
		providerByTenant := getAllByNpcIdProvider(tenantId)
		assert.NotNil(t, providerByTenant)

		providerByNpcId := providerByTenant(npcId)
		assert.NotNil(t, providerByNpcId)

		provider := providerByNpcId(gormDB)
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
	npcId := uint32(1001)

	t.Run("getByIdProvider returns entity provider", func(t *testing.T) {
		provider := getByIdProvider(tenantId)(id)(gormDB)

		// Provider should be callable and return Entity and error
		entity, err := provider()
		assert.Error(t, err) // Expected - no mock expectations
		assert.Equal(t, Entity{}, entity)
	})

	t.Run("getByNpcIdProvider returns entity provider", func(t *testing.T) {
		provider := getByNpcIdProvider(tenantId)(npcId)(gormDB)

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

	t.Run("getAllByNpcIdProvider returns slice provider", func(t *testing.T) {
		provider := getAllByNpcIdProvider(tenantId)(npcId)(gormDB)

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
		assert.Equal(t, uint32(1001), entity.NpcID)
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

		assert.Equal(t, uint32(1001), convertedModel.NpcId())
		assert.Equal(t, "start", convertedModel.StartState())
	})

	t.Run("round trip preserves data", func(t *testing.T) {
		originalModel := createTestModel(t, 2002)
		tenantId := uuid.New()

		entity, err := ToEntity(originalModel, tenantId)
		require.NoError(t, err)
		entity.ID = uuid.New()

		roundTripModel, err := Make(entity)
		require.NoError(t, err)

		assert.Equal(t, originalModel.NpcId(), roundTripModel.NpcId())
		assert.Equal(t, originalModel.StartState(), roundTripModel.StartState())
		assert.Len(t, roundTripModel.States(), len(originalModel.States()))
	})
}
