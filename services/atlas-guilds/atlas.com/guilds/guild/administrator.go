package guild

import (
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func create(db *gorm.DB, tenant tenant.Model, worldId byte, leaderId uint32, name string) (Model, error) {
	e := &Entity{
		TenantId: tenant.Id(),
		WorldId:  worldId,
		Name:     name,
		LeaderId: leaderId,
	}
	err := db.Create(e).Error
	if err != nil {
		return Model{}, err
	}
	return Make(*e)
}

func updateEmblem(db *gorm.DB, tenantId uuid.UUID, guildId uint32, logo uint16, logoColor byte, logoBackground uint16, logoBackgroundColor byte) (Model, error) {
	ge, err := getById(tenantId, guildId)(db)()
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
