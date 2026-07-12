package mock

import (
	"atlas-summons/inventory"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

type ProcessorMock struct {
	GetEquippedWeaponTypeFunc func(characterId uint32) (item.WeaponType, error)
}

var _ inventory.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetEquippedWeaponType(characterId uint32) (item.WeaponType, error) {
	if m.GetEquippedWeaponTypeFunc != nil {
		return m.GetEquippedWeaponTypeFunc(characterId)
	}
	return item.WeaponTypeNone, nil
}
