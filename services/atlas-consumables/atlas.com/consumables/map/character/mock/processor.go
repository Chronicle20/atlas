package mock

import (
	"atlas-consumables/map/character"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

type ProcessorMock struct {
	GetMapFunc            func(characterId uint32) (field.Model, error)
	EnterFunc             func(f field.Model, characterId uint32)
	ExitFunc              func(f field.Model, characterId uint32)
	TransitionMapFunc     func(f field.Model, characterId uint32)
	TransitionChannelFunc func(f field.Model, characterId uint32)
}

var _ character.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetMap(characterId uint32) (field.Model, error) {
	if m.GetMapFunc != nil {
		return m.GetMapFunc(characterId)
	}
	return field.Model{}, nil
}

func (m *ProcessorMock) Enter(f field.Model, characterId uint32) {
	if m.EnterFunc != nil {
		m.EnterFunc(f, characterId)
	}
}

func (m *ProcessorMock) Exit(f field.Model, characterId uint32) {
	if m.ExitFunc != nil {
		m.ExitFunc(f, characterId)
	}
}

func (m *ProcessorMock) TransitionMap(f field.Model, characterId uint32) {
	if m.TransitionMapFunc != nil {
		m.TransitionMapFunc(f, characterId)
	}
}

func (m *ProcessorMock) TransitionChannel(f field.Model, characterId uint32) {
	if m.TransitionChannelFunc != nil {
		m.TransitionChannelFunc(f, characterId)
	}
}
