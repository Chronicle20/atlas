package mock

import (
	"atlas-messengers/messenger"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
)

type ProcessorMock struct {
	CreateFunc               func(transactionID uuid.UUID, characterId uint32) (messenger.Model, error)
	CreateAndEmitFunc        func(input messenger.CreateInput) (messenger.Model, error)
	JoinFunc                 func(transactionID uuid.UUID, messengerId uint32, characterId uint32) (messenger.Model, error)
	JoinAndEmitFunc          func(input messenger.JoinInput) (messenger.Model, error)
	LeaveFunc                func(transactionID uuid.UUID, messengerId uint32, characterId uint32) (messenger.Model, error)
	LeaveAndEmitFunc         func(input messenger.LeaveInput) (messenger.Model, error)
	RequestInviteFunc        func(transactionID uuid.UUID, actorId uint32, characterId uint32) error
	RequestInviteAndEmitFunc func(input messenger.RequestInviteInput) error
	GetByIdFunc              func(messengerId uint32) (messenger.Model, error)
	GetSliceFunc             func(filters ...model.Filter[messenger.Model]) ([]messenger.Model, error)
}

var _ messenger.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) Create(transactionID uuid.UUID, characterId uint32) (messenger.Model, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(transactionID, characterId)
	}
	return messenger.Model{}, nil
}

func (m *ProcessorMock) CreateAndEmit(input messenger.CreateInput) (messenger.Model, error) {
	if m.CreateAndEmitFunc != nil {
		return m.CreateAndEmitFunc(input)
	}
	return messenger.Model{}, nil
}

func (m *ProcessorMock) Join(transactionID uuid.UUID, messengerId uint32, characterId uint32) (messenger.Model, error) {
	if m.JoinFunc != nil {
		return m.JoinFunc(transactionID, messengerId, characterId)
	}
	return messenger.Model{}, nil
}

func (m *ProcessorMock) JoinAndEmit(input messenger.JoinInput) (messenger.Model, error) {
	if m.JoinAndEmitFunc != nil {
		return m.JoinAndEmitFunc(input)
	}
	return messenger.Model{}, nil
}

func (m *ProcessorMock) Leave(transactionID uuid.UUID, messengerId uint32, characterId uint32) (messenger.Model, error) {
	if m.LeaveFunc != nil {
		return m.LeaveFunc(transactionID, messengerId, characterId)
	}
	return messenger.Model{}, nil
}

func (m *ProcessorMock) LeaveAndEmit(input messenger.LeaveInput) (messenger.Model, error) {
	if m.LeaveAndEmitFunc != nil {
		return m.LeaveAndEmitFunc(input)
	}
	return messenger.Model{}, nil
}

func (m *ProcessorMock) RequestInvite(transactionID uuid.UUID, actorId uint32, characterId uint32) error {
	if m.RequestInviteFunc != nil {
		return m.RequestInviteFunc(transactionID, actorId, characterId)
	}
	return nil
}

func (m *ProcessorMock) RequestInviteAndEmit(input messenger.RequestInviteInput) error {
	if m.RequestInviteAndEmitFunc != nil {
		return m.RequestInviteAndEmitFunc(input)
	}
	return nil
}

func (m *ProcessorMock) GetById(messengerId uint32) (messenger.Model, error) {
	if m.GetByIdFunc != nil {
		return m.GetByIdFunc(messengerId)
	}
	return messenger.Model{}, nil
}

func (m *ProcessorMock) GetSlice(filters ...model.Filter[messenger.Model]) ([]messenger.Model, error) {
	if m.GetSliceFunc != nil {
		return m.GetSliceFunc(filters...)
	}
	return nil, nil
}
