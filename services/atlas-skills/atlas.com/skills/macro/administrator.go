package macro

import (
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func deleteByCharacter(db *gorm.DB, characterId uint32) error {
	return db.Where("character_id = ?", characterId).Delete(&Entity{}).Error
}

func create(db *gorm.DB, tenantId uuid.UUID, id uint32, characterId uint32, name string, shout bool, skillId1 uint32, skillId2 uint32, skillId3 uint32) model.Provider[Model] {
	e := Entity{
		TenantId:    tenantId,
		Id:          id,
		CharacterId: characterId,
		Name:        name,
		Shout:       shout,
		SkillId1:    skillId1,
		SkillId2:    skillId2,
		SkillId3:    skillId3,
	}

	err := db.Create(&e).Error
	if err != nil {
		return model.ErrorProvider[Model](err)
	}
	return model.Map(Make)(model.FixedProvider(e))
}
