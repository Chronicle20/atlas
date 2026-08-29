package skill

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	ByCharacterIdProvider(characterId uint32) model.Provider[[]Model]
	// GetByCharacterId returns every skill the character has learned. The
	// caller (evaluating a Maker recipe's reqSkillLevel) filters by skill id
	// itself; a single-fetch (non-draining) implementation would silently
	// drop skills on page 2 and beyond, including the Maker skill.
	GetByCharacterId(characterId uint32) ([]Model, error)
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

// ByCharacterIdProvider fetches every skill for a character. The upstream
// atlas-skills list endpoint is paginated (task-117); this drains every page
// rather than fetching just the first.
func (p *ProcessorImpl) ByCharacterIdProvider(characterId uint32) model.Provider[[]Model] {
	url, err := characterSkillsUrl(p.ctx, characterId)
	if err != nil {
		return model.ErrorProvider[[]Model](err)
	}
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(url, 250, Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) ([]Model, error) {
	return p.ByCharacterIdProvider(characterId)()
}
