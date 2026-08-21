package position

import (
	"context"

	"github.com/sirupsen/logrus"
)

// Processor is atlas-monsters' read-only client for atlas-character. Only
// the minimum surface needed by mob-skill AoE target selection is exposed.
type Processor interface {
	// GetPosition returns the character's last known world coordinates.
	// Errors propagate from the underlying REST call (e.g.
	// requests.ErrNotFound when the character does not exist).
	GetPosition(characterId uint32) (int16, int16, error)
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

// GetPosition fetches the character resource and projects (x, y) out of it.
func (p *ProcessorImpl) GetPosition(characterId uint32) (int16, int16, error) {
	rm, err := requestById(p.ctx, characterId)(p.l, p.ctx)
	if err != nil {
		return 0, 0, err
	}
	return rm.X, rm.Y, nil
}
