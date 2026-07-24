package inventory

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// weaponEquipSlot is the equip-compartment slot occupied by the main weapon
// (equipped-compartment slot -11).
const weaponEquipSlot int16 = -11

// Processor resolves the owner's equipped weapon type for the summon damage
// ceiling. The physical branch of the ceiling formula is weapon-type-aware.
type Processor interface {
	// GetEquippedWeaponType returns the weapon type of the item in the weapon
	// equip slot. If no weapon is equipped (or the lookup fails), it returns
	// item.WeaponTypeNone — the caller treats that as the one-handed-sword fallback.
	GetEquippedWeaponType(characterId uint32) (item.WeaponType, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetEquippedWeaponType(characterId uint32) (item.WeaponType, error) {
	compartment, err := requestEquipCompartment(characterId)(p.l, p.ctx)
	if err != nil {
		return item.WeaponTypeNone, err
	}
	for _, a := range compartment.Assets {
		if a.Slot == weaponEquipSlot {
			return item.GetWeaponType(item.Id(a.TemplateId)), nil
		}
	}
	return item.WeaponTypeNone, nil
}
