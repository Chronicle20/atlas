package character

import (
	"atlas-buffs/buff/stat"
	"context"
	"testing"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func setupProcessorTest(t *testing.T) (Processor, tenant.Model) {
	t.Helper()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}

	ctx := tenant.WithContext(context.Background(), ten)
	processor := NewProcessor(logger, ctx)

	// Reset registry for clean state
	GetRegistry().ResetForTesting()

	return processor, ten
}

func setupProcessorTestChanges() []stat.Model {
	return []stat.Model{
		stat.NewStat("STR", 10),
		stat.NewStat("DEX", 5),
	}
}

func TestProcessor_GetById_NotFound(t *testing.T) {
	processor, _ := setupProcessorTest(t)

	_, err := processor.GetById(9999)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestProcessor_GetById_AfterApply(t *testing.T) {
	processor, _ := setupProcessorTest(t)
	changes := setupProcessorTestChanges()

	worldId := world.Id(0)
	characterId := uint32(1000)
	fromId := uint32(2000)
	sourceId := int32(2001001)
	duration := int32(60)

	// Apply a buff (this will fail to emit to Kafka, but registry state should be updated)
	_ = processor.Apply(worldId, characterId, fromId, sourceId, byte(5), duration, changes)

	// Get the character
	m, err := processor.GetById(characterId)
	assert.NoError(t, err)
	assert.Equal(t, characterId, m.Id())
	assert.Equal(t, worldId, m.WorldId())
	assert.Len(t, m.Buffs(), 1)
}

func TestProcessor_Apply(t *testing.T) {
	processor, ten := setupProcessorTest(t)
	changes := setupProcessorTestChanges()

	worldId := world.Id(0)
	characterId := uint32(1000)
	fromId := uint32(2000)
	sourceId := int32(2001001)
	duration := int32(60)

	// Apply will return an error due to Kafka being unavailable,
	// but registry state should still be updated (message buffer pattern)
	_ = processor.Apply(worldId, characterId, fromId, sourceId, byte(5), duration, changes)

	// Verify buff was added to registry despite Kafka error
	m, err := GetRegistry().Get(ten, characterId)
	assert.NoError(t, err)
	assert.Len(t, m.Buffs(), 1)

	buff := m.Buffs()[sourceId]
	assert.Equal(t, sourceId, buff.SourceId())
	assert.Equal(t, duration, buff.Duration())
}

func TestProcessor_Apply_MultipleBuffs(t *testing.T) {
	processor, ten := setupProcessorTest(t)
	changes := setupProcessorTestChanges()

	worldId := world.Id(0)
	characterId := uint32(1000)
	fromId := uint32(2000)

	// Apply multiple buffs
	_ = processor.Apply(worldId, characterId, fromId, int32(2001001), byte(5), int32(60), changes)
	_ = processor.Apply(worldId, characterId, fromId, int32(2001002), byte(5), int32(120), changes)
	_ = processor.Apply(worldId, characterId, fromId, int32(2001003), byte(5), int32(180), changes)

	m, err := GetRegistry().Get(ten, characterId)
	assert.NoError(t, err)
	assert.Len(t, m.Buffs(), 3)
}

func TestProcessor_Cancel(t *testing.T) {
	processor, ten := setupProcessorTest(t)
	changes := setupProcessorTestChanges()

	worldId := world.Id(0)
	characterId := uint32(1000)
	fromId := uint32(2000)
	sourceId := int32(2001001)
	duration := int32(60)

	// Apply a buff first (ignore Kafka error)
	_ = processor.Apply(worldId, characterId, fromId, sourceId, byte(5), duration, changes)

	// Verify buff exists
	m, _ := GetRegistry().Get(ten, characterId)
	assert.Len(t, m.Buffs(), 1)

	// Cancel the buff (will return Kafka error but registry state is updated)
	_ = processor.Cancel(worldId, characterId, sourceId)

	// Verify buff was removed from registry despite Kafka error
	m, _ = GetRegistry().Get(ten, characterId)
	assert.Len(t, m.Buffs(), 0)
}

func TestProcessor_Cancel_NotFound(t *testing.T) {
	processor, _ := setupProcessorTest(t)

	// Cancel non-existent buff should not error (silent failure)
	err := processor.Cancel(world.Id(0), uint32(9999), int32(12345))
	assert.NoError(t, err)
}

func TestProcessor_Cancel_WrongSourceId(t *testing.T) {
	processor, ten := setupProcessorTest(t)
	changes := setupProcessorTestChanges()

	worldId := world.Id(0)
	characterId := uint32(1000)
	fromId := uint32(2000)
	sourceId := int32(2001001)
	duration := int32(60)

	// Apply a buff
	_ = processor.Apply(worldId, characterId, fromId, sourceId, byte(5), duration, changes)

	// Cancel with wrong sourceId
	err := processor.Cancel(worldId, characterId, int32(9999))
	assert.NoError(t, err) // Should not error, just does nothing

	// Verify original buff still exists
	m, _ := GetRegistry().Get(ten, characterId)
	assert.Len(t, m.Buffs(), 1)
}

func TestProcessor_ExpireBuffs_NoBuffs(t *testing.T) {
	processor, _ := setupProcessorTest(t)

	err := processor.ExpireBuffs()
	assert.NoError(t, err)
}

func TestProcessor_TenantContext(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	ten1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ten2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)

	ctx1 := tenant.WithContext(context.Background(), ten1)
	ctx2 := tenant.WithContext(context.Background(), ten2)

	processor1 := NewProcessor(logger, ctx1)
	processor2 := NewProcessor(logger, ctx2)

	GetRegistry().ResetForTesting()
	changes := setupProcessorTestChanges()

	// Apply buff in tenant1
	_ = processor1.Apply(world.Id(0), uint32(1000), uint32(2000), int32(2001001), byte(5), int32(60), changes)

	// Get from processor1 should work
	m, err := processor1.GetById(uint32(1000))
	assert.NoError(t, err)
	assert.Len(t, m.Buffs(), 1)

	// Get from processor2 should not find it (different tenant)
	_, err = processor2.GetById(uint32(1000))
	assert.ErrorIs(t, err, ErrNotFound)
}
