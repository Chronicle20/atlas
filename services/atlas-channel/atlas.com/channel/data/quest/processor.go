package quest

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	GetById(questId uint32) (Model, error)
	GetAll() ([]Model, error)
	GetAutoStart() ([]Model, error)
	ByIdProvider(questId uint32) model.Provider[Model]
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

func (p *ProcessorImpl) ByIdProvider(questId uint32) model.Provider[Model] {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(questId), Extract)
}

func (p *ProcessorImpl) GetById(questId uint32) (Model, error) {
	return p.ByIdProvider(questId)()
}

// GetAll fetches the complete set of quest definitions. atlas-data's GET
// /data/quests is now paginated (task-117), so this drains every page
// rather than fetching one.
func (p *ProcessorImpl) GetAll() ([]Model, error) {
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(allQuestsUrl(), 250, Extract, model.Filters[Model]())()
}

// GetAutoStart fetches the complete set of auto-start quest definitions.
// atlas-data's GET /data/quests/auto-start is now paginated (task-117), so
// this drains every page rather than fetching one.
func (p *ProcessorImpl) GetAutoStart() ([]Model, error) {
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(autoStartQuestsUrl(), 250, Extract, model.Filters[Model]())()
}
