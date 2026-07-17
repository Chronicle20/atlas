package mock

import (
	"atlas-pets/kafka/message"
	"atlas-pets/pet"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type ProcessorMock struct {
	WithFunc                                 func(opts ...pet.ProcessorOption) pet.Processor
	ByIdProviderFunc                         func(petId uint32) model.Provider[pet.Model]
	GetByIdFunc                              func(petId uint32) (pet.Model, error)
	ByOwnerProviderFunc                      func(ownerId uint32) model.Provider[[]pet.Model]
	GetByOwnerFunc                           func(ownerId uint32) ([]pet.Model, error)
	SpawnedByOwnerProviderFunc               func(ownerId uint32) model.Provider[[]pet.Model]
	HungryByOwnerProviderFunc                func(ownerId uint32) model.Provider[[]pet.Model]
	HungriestByOwnerProviderFunc             func(ownerId uint32) model.Provider[pet.Model]
	CreateAndEmitFunc                        func(i pet.Model) (pet.Model, error)
	CreateFunc                               func(mb *message.Buffer) func(i pet.Model) (pet.Model, error)
	DeleteOnRemoveAndEmitFunc                func(characterId uint32, itemId uint32, slot int16) error
	DeleteOnRemoveFunc                       func(mb *message.Buffer) func(characterId uint32) func(itemId uint32) func(slot int16) error
	DeleteForCharacterAndEmitFunc            func(characterId uint32) error
	DeleteForCharacterFunc                   func(mb *message.Buffer) func(characterId uint32) error
	DeleteFunc                               func(mb *message.Buffer) func(petId uint32) func(ownerId uint32) error
	MoveFunc                                 func(petId uint32, f field.Model, ownerId uint32, x int16, y int16, stance byte) error
	SpawnAndEmitFunc                         func(petId uint32, actorId uint32, lead bool) error
	SpawnFunc                                func(mb *message.Buffer) func(petId uint32) func(actorId uint32) func(lead bool) error
	DespawnAndEmitFunc                       func(petId uint32, actorId uint32, reason string) error
	DespawnFunc                              func(mb *message.Buffer) func(petId uint32) func(actorId uint32) func(reason string) error
	AttemptCommandAndEmitFunc                func(petId uint32, actorId uint32, commandId byte) error
	AttemptCommandFunc                       func(mb *message.Buffer) func(petId uint32) func(actorId uint32) func(commandId byte) error
	EvaluateHungerAndEmitFunc                func(ownerId uint32) error
	EvaluateHungerFunc                       func(mb *message.Buffer) func(ownerId uint32) error
	ClearPositionsFunc                       func(ownerId uint32) error
	AwardClosenessAndEmitFunc                func(petId uint32, amount uint16) error
	AwardClosenessFunc                       func(mb *message.Buffer) func(petId uint32) func(amount uint16) error
	AwardClosenessWithTransactionAndEmitFunc func(transactionId uuid.UUID, petId uint32, amount uint16) error
	AwardClosenessWithTransactionFunc        func(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32, amount uint16) error
	AwardFullnessAndEmitFunc                 func(petId uint32, amount byte) error
	AwardFullnessFunc                        func(mb *message.Buffer) func(petId uint32) func(amount byte) error
	AwardLevelAndEmitFunc                    func(petId uint32, amount byte) error
	AwardLevelFunc                           func(mb *message.Buffer) func(petId uint32) func(amount byte) error
	EvolveAndEmitFunc                        func(transactionId uuid.UUID, petId uint32) error
	EvolveFunc                               func(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32) error
	SetExcludeAndEmitFunc                    func(petId uint32, items []uint32) error
	SetExcludeFunc                           func(mb *message.Buffer) func(petId uint32) func(items []uint32) error
}

var _ pet.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) With(opts ...pet.ProcessorOption) pet.Processor {
	if m.WithFunc != nil {
		return m.WithFunc(opts...)
	}
	return m
}

func (m *ProcessorMock) ByIdProvider(petId uint32) model.Provider[pet.Model] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(petId)
	}
	return model.FixedProvider(pet.Model{})
}

func (m *ProcessorMock) GetById(petId uint32) (pet.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(petId)
	}
	return pet.Model{}, nil
}

func (m *ProcessorMock) ByOwnerProvider(ownerId uint32) model.Provider[[]pet.Model] {
	if m.ByOwnerProviderFunc != nil {
		return m.ByOwnerProviderFunc(ownerId)
	}
	return model.FixedProvider([]pet.Model{})
}

func (m *ProcessorMock) GetByOwner(ownerId uint32) ([]pet.Model, error) {
	if m.GetByOwnerFunc != nil {
		return m.GetByOwnerFunc(ownerId)
	}
	return nil, nil
}

