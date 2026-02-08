package item

import (
	"atlas-gachapons/database"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func CreateItem(db *gorm.DB, m Model) error {
	e := &entity{
		TenantId:   m.TenantId(),
		GachaponId: m.GachaponId(),
		ItemId:     m.ItemId(),
		Quantity:   m.Quantity(),
		Tier:       m.Tier(),
	}
	return db.Create(e).Error
}

func BulkCreateItem(db *gorm.DB, models []Model) error {
	return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		for _, m := range models {
			if err := CreateItem(tx, m); err != nil {
				return err
			}
		}
		return nil
	})
}

func DeleteItem(db *gorm.DB, tenantId uuid.UUID, id uint32) error {
	return db.Where(&entity{TenantId: tenantId, ID: id}).Delete(&entity{}).Error
}

func DeleteAllForTenant(db *gorm.DB, tenantId uuid.UUID) (int64, error) {
	result := db.Unscoped().Where("tenant_id = ?", tenantId).Delete(&entity{})
	return result.RowsAffected, result.Error
}
