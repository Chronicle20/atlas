package databasetest

import (
	"testing"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type fwEntity struct {
	Id       uint32    `gorm:"primaryKey;autoIncrement"`
	TenantId uuid.UUID `gorm:"type:uuid"`
	Name     string
}

func (fwEntity) TableName() string { return "fw_entities" }

type fwOther struct {
	Id   uint32 `gorm:"primaryKey;autoIncrement"`
	Name string
}

func (fwOther) TableName() string { return "fw_others" }

func fwMigration(db *gorm.DB) error { return db.AutoMigrate(&fwEntity{}, &fwOther{}) }

func TestFailWritesOn_FailsNamedVerbOnNamedTableOnly(t *testing.T) {
	db := NewInMemoryTenantDB(t, fwMigration)
	handle := db.WithContext(TenantContext(uuid.New()))

	FailWritesOn(t, db, "fw_entities", WriteCreate)

	require.Error(t, handle.Create(&fwEntity{Name: "blocked"}).Error,
		"create on the named table must fail")
	require.NoError(t, handle.Create(&fwOther{Name: "fine"}).Error,
		"other tables must be unaffected")
	require.NoError(t, handle.Where("1 = 1").Delete(&fwEntity{}).Error,
		"unregistered verbs on the named table must be unaffected")
}

func TestFailWritesOn_DefaultsToAllVerbs(t *testing.T) {
	db := NewInMemoryTenantDB(t, fwMigration)
	handle := db.WithContext(TenantContext(uuid.New()))

	FailWritesOn(t, db, "fw_entities")

	require.Error(t, handle.Create(&fwEntity{Name: "blocked"}).Error)
	require.Error(t, handle.Model(&fwEntity{}).Where("1 = 1").Update("name", "x").Error)
	require.Error(t, handle.Where("1 = 1").Delete(&fwEntity{}).Error)
}

func TestFailWritesOn_DrivesRollbackThroughExecuteTransaction(t *testing.T) {
	db := NewInMemoryTenantDB(t, fwMigration)
	handle := db.WithContext(TenantContext(uuid.New()))
	require.NoError(t, handle.Create(&fwEntity{Name: "original"}).Error)

	// The class-B shape (keys reset): delete-all succeeds, re-create fails,
	// the whole flow must roll back to the pre-flow state.
	FailWritesOn(t, db, "fw_entities", WriteCreate)

	err := database.ExecuteTransaction(handle, func(tx *gorm.DB) error {
		if err := tx.Where("1 = 1").Delete(&fwEntity{}).Error; err != nil {
			return err
		}
		return tx.Create(&fwEntity{Name: "replacement"}).Error
	})
	require.Error(t, err)

	var rows []fwEntity
	require.NoError(t, handle.Find(&rows).Error)
	require.Len(t, rows, 1)
	require.Equal(t, "original", rows[0].Name, "the delete must have rolled back")
}
