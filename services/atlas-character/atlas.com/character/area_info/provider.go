package area_info

import (
	"gorm.io/gorm"
)

func getByCharacterIdAndArea(db *gorm.DB, characterId uint32, area uint16) (Model, error) {
	var e entity
	err := db.Where("character_id = ? AND area = ?", characterId, area).First(&e).Error
	if err != nil {
		return Model{}, err
	}
	return Make(e)
}

func getAllByCharacterId(db *gorm.DB, characterId uint32) ([]Model, error) {
	var es []entity
	err := db.Where("character_id = ?", characterId).Find(&es).Error
	if err != nil {
		return nil, err
	}
	ms := make([]Model, 0, len(es))
	for _, e := range es {
		m, err := Make(e)
		if err != nil {
			return nil, err
		}
		ms = append(ms, m)
	}
	return ms, nil
}
