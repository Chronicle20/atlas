package chalkboard

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

// Processor is the chalkboard REST client used by the mini-game validation
// ladder. HasOpen reports whether the character has an open chalkboard.
type Processor interface {
	HasOpen(characterId uint32) (bool, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// HasOpen fetches the character's chalkboard. A 404 (ErrNotFound) means there
// is genuinely no open chalkboard, so the check does not block the command. Any
// OTHER error (chalkboards service down/erroring) is propagated rather than
// silently fail-opened (EXT-03) — the caller decides how to handle it, instead
// of letting a transient outage wave the CREATE/VISIT chalkboard gate through.
func (p *ProcessorImpl) HasOpen(characterId uint32) (bool, error) {
	_, err := requestById(characterId)(p.l, p.ctx)
	if err != nil {
		if errors.Is(err, requests.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
