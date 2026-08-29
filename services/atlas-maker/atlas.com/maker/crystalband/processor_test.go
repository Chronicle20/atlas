package crystalband_test

import (
	"atlas-maker/crystalband"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
)

func testLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.ErrorLevel)
	return l
}

// band mirrors one row of docs/tasks/task-285-maker-skill-crafting/
// reagent-derivation.md §5.4 — the derived Item.wz/Etc/0426.img table.
type band struct {
	crystalItemId item.Id
	minLevel      uint32
	maxLevel      uint32
}

// derivedBands is the 9-row table from the derivation, verbatim.
var derivedBands = []band{
	{4260000, 31, 50},
	{4260001, 51, 60},
	{4260002, 61, 70},
	{4260003, 71, 80},
	{4260004, 81, 90},
	{4260005, 91, 100},
	{4260006, 101, 110},
	{4260007, 111, 120},
	{4260008, 121, 200},
}

func seedDerivedBands(t *testing.T, db *gorm.DB, tenantId uuid.UUID) {
	t.Helper()
	for _, b := range derivedBands {
		m, err := crystalband.NewBuilder(tenantId).
			SetMinLevel(b.minLevel).
			SetMaxLevel(b.maxLevel).
			SetCrystalItemId(b.crystalItemId).
			SetCount(1).
			Build()
		require.NoError(t, err)
		require.NoError(t, crystalband.CreateCrystalBand(db, m))
	}
}

// TestCrystalForLevelAtEachBand is table-driven, one case per band from the
// derived table, asserting the exact crystal id and count.
func TestCrystalForLevelAtEachBand(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, crystalband.Migration)
	tenantId := uuid.New()
	ctx := databasetest.TenantContext(tenantId)
	seedDerivedBands(t, db, tenantId)

	p := crystalband.NewProcessor(testLogger(), ctx, db)

	for _, b := range derivedBands {
		b := b
		mid := b.minLevel + (b.maxLevel-b.minLevel)/2
		t.Run(fmt.Sprintf("band_%d", b.crystalItemId), func(t *testing.T) {
			itemId, count, err := p.CrystalForLevel(mid)
			require.NoError(t, err)
			assert.Equal(t, b.crystalItemId, itemId)
			assert.EqualValues(t, 1, count)
		})
	}
}

// TestCrystalForLevelAtBandBoundaries checks, for every boundary n in the
// derived table, that n-1, n and n+1 land in the expected bands. Off-by-one
// at a band edge is the defect this table's shape invites.
func TestCrystalForLevelAtBandBoundaries(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, crystalband.Migration)
	tenantId := uuid.New()
	ctx := databasetest.TenantContext(tenantId)
	seedDerivedBands(t, db, tenantId)

	p := crystalband.NewProcessor(testLogger(), ctx, db)

	for i, b := range derivedBands {
		b := b
		t.Run(fmt.Sprintf("min_boundary_%d", b.crystalItemId), func(t *testing.T) {
			itemId, _, err := p.CrystalForLevel(b.minLevel)
			require.NoError(t, err)
			assert.Equal(t, b.crystalItemId, itemId)

			if b.minLevel > 0 {
				belowId, _, belowErr := p.CrystalForLevel(b.minLevel - 1)
				if i == 0 {
					// b.minLevel-1 is below the lowest band entirely.
					assert.ErrorIs(t, belowErr, crystalband.ErrNotFound)
				} else {
					require.NoError(t, belowErr)
					assert.Equal(t, derivedBands[i-1].crystalItemId, belowId)
				}
			}
		})
		t.Run(fmt.Sprintf("max_boundary_%d", b.crystalItemId), func(t *testing.T) {
			itemId, _, err := p.CrystalForLevel(b.maxLevel)
			require.NoError(t, err)
			assert.Equal(t, b.crystalItemId, itemId)

			aboveId, _, aboveErr := p.CrystalForLevel(b.maxLevel + 1)
			if i == len(derivedBands)-1 {
				// b.maxLevel+1 is above the highest band entirely.
				assert.ErrorIs(t, aboveErr, crystalband.ErrNotFound)
			} else {
				require.NoError(t, aboveErr)
				assert.Equal(t, derivedBands[i+1].crystalItemId, aboveId)
			}
		})
	}
}

