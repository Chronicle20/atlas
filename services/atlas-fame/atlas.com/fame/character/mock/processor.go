package mock

import (
	"atlas-fame/character"
	"atlas-fame/kafka/message"

	"github.com/Chronicle20/atlas-constants/world"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
)

type ProcessorMock struct {
	GetByIdFunc                  func(characterId uint32) (character.Model, error)
	ByIdProviderFunc             func(characterId uint32) model.Provider[character.Model]
	RequestChangeFameFunc        func(mb *message.Buffer) func(transactionId uuid.UUID) func(characterId uint32) func(worldId world.Id) func(actorId uint32) func(amount int8) error
	RequestChangeFameAndEmitFunc func(transactionId uuid.UUID, characterId uint32, worldId world.Id, actorId uint32, amount int8) error
}

func (m *ProcessorMock) GetById(characterId uint32) (character.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(characterId)
	}
	return character.Model{}, nil
}

func (m *ProcessorMock) ByIdProvider(characterId uint32) model.Provider[character.Model] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(characterId)
	}
	return func() (character.Model, error) {
		return character.Model{}, nil
	}
}

func (m *ProcessorMock) RequestChangeFame(mb *message.Buffer) func(transactionId uuid.UUID) func(characterId uint32) func(worldId world.Id) func(actorId uint32) func(amount int8) error {
	if m.RequestChangeFameFunc != nil {
		return m.RequestChangeFameFunc(mb)
	}
	return func(transactionId uuid.UUID) func(characterId uint32) func(worldId world.Id) func(actorId uint32) func(amount int8) error {
		return func(characterId uint32) func(worldId world.Id) func(actorId uint32) func(amount int8) error {
			return func(worldId world.Id) func(actorId uint32) func(amount int8) error {
				return func(actorId uint32) func(amount int8) error {
					return func(amount int8) error {
						return nil
					}
				}
			}
		}
	}
}

func (m *ProcessorMock) RequestChangeFameAndEmit(transactionId uuid.UUID, characterId uint32, worldId world.Id, actorId uint32, amount int8) error {
	if m.RequestChangeFameAndEmitFunc != nil {
		return m.RequestChangeFameAndEmitFunc(transactionId, characterId, worldId, actorId, amount)
	}
	return nil
}
