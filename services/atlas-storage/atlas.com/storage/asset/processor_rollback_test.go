package asset

import (
	"errors"
	"testing"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func storagesMigration(db *gorm.DB) error { return db.AutoMigrate(&StorageEntity{}) }

// GetOrCreateStorageId must join an enclosing transaction (re-entrancy), so a
// failing caller discards the storage row it created.
func TestGetOrCreateStorageId_JoinsCallerTransaction(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, storagesMigration)
	ctx := databasetest.TenantContext(uuid.New())
	l, _ := test.NewNullLogger()

	err := database.ExecuteTransaction(db.WithContext(ctx), func(tx *gorm.DB) error {
		id, err := NewProcessor(l, ctx, tx).GetOrCreateStorageId(0, 999)
		require.NoError(t, err)
		require.NotEqual(t, uuid.Nil, id)
		return errors.New("caller fails after storage creation")
	})
	require.Error(t, err)

	var count int64
	require.NoError(t, db.Table("storages").Count(&count).Error)
	require.Zero(t, count, "storage created inside the caller's tx must roll back with it")
}
