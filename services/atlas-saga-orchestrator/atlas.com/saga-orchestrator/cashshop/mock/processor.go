package mock

import (
	"atlas-saga-orchestrator/kafka/message"

	"github.com/google/uuid"
)

// ProcessorMock is a mock implementation of the cashshop.Processor interface.
type ProcessorMock struct {
	AwardCurrencyAndEmitFunc func(transactionId uuid.UUID, accountId uint32, currencyType uint32, amount int32) error
	AwardCurrencyFunc        func(mb *message.Buffer) func(transactionId uuid.UUID, accountId uint32, currencyType uint32, amount int32) error
	AcceptAndEmitFunc        func(transactionId uuid.UUID, characterId uint32, accountId uint32, compartmentId uuid.UUID, compartmentType byte, cashId int64, templateId uint32, quantity uint32, commodityId uint32, purchasedBy uint32, flag uint16) error
	AcceptFunc               func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, accountId uint32, compartmentId uuid.UUID, compartmentType byte, cashId int64, templateId uint32, quantity uint32, commodityId uint32, purchasedBy uint32, flag uint16) error
	ReleaseAndEmitFunc       func(transactionId uuid.UUID, characterId uint32, accountId uint32, compartmentId uuid.UUID, compartmentType byte, assetId uint32, cashId int64, templateId uint32) error
	ReleaseFunc              func(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, accountId uint32, compartmentId uuid.UUID, compartmentType byte, assetId uint32, cashId int64, templateId uint32) error
}

// AwardCurrencyAndEmit is a mock implementation of the cashshop.Processor.AwardCurrencyAndEmit method
func (m *ProcessorMock) AwardCurrencyAndEmit(transactionId uuid.UUID, accountId uint32, currencyType uint32, amount int32) error {
	if m.AwardCurrencyAndEmitFunc != nil {
		return m.AwardCurrencyAndEmitFunc(transactionId, accountId, currencyType, amount)
	}
	return nil
}

// AwardCurrency is a mock implementation of the cashshop.Processor.AwardCurrency method
func (m *ProcessorMock) AwardCurrency(mb *message.Buffer) func(transactionId uuid.UUID, accountId uint32, currencyType uint32, amount int32) error {
	if m.AwardCurrencyFunc != nil {
		return m.AwardCurrencyFunc(mb)
	}
	return func(uuid.UUID, uint32, uint32, int32) error { return nil }
}

// AcceptAndEmit is a mock implementation of the cashshop.Processor.AcceptAndEmit method
func (m *ProcessorMock) AcceptAndEmit(transactionId uuid.UUID, characterId uint32, accountId uint32, compartmentId uuid.UUID, compartmentType byte, cashId int64, templateId uint32, quantity uint32, commodityId uint32, purchasedBy uint32, flag uint16) error {
	if m.AcceptAndEmitFunc != nil {
		return m.AcceptAndEmitFunc(transactionId, characterId, accountId, compartmentId, compartmentType, cashId, templateId, quantity, commodityId, purchasedBy, flag)
	}
	return nil
}

// Accept is a mock implementation of the cashshop.Processor.Accept method
func (m *ProcessorMock) Accept(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, accountId uint32, compartmentId uuid.UUID, compartmentType byte, cashId int64, templateId uint32, quantity uint32, commodityId uint32, purchasedBy uint32, flag uint16) error {
	if m.AcceptFunc != nil {
		return m.AcceptFunc(mb)
	}
	return func(uuid.UUID, uint32, uint32, uuid.UUID, byte, int64, uint32, uint32, uint32, uint32, uint16) error { return nil }
}

// ReleaseAndEmit is a mock implementation of the cashshop.Processor.ReleaseAndEmit method
func (m *ProcessorMock) ReleaseAndEmit(transactionId uuid.UUID, characterId uint32, accountId uint32, compartmentId uuid.UUID, compartmentType byte, assetId uint32, cashId int64, templateId uint32) error {
	if m.ReleaseAndEmitFunc != nil {
		return m.ReleaseAndEmitFunc(transactionId, characterId, accountId, compartmentId, compartmentType, assetId, cashId, templateId)
	}
	return nil
}

// Release is a mock implementation of the cashshop.Processor.Release method
func (m *ProcessorMock) Release(mb *message.Buffer) func(transactionId uuid.UUID, characterId uint32, accountId uint32, compartmentId uuid.UUID, compartmentType byte, assetId uint32, cashId int64, templateId uint32) error {
	if m.ReleaseFunc != nil {
		return m.ReleaseFunc(mb)
	}
	return func(uuid.UUID, uint32, uint32, uuid.UUID, byte, uint32, int64, uint32) error { return nil }
}
