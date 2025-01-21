package member

import (
	"github.com/Chronicle20/atlas-tenant"
	"gorm.io/gorm"
)

func create(db *gorm.DB, tenant tenant.Model, guildId uint32, characterId uint32, name string, jobId uint16, level byte, rank byte) (Model, error) {
	e := &Entity{
		TenantId:    tenant.Id(),
		GuildId:     guildId,
		CharacterId: characterId,
		Name:        name,
		JobId:       jobId,
		Level:       level,
		Rank:        rank,
		Online:      true,
	}
	err := db.Create(e).Error
	if err != nil {
		return Model{}, err
	}
	return Make(*e)
}
