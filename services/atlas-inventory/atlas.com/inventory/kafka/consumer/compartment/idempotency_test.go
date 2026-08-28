package compartment

import (
	"atlas-inventory/asset"
	inventorycompartment "atlas-inventory/compartment"
	assetmsg "atlas-inventory/kafka/message/asset"
	compartment2 "atlas-inventory/kafka/message/compartment"
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
	outboxlib "github.com/Chronicle20/atlas/libs/atlas-outbox"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// mesoMagnetTemplateId is the item from the task-208 report: a cash pet-equip
// that landed in the EQUIP compartment twice off one ACCEPT command.
const mesoMagnetTemplateId = uint32(1812000)

func TestMain(m *testing.M) {
	os.Setenv(string(assetmsg.EnvEventTopicStatus), string(assetmsg.EnvEventTopicStatus))
	os.Setenv(string(compartment2.EnvEventTopicStatus), string(compartment2.EnvEventTopicStatus))
	producertest.InstallNoop()

	mr, err := miniredis.Run()
	if err != nil {
		panic(err)
	}
	defer mr.Close()
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	inventorycompartment.InitReservationRegistry(rc)
	inventorycompartment.InitLockRegistry(rc)
	os.Exit(m.Run())
}

func acceptTestDB(t *testing.T, l logrus.FieldLogger) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxIdleConns(1)
	sqlDB.SetConnMaxLifetime(0)
	sqlDB.SetConnMaxIdleTime(0)

	database.RegisterTenantCallbacks(l, db)
	for _, migrate := range []database.Migrator{
		asset.Migration,
		inventorycompartment.Migration,
		outboxlib.Migration,
		database.IdempotencyMigration,
	} {
		require.NoError(t, migrate(db))
	}
	return db
}

// seedEquipCompartment gives the character an EQUIP compartment with room to
// spare, so a second accept would land in the next free slot rather than fail.
func seedEquipCompartment(t *testing.T, db *gorm.DB, ctx context.Context, characterId uint32) {
	t.Helper()
	require.NoError(t, db.WithContext(ctx).Create(&inventorycompartment.Entity{
		Id:            uuid.New(),
		CharacterId:   characterId,
		InventoryType: inventory.TypeValueEquip,
		Capacity:      24,
	}).Error)
}

func acceptCommand(transactionId uuid.UUID, characterId uint32) compartment2.Command[compartment2.AcceptCommandBody] {
	return compartment2.Command[compartment2.AcceptCommandBody]{
		TransactionId: transactionId,
		CharacterId:   characterId,
		InventoryType: byte(inventory.TypeValueEquip),
		Type:          compartment2.CommandAccept,
		Body: compartment2.AcceptCommandBody{
			TransactionId: transactionId,
			TemplateId:    mesoMagnetTemplateId,
			AssetData: assetmsg.AssetData{
				Quantity:    1,
				CashId:      8834539787427264243,
				CommodityId: 60100000,
				PurchaseBy:  characterId,
			},
		},
	}
}

func countAssets(t *testing.T, db *gorm.DB, ctx context.Context) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.WithContext(ctx).Model(&asset.Entity{}).Count(&n).Error)
	return n
}

// The task-208 regression: atlas-inventory consumed one ACCEPT command twice
// (offset redelivered during a group rebalance) and created two asset rows.
func TestAcceptCommandRedeliveryDoesNotDuplicateTheAsset(t *testing.T) {
	l, _ := test.NewNullLogger()
	db := acceptTestDB(t, l)
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	const characterId = uint32(12)
	seedEquipCompartment(t, db, ctx, characterId)

	cmd := acceptCommand(uuid.New(), characterId)
	handle := handleAcceptCommand(db)

	handle(l, ctx, cmd)
	require.Equal(t, int64(1), countAssets(t, db, ctx), "first delivery must create the asset")

	handle(l, ctx, cmd)
	require.Equal(t, int64(1), countAssets(t, db, ctx), "redelivery must not create a second asset")
}

// A redelivered RELEASE must not delete a second time. The first release frees
// the asset; without a guard the replay re-runs the whole release path.
func TestReleaseCommandRedeliveryIsAppliedOnce(t *testing.T) {
	l, _ := test.NewNullLogger()
	db := acceptTestDB(t, l)
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	const characterId = uint32(12)
	seedEquipCompartment(t, db, ctx, characterId)

	// Land one asset so there is something to release.
	handleAcceptCommand(db)(l, ctx, acceptCommand(uuid.New(), characterId))
	require.Equal(t, int64(1), countAssets(t, db, ctx))

	var seeded asset.Entity
	require.NoError(t, db.WithContext(ctx).First(&seeded).Error)

	releaseTid := uuid.New()
	cmd := compartment2.Command[compartment2.ReleaseCommandBody]{
		TransactionId: releaseTid,
		CharacterId:   characterId,
		InventoryType: byte(inventory.TypeValueEquip),
		Type:          compartment2.CommandRelease,
		Body: compartment2.ReleaseCommandBody{
			TransactionId: releaseTid,
			AssetId:       seeded.Id,
		},
	}

	handle := handleReleaseCommand(db)
	handle(l, ctx, cmd)
	require.Equal(t, int64(0), countAssets(t, db, ctx), "first release must free the asset")

	// The replay must be a no-op rather than a second trip through release.
	handle(l, ctx, cmd)
	require.Equal(t, int64(0), countAssets(t, db, ctx))

	var claims int64
	require.NoError(t, db.WithContext(ctx).Model(&database.IdempotencyEntity{}).
		Where("operation = ?", compartment2.CommandRelease).Count(&claims).Error)
	require.Equal(t, int64(1), claims, "the replay must not create a second claim")
}

// Distinct withdrawals must still both land — the guard keys on the command's
// identity, not merely on "an accept already happened".
func TestDistinctAcceptCommandsBothApply(t *testing.T) {
	l, _ := test.NewNullLogger()
	db := acceptTestDB(t, l)
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	require.NoError(t, err)
	ctx := tenant.WithContext(context.Background(), ten)

	const characterId = uint32(12)
	seedEquipCompartment(t, db, ctx, characterId)

	handle := handleAcceptCommand(db)
	handle(l, ctx, acceptCommand(uuid.New(), characterId))
	handle(l, ctx, acceptCommand(uuid.New(), characterId))

	require.Equal(t, int64(2), countAssets(t, db, ctx))
}
