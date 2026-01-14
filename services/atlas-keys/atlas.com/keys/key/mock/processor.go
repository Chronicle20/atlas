package mock

import (
	"atlas-keys/key"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
)

// ProcessorMock provides a mock implementation of key.Processor for testing.
type ProcessorMock struct {
	ByCharacterIdProviderFunc func(characterId uint32) model.Provider[[]key.Model]
	GetByCharacterIdFunc      func(characterId uint32) ([]key.Model, error)
	ResetFunc                 func(transactionId uuid.UUID, characterId uint32) error
	CreateDefaultFunc         func(transactionId uuid.UUID, characterId uint32) error
	DeleteFunc                func(transactionId uuid.UUID, characterId uint32) error
	ChangeKeyFunc             func(transactionId uuid.UUID, characterId uint32, k int32, theType int8, action int32) error
}

func (m *ProcessorMock) ByCharacterIdProvider(characterId uint32) model.Provider[[]key.Model] {
	if m.ByCharacterIdProviderFunc != nil {
		return m.ByCharacterIdProviderFunc(characterId)
	}
	return func() ([]key.Model, error) {
		return []key.Model{}, nil
	}
}

func (m *ProcessorMock) GetByCharacterId(characterId uint32) ([]key.Model, error) {
	if m.GetByCharacterIdFunc != nil {
		return m.GetByCharacterIdFunc(characterId)
	}
	return []key.Model{}, nil
}

func (m *ProcessorMock) Reset(transactionId uuid.UUID, characterId uint32) error {
	if m.ResetFunc != nil {
		return m.ResetFunc(transactionId, characterId)
	}
	return nil
}

func (m *ProcessorMock) CreateDefault(transactionId uuid.UUID, characterId uint32) error {
	if m.CreateDefaultFunc != nil {
		return m.CreateDefaultFunc(transactionId, characterId)
	}
	return nil
}

func (m *ProcessorMock) Delete(transactionId uuid.UUID, characterId uint32) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(transactionId, characterId)
	}
	return nil
}

func (m *ProcessorMock) ChangeKey(transactionId uuid.UUID, characterId uint32, k int32, theType int8, action int32) error {
	if m.ChangeKeyFunc != nil {
		return m.ChangeKeyFunc(transactionId, characterId, k, theType, action)
	}
	return nil
}
