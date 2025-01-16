package list

import (
	"atlas-buddies/buddy"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func create(db *gorm.DB, t tenant.Model, characterId uint32, capacity byte) (Model, error) {
	e := &Entity{
		TenantId:    t.Id(),
		CharacterId: characterId,
		Capacity:    capacity,
	}

	err := db.Create(e).Error
	if err != nil {
		return Model{}, err
	}
	return Make(*e)
}

func addPendingBuddy(db *gorm.DB, tenantId uuid.UUID, characterId uint32, targetId uint32, targetName string, group string) error {
	return addBuddy(db, tenantId, characterId, targetId, targetName, group, true)
}

func addBuddy(db *gorm.DB, tenantId uuid.UUID, characterId uint32, targetId uint32, targetName string, group string, pending bool) error {
	e, err := byCharacterIdEntityProvider(tenantId, characterId)(db)()
	if err != nil {
		return err
	}

	nb := buddy.Entity{
		CharacterId:   targetId,
		ListId:        e.Id,
		Group:         group,
		CharacterName: targetName,
		ChannelId:     -1,
		Pending:       pending,
	}
	return db.Create(&nb).Error
}

func removeBuddy(db *gorm.DB, tenantId uuid.UUID, characterId uint32, targetId uint32) error {
	e, err := byCharacterIdEntityProvider(tenantId, characterId)(db)()
	if err != nil {
		return err
	}

	var rb buddy.Entity
	for _, b := range e.Buddies {
		if b.CharacterId == targetId {
			rb = b
			break
		}
	}

	if rb.ListId == uuid.Nil {
		return gorm.ErrRecordNotFound
	}

	return db.Delete(&rb).Error
}
