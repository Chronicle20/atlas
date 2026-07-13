package storage

import (
	"testing"

	"atlas-storage/asset"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

// ExpireAndEmit deletes the expired asset then creates its replacement
// (class A-shaped: two writes that must move together). Failing the
// replacement create must restore the deleted asset.
func TestExpireAndEmit_RollsBackDeleteWhenReplacementCreateFails(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration, asset.Migration)
	tid := uuid.New()
	ctx := databasetest.TenantContext(tid)
	l, _ := test.NewNullLogger()

	s, err := Create(l, db.WithContext(ctx), tid)(world.Id(0), 5001)
	require.NoError(t, err)
	a, err := asset.Create(l, db.WithContext(ctx), tid)(asset.NewBuilder(s.Id(), 4000000).SetSlot(0).SetQuantity(1).Build())
	require.NoError(t, err)

	databasetest.FailWritesOn(t, db, "storage_assets", databasetest.WriteCreate)

	p := NewProcessor(l, ctx, db)
	err = p.ExpireAndEmit(uuid.New(), world.Id(0), 5001, a.Id(), false, 4000001, "expired")
	require.Error(t, err, "replacement-create failure must surface as an error")

	var assets int64
	require.NoError(t, db.Table("storage_assets").Count(&assets).Error)
	require.EqualValues(t, 1, assets, "the expired asset's delete must roll back")
}

// MergeAndSort is a loop of quantity-updates + deletes + re-slotting
// (class B). Failing a later delete must roll back the earlier quantity
// updates, restoring the pre-merge stacks.
func TestMergeAndSort_RollsBackQuantityUpdatesWhenDeleteFails(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, Migration, asset.Migration)
	tid := uuid.New()
	ctx := databasetest.TenantContext(tid)
	l, _ := test.NewNullLogger()

	s, err := Create(l, db.WithContext(ctx), tid)(world.Id(0), 5002)
	require.NoError(t, err)
	// Two mergeable stacks of the same consumable: 30 + 30 with slotMax 100
	// (the atlas-data lookup fails in tests, falling back to 100) merge into
	// one stack of 60, deleting the second row.
	_, err = asset.Create(l, db.WithContext(ctx), tid)(asset.NewBuilder(s.Id(), 2000000).SetSlot(0).SetQuantity(30).Build())
	require.NoError(t, err)
	_, err = asset.Create(l, db.WithContext(ctx), tid)(asset.NewBuilder(s.Id(), 2000000).SetSlot(1).SetQuantity(30).Build())
	require.NoError(t, err)

	databasetest.FailWritesOn(t, db, "storage_assets", databasetest.WriteDelete)

	p := NewProcessor(l, ctx, db)
	require.Error(t, p.MergeAndSort(world.Id(0), 5002))

	var quantities []uint32
	require.NoError(t, db.Table("storage_assets").Order("slot").Pluck("quantity", &quantities).Error)
	require.Equal(t, []uint32{30, 30}, quantities, "quantity update must roll back with the failed delete")
}
