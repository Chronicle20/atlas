package crystalband_test

import (
	"atlas-maker/crystalband"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

func TestBuilderRoundTrip(t *testing.T) {
	tenantId := uuid.New()

	m, err := crystalband.NewBuilder(tenantId).
		SetMinLevel(31).
		SetMaxLevel(50).
		SetCrystalItemId(item.Id(4260000)).
		SetCount(1).
		Build()
	require.NoError(t, err)

	assert.Equal(t, tenantId, m.TenantId())
	assert.EqualValues(t, 31, m.MinLevel())
	assert.EqualValues(t, 50, m.MaxLevel())
	assert.Equal(t, item.Id(4260000), m.CrystalItemId())
	assert.EqualValues(t, 1, m.Count())
}

func TestBuilderRejectsMissingIdentity(t *testing.T) {
	t.Run("NilTenant", func(t *testing.T) {
		_, err := crystalband.NewBuilder(uuid.Nil).
			SetMinLevel(31).SetMaxLevel(50).SetCrystalItemId(item.Id(4260000)).SetCount(1).
			Build()
		assert.Error(t, err)
	})
	t.Run("ZeroCrystalItemId", func(t *testing.T) {
		_, err := crystalband.NewBuilder(uuid.New()).
			SetMinLevel(31).SetMaxLevel(50).SetCrystalItemId(item.Id(0)).SetCount(1).
			Build()
		assert.Error(t, err)
	})
	t.Run("ZeroCount", func(t *testing.T) {
		_, err := crystalband.NewBuilder(uuid.New()).
			SetMinLevel(31).SetMaxLevel(50).SetCrystalItemId(item.Id(4260000)).SetCount(0).
			Build()
		assert.Error(t, err)
	})
}

// TestBuilderRejectsInvertedRange guards the invariant CrystalForLevel's
// linear scan of Model.Contains depends on: a band's max cannot be below its
// min.
func TestBuilderRejectsInvertedRange(t *testing.T) {
	_, err := crystalband.NewBuilder(uuid.New()).
		SetMinLevel(50).
		SetMaxLevel(31).
		SetCrystalItemId(item.Id(4260000)).
		SetCount(1).
		Build()
	assert.Error(t, err)
}

// TestModelContainsIsInclusiveAtBothEnds pins the boundary semantics the
// derivation (reagent-derivation.md §5.4) established: bands are inclusive on
// both ends.
func TestModelContainsIsInclusiveAtBothEnds(t *testing.T) {
	m, err := crystalband.NewBuilder(uuid.New()).
		SetMinLevel(31).
		SetMaxLevel(50).
		SetCrystalItemId(item.Id(4260000)).
		SetCount(1).
		Build()
	require.NoError(t, err)

	assert.True(t, m.Contains(31))
	assert.True(t, m.Contains(40))
	assert.True(t, m.Contains(50))
	assert.False(t, m.Contains(30))
	assert.False(t, m.Contains(51))
}
