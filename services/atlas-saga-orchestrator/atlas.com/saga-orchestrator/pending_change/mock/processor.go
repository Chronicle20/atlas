package mock

import (
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// ProcessorMock is a mock implementation of the pending_change.Processor interface.
type ProcessorMock struct {
	ChangeWorldFunc              func(transactionId uuid.UUID, characterId uint32, newWorldId world.Id) error
	ResolveFunc                  func(characterId uint32, id uuid.UUID, status string, reason string) error
	CheckTransferEligibilityFunc func(characterId uint32, destinationWorldId world.Id) (bool, string, error)
}

func (m *ProcessorMock) ChangeWorld(transactionId uuid.UUID, characterId uint32, newWorldId world.Id) error {
	if m.ChangeWorldFunc != nil {
		return m.ChangeWorldFunc(transactionId, characterId, newWorldId)
	}
	return nil
}

func (m *ProcessorMock) Resolve(characterId uint32, id uuid.UUID, status string, reason string) error {
	if m.ResolveFunc != nil {
		return m.ResolveFunc(characterId, id, status, reason)
	}
	return nil
}

func (m *ProcessorMock) CheckTransferEligibility(characterId uint32, destinationWorldId world.Id) (bool, string, error) {
	if m.CheckTransferEligibilityFunc != nil {
		return m.CheckTransferEligibilityFunc(characterId, destinationWorldId)
	}
	return true, "", nil
}