func (m *ProcessorMock) SpawnedByOwnerProvider(ownerId uint32) model.Provider[[]pet.Model] {
	if m.SpawnedByOwnerProviderFunc != nil {
		return m.SpawnedByOwnerProviderFunc(ownerId)
	}
	return model.FixedProvider([]pet.Model{})
}

func (m *ProcessorMock) HungryByOwnerProvider(ownerId uint32) model.Provider[[]pet.Model] {
	if m.HungryByOwnerProviderFunc != nil {
		return m.HungryByOwnerProviderFunc(ownerId)
	}
	return model.FixedProvider([]pet.Model{})
}

func (m *ProcessorMock) HungriestByOwnerProvider(ownerId uint32) model.Provider[pet.Model] {
	if m.HungriestByOwnerProviderFunc != nil {
		return m.HungriestByOwnerProviderFunc(ownerId)
	}
	return model.FixedProvider(pet.Model{})
}

func (m *ProcessorMock) CreateAndEmit(i pet.Model) (pet.Model, error) {
	if m.CreateAndEmitFunc != nil {
		return m.CreateAndEmitFunc(i)
	}
	return pet.Model{}, nil
}

func (m *ProcessorMock) Create(mb *message.Buffer) func(i pet.Model) (pet.Model, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(mb)
	}
	return func(i pet.Model) (pet.Model, error) {
		return pet.Model{}, nil
	}
}

func (m *ProcessorMock) DeleteOnRemoveAndEmit(characterId uint32, itemId uint32, slot int16) error {
	if m.DeleteOnRemoveAndEmitFunc != nil {
		return m.DeleteOnRemoveAndEmitFunc(characterId, itemId, slot)
	}
	return nil
}

func (m *ProcessorMock) DeleteOnRemove(mb *message.Buffer) func(characterId uint32) func(itemId uint32) func(slot int16) error {
	if m.DeleteOnRemoveFunc != nil {
		return m.DeleteOnRemoveFunc(mb)
	}
	return func(characterId uint32) func(itemId uint32) func(slot int16) error {
		return func(itemId uint32) func(slot int16) error {
			return func(slot int16) error {
				return nil
			}
		}
	}
}

func (m *ProcessorMock) DeleteForCharacterAndEmit(characterId uint32) error {
	if m.DeleteForCharacterAndEmitFunc != nil {
		return m.DeleteForCharacterAndEmitFunc(characterId)
	}
	return nil
}

func (m *ProcessorMock) DeleteForCharacter(mb *message.Buffer) func(characterId uint32) error {
	if m.DeleteForCharacterFunc != nil {
		return m.DeleteForCharacterFunc(mb)
	}
	return func(characterId uint32) error {
		return nil
	}
}

func (m *ProcessorMock) Delete(mb *message.Buffer) func(petId uint32) func(ownerId uint32) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(mb)
	}
	return func(petId uint32) func(ownerId uint32) error {
		return func(ownerId uint32) error {
			return nil
		}
	}
}

func (m *ProcessorMock) Move(petId uint32, f field.Model, ownerId uint32, x int16, y int16, stance byte) error {
	if m.MoveFunc != nil {
		return m.MoveFunc(petId, f, ownerId, x, y, stance)
	}
	return nil
}

func (m *ProcessorMock) SpawnAndEmit(petId uint32, actorId uint32, lead bool) error {
	if m.SpawnAndEmitFunc != nil {
		return m.SpawnAndEmitFunc(petId, actorId, lead)
	}
	return nil
}

func (m *ProcessorMock) Spawn(mb *message.Buffer) func(petId uint32) func(actorId uint32) func(lead bool) error {
	if m.SpawnFunc != nil {
		return m.SpawnFunc(mb)
	}
	return func(petId uint32) func(actorId uint32) func(lead bool) error {
		return func(actorId uint32) func(lead bool) error {
			return func(lead bool) error {
				return nil
			}
		}
	}
}

func (m *ProcessorMock) DespawnAndEmit(petId uint32, actorId uint32, reason string) error {
	if m.DespawnAndEmitFunc != nil {
		return m.DespawnAndEmitFunc(petId, actorId, reason)
	}
	return nil
}

func (m *ProcessorMock) Despawn(mb *message.Buffer) func(petId uint32) func(actorId uint32) func(reason string) error {
	if m.DespawnFunc != nil {
		return m.DespawnFunc(mb)
	}
	return func(petId uint32) func(actorId uint32) func(reason string) error {
		return func(actorId uint32) func(reason string) error {
			return func(reason string) error {
				return nil
			}
		}
	}
}

func (m *ProcessorMock) AttemptCommandAndEmit(petId uint32, actorId uint32, commandId byte) error {
	if m.AttemptCommandAndEmitFunc != nil {
		return m.AttemptCommandAndEmitFunc(petId, actorId, commandId)
	}
	return nil
}

