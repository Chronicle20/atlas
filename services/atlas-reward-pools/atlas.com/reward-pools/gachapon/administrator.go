package gachapon

import (
	database "github.com/Chronicle20/atlas/libs/atlas-database"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func CreateGachapon(db *gorm.DB, m Model) error {
	npcIds := make(int64Array, len(m.NpcIds()))
	for i, id := range m.NpcIds() {
		npcIds[i] = int64(id)
	}

	e := &entity{
		Uid:            uuid.New(),
		TenantId:       m.TenantId(),
		ID:             m.Id(),
		Name:           m.Name(),
		NpcIds:         npcIds,
		CommonWeight:   m.CommonWeight(),
		UncommonWeight: m.UncommonWeight(),
		RareWeight:     m.RareWeight(),
		Kind:           m.Kind(),
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

func UpdateGachapon(db *gorm.DB, id string, name string, npcIds []uint32, commonWeight uint32, uncommonWeight uint32, rareWeight uint32) error {
	dbNpcIds := make(int64Array, len(npcIds))
	for i, nid := range npcIds {
		dbNpcIds[i] = int64(nid)
	}
	result := db.Model(&entity{}).
		Where(&entity{ID: id}).
		Updates(map[string]interface{}{
			"name":            name,
			"npc_ids":         dbNpcIds,
			"common_weight":   commonWeight,
			"uncommon_weight": uncommonWeight,
			"rare_weight":     rareWeight,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func DeleteGachapon(db *gorm.DB, id string) error {
	return db.Where(&entity{ID: id}).Delete(&entity{}).Error
}

func DeleteAllForTenant(db *gorm.DB) (int64, error) {
	result := db.Unscoped().Delete(&entity{})
	return result.RowsAffected, result.Error
}
