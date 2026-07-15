package mock

import (
	"atlas-storage/kafka/message"
	"atlas-storage/kafka/message/compartment"
	"atlas-storage/storage"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/google/uuid"
)

type ProcessorMock struct {
	GetOrCreateStorageFunc            func(worldId world.Id, accountId uint32) (storage.Model, error)
	GetStorageByWorldAndAccountIdFunc func(worldId world.Id, accountId uint32) (storage.Model, error)
	CreateStorageFunc                 func(worldId world.Id, accountId uint32) (storage.Model, error)
	DepositFunc                       func(worldId world.Id, accountId uint32, body message.DepositBody) (uint32, error)
	DepositAndEmitFunc                func(transactionId uuid.UUID, worldId world.Id, accountId uint32, body message.DepositBody) (uint32, error)
	WithdrawFunc                      func(body message.WithdrawBody) error
	WithdrawAndEmitFunc               func(transactionId uuid.UUID, worldId world.Id, accountId uint32, body message.WithdrawBody) error
	UpdateMesosFunc                   func(worldId world.Id, accountId uint32, body message.UpdateMesosBody) error
	UpdateMesosAndEmitFunc            func(transactionId uuid.UUID, worldId world.Id, accountId uint32, body message.UpdateMesosBody) error
	DepositRollbackFunc               func(body message.DepositRollbackBody) error
	AcceptFunc                        func(worldId world.Id, accountId uint32, body compartment.AcceptCommandBody) (uint32, int16, error)
	AcceptAndEmitFunc                 func(worldId world.Id, accountId uint32, characterId uint32, body compartment.AcceptCommandBody) error
	ReleaseFunc                       func(body compartment.ReleaseCommandBody) error
	ReleaseAndEmitFunc                func(worldId world.Id, accountId uint32, characterId uint32, body compartment.ReleaseCommandBody) error
	MergeAndSortFunc                  func(worldId world.Id, accountId uint32) error
	ArrangeAndEmitFunc                func(transactionId uuid.UUID, worldId world.Id, accountId uint32) error
	EmitProjectionCreatedEventFunc    func(characterId uint32, accountId uint32, ch channel.Model, npcId uint32) error
	ExpireAndEmitFunc                 func(transactionId uuid.UUID, worldId world.Id, accountId uint32, assetId uint32, isCash bool, replaceItemId uint32, replaceMessage string) error
	DeleteByAccountIdFunc             func(accountId uint32) error
	EmitProjectionDestroyedEventFunc  func(characterId uint32, accountId uint32, worldId world.Id) error
}

var _ storage.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetOrCreateStorage(worldId world.Id, accountId uint32) (storage.Model, error) {
	if m.GetOrCreateStorageFunc != nil {
		return m.GetOrCreateStorageFunc(worldId, accountId)
	}
	return storage.Model{}, nil
}

func (m *ProcessorMock) GetStorageByWorldAndAccountId(worldId world.Id, accountId uint32) (storage.Model, error) {
	if m.GetStorageByWorldAndAccountIdFunc != nil {
		return m.GetStorageByWorldAndAccountIdFunc(worldId, accountId)
	}
	return storage.Model{}, nil
}

func (m *ProcessorMock) CreateStorage(worldId world.Id, accountId uint32) (storage.Model, error) {
	if m.CreateStorageFunc != nil {
		return m.CreateStorageFunc(worldId, accountId)
	}
	return storage.Model{}, nil
}

func (m *ProcessorMock) Deposit(worldId world.Id, accountId uint32, body message.DepositBody) (uint32, error) {
	if m.DepositFunc != nil {
		return m.DepositFunc(worldId, accountId, body)
	}
	return 0, nil
}

func (m *ProcessorMock) DepositAndEmit(transactionId uuid.UUID, worldId world.Id, accountId uint32, body message.DepositBody) (uint32, error) {
	if m.DepositAndEmitFunc != nil {
		return m.DepositAndEmitFunc(transactionId, worldId, accountId, body)
	}
	return 0, nil
}

func (m *ProcessorMock) Withdraw(body message.WithdrawBody) error {
	if m.WithdrawFunc != nil {
		return m.WithdrawFunc(body)
	}
	return nil
}

