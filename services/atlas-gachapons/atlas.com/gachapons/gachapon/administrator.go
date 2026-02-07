package gachapon

import (
	"atlas-gachapons/database"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

func CreateGachapon(db *gorm.DB, m Model) error {
	npcIds := make(pq.Int64Array, len(m.NpcIds()))
	for i, id := range m.NpcIds() {
		npcIds[i] = int64(id)
	}

	e := &entity{
		TenantId:       m.TenantId(),
		ID:             m.Id(),
		Name:           m.Name(),
		NpcIds:         npcIds,
		CommonWeight:   m.CommonWeight(),
		UncommonWeight: m.UncommonWeight(),
		RareWeight:     m.RareWeight(),
	}
	return db.Create(e).Error
}

func BulkCreateGachapon(db *gorm.DB, models []Model) error {
	return database.ExecuteTransaction(db, func(tx *gorm.DB) error {
		for _, m := range models {
			if err := CreateGachapon(tx, m); err != nil {
				return err
			}
		}
		return nil
	})
}

func UpdateGachapon(db *gorm.DB, tenantId uuid.UUID, id string, name string, commonWeight uint32, uncommonWeight uint32, rareWeight uint32) error {
	return db.Model(&entity{}).
		Where(&entity{TenantId: tenantId, ID: id}).
		Updates(map[string]interface{}{
			"name":            name,
			"common_weight":   commonWeight,
			"uncommon_weight": uncommonWeight,
			"rare_weight":     rareWeight,
		}).Error
}

func DeleteGachapon(db *gorm.DB, tenantId uuid.UUID, id string) error {
	return db.Where(&entity{TenantId: tenantId, ID: id}).Delete(&entity{}).Error
}

func DeleteAllForTenant(db *gorm.DB, tenantId uuid.UUID) (int64, error) {
	result := db.Unscoped().Where("tenant_id = ?", tenantId).Delete(&entity{})
	return result.RowsAffected, result.Error
}
