package mock

import (
	"atlas-rates/buffs"
)

type ProcessorMock struct {
	GetActiveBuffsFunc func(characterId uint32) ([]buffs.RestModel, error)
}

var _ buffs.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetActiveBuffs(characterId uint32) ([]buffs.RestModel, error) {
	if m.GetActiveBuffsFunc != nil {
		return m.GetActiveBuffsFunc(characterId)
	}
	return nil, nil
}
