package database_test

import (
	"errors"
	"testing"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type txEntity struct {
	Id       uint32    `gorm:"primaryKey;autoIncrement"`
	TenantId uuid.UUID `gorm:"type:uuid;not null"`
	Name     string    `gorm:"not null"`
}

func (txEntity) TableName() string { return "tx_entities" }

func txMigration(db *gorm.DB) error { return db.AutoMigrate(&txEntity{}) }

func TestExecuteTransaction_RollsBackOnError(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, txMigration)
	handle := db.WithContext(databasetest.TenantContext(uuid.New()))

	err := database.ExecuteTransaction(handle, func(tx *gorm.DB) error {
		if err := tx.Create(&txEntity{Name: "doomed"}).Error; err != nil {
			return err
		}
		return errors.New("boom")
	})
	require.Error(t, err)

	var count int64
	require.NoError(t, handle.Model(&txEntity{}).Count(&count).Error)
	require.Zero(t, count, "write inside failed ExecuteTransaction must roll back")
}

func TestExecuteTransaction_CommitsOnSuccess(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, txMigration)
	handle := db.WithContext(databasetest.TenantContext(uuid.New()))

	require.NoError(t, database.ExecuteTransaction(handle, func(tx *gorm.DB) error {
		return tx.Create(&txEntity{Name: "kept"}).Error
	}))

	var count int64
	require.NoError(t, handle.Model(&txEntity{}).Count(&count).Error)
	require.EqualValues(t, 1, count)
}

func TestExecuteTransaction_NestedJoinsOuterTransaction(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, txMigration)
	handle := db.WithContext(databasetest.TenantContext(uuid.New()))

	err := database.ExecuteTransaction(handle, func(outer *gorm.DB) error {
		innerErr := database.ExecuteTransaction(outer, func(inner *gorm.DB) error {
			return inner.Create(&txEntity{Name: "inner"}).Error
		})
		require.NoError(t, innerErr, "nested call must join, not fail")
		return errors.New("outer fails after inner succeeded")
	})
	require.Error(t, err)

	var count int64
	require.NoError(t, handle.Model(&txEntity{}).Count(&count).Error)
	require.Zero(t, count, "inner write must join the outer tx and roll back with it")
}

func TestExecuteTransaction_TenantCallbacksActiveInsideTransaction(t *testing.T) {
	db := databasetest.NewInMemoryTenantDB(t, txMigration)
	tid := uuid.New()
	handle := db.WithContext(databasetest.TenantContext(tid))

	require.NoError(t, database.ExecuteTransaction(handle, func(tx *gorm.DB) error {
		return tx.Create(&txEntity{Name: "stamped"}).Error
	}))

	var row txEntity
	require.NoError(t, handle.First(&row).Error)
	require.Equal(t, tid, row.TenantId, "tenant create-callback must stamp tenant_id inside the tx")
}
