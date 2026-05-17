package database

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type helperEntity struct {
	ID       uint32    `gorm:"primaryKey;autoIncrement"`
	TenantId uuid.UUID `gorm:"not null"`
	Name     string    `gorm:"not null"`
}

func (helperEntity) TableName() string { return "helper_entities" }

func helperMigration(db *gorm.DB) error { return db.AutoMigrate(&helperEntity{}) }

func TestNewInMemoryTenantDB_RegistersCallbacksAndMigrates(t *testing.T) {
	db := NewInMemoryTenantDB(t, helperMigration)
	tid := uuid.New()
	require.NoError(t, db.Create(&helperEntity{TenantId: tid, Name: "x"}).Error)

	other := uuid.New()
	require.NoError(t, db.Create(&helperEntity{TenantId: other, Name: "y"}).Error)

	var rows []helperEntity
	require.NoError(t, db.WithContext(TenantContext(tid)).Find(&rows).Error)
	assert.Len(t, rows, 1)
	assert.Equal(t, "x", rows[0].Name)
}

func TestTenantContext_CarriesTenant(t *testing.T) {
	tid := uuid.New()
	ctx := TenantContext(tid)
	require.NotNil(t, ctx)
}
