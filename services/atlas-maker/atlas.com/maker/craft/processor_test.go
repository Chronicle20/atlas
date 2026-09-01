package craft_test

import (
	"atlas-maker/character"
	"atlas-maker/compartment"
	"atlas-maker/craft"
	"atlas-maker/data/equipment"
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
	questmock "atlas-maker/quest/mock"
	reagentmock "atlas-maker/reagent/mock"
	recipemock "atlas-maker/recipe/mock"
	skillmock "atlas-maker/skill/mock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	saga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

// spyEmitter records every saga it is given, so a test can assert both
// "exactly one saga was emitted, shaped like X" and "zero sagas were
// emitted" (TestEveryRejectionEmitsNoSaga's whole point). duringEmit, if
// set, runs after the call is recorded but before Emit returns -- ordering_
// test.go uses it to simulate the saga's terminal event being observed (and
// released) synchronously inside the produce call, before the caller ever
// regains control.
type spyEmitter struct {
	calls      []saga.Saga
	err        error
	duringEmit func(saga.Saga)
}

func (e *spyEmitter) Emit(s saga.Saga) error {
	e.calls = append(e.calls, s)
	if e.duringEmit != nil {
		e.duringEmit(s)
	}
	return e.err
}

// deps bundles the Create-only upstream mocks that harness (eligibility_test.go)
// does not construct, so processor_test.go's cases can each configure only
// the ones they care about.
type deps struct {
	rp  *recipemock.ProcessorMock
	rgp *reagentmock.ProcessorMock
	cbp *crystalbandmock.ProcessorMock
	eqp *equipmentmock.ProcessorMock
	em  *spyEmitter
}

func newDeps() *deps {
	return &deps{
		rp:  &recipemock.ProcessorMock{},
		rgp: &reagentmock.ProcessorMock{},
		cbp: &crystalbandmock.ProcessorMock{},
		eqp: &equipmentmock.ProcessorMock{},
		em:  &spyEmitter{},
	}
}

// buildCreateProcessor wires h's character/skill/compartment/quest fixtures
// together with d's recipe/reagent/crystalband/equipment/emitter mocks --
// the full dependency set Create needs, which harness.processor (eligibility
// tests only) does not build.
func buildCreateProcessor(t *testing.T, h *harness, d *deps) craft.Processor {
	t.Helper()
	return buildCreateProcessorWithContext(t, testContext(t), h, d)
}

// buildCreateProcessorWithContext is buildCreateProcessor with the tenant
// context supplied by the caller, so a test that needs to reach the same
// tenant id craftGuard uses internally (ordering_test.go's emit-vs-Track
// races) can build one it already knows the id of, instead of the fresh
// random one testContext mints on every call.
func buildCreateProcessorWithContext(t *testing.T, ctx context.Context, h *harness, d *deps) craft.Processor {
	t.Helper()
	cp := &charactermock.ProcessorMock{
		GetByIdFunc: func(uint32) (character.Model, error) { return h.character, nil },
	}
	sp := &skillmock.ProcessorMock{
		GetByCharacterIdFunc: func(uint32) ([]skill.Model, error) { return h.skills, nil },
	}
	kp := &compartmentmock.ProcessorMock{
		GetByTypeFunc: func(_ uint32, invType inventory.Type) (compartment.Model, error) {
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
		GetByCharacterIdFunc: func(uint32) ([]quest.Model, error) { return h.quests, nil },
	}
	return craft.NewProcessor(testLogger(), ctx, cp, sp, kp, qp, d.rp, d.rgp, d.cbp, d.eqp, d.em)
}

func TestCreateModeOneBuildsSequence(t *testing.T) {
	r := eligibleRecipeFixture(t)
	h := newEligibleHarness(t)
	d := newDeps()
	d.rp.GetByIdFunc = func(item.Id) (recipe.Model, error) { return r, nil }

	p := buildCreateProcessor(t, h, d)

	txId, err := p.Create(h.characterId, craft.Request{
		Mode:         craft.ModeCreate,
		TargetItemId: r.Id(),
		UseCatalyst:  true,
	})
	require.NoError(t, err)
	assert.NotZero(t, txId)

	require.Len(t, d.em.calls, 1)
	steps := d.em.calls[0].Steps

	// RecordCraftManifest -> AwardMesos (negative) -> 1 destroy step (h's
	// fixture holds material 4011001 at count 5 in one slot; the catalyst
	// 4130000 is not held, so BuildCreatePlan resolves no catalyst
	// consumption despite UseCatalyst: true) -> AwardCraftedAsset.
	require.Len(t, steps, 4)

	assert.Equal(t, saga.RecordCraftManifest, steps[0].Action)
	manifest := steps[0].Payload.(saga.CraftManifestPayload)
	assert.EqualValues(t, h.characterId, manifest.CharacterId)
	assert.EqualValues(t, craft.ModeCreate, manifest.Mode)
	assert.EqualValues(t, r.Id(), manifest.TargetItemId)
	assert.False(t, manifest.CatalystUsed)
	assert.Zero(t, manifest.CatalystItemId)
	assert.EqualValues(t, 1200, manifest.MesoCost)

	assert.Equal(t, saga.AwardMesos, steps[1].Action)
	mesos := steps[1].Payload.(saga.AwardMesosPayload)
	assert.EqualValues(t, -1200, mesos.Amount)

	assert.Equal(t, saga.DestroyAssetFromSlot, steps[2].Action)

	last := steps[3]
	assert.Equal(t, saga.AwardCraftedAsset, last.Action)
	award := last.Payload.(saga.AwardCraftedAssetPayload)
	assert.EqualValues(t, r.Tuc(), award.Slots)
	assert.EqualValues(t, r.Id(), award.TemplateId)
}

func TestCreateModeOneNonEquipUsesAwardAsset(t *testing.T) {
	r := buildRecipe(t, itemmake.RestModel{
		Id:            2000010,
		Group:         1,
		ReqLevel:      1,
		ReqSkillLevel: 1,
		ItemNum:       3,
	})
	h := newEligibleHarness(t)
	h.character = buildCharacter(t, 40, 0)
	d := newDeps()
	d.rp.GetByIdFunc = func(item.Id) (recipe.Model, error) { return r, nil }

	p := buildCreateProcessor(t, h, d)

	_, err := p.Create(h.characterId, craft.Request{Mode: craft.ModeCreate, TargetItemId: r.Id()})
	require.NoError(t, err)

	require.Len(t, d.em.calls, 1)
	steps := d.em.calls[0].Steps
	last := steps[len(steps)-1]
	assert.Equal(t, saga.AwardAsset, last.Action)
	award := last.Payload.(saga.AwardItemActionPayload)
	assert.EqualValues(t, r.Id(), award.Item.TemplateId)
	assert.EqualValues(t, 3, award.Item.Quantity)
}

// crystalHarness builds a fully-eligible mode-3 fixture: a group-0 recipe
// whose sole material is the leftover (archive count 1, irrelevant per
// OQ-7) and a character holding at least LeftoverConsumeQuantity of it.
func crystalHarness(t *testing.T) (*harness, recipe.Model) {
	t.Helper()
	r := buildRecipe(t, itemmake.RestModel{
		Id:            4260000,
		Group:         0,
		ReqLevel:      1,
		ReqSkillLevel: 1,
		Meso:          100,
		Recipe:        []itemmake.MaterialRestModel{{ItemId: 4200000, Count: 1}},
		RandomReward: []itemmake.RewardRestModel{
			{ItemId: 4260000, ItemNum: 1, Prob: 100},
		},
	})
	h := newEligibleHarness(t)
	h.character = buildCharacter(t, 40, 100)
	h.etc = compartment.NewBuilder(inventory.TypeValueETC).
		AddAsset(compartment.NewAssetModel(item.Id(4200000), 100, 1)).
		Build()
	return h, r
}

func TestCreateModeThreeConsumesOneHundredLeftover(t *testing.T) {
	h, r := crystalHarness(t)
	d := newDeps()
	d.rp.GetByLeftoverFunc = func(item.Id) (recipe.Model, error) { return r, nil }

	p := buildCreateProcessor(t, h, d)

	_, err := p.Create(h.characterId, craft.Request{Mode: craft.ModeMonsterCrystal, LeftoverItemId: 4200000})
	require.NoError(t, err)

	require.Len(t, d.em.calls, 1)
	var total uint32
	for _, st := range d.em.calls[0].Steps {
		if st.Action == saga.DestroyAssetFromSlot {
			total += st.Payload.(saga.DestroyAssetFromSlotPayload).Quantity
		}
	}
	assert.EqualValues(t, craft.LeftoverConsumeQuantity, total)
}

func TestCreateModeThreeAwardsOneWeightedDraw(t *testing.T) {
	h, r := crystalHarness(t)
	d := newDeps()
	d.rp.GetByLeftoverFunc = func(item.Id) (recipe.Model, error) { return r, nil }

	p := buildCreateProcessor(t, h, d)

	_, err := p.Create(h.characterId, craft.Request{Mode: craft.ModeMonsterCrystal, LeftoverItemId: 4200000})
	require.NoError(t, err)

	require.Len(t, d.em.calls, 1)
	var awards []saga.AwardItemActionPayload
	for _, st := range d.em.calls[0].Steps {
		if st.Action == saga.AwardAsset {
			awards = append(awards, st.Payload.(saga.AwardItemActionPayload))
		}
	}
	require.Len(t, awards, 1)

	rewarded := false
	for _, rw := range r.RandomRewards() {
		if item.Id(awards[0].Item.TemplateId) == rw.ItemId {
			rewarded = true
		}
	}
	assert.True(t, rewarded)
}

// disassembleHarness builds an eligible mode-4 fixture: a Maker skill at
// level 1, an equip held at a known EQUIP slot, and its equipment.Model
// resolved to a reqLevel a seeded crystal band covers.
func disassembleHarness(t *testing.T) (*harness, item.Id, int16) {
	t.Helper()
	equipId := item.Id(1002419)
	slot := int16(5)
	h := newEligibleHarness(t)
	h.character = buildCharacter(t, 40, 0)
	h.equip = compartment.NewBuilder(inventory.TypeValueEquip).
		AddAsset(compartment.NewAssetModel(equipId, 1, slot)).
		Build()
	return h, equipId, slot
}

func TestCreateModeFourVerifiesSlotBeforeDestroying(t *testing.T) {
	h, equipId, _ := disassembleHarness(t)
	d := newDeps()

	p := buildCreateProcessor(t, h, d)

	_, err := p.Create(h.characterId, craft.Request{
		Mode:        craft.ModeDisassemble,
		EquipItemId: equipId,
		SlotPos:     99, // the fixture holds equipId at slot 5, not 99
	})
	require.Error(t, err)
	ce, ok := err.(craft.CraftError)
	require.True(t, ok)
	assert.Equal(t, craft.CodeEquipNotFound, ce.Code)
	assert.Empty(t, d.em.calls)
}

func TestCreateModeFourChargesMeso(t *testing.T) {
	h, equipId, slot := disassembleHarness(t)
	d := newDeps()
	d.eqp.GetByIdFunc = func(item.Id) (equipment.Model, error) {
		return equipment.Extract(equipment.RestModel{Id: uint32(equipId), ReqLevel: 50})
	}
	d.cbp.CrystalForLevelFunc = func(reqLevel uint32) (item.Id, uint32, error) {
		assert.EqualValues(t, 50, reqLevel)
		return item.Id(4260005), 3, nil
	}

	p := buildCreateProcessor(t, h, d)

	_, err := p.Create(h.characterId, craft.Request{
		Mode:        craft.ModeDisassemble,
		EquipItemId: equipId,
		SlotPos:     slot,
	})
	require.NoError(t, err)

	require.Len(t, d.em.calls, 1)
	steps := d.em.calls[0].Steps
	require.NotEmpty(t, steps)
	last := steps[len(steps)-1]
	assert.Equal(t, saga.AwardMesos, last.Action)
	mesos := last.Payload.(saga.AwardMesosPayload)
	assert.EqualValues(t, -craft.DisassembleMesoCharge, mesos.Amount)

	var crystalAwarded bool
	for _, st := range steps {
		if st.Action == saga.AwardAsset {
			award := st.Payload.(saga.AwardItemActionPayload)
			if item.Id(award.Item.TemplateId) == item.Id(4260005) && award.Item.Quantity == 3 {
				crystalAwarded = true
			}
		}
	}
	assert.True(t, crystalAwarded)
}

func TestEveryRejectionEmitsNoSaga(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(t *testing.T, h *harness)
		request craft.Request
		code    craft.Code
	}{
		{
			name:    "level too low",
			mutate:  func(t *testing.T, h *harness) { h.character = buildCharacter(t, 29, 1200) },
			request: craft.Request{Mode: craft.ModeCreate},
			code:    craft.CodeLevelTooLow,
		},
		{
			name:    "insufficient mesos",
			mutate:  func(t *testing.T, h *harness) { h.character = buildCharacter(t, 40, 0) },
			request: craft.Request{Mode: craft.ModeCreate},
			code:    craft.CodeInsufficientMesos,
		},
		{
			name:    "inventory full",
			mutate:  func(t *testing.T, h *harness) { h.accommodate = false },
			request: craft.Request{Mode: craft.ModeCreate},
			code:    craft.CodeInventoryFull,
		},
		{
			name:    "missing reqQuest",
			mutate:  func(t *testing.T, h *harness) { h.quests = nil },
			request: craft.Request{Mode: craft.ModeCreate},
			code:    craft.CodeMissingPrerequisiteQuest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := eligibleRecipeFixture(t)
			h := newEligibleHarness(t)
			tc.mutate(t, h)
			d := newDeps()
			d.rp.GetByIdFunc = func(item.Id) (recipe.Model, error) { return r, nil }

			p := buildCreateProcessor(t, h, d)

			req := tc.request
			req.TargetItemId = r.Id()
			_, err := p.Create(h.characterId, req)
			require.Error(t, err)
			ce, ok := err.(craft.CraftError)
			require.True(t, ok)
			assert.Equal(t, tc.code, ce.Code)
			assert.Empty(t, d.em.calls, "a rejection must emit no saga")
		})
	}

	t.Run("recipe not found", func(t *testing.T) {
		h := newEligibleHarness(t)
		d := newDeps()
		d.rp.GetByIdFunc = func(item.Id) (recipe.Model, error) { return recipe.Model{}, recipe.ErrNotFound }
		p := buildCreateProcessor(t, h, d)
		_, err := p.Create(h.characterId, craft.Request{Mode: craft.ModeCreate, TargetItemId: 9999999})
		require.Error(t, err)
		assert.Equal(t, craft.ErrRecipeNotFound, err)
		assert.Empty(t, d.em.calls)
	})

	t.Run("no crystal mapping", func(t *testing.T) {
		h := newEligibleHarness(t)
		d := newDeps()
		d.rp.GetByLeftoverFunc = func(item.Id) (recipe.Model, error) { return recipe.Model{}, recipe.ErrNoCrystalMapping }
		p := buildCreateProcessor(t, h, d)
		_, err := p.Create(h.characterId, craft.Request{Mode: craft.ModeMonsterCrystal, LeftoverItemId: 4200099})
		require.Error(t, err)
		ce, ok := err.(craft.CraftError)
		require.True(t, ok)
		assert.Equal(t, craft.CodeNoCrystalMapping, ce.Code)
		assert.Empty(t, d.em.calls)
	})

	t.Run("equip not found", func(t *testing.T) {
		h, equipId, _ := disassembleHarness(t)
		d := newDeps()
		p := buildCreateProcessor(t, h, d)
		_, err := p.Create(h.characterId, craft.Request{Mode: craft.ModeDisassemble, EquipItemId: equipId, SlotPos: 1})
		require.Error(t, err)
		ce, ok := err.(craft.CraftError)
		require.True(t, ok)
		assert.Equal(t, craft.CodeEquipNotFound, ce.Code)
		assert.Empty(t, d.em.calls)
	})

	t.Run("craft in progress", func(t *testing.T) {
		r := eligibleRecipeFixture(t)
		h := newEligibleHarness(t)
		d := newDeps()
		d.rp.GetByIdFunc = func(item.Id) (recipe.Model, error) { return r, nil }
		p := buildCreateProcessor(t, h, d)

		_, err := p.Create(h.characterId, craft.Request{Mode: craft.ModeCreate, TargetItemId: r.Id()})
		require.NoError(t, err)
		require.Len(t, d.em.calls, 1)

		_, err = p.Create(h.characterId, craft.Request{Mode: craft.ModeCreate, TargetItemId: r.Id()})
		require.Error(t, err)
		assert.Equal(t, craft.ErrCraftInProgress, err)
		assert.Len(t, d.em.calls, 1, "the second, rejected call must not emit a second saga")
	})
}

