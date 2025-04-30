package commodities

import (
	"context"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	GetByNpcId(npcId uint32) ([]Model, error)
	ByNpcIdProvider(npcId uint32) model.Provider[[]Model]
}

type ProcessorImpl struct {
	l            logrus.FieldLogger
	ctx          context.Context
	db           *gorm.DB
	t            tenant.Model
	GetByNpcIdFn func(npcId uint32) ([]Model, error)
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
	}
	p.GetByNpcIdFn = model.CollapseProvider(p.ByNpcIdProvider)
	return p
}

func (p *ProcessorImpl) GetByNpcId(npcId uint32) ([]Model, error) {
	return p.GetByNpcIdFn(npcId)
}

func (p *ProcessorImpl) ByNpcIdProvider(npcId uint32) model.Provider[[]Model] {
	return model.SliceMap(Make)(getByNpcId(p.t.Id(), npcId)(p.db))(model.ParallelMap())
}
