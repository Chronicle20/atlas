package guild

import (
	"github.com/Chronicle20/atlas-tenant"
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
