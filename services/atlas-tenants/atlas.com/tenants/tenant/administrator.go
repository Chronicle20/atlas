package tenant

import (
	"atlas-tenants/database"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CreateTenant creates a new tenant in the database
func CreateTenant(db *gorm.DB, e Entity) error {
	return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		return tx.Create(&e).Error
	})
}

// UpdateTenant updates an existing tenant in the database
func UpdateTenant(db *gorm.DB, e Entity) error {
	return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		return tx.Save(&e).Error
	})
}

// DeleteTenant deletes a tenant from the database
func DeleteTenant(db *gorm.DB, id uuid.UUID) error {
	var e Entity
	err := db.Where("id = ?", id).First(&e).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("tenant not found")
		}
		return err
	}

	return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		return tx.Delete(&e).Error
	})
}