package location

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// upsertLocation persists the (tenantId, characterId, field) tuple into the
// character_locations table, replacing any existing row for that composite
// primary key. It mirrors the visit/ peer's curried administrator shape:
// db -> tenantId -> characterId -> field -> (entity, error).
func upsertLocation(db *gorm.DB) func(tenantId uuid.UUID) func(characterId uint32) func(f field.Model) (entity, error) {
	return func(tenantId uuid.UUID) func(characterId uint32) func(f field.Model) (entity, error) {
		return func(characterId uint32) func(f field.Model) (entity, error) {
			return func(f field.Model) (entity, error) {
				m := NewBuilder(characterId).SetField(f).Build()
				e := m.ToEntity(tenantId)
				if err := db.Save(&e).Error; err != nil {
					return entity{}, err
				}
				return e, nil
			}
		}
	}
}
