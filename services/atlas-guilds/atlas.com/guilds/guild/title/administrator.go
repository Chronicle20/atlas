package title

import (
	tenant "github.com/Chronicle20/atlas-tenant"
	"gorm.io/gorm"
)

var Defaults = []string{"Master", "Jr. Master", "Member", "Member", "Member"}

func createDefault(db *gorm.DB, tenant tenant.Model, guildId uint32) ([]Model, error) {
	var results = make([]Model, 0)
	for i, v := range Defaults {
		e := Entity{
			TenantId: tenant.Id(),
			GuildId:  guildId,
			Name:     v,
			Index:    byte(i),
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
