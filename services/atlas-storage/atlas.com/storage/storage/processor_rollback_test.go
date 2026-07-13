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
