package saved_location

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func create(db *gorm.DB, tenantId uuid.UUID, m Model) (Model, error) {
	e := &entity{
		ID:           uuid.New(),
		TenantId:     tenantId,
		CharacterId:  m.CharacterId(),
		LocationType: m.LocationType(),
		MapId:        m.MapId(),
		PortalId:     m.PortalId(),
	}

	err := db.Create(e).Error
	if err != nil {
		return Model{}, err
	}
	return modelFromEntity(*e)
}

func upsert(db *gorm.DB, tenantId uuid.UUID, m Model) (Model, error) {
	e := &entity{
		ID:           uuid.New(),
		TenantId:     tenantId,
		CharacterId:  m.CharacterId(),
		LocationType: m.LocationType(),
		MapId:        m.MapId(),
		PortalId:     m.PortalId(),
	}

	err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "character_id"}, {Name: "location_type"}},
		DoUpdates: clause.AssignmentColumns([]string{"map_id", "portal_id"}),
	}).Create(e).Error
	if err != nil {
		return Model{}, err
	}
	return modelFromEntity(*e)
}

func getByCharacterIdAndType(db *gorm.DB, tenantId uuid.UUID, characterId uint32, locationType string) (Model, error) {
	var e entity
	err := db.Where("tenant_id = ? AND character_id = ? AND location_type = ?", tenantId, characterId, locationType).First(&e).Error
	if err != nil {
		return Model{}, err
	}
	return modelFromEntity(e)
}

func deleteByCharacterIdAndType(db *gorm.DB, tenantId uuid.UUID, characterId uint32, locationType string) error {
	return db.Where("tenant_id = ? AND character_id = ? AND location_type = ?", tenantId, characterId, locationType).Delete(&entity{}).Error
}
