package mock

import (
	"atlas-channel/invite"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

type ProcessorMock struct {
	AcceptFunc func(actorId uint32, worldId world.Id, inviteType string, referenceId uint32) error
	RejectFunc func(actorId uint32, worldId world.Id, inviteType string, originatorId uint32) error
}

var _ invite.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) Accept(actorId uint32, worldId world.Id, inviteType string, referenceId uint32) error {
	if m.AcceptFunc != nil {
		return m.AcceptFunc(actorId, worldId, inviteType, referenceId)
	}
	return nil
}

func (m *ProcessorMock) Reject(actorId uint32, worldId world.Id, inviteType string, originatorId uint32) error {
	if m.RejectFunc != nil {
		return m.RejectFunc(actorId, worldId, inviteType, originatorId)
	}
	return nil
}
