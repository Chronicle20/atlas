package npc

import (
	"atlas-npc-conversations/test"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderFunctionCurrying(t *testing.T) {
	db := test.SetupTestDB(t, MigrateTable)
	defer test.CleanupTestDB(t, db)

	id := uuid.New()
	npcId := uint32(1001)

	t.Run("getByIdProvider currying", func(t *testing.T) {
		providerById := getByIdProvider(id)
		assert.NotNil(t, providerById)

		provider := providerById(db)
		assert.NotNil(t, provider)
	})

	t.Run("getByNpcIdProvider currying", func(t *testing.T) {
		providerByNpcId := getByNpcIdProvider(npcId)
		assert.NotNil(t, providerByNpcId)

		provider := providerByNpcId(db)
		assert.NotNil(t, provider)
	})

	t.Run("getAllPagedProvider currying", func(t *testing.T) {
		providerFactory := getAllPagedProvider(model.Page{Number: 1, Size: 50})
		assert.NotNil(t, providerFactory)

		provider := providerFactory(db)
		assert.NotNil(t, provider)
	})

	t.Run("getAllByNpcIdPagedProvider currying", func(t *testing.T) {
		providerFactory := getAllByNpcIdPagedProvider(npcId, model.Page{Number: 1, Size: 50})
		assert.NotNil(t, providerFactory)

		provider := providerFactory(db)
		assert.NotNil(t, provider)
	})
}

func TestProviderFunctionSignatures(t *testing.T) {
	db := test.SetupTestDB(t, MigrateTable)
	defer test.CleanupTestDB(t, db)

	id := uuid.New()
	npcId := uint32(1001)

	t.Run("getByIdProvider returns entity provider", func(t *testing.T) {
		provider := getByIdProvider(id)(db)

		// Provider should be callable and return Entity and error
		entity, err := provider()
		assert.Error(t, err) // Expected - record not found
		assert.Equal(t, Entity{}, entity)
	})

	t.Run("getByNpcIdProvider returns entity provider", func(t *testing.T) {
		provider := getByNpcIdProvider(npcId)(db)

		entity, err := provider()
		assert.Error(t, err) // Expected - record not found
		assert.Equal(t, Entity{}, entity)
	})

	t.Run("getAllPagedProvider returns paged provider", func(t *testing.T) {
		provider := getAllPagedProvider(model.Page{Number: 1, Size: 50})(db)

		paged, err := provider()
		assert.NoError(t, err) // Empty result is not an error
		assert.Empty(t, paged.Items)
	})

	t.Run("getAllByNpcIdPagedProvider returns paged provider", func(t *testing.T) {
		provider := getAllByNpcIdPagedProvider(npcId, model.Page{Number: 1, Size: 50})(db)

		paged, err := provider()
		assert.NoError(t, err) // Empty result is not an error
		assert.Empty(t, paged.Items)
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
