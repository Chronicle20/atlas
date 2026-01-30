package equipable

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func deleteById(db *gorm.DB, tenantId uuid.UUID, id uint32) error {
	return db.Delete(&entity{TenantId: tenantId, ID: id}).Error
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
		CreatedAt:     time.Now(),
	}

	err := db.Create(e).Error
	if err != nil {
		return Model{}, err
	}

	return Make(*e)
}

func update(db *gorm.DB, tenantId uuid.UUID, id uint32, updates map[string]interface{}) (Model, error) {
	var e entity
	err := db.Where("tenant_id = ? AND id = ?", tenantId, id).First(&e).Error
	if err != nil {
		return Model{}, err
	}

	err = db.Model(&e).Updates(updates).Error
	if err != nil {
		return Model{}, err
	}

	return Make(e)
}

func Make(e entity) (Model, error) {
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
		SetOwnerName(e.OwnerName).
		SetLocked(e.Locked).
		SetSpikes(e.Spikes).
		SetKarmaUsed(e.KarmaUsed).
		SetCold(e.Cold).
		SetCanBeTraded(e.CanBeTraded).
		SetLevelType(e.LevelType).
		SetLevel(e.Level).
		SetExperience(e.Experience).
		SetHammersApplied(e.HammersApplied).
		SetExpiration(e.Expiration).
		SetCreatedAt(e.CreatedAt).
		SetEquippedSince(e.EquippedSince).
		Build()
	return r, nil
}

// setEquipped marks the equipment as equipped (sets equippedSince to now)
func setEquipped(db *gorm.DB, tenantId uuid.UUID, id uint32) error {
	now := time.Now()
	return db.Model(&entity{}).
		Where("tenant_id = ? AND id = ?", tenantId, id).
		Update("equipped_since", now).Error
}

// clearEquipped marks the equipment as unequipped (sets equippedSince to null)
func clearEquipped(db *gorm.DB, tenantId uuid.UUID, id uint32) error {
	return db.Model(&entity{}).
		Where("tenant_id = ? AND id = ?", tenantId, id).
		Update("equipped_since", nil).Error
}
