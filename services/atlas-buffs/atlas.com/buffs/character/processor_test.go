package character

import (
	"atlas-buffs/buff/stat"
	character2 "atlas-buffs/kafka/message/character"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
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

	_ = processor.Apply(worldId, channelId, characterId, fromId, sourceId, byte(5), duration, changes, false, false)

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

	_ = processor.Apply(worldId, channelId, characterId, fromId, sourceId, byte(5), duration, changes, false, false)

	m, err := GetRegistry().Get(ctx, characterId)
	assert.NoError(t, err)
	assert.Len(t, m.Buffs(), 1)

	buff := m.Buffs()[srcKey(sourceId)]
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

	_ = processor.Apply(worldId, channelId, characterId, fromId, int32(2001001), byte(5), int32(60), changes, false, false)
	_ = processor.Apply(worldId, channelId, characterId, fromId, int32(2001002), byte(5), int32(120), changes, false, false)
	_ = processor.Apply(worldId, channelId, characterId, fromId, int32(2001003), byte(5), int32(180), changes, false, false)

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

	_ = processor.Apply(worldId, channelId, characterId, fromId, sourceId, byte(5), duration, changes, false, false)

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

	_ = processor.Apply(worldId, channelId, characterId, fromId, sourceId, byte(5), duration, changes, false, false)

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
	_ = processor.Apply(worldId, channel.Id(0), characterId, uint32(2000), int32(2311003), byte(1), int32(60), holy, false, false)

	err := processor.CancelByStatTypes(worldId, characterId, []string{"POISON"})
	assert.NoError(t, err)

	m, _ := GetRegistry().Get(ctx, characterId)
	assert.Len(t, m.Buffs(), 1)
}

