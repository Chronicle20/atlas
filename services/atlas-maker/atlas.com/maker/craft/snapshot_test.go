package craft_test

import (
	"atlas-maker/character"
	"atlas-maker/compartment"
	"atlas-maker/craft"
	"atlas-maker/data/itemmake"
	"atlas-maker/skill"
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

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	skillconst "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

// TestSnapshotSumsQuantityAcrossSlots proves Held sums quantity across every
// slot holding an item, and that Slots reports each slot's individual
// holding in slot order -- the per-slot detail Task 23's consumption plan
// consumes.
func TestSnapshotSumsQuantityAcrossSlots(t *testing.T) {
	etc := compartment.NewBuilder(inventory.TypeValueETC).
		AddAsset(compartment.NewAssetModel(item.Id(4011001), 4, 7)).
		AddAsset(compartment.NewAssetModel(item.Id(4011001), 3, 1)).
		Build()

	cp := &compartmentmock.ProcessorMock{
		GetByTypeFunc: func(_ uint32, invType inventory.Type) (compartment.Model, error) {
			if invType == inventory.TypeValueETC {
				return etc, nil
			}
			return compartment.NewBuilder(invType).Build(), nil
		},
	}

	snap, err := craft.NewSnapshot(cp, 1001)
	require.NoError(t, err)

	assert.EqualValues(t, 7, snap.Held(item.Id(4011001)))

	slots := snap.Slots(item.Id(4011001))
	require.Len(t, slots, 2)
	assert.Equal(t, craft.SlotHolding{Slot: 1, Quantity: 3}, slots[0])
	assert.Equal(t, craft.SlotHolding{Slot: 7, Quantity: 4}, slots[1])
}

// TestSnapshotReadsEachTypeExactlyOnce is the NFR (design §4.2.2) made
// executable: a snapshot build issues exactly one compartment read per
// inventory type, and evaluating multiple recipes against the built
// snapshot issues no further reads.
func TestSnapshotReadsEachTypeExactlyOnce(t *testing.T) {
	seen := map[inventory.Type]int{}
	cp := &compartmentmock.ProcessorMock{
		GetByTypeFunc: func(_ uint32, invType inventory.Type) (compartment.Model, error) {
			seen[invType]++
			return compartment.NewBuilder(invType).Build(), nil
		},
	}

	snap, err := craft.NewSnapshot(cp, 1001)
	require.NoError(t, err)

	require.Len(t, seen, 3)
	assert.Equal(t, 1, seen[inventory.TypeValueEquip])
	assert.Equal(t, 1, seen[inventory.TypeValueUse])
	assert.Equal(t, 1, seen[inventory.TypeValueETC])

	// Evaluating recipes against the already-built snapshot must not issue
	// any further compartment reads. cp (the same mock that built snap) is
	// wired into the Processor as its compartment dependency, so a
	// regression that re-reads compartments per recipe would show up in
	// seen.
	cp.CanAccommodateFunc = func(uint32, []compartment.AccommodationItem) (bool, error) { return true, nil }
	cp2 := &charactermock.ProcessorMock{
		GetByIdFunc: func(uint32) (character.Model, error) { return buildCharacter(t, 40, 0), nil },
	}
	sp := &skillmock.ProcessorMock{
		GetByCharacterIdFunc: func(uint32) ([]skill.Model, error) {
			return []skill.Model{buildSkill(t, uint32(skillconst.BeginnerMaker), 1)}, nil
		},
	}
	qp := &questmock.ProcessorMock{}
	rp := &recipemock.ProcessorMock{}
	rgp := &reagentmock.ProcessorMock{}
	cbp := &crystalbandmock.ProcessorMock{}
	eqp := &equipmentmock.ProcessorMock{}
	p := craft.NewProcessor(testLogger(), testContext(t), cp2, sp, cp, qp, rp, rgp, cbp, eqp, noopEmitter{})

	recipes := []itemmake.RestModel{
		{Id: 3000001, Group: 1, ReqLevel: 1, ReqSkillLevel: 1, ItemNum: 1},
		{Id: 3000002, Group: 1, ReqLevel: 1, ReqSkillLevel: 1, ItemNum: 1},
	}
	for _, rm := range recipes {
		r := buildRecipe(t, rm)
		_, err := p.Evaluate(1001, snap, r)
		require.NoError(t, err)
	}

	require.Len(t, seen, 3)
	assert.Equal(t, 1, seen[inventory.TypeValueEquip])
	assert.Equal(t, 1, seen[inventory.TypeValueUse])
	assert.Equal(t, 1, seen[inventory.TypeValueETC])
}