func (m *ProcessorMock) WithdrawAndEmit(transactionId uuid.UUID, worldId world.Id, accountId uint32, body message.WithdrawBody) error {
	if m.WithdrawAndEmitFunc != nil {
		return m.WithdrawAndEmitFunc(transactionId, worldId, accountId, body)
	}
	return nil
}

func (m *ProcessorMock) UpdateMesos(worldId world.Id, accountId uint32, body message.UpdateMesosBody) error {
	if m.UpdateMesosFunc != nil {
		return m.UpdateMesosFunc(worldId, accountId, body)
	}
	return nil
}

func (m *ProcessorMock) UpdateMesosAndEmit(transactionId uuid.UUID, worldId world.Id, accountId uint32, body message.UpdateMesosBody) error {
	if m.UpdateMesosAndEmitFunc != nil {
		return m.UpdateMesosAndEmitFunc(transactionId, worldId, accountId, body)
	}
	return nil
}

func (m *ProcessorMock) DepositRollback(body message.DepositRollbackBody) error {
	if m.DepositRollbackFunc != nil {
		return m.DepositRollbackFunc(body)
	}
	return nil
}

func (m *ProcessorMock) Accept(worldId world.Id, accountId uint32, body compartment.AcceptCommandBody) (uint32, int16, error) {
	if m.AcceptFunc != nil {
		return m.AcceptFunc(worldId, accountId, body)
	}
	return 0, 0, nil
}

func (m *ProcessorMock) AcceptAndEmit(worldId world.Id, accountId uint32, characterId uint32, body compartment.AcceptCommandBody) error {
	if m.AcceptAndEmitFunc != nil {
		return m.AcceptAndEmitFunc(worldId, accountId, characterId, body)
	}
	return nil
}

func (m *ProcessorMock) Release(body compartment.ReleaseCommandBody) error {
	if m.ReleaseFunc != nil {
		return m.ReleaseFunc(body)
	}
	return nil
}

func (m *ProcessorMock) ReleaseAndEmit(worldId world.Id, accountId uint32, characterId uint32, body compartment.ReleaseCommandBody) error {
	if m.ReleaseAndEmitFunc != nil {
		return m.ReleaseAndEmitFunc(worldId, accountId, characterId, body)
	}
	return nil
}

func (m *ProcessorMock) MergeAndSort(worldId world.Id, accountId uint32) error {
	if m.MergeAndSortFunc != nil {
		return m.MergeAndSortFunc(worldId, accountId)
	}
	return nil
}

func (m *ProcessorMock) ArrangeAndEmit(transactionId uuid.UUID, worldId world.Id, accountId uint32) error {
	if m.ArrangeAndEmitFunc != nil {
		return m.ArrangeAndEmitFunc(transactionId, worldId, accountId)
	}
	return nil
}

func (m *ProcessorMock) EmitProjectionCreatedEvent(characterId uint32, accountId uint32, ch channel.Model, npcId uint32) error {
	if m.EmitProjectionCreatedEventFunc != nil {
		return m.EmitProjectionCreatedEventFunc(characterId, accountId, ch, npcId)
	}
	return nil
}

func (m *ProcessorMock) ExpireAndEmit(transactionId uuid.UUID, worldId world.Id, accountId uint32, assetId uint32, isCash bool, replaceItemId uint32, replaceMessage string) error {
	if m.ExpireAndEmitFunc != nil {
		return m.ExpireAndEmitFunc(transactionId, worldId, accountId, assetId, isCash, replaceItemId, replaceMessage)
	}
	return nil
}

func (m *ProcessorMock) DeleteByAccountId(accountId uint32) error {
	if m.DeleteByAccountIdFunc != nil {
		return m.DeleteByAccountIdFunc(accountId)
	}
	return nil
}

func (m *ProcessorMock) EmitProjectionDestroyedEvent(characterId uint32, accountId uint32, worldId world.Id) error {
	if m.EmitProjectionDestroyedEventFunc != nil {
		return m.EmitProjectionDestroyedEventFunc(characterId, accountId, worldId)
	}
	return nil
}
