package worldbroadcast

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// Processor exposes reads of the atlas-world broadcast-queue resource
// needed by the megaphone/Maple TV command handlers (Task 12).
type Processor interface {
	GetWaitSeconds(worldId world.Id, family string) (uint32, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

// GetWaitSeconds resolves requestQueue(worldId, family) and returns the
// queue's WaitSeconds. Any transport/decode error is returned to the
// caller, never swallowed or defaulted to 0: the handler (Task 12) rejects
// conservatively on error (design §6 "never consume-then-drop").
func (p *ProcessorImpl) GetWaitSeconds(worldId world.Id, family string) (uint32, error) {
	rm, err := requestQueue(worldId, family)(p.l, p.ctx)
	if err != nil {
		return 0, err
	}
	return rm.WaitSeconds, nil
}