// TestManifestStepIsFirstInEveryMode is table-driven over all four modes:
// steps[0].Action is always RecordCraftManifest, and the remaining step
// order is byte-identical to today's (task-285 Task 26a).
func TestManifestStepIsFirstInEveryMode(t *testing.T) {
	t.Run("mode 1 create", func(t *testing.T) {
		r := eligibleRecipeFixture(t)
		h := newEligibleHarness(t)
		d := newDeps()
		d.rp.GetByIdFunc = func(item.Id) (recipe.Model, error) { return r, nil }
		p := buildCreateProcessor(t, h, d)

		_, err := p.Create(h.characterId, craft.Request{Mode: craft.ModeCreate, TargetItemId: r.Id()})
		require.NoError(t, err)
		require.Len(t, d.em.calls, 1)

		wantOrder := []saga.Action{saga.RecordCraftManifest, saga.AwardMesos, saga.DestroyAssetFromSlot, saga.AwardCraftedAsset}
		steps := d.em.calls[0].Steps
		require.Len(t, steps, len(wantOrder))
		for i, a := range wantOrder {
			assert.Equal(t, a, steps[i].Action)
		}
	})

	t.Run("mode 2 create with upgrade", func(t *testing.T) {
		r := eligibleRecipeFixture(t)
		h := newEligibleHarness(t)
		d := newDeps()
		d.rp.GetByIdFunc = func(item.Id) (recipe.Model, error) { return r, nil }
		p := buildCreateProcessor(t, h, d)

		_, err := p.Create(h.characterId, craft.Request{Mode: craft.ModeCreateWithUpgrade, TargetItemId: r.Id()})
		require.NoError(t, err)
		require.Len(t, d.em.calls, 1)

		wantOrder := []saga.Action{saga.RecordCraftManifest, saga.AwardMesos, saga.DestroyAssetFromSlot, saga.AwardCraftedAsset}
		steps := d.em.calls[0].Steps
		require.Len(t, steps, len(wantOrder))
		for i, a := range wantOrder {
			assert.Equal(t, a, steps[i].Action)
		}
	})

	t.Run("mode 3 monster crystal", func(t *testing.T) {
		h, r := crystalHarness(t)
		d := newDeps()
		d.rp.GetByLeftoverFunc = func(item.Id) (recipe.Model, error) { return r, nil }
		p := buildCreateProcessor(t, h, d)

		_, err := p.Create(h.characterId, craft.Request{Mode: craft.ModeMonsterCrystal, LeftoverItemId: 4200000})
		require.NoError(t, err)
		require.Len(t, d.em.calls, 1)

		wantOrder := []saga.Action{saga.RecordCraftManifest, saga.AwardMesos, saga.DestroyAssetFromSlot, saga.AwardAsset}
		steps := d.em.calls[0].Steps
		require.Len(t, steps, len(wantOrder))
		for i, a := range wantOrder {
			assert.Equal(t, a, steps[i].Action)
		}
	})

	t.Run("mode 4 disassemble", func(t *testing.T) {
		h, equipId, slot := disassembleHarness(t)
		d := newDeps()
		d.eqp.GetByIdFunc = func(item.Id) (equipment.Model, error) {
			return equipment.Extract(equipment.RestModel{Id: uint32(equipId), ReqLevel: 50})
		}
		d.cbp.CrystalForLevelFunc = func(uint32) (item.Id, uint32, error) { return item.Id(4260005), 1, nil }
		p := buildCreateProcessor(t, h, d)

		_, err := p.Create(h.characterId, craft.Request{Mode: craft.ModeDisassemble, EquipItemId: equipId, SlotPos: slot})
		require.NoError(t, err)
		require.Len(t, d.em.calls, 1)

		wantOrder := []saga.Action{saga.RecordCraftManifest, saga.DestroyAssetFromSlot, saga.AwardAsset, saga.AwardMesos}
		steps := d.em.calls[0].Steps
		require.Len(t, steps, len(wantOrder))
		for i, a := range wantOrder {
			assert.Equal(t, a, steps[i].Action)
		}
	})
}

