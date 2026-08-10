package storage

import (
	"atlas-storage/asset"
	"atlas-storage/kafka/message"
	"atlas-storage/kafka/message/compartment"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	assetConstants "github.com/Chronicle20/atlas/libs/atlas-constants/asset"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
)

// task-208: Kafka delivery is at-least-once, so a redelivered ACCEPT must not
// deposit a second copy into storage.
func TestAcceptOnceAndEmit_RedeliveryDoesNotDuplicate(t *testing.T) {
	producertest.InstallNoop()

	db := databasetest.NewInMemoryTenantDB(t, Migration, asset.Migration, database.IdempotencyMigration)
	tid := uuid.New()
	ctx := databasetest.TenantContext(tid)
	l, _ := test.NewNullLogger()

	const accountId = uint32(5001)
	body := compartment.AcceptCommandBody{
		TransactionId: uuid.New(),
		TemplateId:    1812000,
		AssetData:     message.AssetData{Quantity: 1},
	}

	p := NewProcessor(l, ctx, db)
	require.NoError(t, p.AcceptOnceAndEmit(world.Id(0), accountId, 12, body))

	var assets int64
	require.NoError(t, db.Table("storage_assets").Count(&assets).Error)
	require.EqualValues(t, 1, assets, "first delivery must deposit the asset")

	require.NoError(t, p.AcceptOnceAndEmit(world.Id(0), accountId, 12, body),
		"a duplicate must be swallowed, not surfaced as an error")

	require.NoError(t, db.Table("storage_assets").Count(&assets).Error)
	require.EqualValues(t, 1, assets, "redelivery must not deposit a second asset")
}

// A distinct deposit shares neither transaction nor key and must still land.
func TestAcceptOnceAndEmit_DistinctCommandsBothApply(t *testing.T) {
	producertest.InstallNoop()

	db := databasetest.NewInMemoryTenantDB(t, Migration, asset.Migration, database.IdempotencyMigration)
	tid := uuid.New()
	ctx := databasetest.TenantContext(tid)
	l, _ := test.NewNullLogger()

	p := NewProcessor(l, ctx, db)
	for i := 0; i < 2; i++ {
		require.NoError(t, p.AcceptOnceAndEmit(world.Id(0), 5001, 12, compartment.AcceptCommandBody{
			TransactionId: uuid.New(),
			TemplateId:    1812000,
			AssetData:     message.AssetData{Quantity: 1},
		}))
	}

	var assets int64
	require.NoError(t, db.Table("storage_assets").Count(&assets).Error)
	require.EqualValues(t, 2, assets)
}

// A redelivered RELEASE must not run the release path twice. The source asset
// is read inside the claim, so the replay short-circuits on the claim rather
// than on a failed lookup.
func TestReleaseOnceAndEmit_RedeliveryIsAppliedOnce(t *testing.T) {
	producertest.InstallNoop()

	db := databasetest.NewInMemoryTenantDB(t, Migration, asset.Migration, database.IdempotencyMigration)
	tid := uuid.New()
	ctx := databasetest.TenantContext(tid)
	l, _ := test.NewNullLogger()

	const accountId = uint32(5001)
	s, err := Create(l, db.WithContext(ctx), tid)(world.Id(0), accountId)
	require.NoError(t, err)
	a, err := asset.Create(l, db.WithContext(ctx), tid)(
		asset.NewBuilder(s.Id(), 1812000).SetSlot(0).SetQuantity(1).Build())
	require.NoError(t, err)

	body := compartment.ReleaseCommandBody{
		TransactionId: uuid.New(),
		AssetId:       assetConstants.Id(a.Id()),
	}

	p := NewProcessor(l, ctx, db)
	require.NoError(t, p.ReleaseOnceAndEmit(world.Id(0), accountId, 12, body))

	var assets int64
	require.NoError(t, db.WithContext(ctx).Model(&asset.Entity{}).Count(&assets).Error)
	require.EqualValues(t, 0, assets, "first release must free the asset")

	require.NoError(t, p.ReleaseOnceAndEmit(world.Id(0), accountId, 12, body),
		"a duplicate release must be swallowed, not surfaced as an error")

	var claims int64
	require.NoError(t, db.Model(&database.IdempotencyEntity{}).
		Where("operation = ?", compartment.CommandRelease).Count(&claims).Error)
	require.EqualValues(t, 1, claims, "the replay must not create a second claim")
}
