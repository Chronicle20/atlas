package model

import (
	"atlas-channel/asset"
	"atlas-channel/character"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

// NewAssetSnapshot flattens a channel asset.Model into the saga DTO
// (decode-time snapshot, PRD Q6). It copies every accessor of asset.Model
// that NewAsset (asset.go:13-39) reads so NewAssetFromSnapshot can rebuild
// the packet item block without re-resolving state from the character
// service after the item may already have been consumed.
func NewAssetSnapshot(a asset.Model) sharedsaga.AssetSnapshot {
	return sharedsaga.AssetSnapshot{
		Slot:           a.Slot(),
		TemplateId:     a.TemplateId(),
		Expiration:     a.Expiration(),
		CashId:         a.CashId(),
		Quantity:       a.Quantity(),
		Flag:           a.Flag(),
		Rechargeable:   a.Rechargeable(),
		Strength:       a.Strength(),
		Dexterity:      a.Dexterity(),
		Intelligence:   a.Intelligence(),
		Luck:           a.Luck(),
		Hp:             a.Hp(),
		Mp:             a.Mp(),
		WeaponAttack:   a.WeaponAttack(),
		MagicAttack:    a.MagicAttack(),
		WeaponDefense:  a.WeaponDefense(),
		MagicDefense:   a.MagicDefense(),
		Accuracy:       a.Accuracy(),
		Avoidability:   a.Avoidability(),
		Hands:          a.Hands(),
		Speed:          a.Speed(),
		Jump:           a.Jump(),
		Slots:          a.Slots(),
		LevelType:      a.LevelType(),
		Level:          a.Level(),
		Experience:     a.Experience(),
		HammersApplied: a.HammersApplied(),
		PetId:          a.PetId(),
		PetName:        a.PetName(),
		PetLevel:       a.PetLevel(),
		Closeness:      a.Closeness(),
		Fullness:       a.Fullness(),
	}
}

// NewAssetFromSnapshot rebuilds the packet item block from a snapshot
// (consumer side, Task 13). The inventory-type predicates are re-derived
// from the snapshot's TemplateId the way asset.go:99-101 does (a.IsEquipment
// etc.), rather than from packetmodel.Asset's own predicates, because those
// depend on fields (e.g. PetId) that are only populated by the Set* calls
// below — reading them beforehand would always observe the zero value.
func NewAssetFromSnapshot(s sharedsaga.AssetSnapshot) packetmodel.Asset {
	base := packetmodel.NewAsset(true, s.Slot, s.TemplateId, s.Expiration)

	t, _ := inventory.TypeFromItemId(item.Id(s.TemplateId))
	isEquip := t == inventory.TypeValueEquip
	isCash := t == inventory.TypeValueCash
	isPet := isCash && s.PetId > 0
	isCashEquipment := isEquip && s.CashId != 0
	isStackable := t == inventory.TypeValueUse || t == inventory.TypeValueSetup || t == inventory.TypeValueETC

	if isEquip {
		base = base.SetEquipmentStats(
			s.Strength, s.Dexterity, s.Intelligence, s.Luck,
			s.Hp, s.Mp,
			s.WeaponAttack, s.MagicAttack, s.WeaponDefense, s.MagicDefense,
			s.Accuracy, s.Avoidability, s.Hands, s.Speed, s.Jump,
		)
		base = base.SetEquipmentMeta(s.Slots, s.LevelType, s.Level, s.Experience, s.HammersApplied, s.Flag)
	}

	if isCashEquipment || isCash {
		base = base.SetCashId(s.CashId)
	}

	if isStackable || (isCash && !isPet) {
		base = base.SetStackableInfo(s.Quantity, s.Flag, s.Rechargeable)
	}

	if isPet {
		base = base.SetPetInfo(s.PetId, s.PetName, s.PetLevel, s.Fullness, s.Closeness)
	}

	return base
}

// NewAvatarSnapshot flattens a character look into the saga DTO (decode-time
// snapshot, avatar megaphone / TV). Same field walk as NewFromCharacter
// (avatar.go:10-39), but into int16-keyed maps rather than slot.Position-keyed
// maps so the DTO survives JSON round-tripping through the saga/kafka layer.
func NewAvatarSnapshot(c character.Model) sharedsaga.AvatarSnapshot {
	equips := make(map[int16]uint32)
	for _, t := range slot.Slots {
		if s, ok := c.Equipment().Get(t.Type); ok {
			if s.CashEquipable != nil {
				equips[int16(s.Position)*-1] = s.CashEquipable.TemplateId()
				continue
			}
			if s.Equipable != nil {
				equips[int16(s.Position)*-1] = s.Equipable.TemplateId()
			}
		}
	}
	maskedEquips := make(map[int16]uint32)
	for _, t := range slot.Slots {
		if s, ok := c.Equipment().Get(t.Type); ok {
			if s.CashEquipable != nil {
				if s.Equipable != nil {
					maskedEquips[int16(s.Position)*-1] = s.Equipable.TemplateId()
				}
			}
		}
	}
	pets := make(map[int8]uint32)
	for _, p := range c.Pets() {
		pets[p.Slot()] = p.TemplateId()
	}

	return sharedsaga.AvatarSnapshot{
		Gender:       c.Gender(),
		SkinColor:    c.SkinColor(),
		Face:         c.Face(),
		Hair:         c.Hair(),
		Equips:       equips,
		MaskedEquips: maskedEquips,
		Pets:         pets,
	}
}

// NewAvatarFromSnapshot rebuilds packetmodel.Avatar from a snapshot (consumer
// side, Task 14). mega is supplied per call site (SET_AVATAR_MEGAPHONE always
// arms mega=true; not carried in the snapshot since it is call-site policy,
// not part of the character's look).
func NewAvatarFromSnapshot(s sharedsaga.AvatarSnapshot, mega bool) packetmodel.Avatar {
	equips := make(map[slot.Position]uint32, len(s.Equips))
	for k, v := range s.Equips {
		equips[slot.Position(k)] = v
	}
	maskedEquips := make(map[slot.Position]uint32, len(s.MaskedEquips))
	for k, v := range s.MaskedEquips {
		maskedEquips[slot.Position(k)] = v
	}
	pets := make(map[int8]uint32, len(s.Pets))
	for k, v := range s.Pets {
		pets[k] = v
	}

	return packetmodel.NewAvatar(s.Gender, s.SkinColor, s.Face, mega, s.Hair, equips, maskedEquips, pets)
}
