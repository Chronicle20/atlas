package mock

import (
	"github.com/google/uuid"
)

// ProcessorMock is a mock implementation of the party.Processor interface.
type ProcessorMock struct {
	RequestLeaveFunc func(transactionId uuid.UUID, characterId uint32, partyId uint32) error
}

func (m *ProcessorMock) RequestLeave(transactionId uuid.UUID, characterId uint32, partyId uint32) error {
	if m.RequestLeaveFunc != nil {
		return m.RequestLeaveFunc(transactionId, characterId, partyId)
	}
	return nil
}
