package title

import (
	"github.com/Chronicle20/atlas-tenant"
	"gorm.io/gorm"
)

var Defaults = []string{"Master", "Jr. Master", "Member", "Member", "Member"}

func createDefault(db *gorm.DB, tenant tenant.Model, guildId uint32) ([]Model, error) {
	return create(db, tenant, guildId, Defaults)
}

func create(db *gorm.DB, tenant tenant.Model, guildId uint32, titles []string) ([]Model, error) {
	var results = make([]Model, 0)
	for i, v := range titles {
		e := Entity{
			TenantId: tenant.Id(),
			GuildId:  guildId,
			Name:     v,
			Index:    byte(i + 1),
		}
		err := db.Create(&e).Error
		if err != nil {
			return nil, err
		}
		r, err := Make(e)
		if err != nil {
			return nil, err
		}

		results = append(results, r)
	}
	return results, nil
}
