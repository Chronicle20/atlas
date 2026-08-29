package craft_test

import (
	"atlas-maker/character"
	"atlas-maker/compartment"
	"atlas-maker/craft"
	"atlas-maker/data/itemmake"
	"atlas-maker/quest"
	"atlas-maker/recipe"
	"atlas-maker/skill"
	"context"
	"testing"

	charactermock "atlas-maker/character/mock"

	compartmentmock "atlas-maker/compartment/mock"

	crystalbandmock "atlas-maker/crystalband/mock"

	equipmentmock "atlas-maker/data/equipment/mock"

	itemmakemock "atlas-maker/data/itemmake/mock"

	questmock "atlas-maker/quest/mock"

	reagentmock "atlas-maker/reagent/mock"

	recipemock "atlas-maker/recipe/mock"

	skillmock "atlas-maker/skill/mock"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	skillconst "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	saga "github.com/Chronicle20/atlas/libs/atlas-saga"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// noopEmitter satisfies craft.SagaEmitter without ever being expected to be
// called by the eligibility/snapshot tests in this file, which never reach
// Create.
type noopEmitter struct{}

func (noopEmitter) Emit(saga.Saga) error { return nil }

// testContext builds a fresh per-call tenant context, so tests that build a
// craft.Processor never collide on the package-wide in-flight guard's
// (tenant, characterId) key.
func testContext(t *testing.T) context.Context {
	t.Helper()
	te, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return tenant.WithContext(context.Background(), te)
}

func testLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	return l
}

// buildRecipe indexes rm through a real recipe.Processor (backed by an
// itemmake mock, on a fresh tenant) to obtain a genuine recipe.Model, since
// recipe.Model's fields are private and this package has no other public
// way to construct one (mirrors recipe/processor_test.go's own fixture
// pattern).
func buildRecipe(t *testing.T, rm itemmake.RestModel) recipe.Model {
	t.Helper()
	im, err := itemmake.Extract(rm)
	require.NoError(t, err)

	te, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), te)

	imp := &itemmakemock.ProcessorMock{
		GetAllFunc: func() ([]itemmake.Model, error) {
			return []itemmake.Model{im}, nil
		},
	}
	p := recipe.NewProcessor(testLogger(), ctx, imp)
	m, err := p.GetById(item.Id(rm.Id))
	require.NoError(t, err)
	return m
}

func buildCharacter(t *testing.T, level byte, meso uint32) character.Model {
	t.Helper()
	m, err := character.Extract(character.RestModel{Level: level, Meso: meso})
	require.NoError(t, err)
	return m
}

func buildSkill(t *testing.T, id uint32, level byte) skill.Model {
	t.Helper()
	m, err := skill.Extract(skill.RestModel{Id: id, Level: level})
	require.NoError(t, err)
	return m
}

func buildQuest(t *testing.T, questId uint32, state byte) quest.Model {
	t.Helper()
	m, err := quest.Extract(quest.RestModel{QuestId: questId, State: state})
	require.NoError(t, err)
	return m
}

// eligibleRecipeFixture mirrors the Task 3/4 reference archive's 1082002
// entry: every optional requirement present, and gating exactly the values
// harness asserts against.
func eligibleRecipeFixture(t *testing.T) recipe.Model {
	t.Helper()
	return buildRecipe(t, itemmake.RestModel{
		Id:            1082002,
		Group:         1,
		ReqLevel:      30,
		ReqSkillLevel: 2,
		ItemNum:       1,
		Tuc:           7,
		Meso:          1200,
		Catalyst:      4130000,
		ReqItem:       4000021,
		ReqEquip:      1002419,
		Recipe:        []itemmake.MaterialRestModel{{ItemId: 4011001, Count: 5}},
		ReqQuest:      []itemmake.QuestReqRestModel{{QuestId: 21614, State: 3}},
	})
}

// harness is a mutable set of upstream fixtures a test starts from a fully
// eligible baseline and then breaks exactly one condition of.
type harness struct {
	characterId      uint32
	character        character.Model
	skills           []skill.Model
	equip            compartment.Model
	use              compartment.Model
	etc              compartment.Model
	quests           []quest.Model
	accommodate      bool
	questCallCount   int
	compartmentCalls int
}

