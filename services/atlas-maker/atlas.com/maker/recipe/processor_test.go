package recipe_test

import (
	"atlas-maker/data/itemmake"
	itemmakemock "atlas-maker/data/itemmake/mock"
	"atlas-maker/recipe"
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	return l
}

func newTestTenant(t *testing.T) tenant.Model {
	te, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	return te
}

func mustExtract(t *testing.T, rm itemmake.RestModel) itemmake.Model {
	t.Helper()
	m, err := itemmake.Extract(rm)
	require.NoError(t, err)
	return m
}

// sixGroupFixture reproduces the Task 3 archive fixture as decoded
// itemmake.Model values: one entry per top-level group digit (0, 1, 2, 4,
// 8, 16), with 4260000 (group 0) pairing leftover 4000000 as its sole
// material and 1082002 (group 1) carrying every optional scalar and list.
func sixGroupFixture(t *testing.T) []itemmake.Model {
	t.Helper()
	return []itemmake.Model{
		mustExtract(t, itemmake.RestModel{
			Id:      4260000,
			Group:   0,
			ItemNum: 1,
			Recipe:  []itemmake.MaterialRestModel{{ItemId: 4000000, Count: 1}},
			RandomReward: []itemmake.RewardRestModel{
				{ItemId: 4260000, ItemNum: 1, Prob: 70},
				{ItemId: 4260001, ItemNum: 1, Prob: 25},
				{ItemId: 4260002, ItemNum: 1, Prob: 5},
			},
		}),
		mustExtract(t, itemmake.RestModel{
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
			Recipe: []itemmake.MaterialRestModel{
				{ItemId: 4011001, Count: 5},
				{ItemId: 4011002, Count: 3},
				{ItemId: 4021007, Count: 1},
			},
			ReqQuest: []itemmake.QuestReqRestModel{{QuestId: 21614, State: 3}},
		}),
		mustExtract(t, itemmake.RestModel{
			Id:       2020000,
			Group:    2,
			ReqLevel: 10,
			ItemNum:  3,
			Meso:     500,
			Recipe:   []itemmake.MaterialRestModel{{ItemId: 4000001, Count: 2}},
		}),
		mustExtract(t, itemmake.RestModel{
			Id:       4030000,
			Group:    4,
			ReqLevel: 15,
			ItemNum:  1,
			Meso:     800,
		}),
		mustExtract(t, itemmake.RestModel{
			Id:       8000000,
			Group:    8,
			ReqLevel: 20,
			ItemNum:  1,
			Meso:     900,
		}),
		mustExtract(t, itemmake.RestModel{
			Id:       16000000,
			Group:    16,
			ReqLevel: 25,
			ItemNum:  1,
			Meso:     1000,
		}),
	}
}

func newProcessor(ctx context.Context, ms []itemmake.Model) recipe.Processor {
	return newCountingProcessor(ctx, ms, nil)
}

func newCountingProcessor(ctx context.Context, ms []itemmake.Model, calls *int) recipe.Processor {
	im := &itemmakemock.ProcessorMock{
		GetAllFunc: func() ([]itemmake.Model, error) {
			if calls != nil {
				*calls++
			}
			return ms, nil
		},
	}
	return recipe.NewProcessor(testLogger(), ctx, im)
}

func TestGetByIdReturnsRecipe(t *testing.T) {
	te := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), te)
	p := newProcessor(ctx, sixGroupFixture(t))

	m, err := p.GetById(item.Id(1082002))
	require.NoError(t, err)

	assert.Equal(t, item.Id(1082002), m.Id())
	assert.EqualValues(t, 1, m.Group())
	assert.EqualValues(t, 30, m.ReqLevel())
	assert.EqualValues(t, 2, m.ReqSkillLevel())
	assert.EqualValues(t, 1, m.ItemNum())
	assert.EqualValues(t, 7, m.Tuc())
	assert.EqualValues(t, 1200, m.Meso())
	assert.Equal(t, item.Id(4130000), m.Catalyst())
	assert.Equal(t, item.Id(4000021), m.ReqItem())
	assert.Equal(t, item.Id(1002419), m.ReqEquip())

	require.Len(t, m.Materials(), 3)
	assert.Equal(t, recipe.Material{ItemId: item.Id(4011001), Count: 5}, m.Materials()[0])
	assert.Equal(t, recipe.Material{ItemId: item.Id(4011002), Count: 3}, m.Materials()[1])
	assert.Equal(t, recipe.Material{ItemId: item.Id(4021007), Count: 1}, m.Materials()[2])

	assert.Empty(t, m.RandomRewards())

	require.Len(t, m.QuestRequirements(), 1)
	assert.Equal(t, recipe.QuestRequirement{QuestId: 21614, State: 3}, m.QuestRequirements()[0])
}

