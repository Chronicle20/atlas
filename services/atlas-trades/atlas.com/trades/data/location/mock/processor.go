package mock

import (
	"atlas-trades/data/location"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// ProcessorMock is the injectable double for the character-location REST
// client. Each Func field defaults to a zero-valued success when left unset.
type ProcessorMock struct {
	FieldOfFunc       func(characterId character.Id) (field.Model, error)
	FieldProviderFunc func(characterId character.Id) model.Provider[field.Model]
}

func (m *ProcessorMock) FieldOf(characterId character.Id) (field.Model, error) {
	if m.FieldOfFunc != nil {
		return m.FieldOfFunc(characterId)
	}
	return field.Model{}, nil
}

func (m *ProcessorMock) FieldProvider(characterId character.Id) model.Provider[field.Model] {
	if m.FieldProviderFunc != nil {
		return m.FieldProviderFunc(characterId)
	}
	return model.FixedProvider(field.Model{})
}

var _ location.Processor = (*ProcessorMock)(nil)
