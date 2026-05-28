package character

import (
	"atlas-buffs/buff/stat"
	"context"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func setupProcessorTest(t *testing.T) (Processor, tenant.Model, context.Context) {
	t.Helper()
	setupTestRegistry(t)

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("Failed to create tenant: %v", err)
	}

	ctx := tenant.WithContext(context.Background(), ten)
	processor := NewProcessor(logger, ctx)

	return processor, ten, ctx
}

func setupProcessorTestChanges() []stat.Model {
	return []stat.Model{
		stat.NewStat("STR", 10),
		stat.NewStat("DEX", 5),
	}
}

func TestProcessor_GetById_NotFound(t *testing.T) {
	processor, _, _ := setupProcessorTest(t)

	_, err := processor.GetById(9999)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestProcessor_GetById_AfterApply(t *testing.T) {
	processor, _, _ := setupProcessorTest(t)
	changes := setupProcessorTestChanges()

	worldId := world.Id(0)
	channelId := channel.Id(0)
	characterId := uint32(1000)
	fromId := uint32(2000)
	sourceId := int32(2001001)
	duration := int32(60)

	_ = processor.Apply(worldId, channelId, characterId, fromId, sourceId, byte(5), duration, changes)

	m, err := processor.GetById(characterId)
	assert.NoError(t, err)
	assert.Equal(t, characterId, m.Id())
	assert.Equal(t, worldId, m.WorldId())
	assert.Len(t, m.Buffs(), 1)
}

func TestProcessor_Apply(t *testing.T) {
	processor, _, ctx := setupProcessorTest(t)
	changes := setupProcessorTestChanges()

	worldId := world.Id(0)
	channelId := channel.Id(0)
	characterId := uint32(1000)
	fromId := uint32(2000)
	sourceId := int32(2001001)
	duration := int32(60)

	_ = processor.Apply(worldId, channelId, characterId, fromId, sourceId, byte(5), duration, changes)

	m, err := GetRegistry().Get(ctx, characterId)
	assert.NoError(t, err)
	assert.Len(t, m.Buffs(), 1)

	buff := m.Buffs()[sourceId]
	assert.Equal(t, sourceId, buff.SourceId())
	assert.Equal(t, duration, buff.Duration())
}

func TestProcessor_Apply_MultipleBuffs(t *testing.T) {
	processor, _, ctx := setupProcessorTest(t)
	changes := setupProcessorTestChanges()

	worldId := world.Id(0)
	channelId := channel.Id(0)
	characterId := uint32(1000)
	fromId := uint32(2000)

	_ = processor.Apply(worldId, channelId, characterId, fromId, int32(2001001), byte(5), int32(60), changes)
	_ = processor.Apply(worldId, channelId, characterId, fromId, int32(2001002), byte(5), int32(120), changes)
	_ = processor.Apply(worldId, channelId, characterId, fromId, int32(2001003), byte(5), int32(180), changes)

	m, err := GetRegistry().Get(ctx, characterId)
	assert.NoError(t, err)
	assert.Len(t, m.Buffs(), 3)
}

func TestProcessor_Cancel(t *testing.T) {
	processor, _, ctx := setupProcessorTest(t)
	changes := setupProcessorTestChanges()

	worldId := world.Id(0)
	channelId := channel.Id(0)
	characterId := uint32(1000)
	fromId := uint32(2000)
	sourceId := int32(2001001)
	duration := int32(60)

	_ = processor.Apply(worldId, channelId, characterId, fromId, sourceId, byte(5), duration, changes)

	m, _ := GetRegistry().Get(ctx, characterId)
	assert.Len(t, m.Buffs(), 1)

	_ = processor.Cancel(worldId, characterId, sourceId)

	m, _ = GetRegistry().Get(ctx, characterId)
	assert.Len(t, m.Buffs(), 0)
}

func TestProcessor_Cancel_NotFound(t *testing.T) {
	processor, _, _ := setupProcessorTest(t)

	err := processor.Cancel(world.Id(0), uint32(9999), int32(12345))
	assert.NoError(t, err)
}

func TestProcessor_Cancel_WrongSourceId(t *testing.T) {
	processor, _, ctx := setupProcessorTest(t)
	changes := setupProcessorTestChanges()

	worldId := world.Id(0)
	channelId := channel.Id(0)
	characterId := uint32(1000)
	fromId := uint32(2000)
	sourceId := int32(2001001)
	duration := int32(60)

	_ = processor.Apply(worldId, channelId, characterId, fromId, sourceId, byte(5), duration, changes)

	err := processor.Cancel(worldId, characterId, int32(9999))
	assert.NoError(t, err)

	m, _ := GetRegistry().Get(ctx, characterId)
	assert.Len(t, m.Buffs(), 1)
}

func TestProcessor_ExpireBuffs_NoBuffs(t *testing.T) {
	processor, _, _ := setupProcessorTest(t)

	err := processor.ExpireBuffs()
	assert.NoError(t, err)
}

func TestProcessor_CancelByStatTypes_EmptyTypes(t *testing.T) {
	processor, _, _ := setupProcessorTest(t)

	err := processor.CancelByStatTypes(world.Id(0), uint32(1000), nil)
	assert.NoError(t, err)
}

func TestProcessor_CancelByStatTypes_NoMatch(t *testing.T) {
	processor, _, ctx := setupProcessorTest(t)

	worldId := world.Id(0)
	characterId := uint32(1000)
	holy := []stat.Model{stat.NewStat("HOLY_SYMBOL", 30)}
	_ = processor.Apply(worldId, channel.Id(0), characterId, uint32(2000), int32(2311003), byte(1), int32(60), holy)

	err := processor.CancelByStatTypes(worldId, characterId, []string{"POISON"})
	assert.NoError(t, err)

	m, _ := GetRegistry().Get(ctx, characterId)
	assert.Len(t, m.Buffs(), 1)
}

func TestProcessor_CancelByStatTypes_MultiMatch(t *testing.T) {
	processor, _, ctx := setupProcessorTest(t)

	worldId := world.Id(0)
	characterId := uint32(1000)

	_ = processor.Apply(worldId, channel.Id(0), characterId, uint32(2000), int32(124), byte(1), int32(60), []stat.Model{stat.NewStat("POISON", -10)})
	_ = processor.Apply(worldId, channel.Id(0), characterId, uint32(2000), int32(125), byte(1), int32(60), []stat.Model{stat.NewStat("CURSE", -50)})
	_ = processor.Apply(worldId, channel.Id(0), characterId, uint32(2000), int32(126), byte(1), int32(60), []stat.Model{stat.NewStat("WEAKEN", -20)})

	err := processor.CancelByStatTypes(worldId, characterId, []string{"POISON", "CURSE", "WEAKEN", "DARKNESS", "SEAL"})
	assert.NoError(t, err)

	m, _ := GetRegistry().Get(ctx, characterId)
	assert.Len(t, m.Buffs(), 0)
}

func TestProcessor_CancelByStatTypes_HolyShieldDoesNotBlockRemoval(t *testing.T) {
	// D5: Holy Shield gates application, not cure. A character with HOLY_SHIELD
	// who somehow has a debuff must still be curable.
	processor, _, ctx := setupProcessorTest(t)

	worldId := world.Id(0)
	characterId := uint32(1000)

	// Insert a POISON buff via the registry directly so the immunity check on
	// Apply can't refuse it once HOLY_SHIELD is present.
	_, _ = GetRegistry().Apply(ctx, worldId, channel.Id(0), characterId, int32(124), byte(1), int32(60), []stat.Model{stat.NewStat("POISON", -10)})
	_, _ = GetRegistry().Apply(ctx, worldId, channel.Id(0), characterId, int32(2311005), byte(1), int32(60), []stat.Model{stat.NewStat("HOLY_SHIELD", 1)})

	err := processor.CancelByStatTypes(worldId, characterId, []string{"POISON"})
	assert.NoError(t, err)

	m, _ := GetRegistry().Get(ctx, characterId)
	assert.Len(t, m.Buffs(), 1)
	_, stillHasHolyShield := m.Buffs()[int32(2311005)]
	assert.True(t, stillHasHolyShield)
}

func TestProcessor_TenantContext(t *testing.T) {
	setupTestRegistry(t)

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	ten1, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ten2, _ := tenant.Create(uuid.New(), "EMS", 83, 1)

	ctx1 := tenant.WithContext(context.Background(), ten1)
	ctx2 := tenant.WithContext(context.Background(), ten2)

	processor1 := NewProcessor(logger, ctx1)
	processor2 := NewProcessor(logger, ctx2)

	changes := setupProcessorTestChanges()

	_ = processor1.Apply(world.Id(0), channel.Id(0), uint32(1000), uint32(2000), int32(2001001), byte(5), int32(60), changes)

	m, err := processor1.GetById(uint32(1000))
	assert.NoError(t, err)
	assert.Len(t, m.Buffs(), 1)

	_, err = processor2.GetById(uint32(1000))
	assert.ErrorIs(t, err, ErrNotFound)
}
