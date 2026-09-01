package craft_test

import (
	"atlas-maker/compartment"
	"atlas-maker/craft"
	"atlas-maker/data/itemmake"
	"testing"

	compartmentmock "atlas-maker/compartment/mock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// buildSnapshot is plan_test.go's fixture helper: a compartment.Processor
// mock returning etc for every inventory type is sufficient since the plan
// only cares about per-item slot holdings, not which compartment they came
// from beyond what inventory.TypeFromItemId already derives.
func buildSnapshot(t *testing.T, etc compartment.Model) craft.Snapshot {
	t.Helper()
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
	return snap
}

func TestPlanResolvesMaterialAcrossMultipleSlots(t *testing.T) {
	etc := compartment.NewBuilder(inventory.TypeValueETC).
		AddAsset(compartment.NewAssetModel(item.Id(4011001), 3, 1)).
		AddAsset(compartment.NewAssetModel(item.Id(4011001), 2, 7)).
		Build()
	snap := buildSnapshot(t, etc)

	r := buildRecipe(t, itemmake.RestModel{
		Id:     1082002,
		Group:  1,
		Recipe: []itemmake.MaterialRestModel{{ItemId: 4011001, Count: 5}},
	})

	plan := craft.BuildCreatePlan(snap, r, nil, false)

	require.Len(t, plan.Consumptions, 2)
	assert.Equal(t, int16(1), plan.Consumptions[0].Slot)
	assert.EqualValues(t, 3, plan.Consumptions[0].Quantity)
	assert.Equal(t, item.Id(4011001), plan.Consumptions[0].TemplateId)
	assert.Equal(t, int16(7), plan.Consumptions[1].Slot)
	assert.EqualValues(t, 2, plan.Consumptions[1].Quantity)
	assert.Equal(t, item.Id(4011001), plan.Consumptions[1].TemplateId)

	var total uint32
	for _, c := range plan.Consumptions {
		total += c.Quantity
	}
	assert.EqualValues(t, 5, total)
}

func TestPlanConsumesFromLowestSlotFirst(t *testing.T) {
	etc := compartment.NewBuilder(inventory.TypeValueETC).
		AddAsset(compartment.NewAssetModel(item.Id(4011001), 1, 7)).
		AddAsset(compartment.NewAssetModel(item.Id(4011001), 1, 1)).
		AddAsset(compartment.NewAssetModel(item.Id(4011001), 1, 4)).
		Build()
	snap := buildSnapshot(t, etc)

	r := buildRecipe(t, itemmake.RestModel{
		Id:     1082003,
		Group:  1,
		Recipe: []itemmake.MaterialRestModel{{ItemId: 4011001, Count: 3}},
	})

	plan := craft.BuildCreatePlan(snap, r, nil, false)

	require.Len(t, plan.Consumptions, 3)
	assert.Equal(t, int16(1), plan.Consumptions[0].Slot)
	assert.Equal(t, int16(4), plan.Consumptions[1].Slot)
	assert.Equal(t, int16(7), plan.Consumptions[2].Slot)
}

func TestPlanDropsUnheldReagents(t *testing.T) {
	etc := compartment.NewBuilder(inventory.TypeValueETC).
		AddAsset(compartment.NewAssetModel(item.Id(4260000), 1, 1)).
		AddAsset(compartment.NewAssetModel(item.Id(4260001), 1, 2)).
		Build()
	snap := buildSnapshot(t, etc)

	r := buildRecipe(t, itemmake.RestModel{Id: 1082004, Group: 1})

	gems := []item.Id{4260000, 4260001, 4260002}
	plan := craft.BuildCreatePlan(snap, r, gems, false)

	require.Len(t, plan.Consumptions, 2)
	held := map[item.Id]bool{}
	for _, c := range plan.Consumptions {
		held[c.TemplateId] = true
	}
	assert.True(t, held[item.Id(4260000)])
	assert.True(t, held[item.Id(4260001)])
	assert.False(t, held[item.Id(4260002)])
}

func TestPlanIncludesCatalystWhenFlagSetAndHeld(t *testing.T) {
	etc := compartment.NewBuilder(inventory.TypeValueETC).
		AddAsset(compartment.NewAssetModel(item.Id(4130000), 1, 3)).
		Build()
	snap := buildSnapshot(t, etc)

	r := buildRecipe(t, itemmake.RestModel{Id: 1082005, Group: 1, Catalyst: 4130000})

	plan := craft.BuildCreatePlan(snap, r, nil, true)

	require.Len(t, plan.Consumptions, 1)
	assert.Equal(t, item.Id(4130000), plan.Consumptions[0].TemplateId)
	assert.EqualValues(t, 1, plan.Consumptions[0].Quantity)
}

func TestPlanOmitsCatalystWhenNotHeld(t *testing.T) {
	etc := compartment.NewBuilder(inventory.TypeValueETC).Build()
	snap := buildSnapshot(t, etc)

	r := buildRecipe(t, itemmake.RestModel{Id: 1082006, Group: 1, Catalyst: 4130000})

	plan := craft.BuildCreatePlan(snap, r, nil, true)

	assert.Empty(t, plan.Consumptions)
}

func TestPlanNeverTrustsClientQuantities(t *testing.T) {
	etc := compartment.NewBuilder(inventory.TypeValueETC).
		AddAsset(compartment.NewAssetModel(item.Id(4260000), 1, 1)).
		Build()
	snap := buildSnapshot(t, etc)

	r := buildRecipe(t, itemmake.RestModel{Id: 1082007, Group: 1})

	// 4260099 is not held at all; 4260000 is held once but named twice.
	gems := []item.Id{item.Id(4260099), item.Id(4260000), item.Id(4260000)}
	plan := craft.BuildCreatePlan(snap, r, gems, false)

	require.Len(t, plan.Consumptions, 1)
	assert.Equal(t, item.Id(4260000), plan.Consumptions[0].TemplateId)
	assert.EqualValues(t, 1, plan.Consumptions[0].Quantity)
}

// TestBuildCreatePlanTagsRoles asserts materials, gems and the catalyst each
// carry their own Role, and that the resolved order (materials -> gems ->
// catalyst) is unchanged from before Role existed (task-285 Task 26a).
func TestBuildCreatePlanTagsRoles(t *testing.T) {
	etc := compartment.NewBuilder(inventory.TypeValueETC).
		AddAsset(compartment.NewAssetModel(item.Id(4011001), 1, 5)).
		AddAsset(compartment.NewAssetModel(item.Id(4260000), 1, 1)).
		AddAsset(compartment.NewAssetModel(item.Id(4130000), 1, 3)).
		Build()
	snap := buildSnapshot(t, etc)

	r := buildRecipe(t, itemmake.RestModel{
		Id:       1082008,
		Group:    1,
		Recipe:   []itemmake.MaterialRestModel{{ItemId: 4011001, Count: 5}},
		Catalyst: 4130000,
	})

	gems := []item.Id{4260000}
	plan := craft.BuildCreatePlan(snap, r, gems, true)

	require.Len(t, plan.Consumptions, 3)
	assert.Equal(t, item.Id(4011001), plan.Consumptions[0].TemplateId)
	assert.Equal(t, craft.RoleMaterial, plan.Consumptions[0].Role)
	assert.Equal(t, item.Id(4260000), plan.Consumptions[1].TemplateId)
	assert.Equal(t, craft.RoleGem, plan.Consumptions[1].Role)
	assert.Equal(t, item.Id(4130000), plan.Consumptions[2].TemplateId)
	assert.Equal(t, craft.RoleCatalyst, plan.Consumptions[2].Role)
}

// TestBuildCrystalPlanTagsMaterialRole covers mode 3's leftover consumption:
// it carries RoleMaterial, matching what the manifest's Materials aggregation
// would expect if mode 3's arm ever populated that field (it does not --
// derivation §5 -- but the Plan's own tagging stays consistent regardless).
func TestBuildCrystalPlanTagsMaterialRole(t *testing.T) {
	etc := compartment.NewBuilder(inventory.TypeValueETC).
		AddAsset(compartment.NewAssetModel(item.Id(4020000), 150, 1)).
		Build()
	snap := buildSnapshot(t, etc)

	plan := craft.BuildCrystalPlan(snap, item.Id(4020000))

	require.Len(t, plan.Consumptions, 1)
	assert.Equal(t, craft.RoleMaterial, plan.Consumptions[0].Role)
	assert.EqualValues(t, craft.LeftoverConsumeQuantity, plan.Consumptions[0].Quantity)
}
