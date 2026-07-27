package mock

import (
	"atlas-rps/configuration"
	"atlas-rps/game"

	"github.com/google/uuid"
)

// ProcessorMock is a test double for configuration.Processor. The GetLadderFunc
// field is used when set; otherwise the method returns a zero Ladder and a nil
// error.
type ProcessorMock struct {
	GetLadderFunc func(tenantId uuid.UUID) (game.Ladder, error)
}

var _ configuration.Processor = (*ProcessorMock)(nil)

func (m *ProcessorMock) GetLadder(tenantId uuid.UUID) (game.Ladder, error) {
	if m.GetLadderFunc != nil {
		return m.GetLadderFunc(tenantId)
	}
	return game.Ladder{}, nil
}
