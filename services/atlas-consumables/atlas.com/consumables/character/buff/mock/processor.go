package mock

import (
	"atlas-consumables/character/buff"
	"atlas-consumables/character/buff/stat"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	ApplyFunc         func(f field.Model, fromId uint32, sourceId int32, level byte, duration int32, statups []stat.Model) model.Operator[uint32]
	CancelFunc        func(f field.Model, characterId uint32, sourceId int32) error
	CancelByTypesFunc func(f field.Model, characterId uint32, types []string) error
}

var _ buff.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) Apply(f field.Model, fromId uint32, sourceId int32, level byte, duration int32, statups []stat.Model) model.Operator[uint32] {
	if m.ApplyFunc != nil {
		return m.ApplyFunc(f, fromId, sourceId, level, duration, statups)
	}
	return func(uint32) error {
		return nil
	}
}

func (m *ProcessorMock) Cancel(f field.Model, characterId uint32, sourceId int32) error {
	if m.CancelFunc != nil {
		return m.CancelFunc(f, characterId, sourceId)
	}
	return nil
}

func (m *ProcessorMock) CancelByTypes(f field.Model, characterId uint32, types []string) error {
	if m.CancelByTypesFunc != nil {
		return m.CancelByTypesFunc(f, characterId, types)
	}
	return nil
}
