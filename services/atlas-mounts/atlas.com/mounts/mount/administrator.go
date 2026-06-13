package mount

import (
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"gorm.io/gorm"
)

// create inserts a new mount row for the given model. The tenant_id is sourced
// from the supplied tenant.Model (the database tenant callback also injects it
// from context when zero, but we set it explicitly to keep the entity complete).
func create(db *gorm.DB) func(t tenant.Model, m Model) (Model, error) {
	return func(t tenant.Model, m Model) (Model, error) {
		e := &Entity{
			TenantId:            t.Id(),
			CharacterId:         m.CharacterId(),
			Id:                  m.Id(),
			Level:               m.Level(),
			Exp:                 m.Exp(),
			Tiredness:           m.Tiredness(),
			LastTirednessTickAt: m.LastTirednessTickAt(),
		}

		if err := db.Create(e).Error; err != nil {
			return Model{}, err
		}
		return Make(*e)
	}
}

// getByCharacterId loads a single mount row scoped to the tenant-in-context
// (the database tenant callback adds the tenant_id predicate) and the given
// character. Returns gorm.ErrRecordNotFound when no row exists.
func getByCharacterId(db *gorm.DB, characterId uint32) (Entity, error) {
	var e Entity
	err := db.Where("character_id = ?", characterId).First(&e).Error
	if err != nil {
		return Entity{}, err
	}
	return e, nil
}

// update persists the progression fields of an existing mount row, keyed by the
// tenant (via callback) and character. It does not touch the id.
func update(db *gorm.DB) func(m Model) error {
	return func(m Model) error {
		return db.Model(&Entity{}).
			Where("character_id = ?", m.CharacterId()).
			Updates(map[string]interface{}{
				"level":                  m.Level(),
				"exp":                    m.Exp(),
				"tiredness":              m.Tiredness(),
				"last_tiredness_tick_at": m.LastTirednessTickAt(),
			}).Error
	}
}
