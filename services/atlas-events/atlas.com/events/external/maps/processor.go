package maps

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	CharacterIdsInMap(f field.Model) ([]uint32, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

// CharacterIdsInMap fetches every character currently in one map instance.
// The upstream list is paginated (task-117), so this drains every page
// rather than fetching just the first -- a truncated list here means the
// evaluation path silently treats some passengers as absent, and an
// unreachable atlas-maps must surface as an error to retry, not as an
// empty (falsely "nobody aboard") result.
func (p *ProcessorImpl) CharacterIdsInMap(f field.Model) ([]uint32, error) {
	return requests.DrainProvider[RestModel, uint32](p.l, p.ctx)(charactersInMapUrl(f), 250, Extract, model.Filters[uint32]())()
}
