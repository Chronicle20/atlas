package asset

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func create(db *gorm.DB, tenantId uuid.UUID, m Model) (Model, error) {
	e := &Entity{
		TenantId:       tenantId,
		CompartmentId:  m.compartmentId,
		Slot:           m.slot,
		TemplateId:     m.templateId,
		Expiration:     m.expiration,
		CreatedAt:      m.createdAt,
		Quantity:       m.quantity,
		OwnerId:        m.ownerId,
		Flag:           m.flag,
		Rechargeable:   m.rechargeable,
		Strength:       m.strength,
		Dexterity:      m.dexterity,
		Intelligence:   m.intelligence,
		Luck:           m.luck,
		Hp:             m.hp,
		Mp:             m.mp,
		WeaponAttack:   m.weaponAttack,
		MagicAttack:    m.magicAttack,
		WeaponDefense:  m.weaponDefense,
		MagicDefense:   m.magicDefense,
		Accuracy:       m.accuracy,
		Avoidability:   m.avoidability,
		Hands:          m.hands,
		Speed:          m.speed,
		Jump:           m.jump,
		Slots:     m.slots,
		LevelType: m.levelType,
		Level:          m.level,
		Experience:     m.experience,
		HammersApplied: m.hammersApplied,
		EquippedSince:  m.equippedSince,
		CashId:         m.cashId,
		CommodityId:    m.commodityId,
		PurchaseBy:     m.purchaseBy,
		PetId:          m.petId,
	}

	err := db.Create(e).Error
	if err != nil {
		return Model{}, err
	}
	return Make(*e)
}

func updateSlot(db *gorm.DB, id uint32, slot int16) error {
	return db.Model(&Entity{Id: id}).Select("Slot").Updates(&Entity{Slot: slot}).Error
}

func updateQuantity(db *gorm.DB, id uint32, quantity uint32) error {
	return db.Model(&Entity{Id: id}).Select("Quantity").Updates(&Entity{Quantity: quantity}).Error
}

func updateEquipmentStats(db *gorm.DB, id uint32, m Model) error {
	return db.Model(&Entity{Id: id}).
		Select("Strength", "Dexterity", "Intelligence", "Luck", "Hp", "Mp",
			"WeaponAttack", "MagicAttack", "WeaponDefense", "MagicDefense",
			"Accuracy", "Avoidability", "Hands", "Speed", "Jump", "Slots",
			"Flag", "LevelType", "Level", "Experience", "HammersApplied", "Expiration").
		Updates(&Entity{
			Strength:       m.strength,
			Dexterity:      m.dexterity,
			Intelligence:   m.intelligence,
			Luck:           m.luck,
			Hp:             m.hp,
			Mp:             m.mp,
			WeaponAttack:   m.weaponAttack,
			MagicAttack:    m.magicAttack,
			WeaponDefense:  m.weaponDefense,
			MagicDefense:   m.magicDefense,
			Accuracy:       m.accuracy,
			Avoidability:   m.avoidability,
			Hands:          m.hands,
			Speed:          m.speed,
			Jump:           m.jump,
			Slots:          m.slots,
			Flag:           m.flag,
			LevelType:      m.levelType,
			Level:          m.level,
			Experience:     m.experience,
			HammersApplied: m.hammersApplied,
			Expiration:     m.expiration,
		}).Error
}

func deleteById(db *gorm.DB, id uint32) error {
	return db.Where("id = ?", id).Delete(&Entity{}).Error
}
