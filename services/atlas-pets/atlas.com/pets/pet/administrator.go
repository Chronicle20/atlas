package pet

import (
	"atlas-pets/pet/exclude"
	"errors"

	tenant "github.com/Chronicle20/atlas-tenant"
	"gorm.io/gorm"
)

func create(db *gorm.DB) func(t tenant.Model, ownerId uint32, m Model) (Model, error) {
	return func(t tenant.Model, ownerId uint32, m Model) (Model, error) {
		s := m.Slot()
		e := &Entity{
			TenantId:   t.Id(),
			OwnerId:    ownerId,
			CashId:     m.CashId(),
			TemplateId: m.TemplateId(),
			Name:       m.Name(),
			Level:      m.Level(),
			Closeness:  m.Closeness(),
			Fullness:   m.Fullness(),
			Expiration: m.Expiration(),
			Slot:       &s,
			Flag:       m.Flag(),
			PurchaseBy: m.PurchaseBy(),
		}

		err := db.Create(e).Error
		if err != nil {
			return Model{}, err
		}
		return Make(*e)
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
