package equipment

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func delete(db *gorm.DB, tenantId uuid.UUID, equipmentId uint32) error {
	return db.Delete(&entity{TenantId: tenantId, ID: equipmentId}).Error
}

func create(db *gorm.DB, tenantId uuid.UUID, itemId uint32, strength uint16, dexterity uint16, intelligence uint16, luck uint16,
	hp uint16, mp uint16, weaponAttack uint16, magicAttack uint16, weaponDefense uint16, magicDefense uint16,
	accuracy uint16, avoidability uint16, hands uint16, speed uint16, jump uint16, slots uint16) (Model, error) {
	e := &entity{
		TenantId:      tenantId,
		ItemId:        itemId,
		Strength:      strength,
		Dexterity:     dexterity,
		Intelligence:  intelligence,
		Luck:          luck,
		Hp:            hp,
		Mp:            mp,
		WeaponAttack:  weaponAttack,
		MagicAttack:   magicAttack,
		WeaponDefense: weaponDefense,
		MagicDefense:  magicDefense,
		Accuracy:      accuracy,
		Avoidability:  avoidability,
		Hands:         hands,
		Speed:         speed,
		Jump:          jump,
		Slots:         slots,
	}

	err := db.Create(e).Error
	if err != nil {
		return Model{}, err
	}

	return makeEquipment(*e)
}

func makeEquipment(e entity) (Model, error) {
	r := NewBuilder(e.ID).
		SetItemId(e.ItemId).
		SetStrength(e.Strength).
		SetDexterity(e.Dexterity).
		SetIntelligence(e.Intelligence).
		SetLuck(e.Luck).
		SetHp(e.Hp).
		SetMp(e.Mp).
		SetWeaponAttack(e.WeaponAttack).
		SetMagicAttack(e.MagicAttack).
		SetWeaponDefense(e.WeaponDefense).
		SetMagicDefense(e.MagicDefense).
		SetAccuracy(e.Accuracy).
		SetAvoidability(e.Avoidability).
		SetHands(e.Hands).
		SetSpeed(e.Speed).
		SetJump(e.Jump).
		SetSlots(e.Slots).
		Build()
	return r, nil
}
