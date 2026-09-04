package area_info

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func upsert(db *gorm.DB, tenantId uuid.UUID, m Model) (Model, error) {
	e := m.ToEntity()
	e.ID = uuid.New()
	e.TenantId = tenantId

	err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "character_id"}, {Name: "area"}},
		DoUpdates: clause.AssignmentColumns([]string{"info"}),
	}).Create(&e).Error
	if err != nil {
		return Model{}, err
	}
	return Make(e)
}
