package crystalband

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
)

func CreateCrystalBand(db *gorm.DB, m Model) error {
	e := m.ToEntity()
	e.Id = uuid.New()
	return db.Create(&e).Error
}

func BulkCreateCrystalBand(db *gorm.DB, models []Model) error {
	return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		for _, m := range models {
			if err := CreateCrystalBand(tx, m); err != nil {
				return err
			}
		}
		return nil
	})
}

func DeleteAllForTenant(db *gorm.DB) (int64, error) {
	result := db.Unscoped().Delete(&entity{})
	return result.RowsAffected, result.Error
}
