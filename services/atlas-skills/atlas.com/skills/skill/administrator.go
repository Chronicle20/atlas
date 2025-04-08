package skill

import (
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type EntityUpdateFunction func() ([]string, func(e *Entity))

func create(db *gorm.DB, tenantId uuid.UUID, characterId uint32, id uint32, level byte, masterLevel byte, expiration time.Time) (Model, error) {
	e := &Entity{
		TenantId:    tenantId,
		CharacterId: characterId,
		Id:          id,
		Level:       level,
		MasterLevel: masterLevel,
		Expiration:  expiration,
	}

	err := db.Create(e).Error
	if err != nil {
		return Model{}, err
	}
	return Make(*e)
}

// Returns a function which accepts a character model,and updates the persisted state of the character given a set of
// modifying functions.
func dynamicUpdate(db *gorm.DB) func(modifiers ...EntityUpdateFunction) func(tenantId uuid.UUID, characterId uint32) model.Operator[Model] {
	return func(modifiers ...EntityUpdateFunction) func(tenantId uuid.UUID, characterId uint32) model.Operator[Model] {
		return func(tenantId uuid.UUID, characterId uint32) model.Operator[Model] {
			return func(s Model) error {
				if len(modifiers) > 0 {
					err := update(db, tenantId, characterId, s.Id(), modifiers...)
					if err != nil {
						return err
					}
				}
				return nil
			}
		}
	}
}

func update(db *gorm.DB, tenantId uuid.UUID, characterId uint32, id uint32, modifiers ...EntityUpdateFunction) error {
	e := &Entity{}

	var columns []string
	for _, modifier := range modifiers {
		c, u := modifier()
		columns = append(columns, c...)
		u(e)
	}
	return db.Model(&Entity{TenantId: tenantId, CharacterId: characterId, Id: id}).Select(columns).Updates(e).Error
}

func SetExpiration(expiration time.Time) EntityUpdateFunction {
	return func() ([]string, func(e *Entity)) {
		return []string{"Expiration"}, func(e *Entity) {
			e.Expiration = expiration
		}
	}
}

func SetMasterLevel(level byte) EntityUpdateFunction {
	return func() ([]string, func(e *Entity)) {
		return []string{"MasterLevel"}, func(e *Entity) {
			e.MasterLevel = level
		}
	}
}

func SetLevel(level byte) EntityUpdateFunction {
	return func() ([]string, func(e *Entity)) {
		return []string{"Level"}, func(e *Entity) {
			e.Level = level
		}
	}
}

func deleteByCharacter(db *gorm.DB, t tenant.Model, characterId uint32) error {
	return db.Where(&Entity{TenantId: t.Id(), CharacterId: characterId}).Delete(&Entity{}).Error
}
