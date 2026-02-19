package title

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var Defaults = []string{"Master", "Jr. Master", "Member", "Member", "Member"}

func createDefault(db *gorm.DB, tenantId uuid.UUID, guildId uint32) ([]Model, error) {
	return createTitles(db, tenantId, guildId, Defaults)
}

func createTitles(db *gorm.DB, tenantId uuid.UUID, guildId uint32, titles []string) ([]Model, error) {
	var results = make([]Model, 0)
	for i, v := range titles {
		e := Entity{
			TenantId: tenantId,
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
