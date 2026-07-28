package mock

import (
	"atlas-channel/reactor"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	InMapModelProviderFunc func(f field.Model) model.Provider[[]reactor.Model]
	ForEachInMapFunc       func(f field.Model, o model.Operator[reactor.Model]) error
	HitFunc                func(f field.Model, reactorId uint32, characterId uint32, stance uint16, skillId uint32) error
}

var _ reactor.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) InMapModelProvider(f field.Model) model.Provider[[]reactor.Model] {
	if m.InMapModelProviderFunc != nil {
		return m.InMapModelProviderFunc(f)
	}
	return model.FixedProvider([]reactor.Model{})
}

func (m *ProcessorMock) ForEachInMap(f field.Model, o model.Operator[reactor.Model]) error {
	if m.ForEachInMapFunc != nil {
		return m.ForEachInMapFunc(f, o)
	}
	return nil
}

func (m *ProcessorMock) Hit(f field.Model, reactorId uint32, characterId uint32, stance uint16, skillId uint32) error {
	if m.HitFunc != nil {
		return m.HitFunc(f, reactorId, characterId, stance, skillId)
	}
	return nil
}
