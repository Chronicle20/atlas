package area_info

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func upsert(db *gorm.DB, tenantId uuid.UUID, m Model) (Model, error) {
	e := &entity{
		ID:          uuid.New(),
		TenantId:    tenantId,
		CharacterId: m.CharacterId(),
		Area:        m.Area(),
		Info:        m.Info(),
	}

	err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "character_id"}, {Name: "area"}},
		DoUpdates: clause.AssignmentColumns([]string{"info"}),
	}).Create(e).Error
	if err != nil {
		return Model{}, err
	}
	return modelFromEntity(*e)
}

func getByCharacterIdAndArea(db *gorm.DB, characterId uint32, area uint16) (Model, error) {
	var e entity
	err := db.Where("character_id = ? AND area = ?", characterId, area).First(&e).Error
	if err != nil {
		return Model{}, err
	}
	return modelFromEntity(e)
}

func getAllByCharacterId(db *gorm.DB, characterId uint32) ([]Model, error) {
	var es []entity
	err := db.Where("character_id = ?", characterId).Find(&es).Error
	if err != nil {
		return nil, err
	}
	ms := make([]Model, 0, len(es))
	for _, e := range es {
		m, err := modelFromEntity(e)
		if err != nil {
			return nil, err
		}
		ms = append(ms, m)
	}
	return ms, nil
}
