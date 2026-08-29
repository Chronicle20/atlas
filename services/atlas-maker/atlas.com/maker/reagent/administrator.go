package reagent

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
)

func CreateReagent(db *gorm.DB, m Model) error {
	e := &entity{
		Id:            uuid.New(),
		TenantId:      m.TenantId(),
		ReagentItemId: uint32(m.ReagentItemId()),
		Stat:          m.Stat(),
		Value:         m.Value(),
	}
	return db.Create(e).Error
}

func BulkCreateReagent(db *gorm.DB, models []Model) error {
	return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		for _, m := range models {
			if err := CreateReagent(tx, m); err != nil {
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
