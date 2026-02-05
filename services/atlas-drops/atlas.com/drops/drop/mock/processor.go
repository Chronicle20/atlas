package mock

import (
	"atlas-drops/drop"
	"atlas-drops/kafka/message"

	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
)

type ProcessorMock struct {
	SpawnFunc                    func(mb *message.Buffer) func(mb *drop.ModelBuilder) (drop.Model, error)
	SpawnAndEmitFunc             func(mb *drop.ModelBuilder) (drop.Model, error)
	SpawnForCharacterFunc        func(mb *message.Buffer) func(mb *drop.ModelBuilder) (drop.Model, error)
	SpawnForCharacterAndEmitFunc func(mb *drop.ModelBuilder) (drop.Model, error)
	ReserveFunc                  func(mb *message.Buffer) func(transactionId uuid.UUID, field field.Model, dropId uint32, characterId uint32, petSlot int8) (drop.Model, error)
	ReserveAndEmitFunc           func(transactionId uuid.UUID, field field.Model, dropId uint32, characterId uint32, petSlot int8) (drop.Model, error)
	CancelReservationFunc        func(mb *message.Buffer) func(transactionId uuid.UUID, field field.Model, dropId uint32, characterId uint32) error
	CancelReservationAndEmitFunc func(transactionId uuid.UUID, field field.Model, dropId uint32, characterId uint32) error
	GatherFunc                   func(mb *message.Buffer) func(transactionId uuid.UUID, field field.Model, dropId uint32, characterId uint32) (drop.Model, error)
	GatherAndEmitFunc            func(transactionId uuid.UUID, field field.Model, dropId uint32, characterId uint32) (drop.Model, error)
	ExpireFunc                   func(mb *message.Buffer) model.Operator[drop.Model]
	ExpireAndEmitFunc            func(m drop.Model) error
	GetByIdFunc                  func(dropId uint32) (drop.Model, error)
	GetForMapFunc                func(field field.Model) ([]drop.Model, error)
	ByIdProviderFunc             func(dropId uint32) model.Provider[drop.Model]
	ForMapProviderFunc           func(field field.Model) model.Provider[[]drop.Model]
}

func (m *ProcessorMock) Spawn(mb *message.Buffer) func(mb *drop.ModelBuilder) (drop.Model, error) {
	if m.SpawnFunc != nil {
		return m.SpawnFunc(mb)
	}
	return func(mb *drop.ModelBuilder) (drop.Model, error) {
		return drop.Model{}, nil
	}
}

func (m *ProcessorMock) SpawnAndEmit(mb *drop.ModelBuilder) (drop.Model, error) {
	if m.SpawnAndEmitFunc != nil {
		return m.SpawnAndEmitFunc(mb)
	}
	return drop.Model{}, nil
}

func (m *ProcessorMock) SpawnForCharacter(mb *message.Buffer) func(mb *drop.ModelBuilder) (drop.Model, error) {
	if m.SpawnForCharacterFunc != nil {
		return m.SpawnForCharacterFunc(mb)
	}
	return func(mb *drop.ModelBuilder) (drop.Model, error) {
		return drop.Model{}, nil
	}
}

func (m *ProcessorMock) SpawnForCharacterAndEmit(mb *drop.ModelBuilder) (drop.Model, error) {
	if m.SpawnForCharacterAndEmitFunc != nil {
		return m.SpawnForCharacterAndEmitFunc(mb)
	}
	return drop.Model{}, nil
}

func (m *ProcessorMock) Reserve(mb *message.Buffer) func(transactionId uuid.UUID, field field.Model, dropId uint32, characterId uint32, petSlot int8) (drop.Model, error) {
	if m.ReserveFunc != nil {
		return m.ReserveFunc(mb)
	}
	return func(transactionId uuid.UUID, field field.Model, dropId uint32, characterId uint32, petSlot int8) (drop.Model, error) {
		return drop.Model{}, nil
	}
}

func (m *ProcessorMock) ReserveAndEmit(transactionId uuid.UUID, field field.Model, dropId uint32, characterId uint32, petSlot int8) (drop.Model, error) {
	if m.ReserveAndEmitFunc != nil {
		return m.ReserveAndEmitFunc(transactionId, field, dropId, characterId, petSlot)
	}
	return drop.Model{}, nil
}

func (m *ProcessorMock) CancelReservation(mb *message.Buffer) func(transactionId uuid.UUID, field field.Model, dropId uint32, characterId uint32) error {
	if m.CancelReservationFunc != nil {
		return m.CancelReservationFunc(mb)
	}
	return func(transactionId uuid.UUID, field field.Model, dropId uint32, characterId uint32) error {
		return nil
	}
}

func (m *ProcessorMock) CancelReservationAndEmit(transactionId uuid.UUID, field field.Model, dropId uint32, characterId uint32) error {
	if m.CancelReservationAndEmitFunc != nil {
		return m.CancelReservationAndEmitFunc(transactionId, field, dropId, characterId)
	}
	return nil
}

func (m *ProcessorMock) Gather(mb *message.Buffer) func(transactionId uuid.UUID, field field.Model, dropId uint32, characterId uint32) (drop.Model, error) {
	if m.GatherFunc != nil {
		return m.GatherFunc(mb)
	}
	return func(transactionId uuid.UUID, field field.Model, dropId uint32, characterId uint32) (drop.Model, error) {
		return drop.Model{}, nil
	}
}

func (m *ProcessorMock) GatherAndEmit(transactionId uuid.UUID, field field.Model, dropId uint32, characterId uint32) (drop.Model, error) {
	if m.GatherAndEmitFunc != nil {
		return m.GatherAndEmitFunc(transactionId, field, dropId, characterId)
	}
	return drop.Model{}, nil
}

func (m *ProcessorMock) Expire(mb *message.Buffer) model.Operator[drop.Model] {
	if m.ExpireFunc != nil {
		return m.ExpireFunc(mb)
	}
	return func(d drop.Model) error {
		return nil
	}
}

func (m *ProcessorMock) ExpireAndEmit(d drop.Model) error {
	if m.ExpireAndEmitFunc != nil {
		return m.ExpireAndEmitFunc(d)
	}
	return nil
}

func (m *ProcessorMock) GetById(dropId uint32) (drop.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(dropId)
	}
	return drop.Model{}, nil
}

func (m *ProcessorMock) GetForMap(field field.Model) ([]drop.Model, error) {
	if m.GetForMapFunc != nil {
		return m.GetForMapFunc(field)
	}
	return []drop.Model{}, nil
}

func (m *ProcessorMock) ByIdProvider(dropId uint32) model.Provider[drop.Model] {
	if m.ByIdProviderFunc != nil {
		return m.ByIdProviderFunc(dropId)
	}
	return model.FixedProvider(drop.Model{})
}

func (m *ProcessorMock) ForMapProvider(field field.Model) model.Provider[[]drop.Model] {
	if m.ForMapProviderFunc != nil {
		return m.ForMapProviderFunc(field)
	}
	return model.FixedProvider([]drop.Model{})
}