// TestModeOneAndTwoManifestsDifferOnlyByMode proves the carrier's whole
// reason to exist (manifest-carrier-derivation.md §2 F2): createOrUpgrade
// never branches on req.Mode when it builds the step list, so mode 1 and
// mode 2 produce byte-identical sagas except for the manifest's Mode field.
func TestModeOneAndTwoManifestsDifferOnlyByMode(t *testing.T) {
	r := eligibleRecipeFixture(t)

	build := func(mode craft.Mode) saga.CraftManifestPayload {
		h := newEligibleHarness(t)
		d := newDeps()
		d.rp.GetByIdFunc = func(item.Id) (recipe.Model, error) { return r, nil }
		p := buildCreateProcessor(t, h, d)

		_, err := p.Create(h.characterId, craft.Request{Mode: mode, TargetItemId: r.Id()})
		require.NoError(t, err)
		require.Len(t, d.em.calls, 1)
		return d.em.calls[0].Steps[0].Payload.(saga.CraftManifestPayload)
	}

	m1 := build(craft.ModeCreate)
	m2 := build(craft.ModeCreateWithUpgrade)

	assert.EqualValues(t, 1, m1.Mode)
	assert.EqualValues(t, 2, m2.Mode)

	m1.Mode = 0
	m2.Mode = 0
	assert.Equal(t, m1, m2)
}