func TestProcessor_CancelByStatTypes_MultiMatch(t *testing.T) {
	processor, _, ctx := setupProcessorTest(t)

	worldId := world.Id(0)
	characterId := uint32(1000)

	_ = processor.Apply(worldId, channel.Id(0), characterId, uint32(2000), int32(124), byte(1), int32(60), []stat.Model{stat.NewStat("POISON", -10)}, false, false)
	_ = processor.Apply(worldId, channel.Id(0), characterId, uint32(2000), int32(125), byte(1), int32(60), []stat.Model{stat.NewStat("CURSE", -50)}, false, false)
	_ = processor.Apply(worldId, channel.Id(0), characterId, uint32(2000), int32(126), byte(1), int32(60), []stat.Model{stat.NewStat("WEAKEN", -20)}, false, false)

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
	_, _ = GetRegistry().Apply(ctx, worldId, channel.Id(0), characterId, int32(124), byte(1), int32(60), []stat.Model{stat.NewStat("POISON", -10)}, false, false)
	_, _ = GetRegistry().Apply(ctx, worldId, channel.Id(0), characterId, int32(2311005), byte(1), int32(60), []stat.Model{stat.NewStat("HOLY_SHIELD", 1)}, false, false)

	err := processor.CancelByStatTypes(worldId, characterId, []string{"POISON"})
	assert.NoError(t, err)

	m, _ := GetRegistry().Get(ctx, characterId)
	assert.Len(t, m.Buffs(), 1)
	_, stillHasHolyShield := m.Buffs()[srcKey(2311005)]
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

	_ = processor1.Apply(world.Id(0), channel.Id(0), uint32(1000), uint32(2000), int32(2001001), byte(5), int32(60), changes, false, false)

	m, err := processor1.GetById(uint32(1000))
	assert.NoError(t, err)
	assert.Len(t, m.Buffs(), 1)

	_, err = processor2.GetById(uint32(1000))
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestProcessor_UpdateStatValue_Increment(t *testing.T) {
	processor, _, ctx := setupProcessorTest(t)

	changes := []stat.Model{stat.NewStat("COMBO", 1)}
	_ = processor.Apply(world.Id(0), channel.Id(0), 1000, 1000, 1111002, byte(20), int32(150000), changes, false, false)

	_ = processor.UpdateStatValue(world.Id(0), channel.Id(0), 1000,
		StatValueUpdate{SourceId: 1111002, StatType: "COMBO", Operation: character2.StatOperationIncrement, Amount: 2, Cap: 6})

	m, err := GetRegistry().Get(ctx, 1000)
	assert.NoError(t, err)
	b := m.Buffs()[srcKey(1111002)]
	assert.Equal(t, int32(3), b.Changes()[0].Amount())
}

func TestProcessor_UpdateStatValue_UnknownOperationIsNoOp(t *testing.T) {
	processor, _, ctx := setupProcessorTest(t)

	changes := []stat.Model{stat.NewStat("COMBO", 1)}
	_ = processor.Apply(world.Id(0), channel.Id(0), 1000, 1000, 1111002, byte(20), int32(150000), changes, false, false)

	err := processor.UpdateStatValue(world.Id(0), channel.Id(0), 1000,
		StatValueUpdate{SourceId: 1111002, StatType: "COMBO", Operation: "MULTIPLY", Amount: 2, Cap: 6})
	assert.NoError(t, err, "unknown operation is a logged no-op, not an error")

	m, err := GetRegistry().Get(ctx, 1000)
	assert.NoError(t, err)
	b := m.Buffs()[srcKey(1111002)]
	assert.Equal(t, int32(1), b.Changes()[0].Amount())
}

func TestProcessor_UpdateStatValue_MissingBuffIsNoOp(t *testing.T) {
	processor, _, _ := setupProcessorTest(t)
	err := processor.UpdateStatValue(world.Id(0), channel.Id(0), 1000,
		StatValueUpdate{SourceId: 1111002, StatType: "COMBO", Operation: character2.StatOperationIncrement, Amount: 1, Cap: 6})
	assert.NoError(t, err, "missing buff is a logged no-op, not an error")
}

// A CreateIfMissing increment against a character with no buffs at all
// stores a NoExpiry buff — the first Energy Charge hit of the cycle.
func TestProcessor_UpdateStatValue_CreateIfMissingStoresBuff(t *testing.T) {
	processor, _, ctx := setupProcessorTest(t)

	err := processor.UpdateStatValue(world.Id(0), channel.Id(0), 1000,
		StatValueUpdate{
			SourceId: 5110001, StatType: "ENERGY_CHARGE",
			Operation: character2.StatOperationIncrement,
			Amount:    102, Cap: 10000, CreateIfMissing: true, Level: 20,
		})
	assert.NoError(t, err)

	m, err := GetRegistry().Get(ctx, 1000)
	assert.NoError(t, err)
	b := m.Buffs()[srcKey(5110001)]
	assert.True(t, b.NoExpiry())
	assert.Equal(t, int32(102), b.Changes()[0].Amount())
}

// Without CreateIfMissing a missing buff stays a logged no-op (Combo).
func TestProcessor_UpdateStatValue_MissingBuffWithoutCreateStoresNothing(t *testing.T) {
	processor, _, ctx := setupProcessorTest(t)

	err := processor.UpdateStatValue(world.Id(0), channel.Id(0), 1000,
		StatValueUpdate{SourceId: 1111002, StatType: "COMBO", Operation: character2.StatOperationIncrement, Amount: 1, Cap: 6})
	assert.NoError(t, err)

	_, err = GetRegistry().Get(ctx, 1000)
	assert.ErrorIs(t, err, ErrNotFound)
}

// TestProcessor_ExpireForCharacter_PrunesLapsedBuff — the CANCEL_DEBUFF
// reconcile path. Duration is MILLISECONDS, so a 1ms buff has lapsed by the
// time the sweep runs and must be pruned. (task-190 FR-2.6.1)
func TestProcessor_ExpireForCharacter_PrunesLapsedBuff(t *testing.T) {
	processor, _, ctx := setupProcessorTest(t)

	const characterId = uint32(5001)
	assert.NoError(t, processor.Apply(world.Id(0), channel.Id(0), characterId, 0, 1002, 1, 1, setupProcessorTestChanges(), false, false))
	time.Sleep(5 * time.Millisecond)

	assert.NoError(t, processor.ExpireForCharacter(world.Id(0), characterId))

	// The sweep pruned it: a second GetExpired finds nothing left to expire.
	assert.Empty(t, GetRegistry().GetExpired(ctx, characterId))
}

// TestProcessor_ExpireForCharacter_NothingExpired — FR-2.9 / NFR-2.1: a nudge
// for a character with nothing lapsed is a no-op. The live buff must survive.
func TestProcessor_ExpireForCharacter_NothingExpired(t *testing.T) {
	processor, _, ctx := setupProcessorTest(t)

	const characterId = uint32(5000)
	const sourceId = int32(1001)
	assert.NoError(t, processor.Apply(world.Id(0), channel.Id(0), characterId, 0, sourceId, 1, 60_000, setupProcessorTestChanges(), false, false))

	assert.NoError(t, processor.ExpireForCharacter(world.Id(0), characterId))

	// Still resident: cancelling it now succeeds and returns the buff.
	cancelled, err := GetRegistry().Cancel(ctx, characterId, sourceId)
	assert.NoError(t, err)
	assert.NotEmpty(t, cancelled)
}

// TestProcessor_ExpireBuffs_StillSweepsFleetWide — the shared helper must not
// change the fleet sweep's behaviour.
func TestProcessor_ExpireBuffs_StillSweepsFleetWide(t *testing.T) {
	processor, _, ctx := setupProcessorTest(t)

	const charA = uint32(5002)
	const charB = uint32(5003)
	assert.NoError(t, processor.Apply(world.Id(0), channel.Id(0), charA, 0, 1003, 1, 1, setupProcessorTestChanges(), false, false))
	assert.NoError(t, processor.Apply(world.Id(0), channel.Id(0), charB, 0, 1004, 1, 1, setupProcessorTestChanges(), false, false))
	time.Sleep(5 * time.Millisecond)

	assert.NoError(t, processor.ExpireBuffs())

	assert.Empty(t, GetRegistry().GetExpired(ctx, charA))
	assert.Empty(t, GetRegistry().GetExpired(ctx, charB))
}
