package skill

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	GetByCharacterId(characterId uint32) ([]Model, error)
	ByCharacterIdProvider(characterId uint32) model.Provider[[]Model]
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
	}
	return p
}

var _ Processor = (*ProcessorImpl)(nil)

// ByCharacterIdProvider fetches every skill for a character. The upstream
// atlas-skills list is now paginated (task-117); GetLevel (a client-side
// scan over the full result) needs the complete set, so this drains every
// page rather than fetching just the first.
func (p *ProcessorImpl) ByCharacterIdProvider(characterId uint32) model.Provider[[]Model] {
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(characterSkillsUrl(characterId), 250, Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetByCharacterId(characterId uint32) ([]Model, error) {
	return p.ByCharacterIdProvider(characterId)()
}

func GetLevel(skills []Model, id skill.Id) byte {
	for _, s := range skills {
		if s.Id() == id {
			return s.Level()
		}
	}
	return 0
}
