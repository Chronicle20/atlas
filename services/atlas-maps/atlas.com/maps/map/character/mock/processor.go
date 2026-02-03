package mock

import (
	"atlas-maps/map/character"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/google/uuid"
)

type Processor struct {
	GetCharactersInMapFunc    func(transactionId uuid.UUID, f field.Model) ([]uint32, error)
	GetMapsWithCharactersFunc func() []character.MapKey
	EnterFunc                 func(transactionId uuid.UUID, f field.Model, characterId uint32)
	ExitFunc                  func(transactionId uuid.UUID, f field.Model, characterId uint32)
}

func (m *Processor) GetCharactersInMap(transactionId uuid.UUID, f field.Model) ([]uint32, error) {
	if m.GetCharactersInMapFunc != nil {
		return m.GetCharactersInMapFunc(transactionId, f)
	}
	return nil, nil
}

func (m *Processor) GetMapsWithCharacters() []character.MapKey {
	if m.GetMapsWithCharactersFunc != nil {
		return m.GetMapsWithCharactersFunc()
	}
	return nil
}

func (m *Processor) Enter(transactionId uuid.UUID, f field.Model, characterId uint32) {
	if m.EnterFunc != nil {
		m.EnterFunc(transactionId, f, characterId)
	}
}

func (m *Processor) Exit(transactionId uuid.UUID, f field.Model, characterId uint32) {
	if m.ExitFunc != nil {
		m.ExitFunc(transactionId, f, characterId)
	}
}