// TestCrystalForLevelBelowLowestBand asserts no match / ErrNotFound for an
// equip whose reqLevel falls outside every seeded band, at both ends. This is
// a product decision, NOT client-derived behaviour: the derivation
// (reagent-derivation.md §5.5) proves the monster-crystal band vector is
// write-only on the client in both gms_v72 and gms_v83 — nothing ever reads
// it back, so the client defines no clamp or fallback to take. Rejecting the
// craft with ErrNotFound is Atlas's own ruling.
func TestCrystalForLevelBelowLowestBand(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, crystalband.Migration)
	tenantId := uuid.New()
	ctx := databasetest.TenantContext(tenantId)
	seedDerivedBands(t, db, tenantId)

	p := crystalband.NewProcessor(testLogger(), ctx, db)

	_, _, err := p.CrystalForLevel(1)
	require.Error(t, err)
	assert.ErrorIs(t, err, crystalband.ErrNotFound)

	_, _, err = p.CrystalForLevel(30)
	require.Error(t, err)
	assert.ErrorIs(t, err, crystalband.ErrNotFound)

	// The symmetric above-200 case: also a product decision, not a client
	// behaviour, for the same reason as above.
	_, _, err = p.CrystalForLevel(201)
	require.Error(t, err)
	assert.ErrorIs(t, err, crystalband.ErrNotFound)
}

// TestCrystalForLevelIsTenantScoped seeds differing bands for two tenants and
// asserts each reads its own.
func TestCrystalForLevelIsTenantScoped(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, crystalband.Migration)
	tenantA := uuid.New()
	tenantB := uuid.New()

	mA, err := crystalband.NewBuilder(tenantA).
		SetMinLevel(31).SetMaxLevel(50).SetCrystalItemId(item.Id(4260000)).SetCount(1).
		Build()
	require.NoError(t, err)
	require.NoError(t, crystalband.CreateCrystalBand(db, mA))

	mB, err := crystalband.NewBuilder(tenantB).
		SetMinLevel(31).SetMaxLevel(50).SetCrystalItemId(item.Id(9999999)).SetCount(7).
		Build()
	require.NoError(t, err)
	require.NoError(t, crystalband.CreateCrystalBand(db, mB))

	itemIdA, countA, err := crystalband.NewProcessor(testLogger(), databasetest.TenantContext(tenantA), db).CrystalForLevel(40)
	require.NoError(t, err)
	assert.Equal(t, item.Id(4260000), itemIdA)
	assert.EqualValues(t, 1, countA)

	itemIdB, countB, err := crystalband.NewProcessor(testLogger(), databasetest.TenantContext(tenantB), db).CrystalForLevel(40)
	require.NoError(t, err)
	assert.Equal(t, item.Id(9999999), itemIdB)
	assert.EqualValues(t, 7, countB)
}

// TestCrystalForLevelNotFoundAcrossTenants asserts the not-found path is also
// what a tenant sees for a level that only matches a band seeded under
// another tenant, mirroring reagent's TestGetByItemIdNotFoundAcrossTenants.
func TestCrystalForLevelNotFoundAcrossTenants(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, crystalband.Migration)
	tenantA := uuid.New()
	tenantB := uuid.New()

	mA, err := crystalband.NewBuilder(tenantA).
		SetMinLevel(31).SetMaxLevel(50).SetCrystalItemId(item.Id(4260000)).SetCount(1).
		Build()
	require.NoError(t, err)
	require.NoError(t, crystalband.CreateCrystalBand(db, mA))

	_, _, err = crystalband.NewProcessor(testLogger(), databasetest.TenantContext(tenantB), db).CrystalForLevel(40)
	require.Error(t, err)
	assert.ErrorIs(t, err, crystalband.ErrNotFound)
}
