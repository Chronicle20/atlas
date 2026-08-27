package account

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type EntityUpdateFunction func() ([]string, func(e *Entity))

func create(db *gorm.DB, tenantId uuid.UUID, name string, password string, gender byte) (Model, error) {
	a := &Entity{
		TenantId: tenantId,
		Name:     name,
		Password: password,
		Gender:   gender,
	}

	err := db.Create(a).Error
	if err != nil {
		return Model{}, err
	}

	return Make(*a)
}

func update(db *gorm.DB) func(modifiers ...EntityUpdateFunction) IdOperator {
	return func(modifiers ...EntityUpdateFunction) IdOperator {
		return func(id uint32) error {
			e := &Entity{}
			var columns []string
			for _, modifier := range modifiers {
				c, u := modifier()
				columns = append(columns, c...)
				u(e)
			}
			return db.Model(&Entity{ID: id}).Select(columns).Updates(e).Error
		}
	}
}

func deleteById(db *gorm.DB) IdOperator {
	return func(id uint32) error {
		return db.Where("id = ?", id).Delete(&Entity{}).Error
	}
}

func updatePic(pic string) EntityUpdateFunction {
	return func() ([]string, func(e *Entity)) {
		cs := []string{"pic"}

		uf := func(e *Entity) {
			e.PIC = pic
		}
		return cs, uf
	}
}

func updateBirthDate(birthDate uint32) EntityUpdateFunction {
	return func() ([]string, func(e *Entity)) {
		cs := []string{"birth_date"}

		uf := func(e *Entity) {
			e.BirthDate = birthDate
		}
		return cs, uf
	}
}

func updatePin(pin string) EntityUpdateFunction {
	return func() ([]string, func(e *Entity)) {
		cs := []string{"pin"}

		uf := func(e *Entity) {
			e.PIN = pin
		}
		return cs, uf
	}
}

func updateTos(tos bool) EntityUpdateFunction {
	return func() ([]string, func(e *Entity)) {
		cs := []string{"tos"}

		uf := func(e *Entity) {
			e.TOS = tos
		}
		return cs, uf
	}
}

func updateGender(gender byte) EntityUpdateFunction {
	return func() ([]string, func(e *Entity)) {
		cs := []string{"gender"}

		uf := func(e *Entity) {
			e.Gender = gender
		}
		return cs, uf
	}
}

func updatePinAttempts(pinAttempts int) EntityUpdateFunction {
	return func() ([]string, func(e *Entity)) {
		cs := []string{"pin_attempts"}

		uf := func(e *Entity) {
			e.PinAttempts = pinAttempts
		}
		return cs, uf
	}
}

func updatePicAttempts(picAttempts int) EntityUpdateFunction {
	return func() ([]string, func(e *Entity)) {
		cs := []string{"pic_attempts"}

		uf := func(e *Entity) {
			e.PicAttempts = picAttempts
		}
		return cs, uf
	}
}

func createCharacterSlot(db *gorm.DB, tenantId uuid.UUID, accountId uint32, worldId byte, slots int16) (CharacterSlotEntity, error) {
	e := &CharacterSlotEntity{
		TenantId:  tenantId,
		AccountId: accountId,
		WorldId:   worldId,
		Slots:     slots,
	}
	err := db.Create(e).Error
	if err != nil {
		return CharacterSlotEntity{}, err
	}
	return *e, nil
}

func updateCharacterSlots(db *gorm.DB, id uint32, slots int16) error {
	return db.Model(&CharacterSlotEntity{ID: id}).Update("slots", slots).Error
}

func MakeCharacterSlot(e CharacterSlotEntity) (CharacterSlotModel, error) {
	return NewCharacterSlotBuilder(e.TenantId, e.AccountId, e.WorldId).
		SetSlots(e.Slots).
		Build(), nil
}

func Make(a Entity) (Model, error) {
	return NewBuilder(a.TenantId, a.Name).
		SetId(a.ID).
		SetPassword(a.Password).
		SetPin(a.PIN).
		SetPic(a.PIC).
		SetBirthDate(a.BirthDate).
		SetPinAttempts(a.PinAttempts).
		SetPicAttempts(a.PicAttempts).
		SetGender(a.Gender).
		SetTOS(a.TOS).
		SetUpdatedAt(a.UpdatedAt).
		Build()
}
