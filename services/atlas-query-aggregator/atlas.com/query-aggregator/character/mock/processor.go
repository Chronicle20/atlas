package mock

import (
	"atlas-query-aggregator/character"
	"github.com/Chronicle20/atlas-model/model"
)

// ProcessorImpl is a mock implementation of the character.ProcessorImpl
type ProcessorImpl struct {
	GetByIdFunc func(decorators ...model.Decorator[character.Model]) func(characterId uint32) (character.Model, error)
}

// GetById returns a function that gets a character by ID
func (m *ProcessorImpl) GetById(decorators ...model.Decorator[character.Model]) func(characterId uint32) (character.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(decorators...)
	}
	return func(characterId uint32) (character.Model, error) {
		return character.NewModelBuilder().Build(), nil
	}
}
