package mock

import (
	"atlas-messengers/invite"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type ProcessorMock struct {
	CreateFunc func(transactionID uuid.UUID, actorId uint32, worldId world.Id, messengerId uint32, targetId uint32) error
}

var _ invite.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) Create(transactionID uuid.UUID, actorId uint32, worldId world.Id, messengerId uint32, targetId uint32) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(transactionID, actorId, worldId, messengerId, targetId)
	}
	return nil
}
