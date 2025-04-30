package shops

import (
	"atlas-npc/commodities"
	"context"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	GetByNpcId(npcId uint32) (Model, error)
	ByNpcIdProvider(npcId uint32) model.Provider[Model]
}

type ProcessorImpl struct {
	l            logrus.FieldLogger
	ctx          context.Context
	db           *gorm.DB
	t            tenant.Model
	GetByNpcIdFn func(npcId uint32) (Model, error)
	cp           commodities.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
		cp:  commodities.NewProcessor(l, ctx, db),
	}
	p.GetByNpcIdFn = model.CollapseProvider(p.ByNpcIdProvider)
	return p
}

func (p *ProcessorImpl) GetByNpcId(npcId uint32) (Model, error) {
	return p.GetByNpcIdFn(npcId)
}

func (p *ProcessorImpl) ByNpcIdProvider(npcId uint32) model.Provider[Model] {
	cms, err := p.cp.GetByNpcId(npcId)
	if err != nil {
		return model.ErrorProvider[Model](err)
	}
	return model.FixedProvider(NewBuilder(npcId).SetCommodities(cms).Build())
}
