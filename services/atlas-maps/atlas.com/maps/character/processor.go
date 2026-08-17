package character

import (
	"context"

	"github.com/sirupsen/logrus"
)

// Processor is atlas-maps' read-only client for atlas-character. Only the
// minimum surface needed by MistTickTask is exposed.
type Processor interface {
	// Snapshot returns the (x, y) world coordinates and current HP of the
	// character with the given id. HP is carried alongside position so the
	// mist tick can skip dead characters without a second REST call.
	// Errors propagate from the underlying REST call (e.g.
	// requests.ErrNotFound when the character does not exist).
	Snapshot(characterId uint32) (int16, int16, uint16, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor constructs a Processor scoped to the supplied tenant context.
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

// Snapshot fetches the character resource and projects (x, y, hp) out of it.
func (p *ProcessorImpl) Snapshot(characterId uint32) (int16, int16, uint16, error) {
	rm, err := requestById(p.ctx, characterId)(p.l, p.ctx)
	if err != nil {
		return 0, 0, 0, err
	}
	return rm.X, rm.Y, rm.Hp, nil
}
