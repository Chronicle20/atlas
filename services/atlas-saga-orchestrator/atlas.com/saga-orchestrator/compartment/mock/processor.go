package mock

import (
	"atlas-saga-orchestrator/compartment"
	"time"

	"github.com/google/uuid"
)

// ProcessorMock is a mock implementation of the compartment.Processor interface
type ProcessorMock struct {
	RequestCreateItemFunc          func(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32, expiration time.Time) error
	RequestDestroyItemFunc         func(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32, removeAll bool) error
	RequestEquipAssetFunc          func(transactionId uuid.UUID, characterId uint32, inventoryType byte, source int16, destination int16) error
	RequestUnequipAssetFunc        func(transactionId uuid.UUID, characterId uint32, inventoryType byte, source int16, destination int16) error
	RequestCreateAndEquipAssetFunc func(transactionId uuid.UUID, payload compartment.CreateAndEquipAssetPayload) error
	RequestAcceptAssetFunc         func(transactionId uuid.UUID, characterId uint32, inventoryType byte, templateId uint32, referenceId uint32, referenceType string, referenceData []byte, quantity uint32) error
	RequestReleaseAssetFunc        func(transactionId uuid.UUID, characterId uint32, inventoryType byte, assetId uint32, quantity uint32) error
}

// RequestCreateItem is a mock implementation of the compartment.Processor.RequestCreateItem method
func (m *ProcessorMock) RequestCreateItem(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32, expiration time.Time) error {
	if m.RequestCreateItemFunc != nil {
		return m.RequestCreateItemFunc(transactionId, characterId, templateId, quantity, expiration)
	}
	return nil
}

// RequestDestroyItem is a mock implementation of the compartment.Processor.RequestDestroyItem method
func (m *ProcessorMock) RequestDestroyItem(transactionId uuid.UUID, characterId uint32, templateId uint32, quantity uint32, removeAll bool) error {
	if m.RequestDestroyItemFunc != nil {
		return m.RequestDestroyItemFunc(transactionId, characterId, templateId, quantity, removeAll)
	}
	return nil
}

// RequestEquipAsset is a mock implementation of the compartment.Processor.RequestEquipAsset method
func (m *ProcessorMock) RequestEquipAsset(transactionId uuid.UUID, characterId uint32, inventoryType byte, source int16, destination int16) error {
	if m.RequestEquipAssetFunc != nil {
		return m.RequestEquipAssetFunc(transactionId, characterId, inventoryType, source, destination)
	}
	return nil
}

// RequestUnequipAsset is a mock implementation of the compartment.Processor.RequestUnequipAsset method
func (m *ProcessorMock) RequestUnequipAsset(transactionId uuid.UUID, characterId uint32, inventoryType byte, source int16, destination int16) error {
	if m.RequestUnequipAssetFunc != nil {
		return m.RequestUnequipAssetFunc(transactionId, characterId, inventoryType, source, destination)
	}
	return nil
}

// RequestCreateAndEquipAsset is a mock implementation of the compartment.Processor.RequestCreateAndEquipAsset method
func (m *ProcessorMock) RequestCreateAndEquipAsset(transactionId uuid.UUID, payload compartment.CreateAndEquipAssetPayload) error {
	if m.RequestCreateAndEquipAssetFunc != nil {
		return m.RequestCreateAndEquipAssetFunc(transactionId, payload)
	}
	return nil
}

// RequestAcceptAsset is a mock implementation of the compartment.Processor.RequestAcceptAsset method
func (m *ProcessorMock) RequestAcceptAsset(transactionId uuid.UUID, characterId uint32, inventoryType byte, templateId uint32, referenceId uint32, referenceType string, referenceData []byte, quantity uint32) error {
	if m.RequestAcceptAssetFunc != nil {
		return m.RequestAcceptAssetFunc(transactionId, characterId, inventoryType, templateId, referenceId, referenceType, referenceData, quantity)
	}
	return nil
}

// RequestReleaseAsset is a mock implementation of the compartment.Processor.RequestReleaseAsset method
func (m *ProcessorMock) RequestReleaseAsset(transactionId uuid.UUID, characterId uint32, inventoryType byte, assetId uint32, quantity uint32) error {
	if m.RequestReleaseAssetFunc != nil {
		return m.RequestReleaseAssetFunc(transactionId, characterId, inventoryType, assetId, quantity)
	}
	return nil
}