func TestGetByIdNotFound(t *testing.T) {
	te := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), te)
	p := newProcessor(ctx, sixGroupFixture(t))

	_, err := p.GetById(item.Id(9999999))
	require.Error(t, err)
	assert.True(t, errors.Is(err, recipe.ErrNotFound))
}

func TestGetByLeftoverResolvesGroupZero(t *testing.T) {
	te := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), te)
	p := newProcessor(ctx, sixGroupFixture(t))

	m, err := p.GetByLeftover(item.Id(4000000))
	require.NoError(t, err)
	assert.Equal(t, item.Id(4260000), m.Id())
	assert.EqualValues(t, 0, m.Group())
}

// TestGetByLeftoverIgnoresNonZeroGroups is the C-6 requirement made
// executable: a non-zero-group recipe that happens to list the same
// leftover as a material must never satisfy GetByLeftover, and removing
// the group-0 entry must surface not-found rather than falling through to
// the group-1 recipe.
func TestGetByLeftoverIgnoresNonZeroGroups(t *testing.T) {
	te := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), te)

	groupZero := mustExtract(t, itemmake.RestModel{
		Id:      4260000,
		Group:   0,
		ItemNum: 1,
		Recipe:  []itemmake.MaterialRestModel{{ItemId: 4000000, Count: 1}},
	})
	groupOneCollision := mustExtract(t, itemmake.RestModel{
		Id:       1382005,
		Group:    1,
		ItemNum:  1,
		ReqEquip: 1382000,
		Recipe:   []itemmake.MaterialRestModel{{ItemId: 4000000, Count: 5}},
	})

	p := newProcessor(ctx, []itemmake.Model{groupZero, groupOneCollision})

	m, err := p.GetByLeftover(item.Id(4000000))
	require.NoError(t, err)
	assert.Equal(t, item.Id(4260000), m.Id(), "GetByLeftover must resolve the group-0 recipe, not the group-1 collision")

	// Remove the group-0 entry and rebuild under a fresh tenant so the
	// existing cache entry cannot leak the prior result.
	te2 := newTestTenant(t)
	ctx2 := tenant.WithContext(context.Background(), te2)
	p2 := newProcessor(ctx2, []itemmake.Model{groupOneCollision})

	_, err = p2.GetByLeftover(item.Id(4000000))
	require.Error(t, err)
	assert.True(t, errors.Is(err, recipe.ErrNoCrystalMapping), "with no group-0 entry, GetByLeftover must not fall through to the group-1 recipe")
}

func TestGetByLeftoverNotFound(t *testing.T) {
	te := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), te)
	p := newProcessor(ctx, sixGroupFixture(t))

	_, err := p.GetByLeftover(item.Id(1))
	require.Error(t, err)
	assert.True(t, errors.Is(err, recipe.ErrNoCrystalMapping))
}

func TestIndexesAreTenantScoped(t *testing.T) {
	teA := newTestTenant(t)
	teB := newTestTenant(t)
	ctxA := tenant.WithContext(context.Background(), teA)
	ctxB := tenant.WithContext(context.Background(), teB)

	onlyInA := mustExtract(t, itemmake.RestModel{Id: 1111111, Group: 0, ItemNum: 1})
	onlyInB := mustExtract(t, itemmake.RestModel{Id: 2222222, Group: 0, ItemNum: 1})

	pA := newProcessor(ctxA, []itemmake.Model{onlyInA})
	pB := newProcessor(ctxB, []itemmake.Model{onlyInB})

	_, err := pA.GetById(item.Id(1111111))
	require.NoError(t, err)
	_, err = pA.GetById(item.Id(2222222))
	require.Error(t, err, "tenant A must not see tenant B's recipe")

	_, err = pB.GetById(item.Id(2222222))
	require.NoError(t, err)
	_, err = pB.GetById(item.Id(1111111))
	require.Error(t, err, "tenant B must not see tenant A's recipe")
}

func TestIndexIsBuiltOnceUntilInvalidated(t *testing.T) {
	te := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), te)

	calls := 0
	p := newCountingProcessor(ctx, sixGroupFixture(t), &calls)

	_, err := p.GetById(item.Id(1082002))
	require.NoError(t, err)
	_, err = p.GetByLeftover(item.Id(4000000))
	require.NoError(t, err)
	_, err = p.GetAll()
	require.NoError(t, err)
	assert.Equal(t, 1, calls, "the upstream catalog must be read once for repeated lookups")

	recipe.Invalidate(te.Id())

	_, err = p.GetById(item.Id(1082002))
	require.NoError(t, err)
	assert.Equal(t, 2, calls, "invalidation must force a rebuild on the next lookup")
}
