package location

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// upsertLocation persists the (tenantId, characterId, field) tuple into the
// character_locations table, replacing any existing row for that composite
// primary key. It mirrors the visit/ peer's curried administrator shape:
// db -> tenantId -> characterId -> field -> (entity, error).
//
// The state discriminator is deliberately NOT written here: this is a position
// write, and CHANGE_MAP must leave liveness as it found it. New rows take the
// column default (OFFLINE); existing rows keep whatever SetState last wrote.
func upsertLocation(db *gorm.DB) func(tenantId uuid.UUID) func(characterId uint32) func(f field.Model) (entity, error) {
	return func(tenantId uuid.UUID) func(characterId uint32) func(f field.Model) (entity, error) {
		return func(characterId uint32) func(f field.Model) (entity, error) {
			return func(f field.Model) (entity, error) {
				m := NewBuilder(characterId).SetField(f).Build()
				e := m.ToEntity(tenantId)
				e.State = ""
				if err := db.Clauses(clause.OnConflict{
					Columns: []clause.Column{{Name: "tenant_id"}, {Name: "character_id"}},
					DoUpdates: clause.AssignmentColumns([]string{
						"world_id", "channel_id", "map_id", "instance", "updated_at",
					}),
				}).Create(&e).Error; err != nil {
					return entity{}, err
				}
				var out entity
				if err := db.Where("tenant_id = ? AND character_id = ?", tenantId, characterId).
					First(&out).Error; err != nil {
					return entity{}, err
				}
				return out, nil
			}
		}
	}
}

// setLocationState writes only the state discriminator for
// (tenantId, characterId), leaving world/channel/map/instance untouched.
//
// conditional=true applies the write only when the row is not already OFFLINE.
// The cash-shop status topic and the character status topic are separate Kafka
// topics with no mutual ordering guarantee, so a late-delivered CHARACTER_EXIT
// could otherwise resurrect a logged-off character as IN_FIELD.
func setLocationState(db *gorm.DB) func(tenantId uuid.UUID) func(characterId uint32) func(state characterconst.PresenceState, conditional bool) error {
	return func(tenantId uuid.UUID) func(characterId uint32) func(state characterconst.PresenceState, conditional bool) error {
		return func(characterId uint32) func(state characterconst.PresenceState, conditional bool) error {
			return func(state characterconst.PresenceState, conditional bool) error {
				q := db.Model(&entity{}).Where("tenant_id = ? AND character_id = ?", tenantId, characterId)
				if conditional {
					q = q.Where("state <> ?", string(characterconst.PresenceStateOffline))
				}
				return q.Update("state", string(state)).Error
			}
		}
	}
}

// deleteLocation removes the character_locations row for (tenantId, characterId).
// Returns nil if the row does not exist (idempotent).
func deleteLocation(db *gorm.DB) func(tenantId uuid.UUID) func(characterId uint32) error {
	return func(tenantId uuid.UUID) func(characterId uint32) error {
		return func(characterId uint32) error {
			return db.Where("tenant_id = ? AND character_id = ?", tenantId, characterId).
				Delete(&entity{}).Error
		}
	}
}
