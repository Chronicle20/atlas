package mock

import (
	"atlas-monster-death/system_message"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
)

type ProcessorMock struct {
	ShowHintFunc func(transactionId uuid.UUID, ch channel.Model, characterId uint32, hint string, width uint16, height uint16) error
}

var _ system_message.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) ShowHint(transactionId uuid.UUID, ch channel.Model, characterId uint32, hint string, width uint16, height uint16) error {
	if m.ShowHintFunc != nil {
		return m.ShowHintFunc(transactionId, ch, characterId, hint, width, height)
	}
	return nil
}