// newEligibleHarness satisfies every condition eligibleRecipeFixture checks:
// level 40 (>= reqLevel 30), a Beginner Maker at level 2 (== reqSkillLevel
// 2), reqItem 4000021 held, reqEquip 1002419 equipped, material 4011001 held
// at 5, quest 21614 at state 3, meso 1200, and an accommodating inventory.
func newEligibleHarness(t *testing.T) *harness {
	t.Helper()
	return &harness{
		characterId: 1001,
		character:   buildCharacter(t, 40, 1200),
		skills:      []skill.Model{buildSkill(t, uint32(skillconst.BeginnerMaker), 2)},
		equip: compartment.NewBuilder(inventory.TypeValueEquip).
			AddAsset(compartment.NewAssetModel(item.Id(1002419), 1, -1)).
			Build(),
		use: compartment.NewBuilder(inventory.TypeValueUse).Build(),
		etc: compartment.NewBuilder(inventory.TypeValueETC).
			AddAsset(compartment.NewAssetModel(item.Id(4011001), 5, 1)).
			AddAsset(compartment.NewAssetModel(item.Id(4000021), 1, 2)).
			Build(),
		quests:      []quest.Model{buildQuest(t, 21614, 3)},
		accommodate: true,
	}
}

func (h *harness) processor(t *testing.T) craft.Processor {
	cp := &charactermock.ProcessorMock{
		GetByIdFunc: func(uint32) (character.Model, error) { return h.character, nil },
	}
	sp := &skillmock.ProcessorMock{
		GetByCharacterIdFunc: func(uint32) ([]skill.Model, error) { return h.skills, nil },
	}
	kp := &compartmentmock.ProcessorMock{
		GetByTypeFunc: func(_ uint32, invType inventory.Type) (compartment.Model, error) {
			h.compartmentCalls++
			switch invType {
			case inventory.TypeValueEquip:
				return h.equip, nil
			case inventory.TypeValueUse:
				return h.use, nil
			default:
				return h.etc, nil
			}
		},
		CanAccommodateFunc: func(uint32, []compartment.AccommodationItem) (bool, error) {
			return h.accommodate, nil
		},
	}
	qp := &questmock.ProcessorMock{
		GetByCharacterIdFunc: func(uint32) ([]quest.Model, error) {
			h.questCallCount++
			return h.quests, nil
		},
	}
	rp := &recipemock.ProcessorMock{}
	rgp := &reagentmock.ProcessorMock{}
	cbp := &crystalbandmock.ProcessorMock{}
	eqp := &equipmentmock.ProcessorMock{}
	return craft.NewProcessor(testLogger(), testContext(t), cp, sp, kp, qp, rp, rgp, cbp, eqp, noopEmitter{})
}

func (h *harness) snapshot(t *testing.T, p craft.Processor) craft.Snapshot {
	t.Helper()
	snap, err := p.NewSnapshot(h.characterId)
	require.NoError(t, err)
	return snap
}

// TestEligibilityOnePerExclusionReason is table-driven, one case per
// exclusion reason (PRD §10), each breaking exactly one condition of an
// otherwise fully-eligible fixture.
func TestEligibilityOnePerExclusionReason(t *testing.T) {
	r := eligibleRecipeFixture(t)

	tests := []struct {
		name   string
		mutate func(t *testing.T, h *harness)
		reason craft.Reason
	}{
		{
			name: "level too low",
			mutate: func(t *testing.T, h *harness) {
				h.character = buildCharacter(t, 29, 1200)
			},
			reason: craft.ReasonLevelTooLow,
		},
		{
			name: "no maker skill",
			mutate: func(t *testing.T, h *harness) {
				h.skills = []skill.Model{buildSkill(t, 9999, 5)}
			},
			reason: craft.ReasonSkillLevelTooLow,
		},
		{
			name: "maker skill level zero",
			mutate: func(t *testing.T, h *harness) {
				h.skills = []skill.Model{buildSkill(t, uint32(skillconst.BeginnerMaker), 0)}
			},
			reason: craft.ReasonSkillLevelTooLow,
		},
		{
			name: "maker skill below recipe",
			mutate: func(t *testing.T, h *harness) {
				h.skills = []skill.Model{buildSkill(t, uint32(skillconst.BeginnerMaker), 1)}
			},
			reason: craft.ReasonSkillLevelTooLow,
		},
		{
			name: "missing material",
			mutate: func(t *testing.T, h *harness) {
				h.etc = compartment.NewBuilder(inventory.TypeValueETC).
					AddAsset(compartment.NewAssetModel(item.Id(4011001), 4, 1)).
					AddAsset(compartment.NewAssetModel(item.Id(4000021), 1, 2)).
					Build()
			},
			reason: craft.ReasonInsufficientMaterials,
		},
		{
			name: "missing reqItem",
			mutate: func(t *testing.T, h *harness) {
				h.etc = compartment.NewBuilder(inventory.TypeValueETC).
					AddAsset(compartment.NewAssetModel(item.Id(4011001), 5, 1)).
					Build()
			},
			reason: craft.ReasonMissingPrerequisiteItem,
		},
		{
			name: "missing reqEquip",
			mutate: func(t *testing.T, h *harness) {
				// Held, but in a positive (stored, not worn) slot.
				h.equip = compartment.NewBuilder(inventory.TypeValueEquip).
					AddAsset(compartment.NewAssetModel(item.Id(1002419), 1, 5)).
					Build()
			},
			reason: craft.ReasonMissingPrerequisiteItem,
		},
		{
			name: "missing reqQuest",
			mutate: func(t *testing.T, h *harness) {
				h.quests = []quest.Model{buildQuest(t, 21614, 1)}
			},
			reason: craft.ReasonMissingPrerequisiteQuest,
		},
		{
			name: "insufficient mesos",
			mutate: func(t *testing.T, h *harness) {
				h.character = buildCharacter(t, 40, 1199)
			},
			reason: craft.ReasonInsufficientMesos,
		},
		{
			name: "inventory full",
			mutate: func(t *testing.T, h *harness) {
				h.accommodate = false
			},
			reason: craft.ReasonInventoryFull,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newEligibleHarness(t)
			tc.mutate(t, h)
			p := h.processor(t)
			snap := h.snapshot(t, p)

			e, err := p.Evaluate(h.characterId, snap, r)
			require.NoError(t, err)

			assert.False(t, e.Eligible)
			assert.Equal(t, tc.reason, e.Reason)
		})
	}
}