// TestCreateManifestMatchesActualConsumption asserts the manifest's
// Materials aggregate by item id and their totals equal the sum of the
// DestroyAssetFromSlot steps' quantities for that id.
func TestCreateManifestMatchesActualConsumption(t *testing.T) {
	r := eligibleRecipeFixture(t)
	h := newEligibleHarness(t)
	d := newDeps()
	d.rp.GetByIdFunc = func(item.Id) (recipe.Model, error) { return r, nil }
	p := buildCreateProcessor(t, h, d)

	_, err := p.Create(h.characterId, craft.Request{Mode: craft.ModeCreate, TargetItemId: r.Id()})
	require.NoError(t, err)
	require.Len(t, d.em.calls, 1)
	steps := d.em.calls[0].Steps
	manifest := steps[0].Payload.(saga.CraftManifestPayload)

	destroyed := map[uint32]uint32{}
	for _, st := range steps {
		if st.Action != saga.DestroyAssetFromSlot {
			continue
		}
		payload := st.Payload.(saga.DestroyAssetFromSlotPayload)
		destroyed[payload.TemplateId] += payload.Quantity
	}

	materials := map[uint32]uint32{}
	for _, m := range manifest.Materials {
		materials[m.ItemId] = m.Count
	}
	assert.Equal(t, destroyed, materials)
	assert.EqualValues(t, r.Id(), manifest.TargetItemId)
	assert.EqualValues(t, r.ItemNum(), manifest.ItemNum)
	assert.EqualValues(t, r.Meso(), manifest.MesoCost)
}

