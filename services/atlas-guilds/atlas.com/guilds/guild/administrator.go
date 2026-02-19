package guild

import (
	"github.com/Chronicle20/atlas-constants/world"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func create(db *gorm.DB, tenantId uuid.UUID, worldId world.Id, leaderId uint32, name string) (Model, error) {
	e := &Entity{
		TenantId: tenantId,
		WorldId:  byte(worldId),
		Name:     name,
		LeaderId: leaderId,
		Capacity: 30,
	}
	err := db.Create(e).Error
	if err != nil {
		return Model{}, err
	}
	return Make(*e)
}

func updateEmblem(db *gorm.DB, guildId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte) (Model, error) {
	ge, err := getById(guildId)(db)()
	if err != nil {
		return Model{}, err
	}

	ge.Logo = logo
	ge.LogoColor = logoColor
	ge.LogoBackground = logoBackground
	ge.LogoBackgroundColor = logoBackgroundColor

	err = db.Save(&ge).Error
	if err != nil {
		return Model{}, err
	}
	return Make(ge)
}

func updateNotice(db *gorm.DB, guildId uint32, notice string) (Model, error) {
	ge, err := getById(guildId)(db)()
	if err != nil {
		return Model{}, err
	}
	ge.Notice = notice
	err = db.Save(&ge).Error
	if err != nil {
		return Model{}, err
	}
	return Make(ge)
}

func updateCapacity(db *gorm.DB, guildId uint32) (Model, error) {
	ge, err := getById(guildId)(db)()
	if err != nil {
		return Model{}, err
	}
	ge.Capacity = ge.Capacity + 5
	err = db.Save(&ge).Error
	if err != nil {
		return Model{}, err
	}
	return Make(ge)
}

func deleteGuild(db *gorm.DB, guildId uint32) error {
	return db.Where("id = ?", guildId).Delete(&Entity{}).Error
}
