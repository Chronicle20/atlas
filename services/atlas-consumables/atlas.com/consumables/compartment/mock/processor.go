package mock

import (
	"atlas-consumables/compartment"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/google/uuid"
)

type ProcessorMock struct {
	RequestReserveFunc        func(transactionId uuid.UUID, characterId uint32, it inventory.Type, reserves []compartment.Reserves) error
	ConsumeItemFunc           func(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error
	DestroyItemFunc           func(characterId uint32, inventoryType inventory.Type, slot int16) error
	CancelItemReservationFunc func(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error
	RequestCreateItemFunc     func(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32, expiration time.Time) error
}

var _ compartment.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) RequestReserve(transactionId uuid.UUID, characterId uint32, it inventory.Type, reserves []compartment.Reserves) error {
	if m.RequestReserveFunc != nil {
		return m.RequestReserveFunc(transactionId, characterId, it, reserves)
	}
	return nil
}

func (m *ProcessorMock) ConsumeItem(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error {
	if m.ConsumeItemFunc != nil {
		return m.ConsumeItemFunc(characterId, inventoryType, transactionId, slot)
	}
	return nil
}

func (m *ProcessorMock) DestroyItem(characterId uint32, inventoryType inventory.Type, slot int16) error {
	if m.DestroyItemFunc != nil {
		return m.DestroyItemFunc(characterId, inventoryType, slot)
	}
	return nil
}

func (m *ProcessorMock) CancelItemReservation(characterId uint32, inventoryType inventory.Type, transactionId uuid.UUID, slot int16) error {
	if m.CancelItemReservationFunc != nil {
		return m.CancelItemReservationFunc(characterId, inventoryType, transactionId, slot)
	}
	return nil
}

func (m *ProcessorMock) RequestCreateItem(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32, expiration time.Time) error {
	if m.RequestCreateItemFunc != nil {
		return m.RequestCreateItemFunc(transactionId, characterId, templateId, quantity, expiration)
	}
	return nil
}