// TestUnheldGemIsDroppedFromManifest covers FR-3.2: a gem the snapshot does
// not hold is dropped entirely, not rejected, and is absent from both the
// manifest and the destroy steps. It also covers the duplicate-id case
// (gemFrequency, plan.go): an id named twice but held once appears once.
func TestUnheldGemIsDroppedFromManifest(t *testing.T) {
	r := eligibleRecipeFixture(t)
	h := newEligibleHarness(t)
	h.etc = compartment.NewBuilder(inventory.TypeValueETC).
		AddAsset(compartment.NewAssetModel(item.Id(4011001), 5, 1)).
		AddAsset(compartment.NewAssetModel(item.Id(4000021), 1, 2)).
		AddAsset(compartment.NewAssetModel(item.Id(4260000), 1, 3)).
		Build()
	d := newDeps()
	d.rp.GetByIdFunc = func(item.Id) (recipe.Model, error) { return r, nil }
	p := buildCreateProcessor(t, h, d)

	// 4260099 is not held at all; 4260000 is held once but named twice.
	gems := []item.Id{4260099, 4260000, 4260000}
	_, err := p.Create(h.characterId, craft.Request{Mode: craft.ModeCreate, TargetItemId: r.Id(), GemItemIds: gems})
	require.NoError(t, err)
	require.Len(t, d.em.calls, 1)
	steps := d.em.calls[0].Steps
	manifest := steps[0].Payload.(saga.CraftManifestPayload)

	require.Len(t, manifest.GemItemIds, 1)
	assert.EqualValues(t, 4260000, manifest.GemItemIds[0])

	var gemDestroyed int
	for _, st := range steps {
		if st.Action != saga.DestroyAssetFromSlot {
			continue
		}
		payload := st.Payload.(saga.DestroyAssetFromSlotPayload)
		if payload.TemplateId == 4260099 {
			t.Fatalf("unheld gem 4260099 must not appear in a destroy step")
		}
		if payload.TemplateId == 4260000 {
			gemDestroyed++
		}
	}
	assert.Equal(t, 1, gemDestroyed, "a gem named twice but held once must appear once")
}

