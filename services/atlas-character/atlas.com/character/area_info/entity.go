package area_info

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migration(db *gorm.DB) error {
	return db.AutoMigrate(&entity{})
}

type entity struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;not null"`
	TenantId    uuid.UUID `gorm:"not null;uniqueIndex:idx_area_info_lookup,priority:1"`
	CharacterId uint32    `gorm:"not null;uniqueIndex:idx_area_info_lookup,priority:2"`
	Area        uint16    `gorm:"not null;uniqueIndex:idx_area_info_lookup,priority:3"`
	Info        string    `gorm:"not null"`
}

func (e entity) TableName() string {
	return "area_infos"
}

// Make converts a persisted entity into its domain Model.
func Make(e entity) (Model, error) {
	return NewBuilder().
		SetId(e.ID).
		SetCharacterId(e.CharacterId).
		SetArea(e.Area).
		SetInfo(e.Info).
		Build()
}

// ToEntity is the inverse of Make for the fields Model owns. TenantId is not
// carried on Model -- upsert (administrator.go) sets it directly from the
// tenant in context when persisting, so it is not part of this round-trip.
func (m Model) ToEntity() entity {
	return entity{
		ID:          m.id,
		CharacterId: m.characterId,
		Area:        m.area,
		Info:        m.info,
	}
}
