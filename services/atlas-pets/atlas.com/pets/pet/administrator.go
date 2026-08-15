package pet

import (
	"atlas-pets/pet/exclude"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func create(db *gorm.DB) func(t tenant.Model, ownerId uint32, m Model) (Model, error) {
	return func(t tenant.Model, ownerId uint32, m Model) (Model, error) {
		e := m.ToEntity(t.Id())
		e.OwnerId = ownerId

		err := db.Create(&e).Error
		if err != nil {
			return Model{}, err
		}
		return Make(e)
	}
}

func updateSlot(db *gorm.DB) func(petId uint32, slot int8) error {
	return func(petId uint32, slot int8) error {
		result := db.Model(&Entity{}).
			Where("id = ?", petId).
			Update("slot", slot)

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return errors.New("no entity found or slot is already set to the given value")
		}

		return nil
	}
}

func updateCloseness(db *gorm.DB) func(petId uint32, closeness uint16) error {
	return func(petId uint32, closeness uint16) error {
		result := db.Model(&Entity{}).
			Where("id = ?", petId).
			Update("closeness", closeness)

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return errors.New("no entity found or closeness is already set to the given value")
		}

		return nil
	}
}

func updateLevel(db *gorm.DB) func(petId uint32, level byte) error {
	return func(petId uint32, level byte) error {
		result := db.Model(&Entity{}).
			Where("id = ?", petId).
			Update("level", level)

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return errors.New("no entity found or level is already set to the given value")
		}

		return nil
	}
}

func updateName(db *gorm.DB) func(petId uint32, name string) error {
	return func(petId uint32, name string) error {
		result := db.Model(&Entity{}).
			Where("id = ?", petId).
			Update("name", name)

		if result.Error != nil {
			return result.Error
		}

		// Deliberately NOT treating RowsAffected == 0 as an error, unlike the
		// sibling update functions above. Kafka is at-least-once: a redelivered
		// RENAME whose value is already applied updates zero rows, and erroring
		// there would fail the orchestrator's rename_pet step on a duplicate
		// that changed nothing (PRD FR-5.5). Existence is proven by the caller's
		// pre-read inside the same transaction.
		return nil
	}
}

func updateOnEvolve(db *gorm.DB) func(petId uint32, templateId uint32, expiration time.Time) error {
	return func(petId uint32, templateId uint32, expiration time.Time) error {
		result := db.Model(&Entity{}).
			Where("id = ?", petId).
			Updates(map[string]interface{}{
				"template_id": templateId,
				"expiration":  expiration,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("no entity found to evolve")
		}
		return nil
	}
}

// updateOnRevive writes ONLY the expiration and the revive transaction id.
// Deliberately not updateOnEvolve: that function also rewrites template_id,
// and a revive must never touch the pet's template.
func updateOnRevive(db *gorm.DB) func(petId uint32, expiration time.Time, transactionId uuid.UUID) error {
	return func(petId uint32, expiration time.Time, transactionId uuid.UUID) error {
		result := db.Model(&Entity{}).
			Where("id = ?", petId).
			Updates(map[string]interface{}{
				"expiration":            expiration,
				"revive_transaction_id": transactionId,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("no entity found to revive")
		}
		return nil
	}
}

func updateFlag(db *gorm.DB) func(petId uint32, flag uint16) error {
	return func(petId uint32, flag uint16) error {
		result := db.Model(&Entity{}).
			Where("id = ?", petId).
			Update("flag", flag)

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return errors.New("no entity found or flag is already set to the given value")
		}

		return nil
	}
}

func updateFullness(db *gorm.DB) func(petId uint32, fullness byte) error {
	return func(petId uint32, fullness byte) error {
		result := db.Model(&Entity{}).
			Where("id = ?", petId).
			Update("fullness", fullness)

		if result.Error != nil {
			return result.Error
		}

		if result.RowsAffected == 0 {
			return errors.New("no entity found or fullness is already set to the given value")
		}

		return nil
	}
}

func deleteById(id uint32) func(db *gorm.DB) error {
	return func(db *gorm.DB) error {
		return db.Where("id = ?", id).Delete(&Entity{}).Error
	}
}

func setExcludes(db *gorm.DB, petId uint32, itemIds []uint32) error {
	// Start a transaction for atomicity
	return db.Transaction(func(tx *gorm.DB) error {
		// Step 1: Delete existing excludes for the pet
		if err := tx.Where("pet_id = ?", petId).Delete(&exclude.Entity{}).Error; err != nil {
			return err
		}

		// Step 2: Create new excludes for the given itemIds
		excludes := make([]exclude.Entity, len(itemIds))
		for i, itemId := range itemIds {
			excludes[i] = exclude.Entity{
				PetId:  petId,
				ItemId: itemId,
			}
		}

		if len(excludes) > 0 {
			if err := tx.Create(&excludes).Error; err != nil {
				return err
			}
		}

		return nil
	})
}