// TestUnheldCatalystLeavesCatalystUnused matches the fact that
// BuildCreatePlan resolves no catalyst consumption when the catalyst is
// unheld even though UseCatalyst was requested.
func TestUnheldCatalystLeavesCatalystUnused(t *testing.T) {
	r := eligibleRecipeFixture(t)
	h := newEligibleHarness(t) // does not hold 4130000, the recipe's catalyst
	d := newDeps()
	d.rp.GetByIdFunc = func(item.Id) (recipe.Model, error) { return r, nil }
	p := buildCreateProcessor(t, h, d)

	_, err := p.Create(h.characterId, craft.Request{Mode: craft.ModeCreate, TargetItemId: r.Id(), UseCatalyst: true})
	require.NoError(t, err)
	require.Len(t, d.em.calls, 1)
	manifest := d.em.calls[0].Steps[0].Payload.(saga.CraftManifestPayload)
	assert.False(t, manifest.CatalystUsed)
	assert.Zero(t, manifest.CatalystItemId)
}

// TestCrystalManifestCarriesDrawnRewardAndLeftover asserts mode 3's manifest
// fields: CrystalItemId equals the award_crystal step's template id,
// LeftoverItemId equals req.LeftoverItemId, and Materials stays empty (the
// MONSTER_CRYSTAL arm takes no material list -- derivation §5).
func TestCrystalManifestCarriesDrawnRewardAndLeftover(t *testing.T) {
	h, r := crystalHarness(t)
	d := newDeps()
	d.rp.GetByLeftoverFunc = func(item.Id) (recipe.Model, error) { return r, nil }
	p := buildCreateProcessor(t, h, d)

	_, err := p.Create(h.characterId, craft.Request{Mode: craft.ModeMonsterCrystal, LeftoverItemId: 4200000})
	require.NoError(t, err)
	require.Len(t, d.em.calls, 1)
	steps := d.em.calls[0].Steps
	manifest := steps[0].Payload.(saga.CraftManifestPayload)

	var awardedItemId uint32
	for _, st := range steps {
		if st.Action == saga.AwardAsset {
			awardedItemId = st.Payload.(saga.AwardItemActionPayload).Item.TemplateId
		}
	}
	require.NotZero(t, awardedItemId)
	assert.EqualValues(t, awardedItemId, manifest.CrystalItemId)
	assert.EqualValues(t, 4200000, manifest.LeftoverItemId)
	assert.Empty(t, manifest.Materials)
	assert.EqualValues(t, 3, manifest.Mode)
}