func (m *ProcessorMock) AttemptCommand(mb *message.Buffer) func(petId uint32) func(actorId uint32) func(commandId byte) error {
	if m.AttemptCommandFunc != nil {
		return m.AttemptCommandFunc(mb)
	}
	return func(petId uint32) func(actorId uint32) func(commandId byte) error {
		return func(actorId uint32) func(commandId byte) error {
			return func(commandId byte) error {
				return nil
			}
		}
	}
}

func (m *ProcessorMock) EvaluateHungerAndEmit(ownerId uint32) error {
	if m.EvaluateHungerAndEmitFunc != nil {
		return m.EvaluateHungerAndEmitFunc(ownerId)
	}
	return nil
}

func (m *ProcessorMock) EvaluateHunger(mb *message.Buffer) func(ownerId uint32) error {
	if m.EvaluateHungerFunc != nil {
		return m.EvaluateHungerFunc(mb)
	}
	return func(ownerId uint32) error {
		return nil
	}
}

func (m *ProcessorMock) ClearPositions(ownerId uint32) error {
	if m.ClearPositionsFunc != nil {
		return m.ClearPositionsFunc(ownerId)
	}
	return nil
}

func (m *ProcessorMock) AwardClosenessAndEmit(petId uint32, amount uint16) error {
	if m.AwardClosenessAndEmitFunc != nil {
		return m.AwardClosenessAndEmitFunc(petId, amount)
	}
	return nil
}

func (m *ProcessorMock) AwardCloseness(mb *message.Buffer) func(petId uint32) func(amount uint16) error {
	if m.AwardClosenessFunc != nil {
		return m.AwardClosenessFunc(mb)
	}
	return func(petId uint32) func(amount uint16) error {
		return func(amount uint16) error {
			return nil
		}
	}
}

func (m *ProcessorMock) AwardClosenessWithTransactionAndEmit(transactionId uuid.UUID, petId uint32, amount uint16) error {
	if m.AwardClosenessWithTransactionAndEmitFunc != nil {
		return m.AwardClosenessWithTransactionAndEmitFunc(transactionId, petId, amount)
	}
	return nil
}

func (m *ProcessorMock) AwardClosenessWithTransaction(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32, amount uint16) error {
	if m.AwardClosenessWithTransactionFunc != nil {
		return m.AwardClosenessWithTransactionFunc(mb)
	}
	return func(transactionId uuid.UUID, petId uint32, amount uint16) error {
		return nil
	}
}

func (m *ProcessorMock) AwardFullnessAndEmit(petId uint32, amount byte) error {
	if m.AwardFullnessAndEmitFunc != nil {
		return m.AwardFullnessAndEmitFunc(petId, amount)
	}
	return nil
}

func (m *ProcessorMock) AwardFullness(mb *message.Buffer) func(petId uint32) func(amount byte) error {
	if m.AwardFullnessFunc != nil {
		return m.AwardFullnessFunc(mb)
	}
	return func(petId uint32) func(amount byte) error {
		return func(amount byte) error {
			return nil
		}
	}
}

func (m *ProcessorMock) AwardLevelAndEmit(petId uint32, amount byte) error {
	if m.AwardLevelAndEmitFunc != nil {
		return m.AwardLevelAndEmitFunc(petId, amount)
	}
	return nil
}

func (m *ProcessorMock) AwardLevel(mb *message.Buffer) func(petId uint32) func(amount byte) error {
	if m.AwardLevelFunc != nil {
		return m.AwardLevelFunc(mb)
	}
	return func(petId uint32) func(amount byte) error {
		return func(amount byte) error {
			return nil
		}
	}
}

func (m *ProcessorMock) EvolveAndEmit(transactionId uuid.UUID, petId uint32) error {
	if m.EvolveAndEmitFunc != nil {
		return m.EvolveAndEmitFunc(transactionId, petId)
	}
	return nil
}

func (m *ProcessorMock) Evolve(mb *message.Buffer) func(transactionId uuid.UUID, petId uint32) error {
	if m.EvolveFunc != nil {
		return m.EvolveFunc(mb)
	}
	return func(transactionId uuid.UUID, petId uint32) error {
		return nil
	}
}

func (m *ProcessorMock) SetExcludeAndEmit(petId uint32, items []uint32) error {
	if m.SetExcludeAndEmitFunc != nil {
		return m.SetExcludeAndEmitFunc(petId, items)
	}
	return nil
}

func (m *ProcessorMock) SetExclude(mb *message.Buffer) func(petId uint32) func(items []uint32) error {
	if m.SetExcludeFunc != nil {
		return m.SetExcludeFunc(mb)
	}
	return func(petId uint32) func(items []uint32) error {
		return func(items []uint32) error {
			return nil
		}
	}
}
