package character

import (
	"context"

	"github.com/sirupsen/logrus"
)

// Processor is atlas-maps' read-only client for atlas-character. Only the
// minimum surface needed by MistTickTask is exposed.
type Processor interface {
	// Position returns the (x, y) world coordinates of the character with
	// the given id. Errors propagate from the underlying REST call (e.g.
	// requests.ErrNotFound when the character does not exist).
	Position(characterId uint32) (int16, int16, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor constructs a Processor scoped to the supplied tenant context.
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

// Position fetches the character resource and projects (x, y) out of it.
func (p *ProcessorImpl) Position(characterId uint32) (int16, int16, error) {
	rm, err := requestById(characterId)(p.l, p.ctx)
	if err != nil {
		return 0, 0, err
	}
	return rm.X, rm.Y, nil
}
