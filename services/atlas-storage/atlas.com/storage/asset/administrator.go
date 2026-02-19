package asset

import (
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func Create(l logrus.FieldLogger, db *gorm.DB, tenantId uuid.UUID) func(m Model) (Model, error) {
	return func(m Model) (Model, error) {
		e := Entity{
			TenantId:       tenantId,
			StorageId:      m.storageId,
			InventoryType:  inventoryTypeFromTemplateId(m.templateId),
			Slot:           m.slot,
			TemplateId:     m.templateId,
			Expiration:     m.expiration,
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
			Slots:          m.slots,
			LevelType:      m.levelType,
			Level:          m.level,
			Experience:     m.experience,
			HammersApplied: m.hammersApplied,
			CashId:         m.cashId,
			CommodityId:    m.commodityId,
			PurchaseBy:     m.purchaseBy,
			PetId:          m.petId,
		}
		err := db.Create(&e).Error
		if err != nil {
			return Model{}, err
		}
		return Make(e), nil
	}
}

func Delete(l logrus.FieldLogger, db *gorm.DB) func(id uint32) error {
	return func(id uint32) error {
		return db.Where("id = ?", id).Delete(&Entity{}).Error
	}
}

func DeleteByStorageId(l logrus.FieldLogger, db *gorm.DB) func(storageId uuid.UUID) error {
	return func(storageId uuid.UUID) error {
		return db.Where("storage_id = ?", storageId).Delete(&Entity{}).Error
	}
}

func UpdateSlot(l logrus.FieldLogger, db *gorm.DB) func(id uint32, slot int16) error {
	return func(id uint32, slot int16) error {
		return db.Model(&Entity{}).
			Where("id = ?", id).
			Update("slot", slot).Error
	}
}

func UpdateQuantity(l logrus.FieldLogger, db *gorm.DB) func(id uint32, quantity uint32) error {
	return func(id uint32, quantity uint32) error {
		return db.Model(&Entity{}).
			Where("id = ?", id).
			Update("quantity", quantity).Error
	}
}
