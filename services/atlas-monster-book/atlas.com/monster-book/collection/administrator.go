package collection

import (
	"errors"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type statsUpdate struct {
	NormalCount     uint16
	SpecialCount    uint16
	BookLevel       uint16
	ExpBonusPercent uint16
}

// upsertStats inserts or updates the per-character collection row.
// Returns true if the row was inserted (vs updated).
//
// We build a Model (with default CoverCardId=0) and project it via ToEntity.
// On insert, CoverCardId=0 is the correct default. On conflict, the
// AssignmentColumns clause omits cover_card_id so any pre-existing cover is
// preserved.
func upsertStats(db *gorm.DB, tenantId uuid.UUID, characterId character.Id, s statsUpdate) (bool, error) {
	m, err := NewModelBuilder().
		SetTenantId(tenantId).
		SetCharacterId(characterId).
		SetNormalCount(s.NormalCount).
		SetSpecialCount(s.SpecialCount).
		SetBookLevel(s.BookLevel).
		SetExpBonusPercent(s.ExpBonusPercent).
		Build()
	if err != nil {
		return false, err
	}
	e := m.ToEntity()
	res := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "tenant_id"}, {Name: "character_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"normal_count", "special_count", "book_level", "exp_bonus_percent", "updated_at",
		}),
	}).Create(&e)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// setCover updates the cover card guarded by lastCoverEventId.
// Returns true if the row was modified, false if duplicate eventId.
func setCover(db *gorm.DB, tenantId uuid.UUID, characterId character.Id, coverCardId item.Id, eventId uuid.UUID) (bool, error) {
	res := db.Model(&entity{}).
		Where("tenant_id = ? AND character_id = ?", tenantId, uint32(characterId)).
		Where("last_cover_event_id IS NULL OR last_cover_event_id <> ?", eventId).
		Updates(map[string]interface{}{
			"cover_card_id":       uint32(coverCardId),
			"last_cover_event_id": eventId,
		})
	if res.Error != nil {
		return false, res.Error
	}
	if res.RowsAffected == 0 {
		// Either the row doesn't exist, or this eventId was already applied.
		// Distinguish by checking existence.
		var count int64
		if err := db.Model(&entity{}).
			Where("tenant_id = ? AND character_id = ?", tenantId, uint32(characterId)).
			Count(&count).Error; err != nil {
			return false, err
		}
		if count == 0 {
			return false, errors.New("collection row does not exist; cover requires owned card")
		}
		return false, nil
	}
	return true, nil
}

func getByCharacter(db *gorm.DB, tenantId uuid.UUID, characterId character.Id) (entity, error) {
	var e entity
	err := db.Where("tenant_id = ? AND character_id = ?", tenantId, uint32(characterId)).First(&e).Error
	return e, err
}

func deleteByCharacter(db *gorm.DB, tenantId uuid.UUID, characterId character.Id) error {
	return db.Where("tenant_id = ? AND character_id = ?", tenantId, uint32(characterId)).Delete(&entity{}).Error
}
