package guild

import (
	"atlas-guilds/guild/member"
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
		Capacity: 30,
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

func updateMemberStatus(db *gorm.DB, tenantId uuid.UUID, guildId uint32, characterId uint32, online bool) error {
	ge, err := getById(tenantId, guildId)(db)()
	if err != nil {
		return err
	}

	var gm *member.Entity
	for _, pm := range ge.Members {
		if pm.CharacterId == characterId {
			gm = &pm
		}
	}
	if gm == nil {
		return nil
	}

	gm.Online = online
	return db.Save(gm).Error
}

func updateMemberTitle(db *gorm.DB, tenantId uuid.UUID, guildId uint32, characterId uint32, title byte) error {
	ge, err := getById(tenantId, guildId)(db)()
	if err != nil {
		return err
	}

	var gm *member.Entity
	for _, pm := range ge.Members {
		if pm.CharacterId == characterId {
			gm = &pm
		}
	}
	if gm == nil {
		return nil
	}

	gm.Title = title
	return db.Save(gm).Error
}

func updateNotice(db *gorm.DB, tenantId uuid.UUID, guildId uint32, notice string) (Model, error) {
	ge, err := getById(tenantId, guildId)(db)()
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

func deleteGuild(db *gorm.DB, tenantId uuid.UUID, guildId uint32) error {
	return db.Where("tenant_id = ? AND id = ?", tenantId, guildId).Delete(&Entity{}).Error
}