// TestEligibilityAllConditionsMetIsEligible proves the fully-eligible
// fixture returns Eligible with an empty Reason.
func TestEligibilityAllConditionsMetIsEligible(t *testing.T) {
	r := eligibleRecipeFixture(t)
	h := newEligibleHarness(t)
	p := h.processor(t)
	snap := h.snapshot(t, p)

	e, err := p.Evaluate(h.characterId, snap, r)
	require.NoError(t, err)

	assert.True(t, e.Eligible)
	assert.Empty(t, e.Reason)
}

// TestAnyMakerVariantSatisfiesTheSkillGate proves each of the four Maker
// skill identities, alone, satisfies a reqSkillLevel of 2 at level 2.
func TestAnyMakerVariantSatisfiesTheSkillGate(t *testing.T) {
	r := eligibleRecipeFixture(t)

	identities := []skillconst.Identity{
		skillconst.BeginnerMaker,
		skillconst.NoblesseMaker,
		skillconst.LegendMaker,
		skillconst.EvanMaker,
	}

	for _, id := range identities {
		t.Run(skillconst.IdentityName(id), func(t *testing.T) {
			h := newEligibleHarness(t)
			h.skills = []skill.Model{buildSkill(t, uint32(id), 2)}
			p := h.processor(t)
			snap := h.snapshot(t, p)

			e, err := p.Evaluate(h.characterId, snap, r)
			require.NoError(t, err)

			assert.True(t, e.Eligible)
		})
	}
}

// TestReqQuestIsOnlyReadWhenTheRecipeCarriesOne proves atlas-quests is never
// called for a recipe with no reqQuest (C-5): reading quests for every
// recipe is the cost the check order's step 3 exists to avoid.
func TestReqQuestIsOnlyReadWhenTheRecipeCarriesOne(t *testing.T) {
	r := buildRecipe(t, itemmake.RestModel{
		Id:            2000001,
		Group:         1,
		ReqLevel:      1,
		ReqSkillLevel: 1,
		ItemNum:       1,
	})

	h := newEligibleHarness(t)
	h.character = buildCharacter(t, 40, 0)
	h.etc = compartment.NewBuilder(inventory.TypeValueETC).Build()
	p := h.processor(t)
	snap := h.snapshot(t, p)

	e, err := p.Evaluate(h.characterId, snap, r)
	require.NoError(t, err)

	assert.True(t, e.Eligible)
	assert.Zero(t, h.questCallCount, "atlas-quests must not be called for a recipe with no reqQuest")
}

// TestMaterialCountSumsAcrossStacks proves a 5-count material requirement is
// satisfied by 3+2 held across two slots -- the case a naive single-asset
// read (rather than a sum across the snapshot) would reject.
func TestMaterialCountSumsAcrossStacks(t *testing.T) {
	r := buildRecipe(t, itemmake.RestModel{
		Id:            2000002,
		Group:         1,
		ReqLevel:      1,
		ReqSkillLevel: 1,
		ItemNum:       1,
		Recipe:        []itemmake.MaterialRestModel{{ItemId: 4011001, Count: 5}},
	})

	h := newEligibleHarness(t)
	h.character = buildCharacter(t, 40, 0)
	h.etc = compartment.NewBuilder(inventory.TypeValueETC).
		AddAsset(compartment.NewAssetModel(item.Id(4011001), 3, 1)).
		AddAsset(compartment.NewAssetModel(item.Id(4011001), 2, 2)).
		Build()
	p := h.processor(t)
	snap := h.snapshot(t, p)

	assert.EqualValues(t, 5, snap.Held(item.Id(4011001)))

	e, err := p.Evaluate(h.characterId, snap, r)
	require.NoError(t, err)

	assert.True(t, e.Eligible)
}