// TestDisassembleManifestCarriesCrystalsAndZeroMeso asserts mode 4's
// manifest against the DisassembleMesoCharge constant, not a literal, so a
// future retune does not silently pass.
func TestDisassembleManifestCarriesCrystalsAndZeroMeso(t *testing.T) {
	h, equipId, slot := disassembleHarness(t)
	d := newDeps()
	d.eqp.GetByIdFunc = func(item.Id) (equipment.Model, error) {
		return equipment.Extract(equipment.RestModel{Id: uint32(equipId), ReqLevel: 50})
	}
	d.cbp.CrystalForLevelFunc = func(uint32) (item.Id, uint32, error) { return item.Id(4260005), 3, nil }
	p := buildCreateProcessor(t, h, d)

	_, err := p.Create(h.characterId, craft.Request{Mode: craft.ModeDisassemble, EquipItemId: equipId, SlotPos: slot})
	require.NoError(t, err)
	require.Len(t, d.em.calls, 1)
	manifest := d.em.calls[0].Steps[0].Payload.(saga.CraftManifestPayload)

	assert.EqualValues(t, equipId, manifest.DisassembledItemId)
	require.Len(t, manifest.Crystals, 1)
	assert.EqualValues(t, 4260005, manifest.Crystals[0].ItemId)
	assert.EqualValues(t, 3, manifest.Crystals[0].Count)
	assert.EqualValues(t, craft.DisassembleMesoCharge, manifest.MesoCost)
	assert.EqualValues(t, 4, manifest.Mode)
}

