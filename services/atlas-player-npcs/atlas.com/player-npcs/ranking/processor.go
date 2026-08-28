package ranking

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	// GetByCharacterId reads a character's computed ranking for worldId.
	// A character with no computed ranking (404) is not an error: the
	// zero-value Model is returned so deploy can still proceed (a newly
	// leveled character may not have a rank yet).
	GetByCharacterId(characterId uint32, worldId world.Id) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) GetByCharacterId(characterId uint32, worldId world.Id) (Model, error) {
	rm, err := requestById(p.ctx, characterId, worldId)(p.l, p.ctx)
	if err != nil {
		if errors.Is(err, requests.ErrNotFound) {
			return Model{}, nil
		}
		return Model{}, err
	}
	return Extract(rm)
}