// TestEveryRejectionStillEmitsNoSaga extends the rejection sweep so the new
// manifest step does not create an emission on a rejected craft.
func TestEveryRejectionStillEmitsNoSaga(t *testing.T) {
	r := eligibleRecipeFixture(t)
	h := newEligibleHarness(t)
	h.character = buildCharacter(t, 29, 1200) // level too low
	d := newDeps()
	d.rp.GetByIdFunc = func(item.Id) (recipe.Model, error) { return r, nil }
	p := buildCreateProcessor(t, h, d)

	_, err := p.Create(h.characterId, craft.Request{Mode: craft.ModeCreate, TargetItemId: r.Id()})
	require.Error(t, err)
	assert.Empty(t, d.em.calls, "a rejected craft must never emit even the manifest step")
}

func TestEveryStepUsesACompensableAction(t *testing.T) {
	r := eligibleRecipeFixture(t)
	h := newEligibleHarness(t)
	d := newDeps()
	d.rp.GetByIdFunc = func(item.Id) (recipe.Model, error) { return r, nil }
	p := buildCreateProcessor(t, h, d)
	_, err := p.Create(h.characterId, craft.Request{Mode: craft.ModeCreate, TargetItemId: r.Id(), UseCatalyst: true})
	require.NoError(t, err)

	hCrystal, rCrystal := crystalHarness(t)
	dCrystal := newDeps()
	dCrystal.rp.GetByLeftoverFunc = func(item.Id) (recipe.Model, error) { return rCrystal, nil }
	pCrystal := buildCreateProcessor(t, hCrystal, dCrystal)
	_, err = pCrystal.Create(hCrystal.characterId, craft.Request{Mode: craft.ModeMonsterCrystal, LeftoverItemId: 4200000})
	require.NoError(t, err)

	hDis, equipId, slot := disassembleHarness(t)
	dDis := newDeps()
	dDis.eqp.GetByIdFunc = func(item.Id) (equipment.Model, error) {
		return equipment.Extract(equipment.RestModel{Id: uint32(equipId), ReqLevel: 50})
	}
	dDis.cbp.CrystalForLevelFunc = func(uint32) (item.Id, uint32, error) { return item.Id(4260005), 1, nil }
	pDis := buildCreateProcessor(t, hDis, dDis)
	_, err = pDis.Create(hDis.characterId, craft.Request{Mode: craft.ModeDisassemble, EquipItemId: equipId, SlotPos: slot})
	require.NoError(t, err)

	for _, e := range []*spyEmitter{d.em, dCrystal.em, dDis.em} {
		require.Len(t, e.calls, 1)
		for _, st := range e.calls[0].Steps {
			assert.NotEqual(t, saga.DestroyAllAssets, st.Action)
			// RecordCraftManifest is exempt: it is self-completing and
			// commits no mutation (it only carries data alongside the
			// saga), so it has nothing to compensate (task-285 Task 26a).
			if st.Action == saga.RecordCraftManifest {
				continue
			}
			assert.True(t, craft.CompensableActions[st.Action], "action %q is not in the compensable set", st.Action)
		}
	}
}

// TestNonZeroWorldAndChannelReachEmittedSaga guards against WorldId/ChannelId
// silently dropping to their zero value on the way to AwardMesosPayload:
// Request carries the calling channel's own scope (processor.go's Request
// doc), so a non-zero value driven in must survive verbatim onto the
// emitted saga's AwardMesos step.
func TestNonZeroWorldAndChannelReachEmittedSaga(t *testing.T) {
	r := eligibleRecipeFixture(t)
	h := newEligibleHarness(t)
	d := newDeps()
	d.rp.GetByIdFunc = func(item.Id) (recipe.Model, error) { return r, nil }
	p := buildCreateProcessor(t, h, d)

	txId, err := p.Create(h.characterId, craft.Request{
		Mode:         craft.ModeCreate,
		TargetItemId: r.Id(),
		WorldId:      world.Id(3),
		ChannelId:    channel.Id(7),
	})
	require.NoError(t, err)
	assert.NotZero(t, txId)

	require.Len(t, d.em.calls, 1)
	steps := d.em.calls[0].Steps
	var mesos saga.AwardMesosPayload
	found := false
	for _, st := range steps {
		if st.Action == saga.AwardMesos {
			mesos = st.Payload.(saga.AwardMesosPayload)
			found = true
			break
		}
	}
	require.True(t, found, "expected an AwardMesos step in the emitted saga")
	assert.Equal(t, world.Id(3), mesos.WorldId)
	assert.Equal(t, channel.Id(7), mesos.ChannelId)
}
